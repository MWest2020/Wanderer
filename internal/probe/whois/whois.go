// Package whois is a thin RDAP-based WHOIS probe. It calls a public
// RDAP endpoint (rdap.org by default), pulls registrant identity,
// registrar, reseller, status, and expiry out of the response, and
// emits Findings for each. Entities are parsed recursively
// (entities[].entities[]) because a reseller commonly sits under the
// registrar entity. On any failure (network error, non-200 status,
// parse error) the probe emits a single `whois.unavailable` Finding so
// the rest of the scan continues.
package whois

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/pkg/models"
)

// defaultBaseURL is the RDAP bootstrap endpoint used when a caller
// (Probe or the scanner's NS-holder lookup) does not override it.
const defaultBaseURL = "https://rdap.org/domain/"

// Probe is the WHOIS / RDAP probe.
type Probe struct {
	// BaseURL overrides the RDAP endpoint; tests inject an httptest
	// server URL here. The empty string means rdap.org/domain/.
	BaseURL string
}

// New returns a Probe with default settings.
func New() *Probe { return &Probe{} }

// ID implements probe.Probe.
func (*Probe) ID() string { return "whois" }

// Run implements probe.Probe.
func (p *Probe) Run(ctx context.Context, target models.Target, cfg probe.Config) ([]models.Finding, error) {
	if target.Domain == "" {
		return nil, fmt.Errorf("whois: empty domain")
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	doc, err := fetchDoc(ctx, target.Domain, p.BaseURL, client, cfg.UserAgent)
	if err != nil {
		return []models.Finding{unavailable(target.Domain, err.Error())}, nil
	}
	return buildFindings(target.Domain, doc), nil
}

// LookupRegistrantStatus performs its own RDAP fetch for domain and
// classifies the registrant entity as "present" (an identifiable name
// was published), "proxied" (the vcard is empty or reads as a
// redaction/privacy placeholder), or "absent" (no registrant entity at
// all). It is used by the scanner's NS-holder-transparency check — one
// call per unique registrable domain among a target's nameservers — a
// domain wanderer is not scanning, so this intentionally returns a
// classification rather than a Finding; the caller shapes that.
func LookupRegistrantStatus(ctx context.Context, domain, baseURL string, client *http.Client, userAgent string) (string, error) {
	doc, err := fetchDoc(ctx, domain, baseURL, client, userAgent)
	if err != nil {
		return "", err
	}
	registrant := findFirst(doc.Entities, "registrant")
	if registrant == nil {
		return "absent", nil
	}
	name := vcardValue(registrant.VCardArray, "fn")
	if name == "" || looksRedacted(name) {
		return "proxied", nil
	}
	return "present", nil
}

// fetchDoc performs the RDAP HTTP GET for domain and returns the
// parsed document. It returns a plain error (not a Finding) on any
// failure (network, non-200, malformed JSON) so callers can shape
// their own *.unavailable Finding with the ProbeID appropriate to
// their context (whois.unavailable for the per-target probe,
// whois.ns_holder.unavailable for the scanner's per-nameserver-domain
// lookup).
func fetchDoc(ctx context.Context, domain, baseURL string, client *http.Client, userAgent string) (rdapDoc, error) {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	url := strings.TrimRight(baseURL, "/") + "/" + domain

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return rdapDoc{}, err
	}
	req.Header.Set("Accept", "application/rdap+json, application/json")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}
	resp, err := client.Do(req)
	if err != nil {
		return rdapDoc{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return rdapDoc{}, fmt.Errorf("rdap HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return rdapDoc{}, err
	}
	var doc rdapDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return rdapDoc{}, fmt.Errorf("rdap: parse: %w", err)
	}
	return doc, nil
}

// rdapDoc is the slice of an RDAP domain document we read.
type rdapDoc struct {
	Entities []rdapEntity `json:"entities"`
	Status   []string     `json:"status"`
	Events   []rdapEvent  `json:"events"`
	// Redacted is the RFC 9537 redacted-fields array. Its presence is
	// recorded as evidence only; it never decides an outcome by
	// itself — the registry_redaction TLD list (run 06) does.
	Redacted []any `json:"redacted"`
}

