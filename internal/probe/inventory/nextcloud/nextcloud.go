// Package nextcloud inspects a local Nextcloud installation by
// shelling out to its `occ` admin CLI. The inspector runs four
// `occ` commands and emits one Finding family per data source:
//
//   - inventory.nextcloud.app             — every enabled / disabled
//     app from `occ app:list`
//   - inventory.nextcloud.version         — `versionstring` from
//     `occ status`
//   - inventory.nextcloud.trusted_domain  — every domain from
//     `occ config:list system`
//   - inventory.nextcloud.objectstore     — every S3-style backend
//     from the same source,
//     annotated with geoip
//   - inventory.nextcloud.oidc_provider   — every IdP from
//     `occ user_oidc:provider list`
//   - inventory.nextcloud.oidc.unavailable — emitted instead when
//     user_oidc is absent
//
// On hosts without `occ` (or without a configured path) every
// query returns an error and the inspector reports unavailable
// up the inventory stack.
package nextcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// HostResolver annotates a hostname with ASN / organisation /
// country information. The same interface is implemented by
// egress.IPResolver; the inspector takes this dependency without
// importing the egress package directly to keep the inventory
// modus independent.
type HostResolver interface {
	Resolve(host string) (asn uint, organisation, country string, ok bool)
}

// SupportedNextcloudMajors lists the Nextcloud major versions
// the parser has been written against. The version probe emits a
// `supported` attribute so an operator can see at a glance
// whether their installation falls inside the contract.
var SupportedNextcloudMajors = []int{28, 29, 30}

// Nextcloud is the inspector.
type Nextcloud struct {
	OccPath string
	RunAs   string
	// Resolver enriches `inventory.nextcloud.objectstore` Findings
	// with the endpoint hostname's ASN / country. Nil resolver →
	// the geoip attributes stay absent.
	Resolver HostResolver

	// QueryFunc is the legacy hook that returns the raw
	// `occ app:list --output=json` output. When nil the
	// inspector shells out at runtime. Kept for the unit tests
	// that pin the app-list parser; new query types use the more
	// specific *FuncOverrides instead.
	QueryFunc func(ctx context.Context) (string, error)

	// Per-query overrides for unit tests. When nil, the inspector
	// shells out via `occ`. Each one returns the raw `occ` stdout.
	StatusFunc       func(ctx context.Context) (string, error)
	ConfigSystemFunc func(ctx context.Context) (string, error)
	OIDCProviderFunc func(ctx context.Context) (string, error)
	OIDCAppListFunc  func(ctx context.Context) (string, error)
}

func (Nextcloud) ID() string { return "nextcloud" }

func (n Nextcloud) Available() (bool, string) {
	if n.QueryFunc != nil {
		return true, ""
	}
	if n.OccPath == "" {
		return false, "occ path not configured"
	}
	if _, err := exec.LookPath(n.OccPath); err != nil {
		return false, "occ not callable: " + err.Error()
	}
	return true, ""
}

func (n Nextcloud) Inspect(ctx context.Context) ([]models.Finding, error) {
	var out []models.Finding

	// 1. App list — the legacy query. Failure is fatal because
	// every other query is optional surface; if app:list fails
	// the host probably has no usable `occ`.
	appsRaw, err := n.runApps(ctx)
	if err != nil {
		return nil, fmt.Errorf("nextcloud: %w", err)
	}
	apps, err := Parse(appsRaw)
	if err != nil {
		return nil, err
	}
	out = append(out, apps...)

	// 2. Version. Best-effort: a parse error here downgrades to
	// an inventory.nextcloud.version.error meta-finding so the
	// rest of the inspector still reports.
	if raw, err := n.runStatus(ctx); err == nil {
		if v, err := ParseStatus(raw); err == nil {
			out = append(out, v)
		}
	}

	// 3. Trusted domains + objectstore — both come from the same
	// `occ config:list system` payload.
	if raw, err := n.runConfigSystem(ctx); err == nil {
		td, store := ParseSystemConfig(raw)
		out = append(out, td...)
		out = append(out, n.annotateObjectstore(store)...)
	}

	// 4. OIDC providers. Falls back to the alternative-app
	// detection path when `user_oidc:provider list` errors.
	out = append(out, n.runOIDC(ctx, appsRaw)...)

	// Stable order — Findings are deduplicated and tested by
	// signature in the assessor, but a stable sort makes the
	// JSON output diffable.
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].ProbeID != out[j].ProbeID {
			return out[i].ProbeID < out[j].ProbeID
		}
		return out[i].Subject < out[j].Subject
	})
	return out, nil
}

