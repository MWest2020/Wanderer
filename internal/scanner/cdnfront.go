package scanner

import (
	"fmt"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// CDN / front detection — the high-signal observed fact "X's apex is
// fronted by Cloudflare (US)". The hosting-identity signal reads the apex
// IP's ASN org and says "hosted at Cloudflare"; for a CDN-fronted site
// that is the edge, not the origin. This synthesis step correlates the
// apex ip.asn org with the http.response server header against a curated
// CDN-signature table and states plainly when the apex sits behind an
// edge, with wand.technologie.no_us_hyperscaler annotating the
// US-hyperscaler-reach score behind it. Second Wave-2 lead
// (research-high-signal-observability), after the third-party origin map.

// cdnSignature recognises a CDN/edge from the apex ASN organisation and/or
// the HTTP server header. A header match is a strong, specific tell; an
// ASN-org match catches edges that strip their server header. An empty
// match field is never tested (it would match everything), so an entry
// with only a serverMatch is detected on the header alone.
type cdnSignature struct {
	name        string
	orgMatch    string // substring of the ASN org; "" = do not match on org
	serverMatch string // substring of the server header; "" = do not match on server
}

// cdnSignatures is deliberately small and conservative — entries only
// where the server header or a distinctive ASN org is a strong tell, so a
// "fronted by" claim is rarely wrong. Grown in-repo like the mail/DNS/
// hosting tables, no third-party dependency. CloudFront is server-only:
// the AWS ASN org alone cannot tell an EC2 origin from a CloudFront edge.
var cdnSignatures = []cdnSignature{
	{"Cloudflare", "cloudflare", "cloudflare"},
	{"Fastly", "fastly", "fastly"},
	{"Akamai", "akamai", "akamaighost"},
	{"Amazon CloudFront", "", "cloudfront"},
	{"Vercel", "", "vercel"},
	{"Netlify", "", "netlify"},
	{"Sucuri", "", "sucuri"},
	{"BunnyCDN", "bunny", "bunnycdn"},
	{"Imperva (Incapsula)", "incapsula", "incapsula"},
	{"StackPath", "highwinds", "stackpath"},
	{"KeyCDN", "keycdn", "keycdn"},
}

// cdnFront detects the edge fronting an apex from its ASN organisation and
// server header, returning the recognisable edge name and which signal(s)
// fired ("asn", "server"). Returns "" when no signature matches. First
// matching signature wins.
func cdnFront(asnOrg, serverHeader string) (string, []string) {
	org := strings.ToLower(strings.TrimSpace(asnOrg))
	srv := strings.ToLower(strings.TrimSpace(serverHeader))
	for _, s := range cdnSignatures {
		var signals []string
		if s.orgMatch != "" && org != "" && strings.Contains(org, s.orgMatch) {
			signals = append(signals, "asn")
		}
		if s.serverMatch != "" && srv != "" && strings.Contains(srv, s.serverMatch) {
			signals = append(signals, "server")
		}
		if len(signals) > 0 {
			return s.name, signals
		}
	}
	return "", nil
}

// synthesiseCDNFront correlates the apex ip.asn (org + country), the apex
// http.response server header, and the apex tls.issuer into one observed
// aggregate Finding stating whether the apex is fronted by a CDN/edge and,
// when it is, who and where. It is an observation, not a verdict — the
// US-hyperscaler-reach score is the rule's job. The bool is false when the
// apex produced neither an ip.asn nor an http.response finding (the IP and
// HTTP probes did not run); there is no basis for a front statement.
func synthesiseCDNFront(target models.Target, findings []models.Finding) (models.Finding, bool) {
	apex := normHost(target.Domain)

	var asnOrg, country, server, issuerOrg string
	sawASN, sawResponse := false, false
	for _, f := range findings {
		if normHost(f.Subject) != apex {
			continue
		}
		switch f.ProbeID {
		case "ip.asn":
			if !sawASN {
				sawASN = true
				asnOrg = stringAttr(f.Attributes, "organisation")
				country = stringAttr(f.Attributes, "country")
			}
		case "http.response":
			if !sawResponse {
				sawResponse = true
				server = stringAttr(f.Attributes, "server")
			}
		case "tls.issuer":
			if issuerOrg == "" {
				issuerOrg = firstStringAttr(f.Attributes, "issuer_o")
			}
		}
	}
	if !sawASN && !sawResponse {
		return models.Finding{}, false
	}

	finding := models.Finding{
		ProbeID:       "http.cdn_front",
		DimensionHint: models.DimensionTechnologie,
		Subject:       target.Domain,
		Severity:      models.SeverityObservation,
	}

	edge, signals := cdnFront(asnOrg, server)
	if edge == "" {
		summary := fmt.Sprintf("no CDN/edge front detected for %s — apex served directly", target.Domain)
		finding.Attributes = map[string]any{
			"summary": summary,
			"fronted": false,
		}
		finding.Evidence = []byte(summary)
		return finding, true
	}

	loc := country
	if loc == "" {
		loc = "country undetermined (anycast?)"
	}
	summary := fmt.Sprintf("%s's apex is fronted by %s (%s)", target.Domain, edge, loc)
	attrs := map[string]any{
		"summary": summary,
		"fronted": true,
		"edge":    edge,
		"signals": signals,
	}
	if country != "" {
		attrs["country"] = country
	}
	if asnOrg != "" {
		attrs["asn_org"] = asnOrg
	}
	if server != "" {
		attrs["server"] = server
	}
	if issuerOrg != "" {
		attrs["issuer_o"] = issuerOrg
	}
	finding.Attributes = attrs
	finding.Evidence = []byte(summary)
	return finding, true
}

// firstStringAttr reads the first element of a string-slice attribute,
// tolerant of the in-memory ([]string) and JSON-reloaded ([]any) shapes.
// tls.issuer carries issuer_o as a []string (X.509 Organization).
func firstStringAttr(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	switch v := attrs[key].(type) {
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	case []any:
		if len(v) > 0 {
			if s, ok := v[0].(string); ok {
				return s
			}
		}
	case string:
		return v
	}
	return ""
}