// rdapEntity is one RDAP entity. Entities SHALL be parsed recursively:
// a reseller commonly sits under the registrar entity's own Entities.
type rdapEntity struct {
	Roles      []string     `json:"roles"`
	VCardArray []any        `json:"vcardArray"`
	Entities   []rdapEntity `json:"entities,omitempty"`
}

// rdapEvent is one entry of the RDAP "events" array.
type rdapEvent struct {
	Action string `json:"eventAction"`
	Date   string `json:"eventDate"`
}

// buildFindings turns a parsed RDAP document into the probe's
// Findings. Redacted or missing fields are emitted as explicit
// redacted/absent values, never omitted silently.
func buildFindings(domain string, doc rdapDoc) []models.Finding {
	registrant := findFirst(doc.Entities, "registrant")
	registrar := findFirst(doc.Entities, "registrar")
	reseller := findFirst(doc.Entities, "reseller")

	var out []models.Finding

	// Legacy findings, kept as-is: wand.juridisch.registrar_jurisdiction
	// reads whois.registrant's country attribute.
	if registrant != nil {
		if country := vcardCountry(registrant.VCardArray); country != "" {
			out = append(out, models.Finding{
				ProbeID:       "whois.registrant",
				DimensionHint: models.DimensionJuridisch,
				Subject:       domain,
				Severity:      models.SeverityFinding,
				Attributes: map[string]any{
					"country": strings.ToUpper(country),
				},
			})
		}
	}
	if registrar != nil {
		if name := vcardValue(registrar.VCardArray, "fn"); name != "" {
			out = append(out, models.Finding{
				ProbeID:  "whois.registrar",
				Subject:  domain,
				Severity: models.SeverityInfo,
				Attributes: map[string]any{
					"name": name,
				},
			})
		}
	}

	out = append(out, registrantIdentityFinding(domain, registrant, len(doc.Redacted) > 0))
	out = append(out, resellerFinding(domain, reseller))
	out = append(out, statusFinding(domain, doc.Status))
	out = append(out, expiryFinding(domain, doc.Events))
	return out
}

// registrantIdentityFinding emits whois.registrant_identity: the
// registrant's name, its vCard KIND (individual/org/…), and a
// privacy_proxy flag. The flag is a coarse probe-level signal — it
// only catches the generic redaction placeholders registries publish
// (e.g. "REDACTED FOR PRIVACY"); matching specific commercial privacy
// services and per-TLD registry redaction is the assessor's job
// (privacy_proxies.yaml / registry_redaction.yaml, run 06).
func registrantIdentityFinding(domain string, e *rdapEntity, sawRedactedArray bool) models.Finding {
	name := "absent"
	kind := "absent"
	proxy := false
	if e != nil {
		if v := vcardValue(e.VCardArray, "fn"); v != "" {
			name = v
			proxy = looksRedacted(v)
		} else {
			// A registrant entity with no fn at all: still not an
			// identifiable name, so treat like a redaction.
			proxy = true
		}
		if v := vcardValue(e.VCardArray, "kind"); v != "" {
			kind = v
		}
	}
	attrs := map[string]any{
		"name":          name,
		"kind":          kind,
		"privacy_proxy": proxy,
	}
	if sawRedactedArray {
		attrs["rfc9537_redacted"] = true
	}
	return models.Finding{
		ProbeID:       "whois.registrant_identity",
		DimensionHint: models.DimensionAccountability,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes:    attrs,
	}
}

// resellerFinding emits whois.reseller: whether a reseller entity was
// found anywhere in the (recursive) entity tree, and its name.
func resellerFinding(domain string, e *rdapEntity) models.Finding {
	present := e != nil
	name := "absent"
	if present {
		if v := vcardValue(e.VCardArray, "fn"); v != "" {
			name = v
		}
	}
	return models.Finding{
		ProbeID:       "whois.reseller",
		DimensionHint: models.DimensionAccountability,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"present": present,
			"name":    name,
		},
	}
}

// statusFinding emits whois.status: the RDAP domain status codes,
// explicitly empty (never omitted) when the registry publishes none.
func statusFinding(domain string, codes []string) models.Finding {
	if codes == nil {
		codes = []string{}
	}
	return models.Finding{
		ProbeID:       "whois.status",
		DimensionHint: models.DimensionAccountability,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"codes": codes,
		},
	}
}