// runApps runs `occ app:list`. Falls back to the legacy
// QueryFunc hook for tests that predate the multi-query
// inspector.
func (n Nextcloud) runApps(ctx context.Context) (string, error) {
	if n.QueryFunc != nil {
		return n.QueryFunc(ctx)
	}
	return n.runOcc(ctx, "app:list", "--output=json")
}

func (n Nextcloud) runStatus(ctx context.Context) (string, error) {
	if n.StatusFunc != nil {
		return n.StatusFunc(ctx)
	}
	return n.runOcc(ctx, "status", "--output=json")
}

func (n Nextcloud) runConfigSystem(ctx context.Context) (string, error) {
	if n.ConfigSystemFunc != nil {
		return n.ConfigSystemFunc(ctx)
	}
	return n.runOcc(ctx, "config:list", "system", "--output=json")
}

func (n Nextcloud) runOIDCProvider(ctx context.Context) (string, error) {
	if n.OIDCProviderFunc != nil {
		return n.OIDCProviderFunc(ctx)
	}
	return n.runOcc(ctx, "user_oidc:provider", "--output=json")
}

func (n Nextcloud) runOcc(ctx context.Context, args ...string) (string, error) {
	cmd := n.OccPath
	finalArgs := args
	if n.RunAs != "" {
		finalArgs = append([]string{"-u", n.RunAs, n.OccPath}, args...)
		cmd = "sudo"
	}
	out, err := exec.CommandContext(ctx, cmd, finalArgs...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// runOIDC tries the canonical `user_oidc` app first; on failure,
// looks at the app list for a known alternative
// (`oidc_login`, `social_login`) and emits an unavailable
// meta-finding naming what it found.
func (n Nextcloud) runOIDC(ctx context.Context, appsRaw string) []models.Finding {
	raw, err := n.runOIDCProvider(ctx)
	if err == nil {
		providers, parseErr := ParseOIDCProviders(raw)
		if parseErr == nil {
			return providers
		}
		// Fall through to the unavailable path with the parse
		// error preserved as the meta-finding's reason.
		return []models.Finding{oidcUnavailable("user_oidc parse error: "+parseErr.Error(), "")}
	}

	// `user_oidc:provider list` failed. Scan the app list for a
	// known alternative OIDC implementation so operators see
	// "we couldn't probe because you're on social_login, not
	// because Wanderer is broken".
	alt := detectAlternativeOIDCApp(appsRaw)
	return []models.Finding{oidcUnavailable(err.Error(), alt)}
}

func detectAlternativeOIDCApp(appsRaw string) string {
	var doc occListJSON
	if err := json.Unmarshal([]byte(appsRaw), &doc); err != nil {
		return ""
	}
	for _, alt := range []string{"oidc_login", "social_login", "user_saml"} {
		if _, ok := doc.Enabled[alt]; ok {
			return alt
		}
		if _, ok := doc.Disabled[alt]; ok {
			return alt
		}
	}
	return ""
}

func oidcUnavailable(reason, alternativeApp string) models.Finding {
	attrs := map[string]any{
		"unavailable": true,
		"reason":      reason,
	}
	if alternativeApp != "" {
		attrs["alternative_app"] = alternativeApp
	}
	return models.Finding{
		ProbeID:       "inventory.nextcloud.oidc.unavailable",
		Subject:       "user_oidc",
		Severity:      models.SeverityInfo,
		SourceModus:   models.SourceModusInventory,
		DimensionHint: models.DimensionTechnologie,
		Attributes:    attrs,
	}
}

// annotateObjectstore walks the raw objectstore findings the
// system-config parser emitted and enriches each with ASN /
// country data when a HostResolver is wired. The endpoint
// hostname is the input to Resolve; absent a host (the bucket
// uses S3's default endpoint) the finding stays unannotated.
func (n Nextcloud) annotateObjectstore(in []models.Finding) []models.Finding {
	if n.Resolver == nil || len(in) == 0 {
		return in
	}
	out := make([]models.Finding, 0, len(in))
	for _, f := range in {
		host, _ := f.Attributes["endpoint_host"].(string)
		if host != "" {
			if asn, org, country, ok := n.Resolver.Resolve(host); ok {
				if f.Attributes == nil {
					f.Attributes = map[string]any{}
				}
				f.Attributes["asn"] = asn
				f.Attributes["asn_organisation"] = org
				f.Attributes["country"] = country
			}
		}
		out = append(out, f)
	}
	return out
}

// occListJSON is the `{enabled: {}, disabled: {}}` shape
// `occ app:list` emits as of Nextcloud 28..30.
type occListJSON struct {
	Enabled  map[string]string `json:"enabled"`
	Disabled map[string]string `json:"disabled"`
}

// Parse converts `occ app:list --output=json` output to Findings.
func Parse(raw string) ([]models.Finding, error) {
	var doc occListJSON
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, fmt.Errorf("nextcloud parse: %w", err)
	}
	var out []models.Finding
	for app, version := range doc.Enabled {
		out = append(out, finding(app, version, true))
	}
	for app, version := range doc.Disabled {
		out = append(out, finding(app, version, false))
	}
	return out, nil
}

func finding(app, version string, enabled bool) models.Finding {
	return models.Finding{
		ProbeID:       "inventory.nextcloud.app",
		DimensionHint: models.DimensionTechnologie,
		Subject:       app,
		Severity:      models.SeverityInfo,
		SourceModus:   models.SourceModusInventory,
		Attributes: map[string]any{
			"version": version,
			"enabled": enabled,
		},
	}
}

// occStatusJSON is the subset of `occ status --output=json` we
// care about. Other fields (installed / installedversion / etc.)
// are present in the JSON but ignored to keep the contract narrow.
type occStatusJSON struct {
	VersionString string `json:"versionstring"`
	Version       string `json:"version"`
	Edition       string `json:"edition,omitempty"`
	ProductName   string `json:"productname,omitempty"`
}

// ParseStatus converts `occ status --output=json` to a single
// inventory.nextcloud.version Finding.
func ParseStatus(raw string) (models.Finding, error) {
	var doc occStatusJSON
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return models.Finding{}, fmt.Errorf("nextcloud parse status: %w", err)
	}
	if doc.VersionString == "" {
		return models.Finding{}, errors.New("nextcloud parse status: empty versionstring")
	}
	major := majorOf(doc.VersionString)
	supported := false
	for _, m := range SupportedNextcloudMajors {
		if m == major {
			supported = true
			break
		}
	}
	return models.Finding{
		ProbeID:       "inventory.nextcloud.version",
		Subject:       doc.VersionString,
		Severity:      models.SeverityInfo,
		SourceModus:   models.SourceModusInventory,
		DimensionHint: models.DimensionTechnologie,
		Attributes: map[string]any{
			"version":       doc.Version,
			"versionstring": doc.VersionString,
			"edition":       doc.Edition,
			"productname":   doc.ProductName,
			"major":         major,
			"supported":     supported,
		},
	}, nil
}

