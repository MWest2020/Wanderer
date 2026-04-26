package drift

import (
	"sort"

	"github.com/MWest2020/wanderer/pkg/models"
)

// Rule returns drift Findings produced by comparing prev and curr.
// A rule that finds nothing returns nil.
type Rule func(prev, curr *models.Scan) []models.Finding

// DefaultRules is the seed set of drift rules. Adding a rule here is
// a one-file edit plus a test.
var DefaultRules = []Rule{
	tlsIssuerChanged,
	tlsDaysLeftDropped,
	dnsMXSetChanged,
	dnsNSSetChanged,
	ipCountryChanged,
	httpThirdPartyChanged,
}

// emit builds a drift Finding with the standard cross-scan attributes
// already attached. Rules call this to keep the metadata identical
// across the rule set.
func emit(prev, curr *models.Scan, probeID string, dim models.DimensionHint, sev models.Severity, subject string, extra map[string]any) models.Finding {
	attrs := map[string]any{
		"source_modus": SourceModusDrift,
		"prev_scan_id": prev.ID,
		"curr_scan_id": curr.ID,
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return models.Finding{
		ProbeID:       probeID,
		DimensionHint: dim,
		Subject:       subject,
		Severity:      sev,
		Attributes:    attrs,
	}
}

func firstFinding(scan *models.Scan, probeID string) (models.Finding, bool) {
	for _, f := range scan.Findings {
		if f.ProbeID == probeID {
			return f, true
		}
	}
	return models.Finding{}, false
}

func attrString(f models.Finding, key string) string {
	if f.Attributes == nil {
		return ""
	}
	if s, ok := f.Attributes[key].(string); ok {
		return s
	}
	return ""
}

func attrInt(f models.Finding, key string) (int, bool) {
	if f.Attributes == nil {
		return 0, false
	}
	switch v := f.Attributes[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	}
	return 0, false
}

// hostsByProbe collects sorted, deduped hosts across all findings
// matching probeID, reading the host from attrKey on each. The
// stable order keeps drift output deterministic.
func hostsByProbe(scan *models.Scan, probeID, attrKey string) []string {
	seen := map[string]bool{}
	for _, f := range scan.Findings {
		if f.ProbeID != probeID {
			continue
		}
		host := attrString(f, attrKey)
		if host == "" {
			continue
		}
		seen[host] = true
	}
	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

// thirdPartyHosts collects subjects of http.third_party findings.
func thirdPartyHosts(scan *models.Scan) []string {
	seen := map[string]bool{}
	for _, f := range scan.Findings {
		if f.ProbeID != "http.third_party" {
			continue
		}
		if f.Subject != "" {
			seen[f.Subject] = true
		}
	}
	out := make([]string, 0, len(seen))
	for h := range seen {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

// setDiff returns (added, removed) between prev and curr (both sorted).
func setDiff(prev, curr []string) (added, removed []string) {
	in := func(s []string, v string) bool {
		for _, x := range s {
			if x == v {
				return true
			}
		}
		return false
	}
	for _, v := range curr {
		if !in(prev, v) {
			added = append(added, v)
		}
	}
	for _, v := range prev {
		if !in(curr, v) {
			removed = append(removed, v)
		}
	}
	return added, removed
}

// ----- rule implementations -----

func tlsIssuerChanged(prev, curr *models.Scan) []models.Finding {
	pf, okp := firstFinding(prev, "tls.issuer")
	cf, okc := firstFinding(curr, "tls.issuer")
	if !okp || !okc {
		return nil
	}
	pIssuer := attrString(pf, "issuer_cn")
	cIssuer := attrString(cf, "issuer_cn")
	if pIssuer == "" || cIssuer == "" || pIssuer == cIssuer {
		return nil
	}
	return []models.Finding{emit(prev, curr,
		"drift.tls.issuer_changed", models.DimensionJuridisch,
		models.SeverityFinding, cf.Subject,
		map[string]any{"prev_issuer_cn": pIssuer, "curr_issuer_cn": cIssuer},
	)}
}

func tlsDaysLeftDropped(prev, curr *models.Scan) []models.Finding {
	pf, okp := firstFinding(prev, "tls.validity")
	cf, okc := firstFinding(curr, "tls.validity")
	if !okp || !okc {
		return nil
	}
	prevDays, hp := attrInt(pf, "days_left")
	currDays, hc := attrInt(cf, "days_left")
	if !hp || !hc {
		return nil
	}
	// Fire when crossing the 30-day threshold downward.
	if prevDays >= 30 && currDays < 30 {
		return []models.Finding{emit(prev, curr,
			"drift.tls.days_left_dropped", models.DimensionOperationeel,
			models.SeverityConcern, cf.Subject,
			map[string]any{"prev_days_left": prevDays, "curr_days_left": currDays},
		)}
	}
	return nil
}

func dnsMXSetChanged(prev, curr *models.Scan) []models.Finding {
	pHosts := hostsByProbe(prev, "dns.mx", "host")
	cHosts := hostsByProbe(curr, "dns.mx", "host")
	if len(pHosts) == 0 && len(cHosts) == 0 {
		return nil
	}
	added, removed := setDiff(pHosts, cHosts)
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}
	subject := ""
	if f, ok := firstFinding(curr, "dns.mx"); ok {
		subject = f.Subject
	}
	return []models.Finding{emit(prev, curr,
		"drift.dns.mx_set_changed", models.DimensionDataAI,
		models.SeverityObservation, subject,
		map[string]any{"added": added, "removed": removed},
	)}
}

func dnsNSSetChanged(prev, curr *models.Scan) []models.Finding {
	pHosts := hostsByProbe(prev, "dns.ns", "host")
	cHosts := hostsByProbe(curr, "dns.ns", "host")
	if len(pHosts) == 0 && len(cHosts) == 0 {
		return nil
	}
	added, removed := setDiff(pHosts, cHosts)
	if len(added) == 0 && len(removed) == 0 {
		return nil
	}
	subject := ""
	if f, ok := firstFinding(curr, "dns.ns"); ok {
		subject = f.Subject
	}
	return []models.Finding{emit(prev, curr,
		"drift.dns.ns_set_changed", models.DimensionOperationeel,
		models.SeverityObservation, subject,
		map[string]any{"added": added, "removed": removed},
	)}
}

func ipCountryChanged(prev, curr *models.Scan) []models.Finding {
	prevByHost := map[string]string{}
	for _, f := range prev.Findings {
		if f.ProbeID != "ip.asn" || f.Subject == "" {
			continue
		}
		prevByHost[f.Subject] = attrString(f, "country")
	}
	var out []models.Finding
	for _, f := range curr.Findings {
		if f.ProbeID != "ip.asn" || f.Subject == "" {
			continue
		}
		curC := attrString(f, "country")
		prevC, hadPrev := prevByHost[f.Subject]
		if !hadPrev || prevC == "" || curC == "" || prevC == curC {
			continue
		}
		out = append(out, emit(prev, curr,
			"drift.ip.country_changed", models.DimensionJuridisch,
			models.SeverityFinding, f.Subject,
			map[string]any{"prev_country": prevC, "curr_country": curC},
		))
	}
	return out
}

func httpThirdPartyChanged(prev, curr *models.Scan) []models.Finding {
	pHosts := thirdPartyHosts(prev)
	cHosts := thirdPartyHosts(curr)
	added, removed := setDiff(pHosts, cHosts)
	subject := ""
	if f, ok := firstFinding(curr, "http.response"); ok {
		subject = f.Subject
	}
	var out []models.Finding
	if len(added) > 0 {
		out = append(out, emit(prev, curr,
			"drift.http.third_party_added", models.DimensionTechnologie,
			models.SeverityObservation, subject,
			map[string]any{"hosts": added},
		))
	}
	if len(removed) > 0 {
		out = append(out, emit(prev, curr,
			"drift.http.third_party_removed", models.DimensionTechnologie,
			models.SeverityInfo, subject,
			map[string]any{"hosts": removed},
		))
	}
	return out
}
