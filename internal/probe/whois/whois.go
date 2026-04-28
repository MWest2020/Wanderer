// Package whois is a thin RDAP-based WHOIS probe. It calls a public
// RDAP endpoint (rdap.org by default), pulls registrant country and
// registrar name out of the response, and emits two Findings —
// `whois.registrant` and `whois.registrar`. On any failure (network
// error, non-200 status, parse error) the probe emits a single
// `whois.unavailable` Finding so the rest of the scan continues.
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
	endpoint := p.BaseURL
	if endpoint == "" {
		endpoint = "https://rdap.org/domain/"
	}
	url := strings.TrimRight(endpoint, "/") + "/" + target.Domain

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return []models.Finding{unavailable(target.Domain, err.Error())}, nil
	}
	req.Header.Set("Accept", "application/rdap+json, application/json")
	if ua := cfg.UserAgent; ua != "" {
		req.Header.Set("User-Agent", ua)
	}
	resp, err := client.Do(req)
	if err != nil {
		return []models.Finding{unavailable(target.Domain, err.Error())}, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return []models.Finding{unavailable(target.Domain, fmt.Sprintf("rdap HTTP %d", resp.StatusCode))}, nil
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return []models.Finding{unavailable(target.Domain, err.Error())}, nil
	}
	return parse(target.Domain, body), nil
}

// rdapDoc is the slice of an RDAP domain document we read.
type rdapDoc struct {
	Entities []rdapEntity `json:"entities"`
}

type rdapEntity struct {
	Roles      []string   `json:"roles"`
	VCardArray []any      `json:"vcardArray"`
	Names      []string   `json:"names,omitempty"`
}

// parse extracts registrant and registrar information. The vcardArray
// is the canonical RDAP carrier of registrant data: a two-element
// array `["vcard", [...properties...]]` where each property is itself
// `[name, params, type, value]`. We pull `country-name` from the
// `adr` properties.
func parse(domain string, body []byte) []models.Finding {
	var doc rdapDoc
	if err := json.Unmarshal(body, &doc); err != nil {
		return []models.Finding{unavailable(domain, "rdap: parse: "+err.Error())}
	}
	var out []models.Finding
	for _, e := range doc.Entities {
		if hasRole(e.Roles, "registrant") {
			country := vcardCountry(e.VCardArray)
			if country != "" {
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
		if hasRole(e.Roles, "registrar") {
			name := vcardName(e.VCardArray)
			if name != "" {
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
	}
	if len(out) == 0 {
		return []models.Finding{unavailable(domain, "rdap: no registrant or registrar entities found")}
	}
	return out
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

// vcardName returns the `fn` (formatted name) entry of a vCard.
func vcardName(vcard []any) string {
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
		if !strings.EqualFold(name, "fn") {
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