// majorOf returns the major component of `25.0.5.1`, etc.
// Returns 0 when the prefix is not numeric.
func majorOf(s string) int {
	for i, c := range s {
		if c < '0' || c > '9' {
			s = s[:i]
			break
		}
	}
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}

// occSystemConfigJSON captures the `system` map from
// `occ config:list system --output=json`. The top-level key is
// always `system` per `occ`'s convention.
type occSystemConfigJSON struct {
	System map[string]any `json:"system"`
}

// ParseSystemConfig walks the `system` config map and returns:
//   - one inventory.nextcloud.trusted_domain Finding per entry
//   - one inventory.nextcloud.objectstore Finding per backend
//     (zero or one in practice, but Nextcloud supports multi)
func ParseSystemConfig(raw string) ([]models.Finding, []models.Finding) {
	var doc occSystemConfigJSON
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, nil
	}

	var trusted, stores []models.Finding

	// Trusted domains is `[]string` keyed at `trusted_domains`.
	if raw, ok := doc.System["trusted_domains"]; ok {
		if items, ok := raw.([]any); ok {
			for _, v := range items {
				if s, ok := v.(string); ok && s != "" {
					trusted = append(trusted, models.Finding{
						ProbeID:       "inventory.nextcloud.trusted_domain",
						Subject:       s,
						Severity:      models.SeverityInfo,
						SourceModus:   models.SourceModusInventory,
						DimensionHint: models.DimensionTechnologie,
						Attributes:    map[string]any{},
					})
				}
			}
		}
	}

	// Objectstore can be either `{class, arguments: {...}}` (one
	// backend) or `multibucket` (a map of name → backend).
	if raw, ok := doc.System["objectstore"]; ok {
		stores = append(stores, parseObjectstoreEntry(raw, "")...)
	}
	if raw, ok := doc.System["objectstore_multibucket"]; ok {
		stores = append(stores, parseObjectstoreEntry(raw, "")...)
	}

	return trusted, stores
}

