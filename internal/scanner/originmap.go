package scanner

import (
	"fmt"
	"sort"
	"strings"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Web third-party origin map — the high-signal observed fact "this page
// loads fonts from Google (US), a script bundle from jsDelivr, …". The
// scanner already collects the pieces (http.third_party hosts + their
// resource kinds + the ip.asn lookups on them via the Related expansion);
// this synthesis step turns them into one plain, vendor-grouped Finding
// that leads, with wand.technologie.third_parties_eea annotating the
// in/out-EEA count behind it. First Wave-2 lead
// (research-high-signal-observability) — a map of many vendors, the
// richer cousin of the four Wave-1 who/where twins.

// thirdPartyVendorSuffixes maps a known third-party host suffix to a
// recognisable vendor. Deliberately small and curated — the common page
// third parties a public-sector site actually loads — and grown in-repo
// like the mail/DNS operator tables, not via a third-party dependency.
// Anything not listed falls back to the ASN organisation (see
// thirdPartyVendor).
var thirdPartyVendorSuffixes = []operatorSuffix{
	{"fonts.googleapis.com", "Google Fonts"},
	{"fonts.gstatic.com", "Google Fonts"},
	{"ajax.googleapis.com", "Google Hosted Libraries"},
	{"google-analytics.com", "Google Analytics"},
	{"analytics.google.com", "Google Analytics"},
	{"googletagmanager.com", "Google Tag Manager"},
	{"googlesyndication.com", "Google Ads"},
	{"doubleclick.net", "Google (DoubleClick)"},
	{"gstatic.com", "Google"},
	{"youtube.com", "YouTube (Google)"},
	{"ytimg.com", "YouTube (Google)"},
	{"cdn.jsdelivr.net", "jsDelivr"},
	{"cdnjs.cloudflare.com", "cdnjs (Cloudflare)"},
	{"unpkg.com", "unpkg"},
	{"code.jquery.com", "jQuery CDN"},
	{"bootstrapcdn.com", "BootstrapCDN"},
	{"connect.facebook.net", "Meta (Facebook)"},
	{"facebook.net", "Meta (Facebook)"},
	{"hotjar.com", "Hotjar"},
	{"hotjar.io", "Hotjar"},
	{"plausible.io", "Plausible"},
	{"matomo.cloud", "Matomo"},
}

// thirdPartyVendor resolves a third-party host to a recognisable vendor
// name via the curated suffix table, with the ASN organisation as the
// fallback so an unlisted host still gets a name. Returns "" only when
// neither knows.
func thirdPartyVendor(host, asnOrg string) string {
	return operatorBySuffix(host, thirdPartyVendorSuffixes, asnOrg)
}

// vendorEntry is one vendor in the origin map: who it is, what it serves
// (the union of resource kinds across its hosts), where it sits, and the
// raw hosts behind it as evidence. Country is the observed fact; whether
// it is non-EEA is the rule's judgment, not the scanner's.
type vendorEntry struct {
	Vendor  string   `json:"vendor"`
	Kinds   []string `json:"kinds,omitempty"`
	Country string   `json:"country,omitempty"`
	Hosts   []string `json:"hosts"`
	ASOrgs  []string `json:"organisations,omitempty"`
}

// synthesiseOriginMap correlates the http.third_party hosts (and the
// resource kinds they serve) with the ip.asn lookups the IP probe ran on
// them, groups them by recognisable vendor, and returns one observed
// aggregate Finding mapping what the page pulls in and from where. It is
// an observation, not a verdict — the in/out-EEA count is the rule's job.
// The bool is false when there is no http.third_party finding at all (the
// HTTP probe did not run); a page that loads no third parties still yields
// a Finding that says so.
func synthesiseOriginMap(target models.Target, findings []models.Finding) (models.Finding, bool) {
	// ip.asn lookups, keyed by host → first seen country/org.
	type geo struct {
		country string
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
		asnByHost[h] = geo{
			country: stringAttr(f.Attributes, "country"),
			org:     stringAttr(f.Attributes, "organisation"),
		}
	}

	// Walk the third-party hosts, grouping by vendor. We accumulate into
	// maps/sets keyed by vendor so several hosts from one vendor collapse
	// into a single entry with the union of their kinds and countries.
	type acc struct {
		kinds   map[string]bool
		hosts   map[string]bool
		orgs    map[string]bool
		country string
	}
	byVendor := map[string]*acc{}
	var order []string // first-seen vendor order, stabilised later
	sawResponse := false
	for _, f := range findings {
		if f.ProbeID == "http.response" {
			// The page was fetched — a basis for "loads no third parties"
			// even when none are found.
			sawResponse = true
		}
		if f.ProbeID != "http.third_party" {
			continue
		}
		host := normHost(f.Subject)
		if host == "" {
			continue
		}
		g := asnByHost[host]
		vendor := thirdPartyVendor(host, g.org)
		if vendor == "" {
			vendor = host
		}
		a := byVendor[vendor]
		if a == nil {
			a = &acc{kinds: map[string]bool{}, hosts: map[string]bool{}, orgs: map[string]bool{}}
			byVendor[vendor] = a
			order = append(order, vendor)
		}
		for _, k := range kindsAttr(f.Attributes) {
			a.kinds[k] = true
		}
		a.hosts[host] = true
		if g.org != "" {
			a.orgs[g.org] = true
		}
		if g.country != "" && a.country == "" {
			a.country = g.country
		}
	}
	// Nothing to synthesise unless the HTTP probe fetched the page or
	// found at least one third party — otherwise the probe did not run (or
	// the fetch failed) and there is no basis for a map.
	if len(byVendor) == 0 && !sawResponse {
		return models.Finding{}, false
	}

	finding := models.Finding{
		ProbeID:       "http.origin_map",
		DimensionHint: models.DimensionTechnologie,
		Subject:       target.Domain,
		Severity:      models.SeverityObservation,
	}

	if len(byVendor) == 0 {
		summary := fmt.Sprintf("%s loads no third-party resources", target.Domain)
		finding.Attributes = map[string]any{
			"summary":          summary,
			"no_third_parties": true,
		}
		finding.Evidence = []byte(summary)
		return finding, true
	}

	sort.Strings(order)
	var entries []vendorEntry
	for _, v := range order {
		a := byVendor[v]
		entries = append(entries, vendorEntry{
			Vendor:  v,
			Kinds:   sortedKeys(a.kinds),
			Country: a.country,
			Hosts:   sortedKeys(a.hosts),
			ASOrgs:  sortedKeys(a.orgs),
		})
	}

	summary := fmt.Sprintf("%s loads %s", target.Domain, joinVendors(entries))
	finding.Attributes = map[string]any{
		"summary": summary,
		"vendors": vendorsAsAttr(entries),
	}
	finding.Evidence = []byte(summary)
	return finding, true
}

// joinVendors renders the origin map as a phrase, leading with what each
// vendor serves — "fonts from Google Fonts (US), a script from jsDelivr
// (country undetermined)".
func joinVendors(entries []vendorEntry) string {
	var parts []string
	for _, e := range entries {
		country := e.Country
		if country == "" {
			country = "country undetermined"
		}
		serves := joinKinds(e.Kinds)
		if serves != "" {
			parts = append(parts, fmt.Sprintf("%s from %s (%s)", serves, e.Vendor, country))
		} else {
			parts = append(parts, fmt.Sprintf("%s (%s)", e.Vendor, country))
		}
	}
	switch len(parts) {
	case 0:
		return "no recognised vendors"
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

// joinKinds renders the resource kinds a vendor serves in human form —
// "fonts", "a script", "scripts and styles". The HTML element kinds
// (script/link/img/iframe/source) are mapped to readable nouns.
func joinKinds(kinds []string) string {
	var nouns []string
	seen := map[string]bool{}
	for _, k := range kinds {
		n := kindNoun(k)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		nouns = append(nouns, n)
	}
	switch len(nouns) {
	case 0:
		return ""
	case 1:
		return nouns[0]
	default:
		return strings.Join(nouns[:len(nouns)-1], ", ") + " and " + nouns[len(nouns)-1]
	}
}

func kindNoun(kind string) string {
	switch kind {
	case "script":
		return "scripts"
	case "link":
		return "styles/assets"
	case "img":
		return "images"
	case "iframe":
		return "embeds"
	case "source":
		return "media"
	default:
		return kind
	}
}

// vendorsAsAttr converts the typed entries to JSON-serialisable maps so
// the Finding's Attributes stay a plain map[string]any.
func vendorsAsAttr(entries []vendorEntry) []map[string]any {
	out := make([]map[string]any, 0, len(entries))
	for _, e := range entries {
		m := map[string]any{
			"vendor": e.Vendor,
			"hosts":  e.Hosts,
		}
		if len(e.Kinds) > 0 {
			m["kinds"] = e.Kinds
		}
		if e.Country != "" {
			m["country"] = e.Country
		}
		if len(e.ASOrgs) > 0 {
			m["organisations"] = e.ASOrgs
		}
		out = append(out, m)
	}
	return out
}

// kindsAttr reads the http.third_party "kinds" attribute, tolerant of the
// in-memory ([]string) and JSON-reloaded ([]any) shapes.
func kindsAttr(attrs map[string]any) []string {
	if attrs == nil {
		return nil
	}
	switch v := attrs["kinds"].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, k := range v {
			if s, ok := k.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
