package wand

import (
	"fmt"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// httpExposure scores the target's passive HTTP exposure posture from
// signals the HTTP probe already observed: the baseline security
// headers (is transport security enforced, is clickjacking/MIME/
// referrer policy set) and whether the server leaks its stack version
// in the Server / X-Powered-By banner. This is the "what is exposed /
// misusable" axis — observed, not actively exploited.
func httpExposure() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.operationeel.http_exposure",
		Dimension:   models.DimensionOperationeel,
		Description: "The site enforces the baseline HTTP security headers and does not leak its stack version.",
		Rationale: "Missing security headers leave a site open to transport " +
			"downgrade (no HSTS), clickjacking (no X-Frame-Options/CSP), and " +
			"MIME/referrer leakage; a Server / X-Powered-By banner that names " +
			"the exact stack version hands an attacker the version to target. " +
			"Both are passively observable and cheap to fix.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var secHdr, resp models.Finding
			for _, f := range findings {
				switch f.ProbeID {
				case "http.security_headers":
					if assessor.IsEvidenceLike(f) {
						secHdr = f
					}
				case "http.response":
					resp = f
				}
			}
			if secHdr.ProbeID == "" {
				return assessor.RuleResult{
					Score:   models.ScoreOnbekend,
					Verdict: "no http.security_headers finding — the HTTP probe did not run or the fetch failed",
				}
			}
			missing := attrStrings(secHdr.Attributes["missing"])
			evidence := []string{secHdr.ID}
			if resp.ID != "" {
				evidence = append(evidence, resp.ID)
			}
			banner := bannerDisclosure(resp)

			verdict := func(base string) string {
				if banner != "" {
					return base + "; " + banner
				}
				return base
			}
			switch {
			case sliceContains(missing, "Strict-Transport-Security"):
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  verdict(fmt.Sprintf("no HSTS — transport security is not enforced (missing: %s)", strings.Join(missing, ", "))),
					Evidence: evidence,
				}
			case len(missing) > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  verdict(fmt.Sprintf("HSTS present; %d baseline header(s) missing: %s", len(missing), strings.Join(missing, ", "))),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  verdict("all baseline HTTP security headers present"),
					Evidence: evidence,
				}
			}
		},
	}
}

// bannerDisclosure returns a human note when the response leaks a
// Server / X-Powered-By stack identity, or "" when it does not.
func bannerDisclosure(resp models.Finding) string {
	if resp.ProbeID == "" {
		return ""
	}
	var parts []string
	if s := stringFromAttr(resp.Attributes, "server"); s != "" {
		parts = append(parts, "Server: "+s)
	}
	if p := stringFromAttr(resp.Attributes, "powered_by"); p != "" {
		parts = append(parts, "X-Powered-By: "+p)
	}
	if len(parts) == 0 {
		return ""
	}
	return "discloses stack (" + strings.Join(parts, ", ") + ")"
}

// attrStrings extracts a []string attribute, tolerating both the
// in-memory []string and the []any a JSON round-trip through the store
// produces.
func attrStrings(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, 0, len(s))
		for _, e := range s {
			if str, ok := e.(string); ok {
				out = append(out, str)
			}
		}
		return out
	}
	return nil
}

func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