// expiryFinding emits whois.expiry: the "expiration" event date, or
// an explicit absent when the registry publishes no such event (as
// SIDN does not for .nl — see the rijksoverheid.nl fixture).
func expiryFinding(domain string, events []rdapEvent) models.Finding {
	for _, ev := range events {
		if strings.EqualFold(ev.Action, "expiration") {
			return models.Finding{
				ProbeID:       "whois.expiry",
				DimensionHint: models.DimensionAccountability,
				Subject:       domain,
				Severity:      models.SeverityObservation,
				Attributes: map[string]any{
					"present": true,
					"date":    ev.Date,
				},
			}
		}
	}
	return models.Finding{
		ProbeID:       "whois.expiry",
		DimensionHint: models.DimensionAccountability,
		Subject:       domain,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"present": false,
			"date":    "absent",
		},
	}
}

// looksRedacted reports whether name reads like a generic
// registry/registrar redaction placeholder rather than an
// identifiable name. Deliberately coarse: it is a probe-level signal,
// not the definitive privacy-proxy classification (see
// registrantIdentityFinding's doc comment).
func looksRedacted(name string) bool {
	n := strings.ToUpper(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	for _, marker := range []string{"REDACTED", "PRIVACY", "WITHHELD", "DATA PROTECTED", "NOT DISCLOSED"} {
		if strings.Contains(n, marker) {
			return true
		}
	}
	return false
}

// findFirst walks entities recursively (entities[].entities[]) and
// returns a pointer to the first entity carrying the given role, or
// nil when none is found. Depth-first, so a top-level entity is
// preferred over a same-role entity nested deeper.
func findFirst(entities []rdapEntity, role string) *rdapEntity {
	for i := range entities {
		if hasRole(entities[i].Roles, role) {
			return &entities[i]
		}
	}
	for i := range entities {
		if found := findFirst(entities[i].Entities, role); found != nil {
			return found
		}
	}
	return nil
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if strings.EqualFold(r, want) {
			return true
		}
	}
	return false
}

// vcardCountry walks the vCard array looking for an `adr` property
// whose params contain a `cc` (country code) or a country-name
// segment in the value array. Best-effort; RDAP servers vary wildly
// in how they populate vCard.
func vcardCountry(vcard []any) string {
	if len(vcard) < 2 {
		return ""
	}
	props, ok := vcard[1].([]any)
	if !ok {
		return ""
	}
	for _, raw := range props {
		prop, ok := raw.([]any)
		if !ok || len(prop) < 4 {
			continue
		}
		name, _ := prop[0].(string)
		if !strings.EqualFold(name, "adr") {
			continue
		}
		// Check params for cc.
		if params, ok := prop[1].(map[string]any); ok {
			if cc, ok := params["cc"].(string); ok && cc != "" {
				return cc
			}
		}
		// Fall through: the value (prop[3]) is a 7-element array
		// per vCard adr; the country name is the last element.
		if vals, ok := prop[3].([]any); ok && len(vals) >= 7 {
			if c, ok := vals[6].(string); ok && c != "" {
				return c
			}
		}
	}
	return ""
}

// vcardValue returns the value of the named vCard property (e.g. "fn",
// "kind") — its 4th array element per the RDAP vCard encoding
// `[name, params, type, value]`.
func vcardValue(vcard []any, propName string) string {
	if len(vcard) < 2 {
		return ""
	}
	props, ok := vcard[1].([]any)
	if !ok {
		return ""
	}
	for _, raw := range props {
		prop, ok := raw.([]any)
		if !ok || len(prop) < 4 {
			continue
		}
		name, _ := prop[0].(string)
		if !strings.EqualFold(name, propName) {
			continue
		}
		if v, ok := prop[3].(string); ok {
			return v
		}
	}
	return ""
}

func unavailable(domain, reason string) models.Finding {
	return models.Finding{
		ProbeID:  "whois.unavailable",
		Subject:  domain,
		Severity: models.SeverityInfo,
		Attributes: map[string]any{
			"reason": reason,
		},
	}
}