func parseObjectstoreEntry(raw any, label string) []models.Finding {
	entry, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	class, _ := entry["class"].(string)
	args, _ := entry["arguments"].(map[string]any)

	host, _ := args["hostname"].(string)
	bucket, _ := args["bucket"].(string)
	endpoint, _ := args["endpoint"].(string)
	region, _ := args["region"].(string)

	subject := bucket
	if subject == "" {
		subject = endpoint
	}
	if subject == "" {
		subject = class
	}
	if label != "" {
		subject = label + ":" + subject
	}

	if endpoint != "" && host == "" {
		// Some Nextcloud installs put the full URL in `endpoint`
		// rather than splitting into hostname.
		if u, err := url.Parse(endpoint); err == nil {
			host = u.Hostname()
		}
	}

	attrs := map[string]any{
		"class":         class,
		"bucket":        bucket,
		"region":        region,
		"endpoint":      endpoint,
		"endpoint_host": host,
	}

	return []models.Finding{{
		ProbeID:       "inventory.nextcloud.objectstore",
		Subject:       subject,
		Severity:      models.SeverityInfo,
		SourceModus:   models.SourceModusInventory,
		DimensionHint: models.DimensionTechnologie,
		Attributes:    attrs,
	}}
}

// occOIDCProviderJSON is the shape `occ user_oidc:provider`
// returns: an array of `{identifier, clientId, discoveryEndpoint}`
// objects per provider.
type occOIDCProviderJSON struct {
	Identifier        string `json:"identifier"`
	ClientID          string `json:"clientId"`
	DiscoveryEndpoint string `json:"discoveryEndpoint"`
}

// ParseOIDCProviders converts `occ user_oidc:provider --output=json`
// to one inventory.nextcloud.oidc_provider Finding per IdP.
func ParseOIDCProviders(raw string) ([]models.Finding, error) {
	var providers []occOIDCProviderJSON
	if err := json.Unmarshal([]byte(raw), &providers); err != nil {
		return nil, fmt.Errorf("nextcloud parse oidc: %w", err)
	}
	var out []models.Finding
	for _, p := range providers {
		host := ""
		if p.DiscoveryEndpoint != "" {
			if u, err := url.Parse(p.DiscoveryEndpoint); err == nil {
				host = u.Hostname()
			}
		}
		attrs := map[string]any{
			"client_id":          p.ClientID,
			"discovery_endpoint": p.DiscoveryEndpoint,
			"issuer_host":        host,
		}
		// Issuer URL is the discovery endpoint minus the well-known
		// suffix for readability.
		issuer := strings.TrimSuffix(p.DiscoveryEndpoint, "/.well-known/openid-configuration")
		attrs["issuer_url"] = issuer
		out = append(out, models.Finding{
			ProbeID:       "inventory.nextcloud.oidc_provider",
			Subject:       p.Identifier,
			Severity:      models.SeverityInfo,
			SourceModus:   models.SourceModusInventory,
			DimensionHint: models.DimensionTechnologie,
			Attributes:    attrs,
		})
	}
	return out, nil
}
