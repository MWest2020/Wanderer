package scanner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Hosting identity — the high-signal observed fact "X is hosted at
// <operator> (<country>)". The scanner already collects the pieces (the
// apex dns.a/dns.aaaa addresses + the ip.asn lookups on them); this
// synthesis step turns the ASN organisation — already a "who", just
// unpolished — into one plain, self-evident Finding that leads, with
// wand.juridisch.apex_ip_eea annotating the EEA-jurisdiction score behind
// it. The fourth and last cheap who/where twin
// (research-high-signal-observability, Wave 1), after transit, mail, DNS.

// hostingOperators maps a substring of an ASN organisation to a
// recognisable host name. Unlike the mail/DNS suffix tables this matches
// on the ASN org (the apex has no operator hostname to map); the org is
// already a name, this just friendly-names the common, ugly ones
// (HETZNER-AS → Hetzner, AMAZON-02 → AWS). Deliberately small and curated,
// grown in-repo, no third-party dependency. First match wins; anything
// unlisted falls back to the raw org (see hostingOperator).
var hostingOperators = []struct{ match, operator string }{
	{"hetzner", "Hetzner"},
	{"amazon", "AWS"},
	{"aws", "AWS"},
	{"cloudflare", "Cloudflare"},
	{"microsoft", "Microsoft Azure"},
	{"azure", "Microsoft Azure"},
	{"google", "Google Cloud"},
	{"ovh", "OVH"},
	{"leaseweb", "Leaseweb"},
	{"digitalocean", "DigitalOcean"},
	{"linode", "Akamai (Linode)"},
	{"akamai", "Akamai"},
	{"fastly", "Fastly"},
	{"oracle", "Oracle Cloud"},
	{"scaleway", "Scaleway"},
	{"online s.a.s", "Scaleway"},
	{"vultr", "Vultr"},
	{"gandi", "Gandi"},
	{"transip", "TransIP"},
	{"cyso", "Cyso"},
}

// hostingOperator resolves an ASN organisation to a recognisable host
// name. The curated table is the hint; the raw organisation is the
// fallback so an unlisted host still gets its honest name. Returns "" only
// when the organisation is empty.
func hostingOperator(asnOrg string) string {
	org := strings.TrimSpace(asnOrg)
	if org == "" {
		return ""
	}
	l := strings.ToLower(org)
	for _, e := range hostingOperators {
		if strings.Contains(l, e.match) {
			return e.operator
		}
	}
	return org
}

// hostingRoute is one apex address resolved to who hosts it and where.
type hostingRoute struct {
	Address  string `json:"address"`
	Operator string `json:"operator,omitempty"`
	Country  string `json:"country,omitempty"`
	ASN      uint   `json:"asn,omitempty"`
	ASOrg    string `json:"organisation,omitempty"`
}

// synthesiseHostingIdentity correlates the apex dns.a/dns.aaaa addresses
// with the ip.asn lookups the IP probe ran on the apex and returns one
// observed aggregate Finding stating who hosts the target's front door and
// where. It is an observation, not a verdict — the EEA score is the rule's
// job. The bool is false when the dns probe produced no apex A/AAAA
// finding at all (it did not run); a domain whose apex cannot be resolved,
// or one with no GeoIP, still yields a Finding that says so.
func synthesiseHostingIdentity(target models.Target, findings []models.Finding) (models.Finding, bool) {
	apex := normHost(target.Domain)

	// ip.asn lookups on the apex, keyed by address → first seen
	// country/org. The IP probe emits one ip.asn per (host, address).
	type geo struct {
		country string
		asn     uint
		org     string
	}
	asnByAddr := map[string]geo{}
	for _, f := range findings {
		if f.ProbeID != "ip.asn" || normHost(f.Subject) != apex {
			continue
		}
		addr := stringAttr(f.Attributes, "address")
		if addr == "" {
			continue
		}
		if _, ok := asnByAddr[addr]; ok {
			continue
		}
		asnByAddr[addr] = geo{
			country: stringAttr(f.Attributes, "country"),
			org:     stringAttr(f.Attributes, "organisation"),
			asn:     uintAttr(f.Attributes, "asn"),
		}
	}

	// Walk the apex A/AAAA findings: sawApex gates the bool; an address
	// attribute means the apex actually resolved (vs no_answer / error).
	var addrs []string
	seenAddr := map[string]bool{}
	sawApex := false
	apexResolved := false
	for _, f := range findings {
		if (f.ProbeID != "dns.a" && f.ProbeID != "dns.aaaa") || normHost(f.Subject) != apex {
			continue
		}
		sawApex = true
		addr := stringAttr(f.Attributes, "address")
		if addr == "" || seenAddr[addr] {
			continue
		}
		seenAddr[addr] = true
		apexResolved = true
		addrs = append(addrs, addr)
	}
	if !sawApex {
		// The dns probe never ran (or produced no apex A/AAAA finding);
		// nothing to synthesise.
		return models.Finding{}, false
	}

	finding := models.Finding{
		ProbeID:       "ip.hosting",
		DimensionHint: models.DimensionJuridisch,
		Subject:       target.Domain,
		Severity:      models.SeverityObservation,
	}

	if !apexResolved {
		finding.Attributes = map[string]any{
			"summary":      fmt.Sprintf("no resolvable apex host for %s", target.Domain),
			"no_apex_host": true,
		}
		finding.Evidence = []byte(finding.Attributes["summary"].(string))
		return finding, true
	}
	if len(asnByAddr) == 0 {
		// Apex resolved but no ip.asn — the IP probe did not run or there
		// is no GeoIP database. The operator cannot be named.
		summary := fmt.Sprintf("hosting operator for %s undetermined (no GeoIP)", target.Domain)
		finding.Attributes = map[string]any{
			"summary":               summary,
			"operator_undetermined": true,
		}
		finding.Evidence = []byte(summary)
		return finding, true
	}

	var routes []hostingRoute
	for _, addr := range addrs {
		g := asnByAddr[addr]
		routes = append(routes, hostingRoute{
			Address:  addr,
			Operator: hostingOperator(g.org),
			Country:  g.country,
			ASN:      g.asn,
			ASOrg:    g.org,
		})
	}
	sort.SliceStable(routes, func(i, j int) bool {
		return routes[i].Address < routes[j].Address
	})

	summary := fmt.Sprintf("%s is hosted at %s", target.Domain, joinHostingOperators(routes))
	finding.Attributes = map[string]any{
		"summary": summary,
		"routes":  hostingRoutesAsAttr(routes),
	}
	finding.Evidence = []byte(summary)
	return finding, true
}

// joinHostingOperators renders the distinct "operator (country)" pairs —
// "Hetzner (DE)" or "AWS (US) and Cloudflare (US)". Unknown operators
// degrade to the address; unknown countries read "country undetermined".
func joinHostingOperators(routes []hostingRoute) string {
	var parts []string
	seen := map[string]bool{}
	for _, r := range routes {
		name := r.Operator
		if name == "" {
			name = r.Address
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

// hostingRoutesAsAttr converts the typed routes to JSON-serialisable maps
// so the Finding's Attributes stay a plain map[string]any (the store +
// exporters marshal it directly).
func hostingRoutesAsAttr(routes []hostingRoute) []map[string]any {
	out := make([]map[string]any, 0, len(routes))
	for _, r := range routes {
		m := map[string]any{"address": r.Address}
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
