package scanner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// DNS hosting — the high-signal observed fact "DNS for X is run by
// <operator> (<country>)". The scanner already collects the pieces
// (dns.ns hosts + ip.asn lookups on them, via the Related expansion);
// this synthesis step turns them into one plain, self-evident Finding
// that leads, with wand.juridisch.ns_vendor_jurisdiction annotating the
// EEA-jurisdiction score behind it. Structural twin of mail routing
// (research-high-signal-observability, Wave 1).

// dnsOperatorSuffixes maps a known NS-host suffix to a recognisable
// managed-DNS operator. Deliberately small and curated — the common
// providers a public-sector domain actually uses — and grown in-repo the
// way the mail-operator list grows, not via a third-party dependency.
// Anything not listed falls back to the ASN organisation (see
// dnsOperator).
var dnsOperatorSuffixes = []operatorSuffix{
	{"cloudflare.com", "Cloudflare"},
	{"amazonaws.com", "AWS Route 53"},
	{"azure-dns.com", "Azure DNS"},
	{"azure-dns.net", "Azure DNS"},
	{"azure-dns.org", "Azure DNS"},
	{"azure-dns.info", "Azure DNS"},
	{"googledomains.com", "Google Cloud DNS"},
	{"google.com", "Google Cloud DNS"},
	{"nsone.net", "NS1"},
	{"ultradns.com", "UltraDNS"},
	{"ultradns.net", "UltraDNS"},
	{"ultradns.org", "UltraDNS"},
	{"dnsmadeeasy.com", "DNS Made Easy"},
	{"akam.net", "Akamai (Edge DNS)"},
	{"akamai.net", "Akamai (Edge DNS)"},
	{"transip.net", "TransIP"},
	{"transip.nl", "TransIP"},
	{"transip.eu", "TransIP"},
	{"antagonist.nl", "Antagonist"},
	{"is.nl", "Internet Service Europe"},
	{"domaincontrol.com", "GoDaddy"},
}

// dnsOperator resolves an NS host to a recognisable managed-DNS operator
// name. The curated suffix table is the hint; the ASN organisation is the
// fallback so unlisted operators still get a name. AWS Route 53's NS
// hosts follow the awsdns-NN.{com,net,org,co.uk} family, so they are
// matched on the awsdns- label rather than enumerated.
func dnsOperator(nsHost, asnOrg string) string {
	if h := normHost(nsHost); strings.Contains(h, "awsdns-") {
		return "AWS Route 53"
	}
	return operatorBySuffix(nsHost, dnsOperatorSuffixes, asnOrg)
}

// dnsRoute is one authoritative NS host resolved to who runs it and
// where. Unlike a mail route there is no preference — NS records are an
// unordered set.
type dnsRoute struct {
	Host     string `json:"host"`
	Operator string `json:"operator,omitempty"`
	Country  string `json:"country,omitempty"`
	ASN      uint   `json:"asn,omitempty"`
	ASOrg    string `json:"organisation,omitempty"`
}

// synthesiseDNSHosting correlates the dns.ns hosts with the ip.asn
// lookups the IP probe ran on them (the scanner adds NS hosts to
// Target.Related) and returns one observed aggregate Finding stating who
// runs the target's authoritative DNS and where. It is an observation,
// not a verdict — the EEA score is the rule's job. The bool is false when
// there is no dns.ns finding at all (the dns probe did not run); a domain
// whose NS records cannot be resolved still yields a Finding that says so.
func synthesiseDNSHosting(target models.Target, findings []models.Finding) (models.Finding, bool) {
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

	var routes []dnsRoute
	seenHost := map[string]bool{}
	sawNS := false
	for _, f := range findings {
		if f.ProbeID != "dns.ns" {
			continue
		}
		sawNS = true
		host := normHost(stringAttr(f.Attributes, "host"))
		if host == "" || seenHost[host] {
			// lookup error / no_answer, or a duplicate NS host — nothing
			// new to route.
			continue
		}
		seenHost[host] = true
		g := asnByHost[host]
		routes = append(routes, dnsRoute{
			Host:     host,
			Operator: dnsOperator(host, g.org),
			Country:  g.country,
			ASN:      g.asn,
			ASOrg:    g.org,
		})
	}
	if !sawNS {
		// The dns probe never ran (or produced no dns.ns finding at all);
		// nothing to synthesise.
		return models.Finding{}, false
	}

	finding := models.Finding{
		ProbeID:       "dns.ns_hosting",
		DimensionHint: models.DimensionOperationeel,
		Subject:       target.Domain,
		Severity:      models.SeverityObservation,
	}

	if len(routes) == 0 {
		finding.Attributes = map[string]any{
			"summary":              fmt.Sprintf("no resolvable authoritative DNS for %s", target.Domain),
			"no_authoritative_dns": true,
		}
		finding.Evidence = []byte(finding.Attributes["summary"].(string))
		return finding, true
	}

	sort.SliceStable(routes, func(i, j int) bool {
		return routes[i].Host < routes[j].Host
	})

	summary := fmt.Sprintf("DNS for %s is run by %s", target.Domain, joinDNSOperators(routes))
	finding.Attributes = map[string]any{
		"summary": summary,
		"routes":  dnsRoutesAsAttr(routes),
	}
	finding.Evidence = []byte(summary)
	return finding, true
}

// joinDNSOperators renders the distinct "operator (country)" pairs —
// "Cloudflare (US)" or "TransIP (NL) and Cloudflare (US)". Unknown
// operators degrade to the raw host; unknown countries read "country
// undetermined".
func joinDNSOperators(routes []dnsRoute) string {
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

// dnsRoutesAsAttr converts the typed routes to JSON-serialisable maps so
// the Finding's Attributes stay a plain map[string]any (the store +
// exporters marshal it directly).
func dnsRoutesAsAttr(routes []dnsRoute) []map[string]any {
	out := make([]map[string]any, 0, len(routes))
	for _, r := range routes {
		m := map[string]any{"host": r.Host}
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
