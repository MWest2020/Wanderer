package scanner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Mail routing — the high-signal observed fact "inbound mail for X lands
// at <operator> (<country>)". The scanner already collects the pieces
// (dns.mx hosts + ip.asn lookups on them); this synthesis step turns
// them into one plain, self-evident Finding that leads, with
// wand.juridisch.mx_vendor_jurisdiction annotating the EEA-jurisdiction
// score behind it (research-high-signal-observability, Wave 1).

// operatorSuffix maps a known host suffix to a recognisable operator
// name. Shared by the mail- and DNS-hosting synthesis (the suffix→name
// table is the same machinery for both, only the entries differ).
type operatorSuffix struct{ suffix, operator string }

// operatorBySuffix resolves a host to a recognisable operator name: the
// curated suffix table is the hint, the ASN organisation the fallback so
// unlisted operators still get a name. Returns "" only when neither
// knows. Matching is on a label boundary so "notgoogle.com" never
// matches "google.com".
func operatorBySuffix(host string, table []operatorSuffix, asnOrg string) string {
	h := normHost(host)
	for _, e := range table {
		if h == e.suffix || strings.HasSuffix(h, "."+e.suffix) {
			return e.operator
		}
	}
	return strings.TrimSpace(asnOrg)
}

// mailOperatorSuffixes maps a known MX-host suffix to a recognisable
// operator name. Deliberately small and curated — the common operators a
// public-sector domain actually uses — and grown in-repo the way the
// egress vendor list grows, not via a third-party dependency. Anything
// not listed falls back to the ASN organisation (see mailOperator).
var mailOperatorSuffixes = []operatorSuffix{
	{"aspmx.l.google.com", "Google Workspace"},
	{"googlemail.com", "Google Workspace"},
	{"google.com", "Google Workspace"},
	{"mail.protection.outlook.com", "Microsoft 365"},
	{"protection.outlook.com", "Microsoft 365"},
	{"protonmail.ch", "Proton"},
	{"protonmail.com", "Proton"},
	{"zoho.eu", "Zoho"},
	{"zoho.com", "Zoho"},
	{"mimecast.com", "Mimecast"},
	{"messagelabs.com", "Broadcom (Symantec.cloud)"},
	{"antispamcloud.com", "SpamExperts"},
	{"transip.email", "TransIP"},
	{"transip.nl", "TransIP"},
}

// mailOperator resolves an MX host to a recognisable operator name via
// the curated mail-operator table, with the ASN organisation as fallback.
func mailOperator(mxHost, asnOrg string) string {
	return operatorBySuffix(mxHost, mailOperatorSuffixes, asnOrg)
}

// normHost lowercases and strips a trailing dot — MX hosts arrive
// fully-qualified ("aspmx.l.google.com."), ip.asn subjects do not.
func normHost(h string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(h), "."))
}

// mailRoute is one MX host resolved to who runs it and where.
type mailRoute struct {
	Host       string `json:"host"`
	Preference int    `json:"preference"`
	Operator   string `json:"operator,omitempty"`
	Country    string `json:"country,omitempty"`
	ASN        uint   `json:"asn,omitempty"`
	ASOrg      string `json:"organisation,omitempty"`
}

