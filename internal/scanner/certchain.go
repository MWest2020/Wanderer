package scanner

import (
	"fmt"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// TLS-chain geography — the high-signal observed fact "the TLS certificate
// for X is issued by Let's Encrypt (US); chain ← ISRG Root X1". The
// cert_issuer_eea rule scores the issuer country but never names the CA —
// the authority that controls the site's cryptographic identity. This
// synthesis step reads the apex tls.issuer + tls.chain, names the CA, and
// states the chain plainly, with wand.juridisch.cert_issuer_eea annotating
// the EEA score behind it. The seventh and last who/where signal
// (research-high-signal-observability), closing Wave 2.

// certAuthorities maps a substring of the issuer organisation or common
// name to a recognisable Certificate Authority. Deliberately small and
// curated — the CAs a public-sector domain actually uses — and grown
// in-repo like the mail/DNS/hosting tables, no third-party dependency.
// Anything unlisted falls back to the raw issuer org/CN. First match wins.
var certAuthorities = []struct{ match, ca string }{
	{"let's encrypt", "Let's Encrypt"},
	{"lets encrypt", "Let's Encrypt"},
	{"isrg", "Let's Encrypt"},
	{"digicert", "DigiCert"},
	{"sectigo", "Sectigo"},
	{"comodo", "Sectigo (Comodo)"},
	{"globalsign", "GlobalSign"},
	{"godaddy", "GoDaddy"},
	{"starfield", "GoDaddy (Starfield)"},
	{"entrust", "Entrust"},
	{"google trust services", "Google Trust Services"},
	{"goog", "Google Trust Services"},
	{"amazon", "Amazon"},
	{"cloudflare", "Cloudflare"},
	{"identrust", "IdenTrust"},
	{"buypass", "Buypass"},
	{"actalis", "Actalis"},
	{"zerossl", "ZeroSSL"},
	{"certum", "Certum"},
}

// certAuthority resolves a leaf certificate's issuer organisation / common
// name to a recognisable CA. The curated table is the hint; the raw issuer
// org (or CN when the org is empty) is the fallback so an unlisted
// authority still gets its honest name. Returns "" only when both are
// empty.
func certAuthority(issuerOrg, issuerCN string) string {
	for _, e := range certAuthorities {
		if issuerOrg != "" && strings.Contains(strings.ToLower(issuerOrg), e.match) {
			return e.ca
		}
		if issuerCN != "" && strings.Contains(strings.ToLower(issuerCN), e.match) {
			return e.ca
		}
	}
	if issuerOrg != "" {
		return issuerOrg
	}
	return issuerCN
}

// synthesiseCertChain reads the apex tls.issuer (issuer org/CN/country) and
// tls.chain (intermediate details) into one observed aggregate Finding
// stating who issued the certificate and where, plus the chain to the
// presented intermediates. It is an observation, not a verdict — the EEA
// score is the rule's job. The bool is false when there is no tls.issuer
// finding for the apex (the TLS probe failed or the service is plain
// HTTP); there is no CA to name.
func synthesiseCertChain(target models.Target, findings []models.Finding) (models.Finding, bool) {
	apex := normHost(target.Domain)

	var issuerOrg, issuerCN, issuerCountry string
	var intermediates []certLink
	sawIssuer := false
	for _, f := range findings {
		if normHost(f.Subject) != apex {
			continue
		}
		switch f.ProbeID {
		case "tls.issuer":
			if !sawIssuer {
				sawIssuer = true
				issuerOrg = firstStringAttr(f.Attributes, "issuer_o")
				issuerCN = stringAttr(f.Attributes, "issuer_cn")
				issuerCountry = firstStringAttr(f.Attributes, "issuer_country")
			}
		case "tls.chain":
			if intermediates == nil {
				intermediates = chainLinks(f.Attributes)
			}
		}
	}
	if !sawIssuer {
		return models.Finding{}, false
	}

	ca := certAuthority(issuerOrg, issuerCN)
	loc := issuerCountry
	if loc == "" {
		loc = "jurisdiction undetermined"
	}
	summary := fmt.Sprintf("the TLS certificate for %s is issued by %s (%s)", target.Domain, caOrUnknown(ca), loc)
	if chain := renderChain(intermediates); chain != "" {
		summary += "; chain ← " + chain
	}

	attrs := map[string]any{
		"summary": summary,
		"ca":      ca,
	}
	if issuerCountry != "" {
		attrs["country"] = issuerCountry
	}
	if issuerOrg != "" {
		attrs["issuer_o"] = issuerOrg
	}
	if issuerCN != "" {
		attrs["issuer_cn"] = issuerCN
	}
	if len(intermediates) > 0 {
		attrs["chain"] = linksAsAttr(intermediates)
	}

	finding := models.Finding{
		ProbeID:       "tls.chain_geography",
		DimensionHint: models.DimensionJuridisch,
		Subject:       target.Domain,
		Severity:      models.SeverityObservation,
		Attributes:    attrs,
		Evidence:      []byte(summary),
	}
	return finding, true
}

// certLink is one intermediate CA in the presented chain.
type certLink struct {
	CN      string `json:"cn"`
	Org     string `json:"organisation,omitempty"`
	Country string `json:"country,omitempty"`
}

// chainLinks reads the tls.chain intermediate_details attribute (tolerant
// of the in-memory []map[string]any and JSON-reloaded []any shapes),
// falling back to the bare intermediates (CN-only) list.
func chainLinks(attrs map[string]any) []certLink {
	if attrs == nil {
		return nil
	}
	toLink := func(m map[string]any) certLink {
		l := certLink{}
		l.CN, _ = m["cn"].(string)
		l.Org, _ = m["organisation"].(string)
		l.Country, _ = m["country"].(string)
		return l
	}
	switch d := attrs["intermediate_details"].(type) {
	case []map[string]any:
		out := make([]certLink, 0, len(d))
		for _, m := range d {
			out = append(out, toLink(m))
		}
		return out
	case []any:
		out := make([]certLink, 0, len(d))
		for _, r := range d {
			if m, ok := r.(map[string]any); ok {
				out = append(out, toLink(m))
			}
		}
		return out
	}
	// Fallback: CN-only intermediates.
	for _, cns := range []string{"intermediates"} {
		switch v := attrs[cns].(type) {
		case []string:
			out := make([]certLink, 0, len(v))
			for _, cn := range v {
				out = append(out, certLink{CN: cn})
			}
			return out
		case []any:
			out := make([]certLink, 0, len(v))
			for _, r := range v {
				if cn, ok := r.(string); ok {
					out = append(out, certLink{CN: cn})
				}
			}
			return out
		}
	}
	return nil
}

// renderChain renders the intermediate links as "ISRG Root X1 (US)" joined
// by " ← ", naming the CA brand where the org resolves one, with the
// country when known.
func renderChain(links []certLink) string {
	var parts []string
	for _, l := range links {
		name := certAuthority(l.Org, l.CN)
		if name == "" {
			name = l.CN
		}
		if name == "" {
			continue
		}
		if l.Country != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", name, l.Country))
		} else {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, " ← ")
}

func linksAsAttr(links []certLink) []map[string]any {
	out := make([]map[string]any, 0, len(links))
	for _, l := range links {
		m := map[string]any{"cn": l.CN}
		if l.Org != "" {
			m["organisation"] = l.Org
		}
		if l.Country != "" {
			m["country"] = l.Country
		}
		out = append(out, m)
	}
	return out
}

func caOrUnknown(ca string) string {
	if ca == "" {
		return "an undetermined authority"
	}
	return ca
}