// synthesiseMailRouting correlates the dns.mx hosts with the ip.asn
// lookups the IP probe ran on them and returns one observed aggregate
// Finding stating where the target's inbound mail lands and who runs it.
// It is an observation, not a verdict — the EEA score is the rule's job.
// The bool is false when there is no dns.mx finding at all (the dns probe
// did not run); a domain with no MX records still yields a Finding that
// says so.
func synthesiseMailRouting(target models.Target, findings []models.Finding) (models.Finding, bool) {
	// ip.asn lookups, keyed by host → first seen country/org. The IP
	// probe may emit several addresses per host; the first geo-attributed
	// one is enough to name the operator and jurisdiction.
	type geo struct {
		country string
		asn     uint
		org     string
	}
	asnByHost := map[string]geo{}
	for _, f := range findings {
		if f.ProbeID != "ip.asn" {
			continue
		}
		h := normHost(f.Subject)
		if _, ok := asnByHost[h]; ok {
			continue
		}
		g := geo{
			country: stringAttr(f.Attributes, "country"),
			org:     stringAttr(f.Attributes, "organisation"),
		}
		g.asn = uintAttr(f.Attributes, "asn")
		asnByHost[h] = g
	}

	var routes []mailRoute
	sawMX := false
	for _, f := range findings {
		if f.ProbeID != "dns.mx" {
			continue
		}
		sawMX = true
		host := normHost(stringAttr(f.Attributes, "host"))
		if host == "" {
			// no_answer / lookup error — no inbound mail to route.
			continue
		}
		g := asnByHost[host]
		routes = append(routes, mailRoute{
			Host:       host,
			Preference: intAttr(f.Attributes, "preference"),
			Operator:   mailOperator(host, g.org),
			Country:    g.country,
			ASN:        g.asn,
			ASOrg:      g.org,
		})
	}
	if !sawMX {
		// The dns probe never ran (or produced no dns.mx finding at
		// all); nothing to synthesise.
		return models.Finding{}, false
	}

	finding := models.Finding{
		ProbeID:       "dns.mx_routing",
		DimensionHint: models.DimensionDataAI,
		Subject:       target.Domain,
		Severity:      models.SeverityObservation,
	}

	if len(routes) == 0 {
		finding.Attributes = map[string]any{
			"summary":         fmt.Sprintf("no inbound mail routing for %s (no MX / null MX)", target.Domain),
			"no_inbound_mail": true,
		}
		finding.Evidence = []byte(finding.Attributes["summary"].(string))
		return finding, true
	}

	sort.SliceStable(routes, func(i, j int) bool {
		if routes[i].Preference != routes[j].Preference {
			return routes[i].Preference < routes[j].Preference
		}
		return routes[i].Host < routes[j].Host
	})

	summary := fmt.Sprintf("inbound mail for %s lands at %s", target.Domain, joinOperators(routes))
	finding.Attributes = map[string]any{
		"summary": summary,
		"routes":  routesAsAttr(routes),
	}
	finding.Evidence = []byte(summary)
	return finding, true
}

// joinOperators renders the distinct "operator (country)" pairs in
// preference order — "Google Workspace (US)" or "Microsoft 365 (IE) and
// Proton (CH)". Unknown operators degrade to the raw host; unknown
// countries read "country undetermined".
func joinOperators(routes []mailRoute) string {
	var parts []string
	seen := map[string]bool{}
	for _, r := range routes {
		name := r.Operator
		if name == "" {
			name = r.Host
		}
		country := r.Country
		if country == "" {
			country = "country undetermined"
		}
		pair := fmt.Sprintf("%s (%s)", name, country)
		if seen[pair] {
			continue
		}
		seen[pair] = true
		parts = append(parts, pair)
	}
	switch len(parts) {
	case 0:
		return "an undetermined operator"
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

// routesAsAttr converts the typed routes to JSON-serialisable maps so the
// Finding's Attributes stay a plain map[string]any (the store + exporters
// marshal it directly).
func routesAsAttr(routes []mailRoute) []map[string]any {
	out := make([]map[string]any, 0, len(routes))
	for _, r := range routes {
		m := map[string]any{"host": r.Host, "preference": r.Preference}
		if r.Operator != "" {
			m["operator"] = r.Operator
		}
		if r.Country != "" {
			m["country"] = r.Country
		}
		if r.ASN != 0 {
			m["asn"] = r.ASN
		}
		if r.ASOrg != "" {
			m["organisation"] = r.ASOrg
		}
		out = append(out, m)
	}
	return out
}

func stringAttr(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	s, _ := attrs[key].(string)
	return s
}

func intAttr(attrs map[string]any, key string) int {
	if attrs == nil {
		return 0
	}
	switch v := attrs[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

func uintAttr(attrs map[string]any, key string) uint {
	if attrs == nil {
		return 0
	}
	switch v := attrs[key].(type) {
	case uint:
		return v
	case int:
		if v >= 0 {
			return uint(v)
		}
	case float64:
		if v >= 0 {
			return uint(v)
		}
	}
	return 0
}
