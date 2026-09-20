package wand

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/internal/probe"
	"github.com/MWest2020/wanderer/internal/probe/whois"
	"github.com/MWest2020/wanderer/pkg/models"
)

func expectedRegistrantFinding(domain string, names []string) models.Finding {
	if names == nil {
		names = []string{}
	}
	return models.Finding{
		ProbeID:       "config.expected_registrant",
		DimensionHint: models.DimensionAccountability,
		Subject:       domain,
		Severity:      models.SeverityInfo,
		Attributes: map[string]any{
			"expected_registrant": names,
		},
	}
}

func registrantIdentityFinding(id, domain, name string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "whois.registrant_identity", Subject: domain,
		DimensionHint: models.DimensionAccountability,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"name": name,
		},
	}
}

func whoisUnavailableFinding(id, domain string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "whois.unavailable", Subject: domain,
		Severity: models.SeverityInfo,
	}
}

func resellerFinding(id, domain string, present bool, name string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "whois.reseller", Subject: domain,
		DimensionHint: models.DimensionAccountability,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"present": present,
			"name":    name,
		},
	}
}

func soaFinding(id, domain string, attrs map[string]any) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "dns.soa", Subject: domain,
		DimensionHint: models.DimensionAccountability,
		Severity:      models.SeverityObservation,
		Attributes:    attrs,
	}
}

func securitytxtFinding(id, domain string, attrs map[string]any) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "http.securitytxt", Subject: domain,
		DimensionHint: models.DimensionAccountability,
		Severity:      models.SeverityObservation,
		Attributes:    attrs,
	}
}

func nsHolderFinding(id, domain, status string) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "whois.ns_holder", Subject: domain,
		DimensionHint: models.DimensionAccountability,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"registrant": status,
		},
	}
}

// ---------- registrant_identifiable ----------

func TestRegistrantIdentifiable(t *testing.T) {
	r := ruleByID(t, "wand.accountability.registrant_identifiable")

	t.Run("TLD on redaction list scores onbekend regardless of vcard", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("gemeente.nl", []string{"Gemeente Voorbeeld"}),
			registrantIdentityFinding("f1", "gemeente.nl", "Gemeente Voorbeeld"), // even a matching name
		})
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
		if got.Reason != assessor.ReasonRegistryRedacted {
			t.Fatalf("reason = %q, want %q", got.Reason, assessor.ReasonRegistryRedacted)
		}
		if !strings.Contains(got.Verdict, "SIDN") || !strings.Contains(got.Verdict, ".nl") {
			t.Errorf("verdict = %q, want it to name the registry and the TLD", got.Verdict)
		}
	})

	t.Run("TLD on redaction list even when RDAP itself failed", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("gemeente.nl", nil),
			whoisUnavailableFinding("f1", "gemeente.nl"),
		})
		if got.Score != models.ScoreOnbekend || got.Reason != assessor.ReasonRegistryRedacted {
			t.Fatalf("score = %s reason = %q, want onbekend/registry_redacted", got.Score, got.Reason)
		}
	})

	t.Run("commercial privacy proxy scores afhankelijk", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("example.com", nil),
			registrantIdentityFinding("f1", "example.com", "Domains By Proxy, LLC"),
		})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
		if got.Reason != "" {
			t.Errorf("reason = %q, want empty (proxy is not a reason code)", got.Reason)
		}
		if !strings.Contains(got.Verdict, "Domains By Proxy") {
			t.Errorf("verdict = %q, want it to name the proxy", got.Verdict)
		}
	})

	t.Run("declared name matches (case-insensitive, legal form normalised)", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("example.com", []string{"Gemeente Voorbeeld"}),
			registrantIdentityFinding("f1", "example.com", "gemeente voorbeeld B.V."),
		})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
	})

	t.Run("declared name matches with Stichting prefix normalised", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("example.com", []string{"Stichting Voorbeeld"}),
			registrantIdentityFinding("f1", "example.com", "VOORBEELD"),
		})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
	})

	t.Run("no fuzzy matching: near-miss does not match", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("example.com", []string{"Gemeente Voorbeeld"}),
			registrantIdentityFinding("f1", "example.com", "Gemeente Voorbeld"), // typo
		})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende (present but unmatched)", got.Score)
		}
	})

	t.Run("present but unmatched scores voldoende", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("example.com", []string{"Gemeente Voorbeeld"}),
			registrantIdentityFinding("f1", "example.com", "Some Other Organisation"),
		})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
	})

	t.Run("RDAP unavailable scores onbekend with probe_unavailable", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("example.com", nil),
			whoisUnavailableFinding("f1", "example.com"),
		})
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
		if got.Reason != assessor.ReasonProbeUnavailable {
			t.Fatalf("reason = %q, want %q", got.Reason, assessor.ReasonProbeUnavailable)
		}
	})

	t.Run("no evidence at all scores onbekend with probe_unavailable", func(t *testing.T) {
		got := r.Match(nil)
		if got.Score != models.ScoreOnbekend || got.Reason != assessor.ReasonProbeUnavailable {
			t.Fatalf("score = %s reason = %q, want onbekend/probe_unavailable", got.Score, got.Reason)
		}
	})

	t.Run("no identifiable registrant name published scores afhankelijk", func(t *testing.T) {
		got := r.Match([]models.Finding{
			expectedRegistrantFinding("example.com", nil),
			registrantIdentityFinding("f1", "example.com", "absent"),
		})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
	})
}

// ---------- no_reseller ----------

func TestNoReseller(t *testing.T) {
	r := ruleByID(t, "wand.accountability.no_reseller")

	t.Run("reseller present scores afhankelijk and names it", func(t *testing.T) {
		got := r.Match([]models.Finding{resellerFinding("f1", "example.nl", true, "Cheap Domains BV")})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
		if !strings.Contains(got.Verdict, "Cheap Domains BV") {
			t.Errorf("verdict = %q, want it to name the reseller", got.Verdict)
		}
	})

	t.Run("direct registrar relationship scores soeverein", func(t *testing.T) {
		got := r.Match([]models.Finding{resellerFinding("f1", "example.nl", false, "absent")})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
	})

	t.Run("unparseable / missing whois.reseller scores onbekend", func(t *testing.T) {
		got := r.Match(nil)
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
	})
}

// ---------- soa_rname ----------

func TestSoaRname(t *testing.T) {
	r := ruleByID(t, "wand.accountability.soa_rname")

	t.Run("resolves with MX scores soeverein and disclaims delivery", func(t *testing.T) {
		got := r.Match([]models.Finding{soaFinding("f1", "example.nl", map[string]any{
			"mailbox_domain":          "voorbeeld.nl",
			"mailbox_domain_resolves": true,
			"mailbox_domain_has_mx":   true,
		})})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
		if !strings.Contains(got.Verdict, "delivery itself was not verified") {
			t.Errorf("verdict = %q, must disclaim delivery verification", got.Verdict)
		}
	})

	t.Run("resolves without MX scores voldoende", func(t *testing.T) {
		got := r.Match([]models.Finding{soaFinding("f1", "example.nl", map[string]any{
			"mailbox_domain":          "voorbeeld.nl",
			"mailbox_domain_resolves": true,
			"mailbox_domain_has_mx":   false,
		})})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
	})

	t.Run("dangling RNAME domain scores afhankelijk", func(t *testing.T) {
		got := r.Match([]models.Finding{soaFinding("f1", "example.nl", map[string]any{
			"mailbox_domain":          "dangling.voorbeeld.nl",
			"mailbox_domain_resolves": false,
			"mailbox_domain_has_mx":   false,
		})})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
		if !strings.Contains(got.Verdict, "dangling.voorbeeld.nl") {
			t.Errorf("verdict = %q, want it to name the dangling domain", got.Verdict)
		}
	})

	t.Run("unparseable RNAME scores onbekend", func(t *testing.T) {
		got := r.Match([]models.Finding{soaFinding("f1", "example.nl", map[string]any{
			"rname_parse_error": "no unescaped dot",
		})})
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
	})

	t.Run("SOA query failed scores onbekend", func(t *testing.T) {
		got := r.Match(nil)
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
	})
}

// ---------- securitytxt ----------

func TestSecuritytxt(t *testing.T) {
	r := ruleByID(t, "wand.accountability.securitytxt")

	future := time.Now().Add(30 * 24 * time.Hour).UTC().Format(time.RFC3339)
	past := time.Now().Add(-30 * 24 * time.Hour).UTC().Format(time.RFC3339)

	t.Run("present, parseable, Contact and unexpired Expires scores soeverein", func(t *testing.T) {
		got := r.Match([]models.Finding{securitytxtFinding("f1", "example.nl", map[string]any{
			"present":   true,
			"parseable": true,
			"contact":   []string{"mailto:security@example.nl"},
			"expires":   future,
		})})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
	})

	t.Run("present but expired scores voldoende and names the date", func(t *testing.T) {
		got := r.Match([]models.Finding{securitytxtFinding("f1", "example.nl", map[string]any{
			"present":   true,
			"parseable": true,
			"contact":   []string{"mailto:security@example.nl"},
			"expires":   past,
		})})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
	})

	t.Run("present but missing Contact scores voldoende", func(t *testing.T) {
		got := r.Match([]models.Finding{securitytxtFinding("f1", "example.nl", map[string]any{
			"present":   true,
			"parseable": true,
			"expires":   future,
		})})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
	})

	t.Run("present but unparseable (HTML) scores voldoende", func(t *testing.T) {
		got := r.Match([]models.Finding{securitytxtFinding("f1", "example.nl", map[string]any{
			"present":   true,
			"parseable": false,
		})})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
	})

	t.Run("absent (404) is a valid observation, scores afhankelijk", func(t *testing.T) {
		got := r.Match([]models.Finding{securitytxtFinding("f1", "example.nl", map[string]any{
			"present": false,
		})})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
	})

	t.Run("transport failure scores onbekend", func(t *testing.T) {
		got := r.Match(nil)
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
	})
}

// ---------- ns_holder_transparent ----------

func TestNsHolderTransparent(t *testing.T) {
	r := ruleByID(t, "wand.accountability.ns_holder_transparent")

	t.Run("all identifiable scores soeverein", func(t *testing.T) {
		got := r.Match([]models.Finding{
			nsHolderFinding("f1", "provider-a.nl", "present"),
			nsHolderFinding("f2", "provider-b.nl", "present"),
		})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
	})

	t.Run("mixed transparency scores voldoende and names the opaque domain", func(t *testing.T) {
		got := r.Match([]models.Finding{
			nsHolderFinding("f1", "provider-a.nl", "present"),
			nsHolderFinding("f2", "provider-b.nl", "proxied"),
		})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
		if !strings.Contains(got.Verdict, "provider-b.nl") {
			t.Errorf("verdict = %q, want it to name the opaque domain", got.Verdict)
		}
	})

	t.Run("none identifiable scores afhankelijk", func(t *testing.T) {
		got := r.Match([]models.Finding{
			nsHolderFinding("f1", "provider-a.nl", "proxied"),
			nsHolderFinding("f2", "provider-b.nl", "absent"),
		})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
	})

	t.Run("no NS holder lookup succeeded scores onbekend", func(t *testing.T) {
		got := r.Match([]models.Finding{
			{ID: "f1", ProbeID: "whois.ns_holder.unavailable", Subject: "provider-a.nl"},
		})
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
	})
}

// ---------- domain_expiry ----------

func expiryFinding(id, domain string, attrs map[string]any) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "whois.expiry", Subject: domain,
		DimensionHint: models.DimensionAccountability,
		Severity:      models.SeverityObservation,
		Attributes:    attrs,
	}
}

func TestDomainExpiry(t *testing.T) {
	r := ruleByID(t, "wand.operationeel.domain_expiry")

	future := func(days int) string {
		return time.Now().Add(time.Duration(days) * 24 * time.Hour).UTC().Format(time.RFC3339)
	}

	t.Run("beyond 90 days scores soeverein", func(t *testing.T) {
		got := r.Match([]models.Finding{expiryFinding("f1", "example.nl", map[string]any{
			"present": true, "date": future(120),
		})})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
	})

	t.Run("within 90 days scores voldoende", func(t *testing.T) {
		got := r.Match([]models.Finding{expiryFinding("f1", "example.nl", map[string]any{
			"present": true, "date": future(60),
		})})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
	})

	t.Run("within 30 days scores afhankelijk and names the date", func(t *testing.T) {
		got := r.Match([]models.Finding{expiryFinding("f1", "example.nl", map[string]any{
			"present": true, "date": future(12),
		})})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
		if !strings.Contains(got.Verdict, time.Now().Add(12*24*time.Hour).UTC().Format("2006-01-02")) {
			t.Errorf("verdict = %q, want it to name the date", got.Verdict)
		}
	})

	t.Run("names the UTC calendar date regardless of the RDAP event's own zone", func(t *testing.T) {
		// A fixed, non-UTC zone stands in for whatever zone happens to
		// be on the scanning machine or embedded in the RDAP event —
		// the named date must not shift with either. 23:00 in UTC-5 on
		// the 30th is 04:00 UTC on the 1st: naive same-zone formatting
		// would name the wrong day.
		fixed := time.FixedZone("UTC-5", -5*60*60)
		instant := time.Now().Add(12 * 24 * time.Hour)
		wantDate := instant.UTC().Format("2006-01-02")
		got := r.Match([]models.Finding{expiryFinding("f1", "example.nl", map[string]any{
			"present": true, "date": instant.In(fixed).Format(time.RFC3339),
		})})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
		if !strings.Contains(got.Verdict, wantDate) {
			t.Errorf("verdict = %q, want it to name the UTC date %s", got.Verdict, wantDate)
		}
	})

	t.Run("already past scores afhankelijk", func(t *testing.T) {
		got := r.Match([]models.Finding{expiryFinding("f1", "example.nl", map[string]any{
			"present": true, "date": future(-5),
		})})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
	})

	t.Run("registry publishes no expiry scores onbekend with not_published_by_registry", func(t *testing.T) {
		got := r.Match([]models.Finding{expiryFinding("f1", "rijksoverheid.nl", map[string]any{
			"present": false, "date": "absent",
		})})
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
		if got.Reason != assessor.ReasonNotPublishedByRegistry {
			t.Fatalf("reason = %q, want %q", got.Reason, assessor.ReasonNotPublishedByRegistry)
		}
	})

	t.Run("RDAP unavailable scores onbekend with probe_unavailable", func(t *testing.T) {
		got := r.Match([]models.Finding{whoisUnavailableFinding("f1", "example.nl")})
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
		if got.Reason != assessor.ReasonProbeUnavailable {
			t.Fatalf("reason = %q, want %q", got.Reason, assessor.ReasonProbeUnavailable)
		}
	})

	t.Run("no evidence at all scores onbekend with probe_unavailable", func(t *testing.T) {
		got := r.Match(nil)
		if got.Score != models.ScoreOnbekend || got.Reason != assessor.ReasonProbeUnavailable {
			t.Fatalf("score = %s reason = %q, want onbekend/probe_unavailable", got.Score, got.Reason)
		}
	})
}

// TestDomainExpiryThresholdsMatchComparison derives its boundary days
// from r.Thresholds rather than from literal 90/30 in the test, so
// that if Match's comparison and the rule's declared Thresholds ever
// diverge (e.g. someone edits one without the other), this test fails
// instead of silently passing against a stale hand-copied number.
func TestDomainExpiryThresholdsMatchComparison(t *testing.T) {
	r := ruleByID(t, "wand.operationeel.domain_expiry")

	byName := map[string]assessor.Threshold{}
	for _, th := range r.Thresholds {
		byName[th.Name] = th
	}
	safe, ok := byName["domain_expiry_safe_days"]
	if !ok {
		t.Fatal("no domain_expiry_safe_days threshold declared")
	}
	urgent, ok := byName["domain_expiry_urgent_days"]
	if !ok {
		t.Fatal("no domain_expiry_urgent_days threshold declared")
	}

	future := func(days int) string {
		return time.Now().Add(time.Duration(days) * 24 * time.Hour).UTC().Format(time.RFC3339)
	}
	matchDays := func(days int) models.Score {
		return r.Match([]models.Finding{expiryFinding("f1", "example.nl", map[string]any{
			"present": true, "date": future(days),
		})}).Score
	}

	// +2, not +1: daysLeft is computed from time.Until at Match-time,
	// a moment after future() stamped the date, so a +1 margin can
	// truncate back down to the threshold itself and flip the verdict.
	if got := matchDays(int(safe.Value) + 2); got != models.ScoreSoeverein {
		t.Errorf("two days beyond the declared safe threshold (%d): score = %s, want soeverein", int(safe.Value)+2, got)
	}
	if got := matchDays(int(urgent.Value) + 2); got != models.ScoreVoldoende {
		t.Errorf("two days beyond the declared urgent threshold (%d): score = %s, want voldoende", int(urgent.Value)+2, got)
	}
	if got := matchDays(int(urgent.Value)); got != models.ScoreAfhankelijk {
		t.Errorf("at the declared urgent threshold (%d): score = %s, want afhankelijk", int(urgent.Value), got)
	}
}

// ---------- variant_convergence ----------

func variantPath(hostPart, family, scheme, status, finalOrigin, reason string) map[string]any {
	m := map[string]any{
		"host_part": hostPart, "family": family, "scheme": scheme, "status": status,
	}
	if finalOrigin != "" {
		m["final_origin"] = finalOrigin
	}
	if reason != "" {
		m["reason"] = reason
	}
	return m
}

func variantsFinding(id, domain string, paths []map[string]any) models.Finding {
	return models.Finding{
		ID: id, ProbeID: "http.variants", Subject: domain,
		DimensionHint: models.DimensionOperationeel,
		Severity:      models.SeverityObservation,
		Attributes: map[string]any{
			"paths": paths,
		},
	}
}

// allEightConverge builds 8 paths (apex/www x v4/v6 x http/https) that
// all land on origin.
func allEightConverge(origin string) []map[string]any {
	var out []map[string]any
	for _, host := range []string{"apex", "www"} {
		for _, fam := range []string{"v4", "v6"} {
			for _, scheme := range []string{"http", "https"} {
				out = append(out, variantPath(host, fam, scheme, "reachable", origin, ""))
			}
		}
	}
	return out
}

func TestVariantConvergence(t *testing.T) {
	r := ruleByID(t, "wand.operationeel.variant_convergence")

	t.Run("full convergence scores soeverein", func(t *testing.T) {
		got := r.Match([]models.Finding{
			variantsFinding("f1", "voorbeeld.nl", allEightConverge("https://www.voorbeeld.nl")),
		})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein", got.Score)
		}
		if got.Reason != "" {
			t.Errorf("reason = %q, want empty (scanner has full IPv6 reach)", got.Reason)
		}
	})

	t.Run("plain HTTP serving content without redirect scores afhankelijk", func(t *testing.T) {
		paths := allEightConverge("https://www.voorbeeld.nl")
		paths[0] = variantPath("apex", "v4", "http", "reachable", "http://voorbeeld.nl", "")
		got := r.Match([]models.Finding{variantsFinding("f1", "voorbeeld.nl", paths)})
		if got.Score != models.ScoreAfhankelijk {
			t.Fatalf("score = %s, want afhankelijk", got.Score)
		}
		if !strings.Contains(got.Verdict, "apex/v4/http") {
			t.Errorf("verdict = %q, want it to name the offending path", got.Verdict)
		}
	})

	t.Run("scanner without IPv6 does not blame the target", func(t *testing.T) {
		var paths []map[string]any
		for _, host := range []string{"apex", "www"} {
			for _, scheme := range []string{"http", "https"} {
				paths = append(paths, variantPath(host, "v4", scheme, "reachable", "https://www.voorbeeld.nl", ""))
				paths = append(paths, variantPath(host, "v6", scheme, "not_tested", "", "scanner_no_ipv6"))
			}
		}
		got := r.Match([]models.Finding{variantsFinding("f1", "voorbeeld.nl", paths)})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein on the observed v4 paths, not voldoende for a dead v6 family", got.Score)
		}
		if got.Reason != assessor.ReasonScannerNoIPv6 {
			t.Fatalf("reason = %q, want %q", got.Reason, assessor.ReasonScannerNoIPv6)
		}
		if !strings.Contains(got.Verdict, "4 of 8") {
			t.Errorf("verdict = %q, want it to state 4 of 8 paths observed", got.Verdict)
		}
	})

	t.Run("diverging paths score voldoende", func(t *testing.T) {
		paths := allEightConverge("https://www.voorbeeld.nl")
		paths[0] = variantPath("apex", "v4", "http", "reachable", "https://apex.voorbeeld.nl", "")
		got := r.Match([]models.Finding{variantsFinding("f1", "voorbeeld.nl", paths)})
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende", got.Score)
		}
	})

	t.Run("dead v6 family with a published AAAA record scores voldoende", func(t *testing.T) {
		var paths []map[string]any
		for _, host := range []string{"apex", "www"} {
			for _, scheme := range []string{"http", "https"} {
				paths = append(paths, variantPath(host, "v4", scheme, "reachable", "https://www.voorbeeld.nl", ""))
				paths = append(paths, variantPath(host, "v6", scheme, "unreachable", "", ""))
			}
		}
		findings := []models.Finding{
			variantsFinding("f1", "voorbeeld.nl", paths),
			{ID: "f2", ProbeID: "dns.aaaa", Subject: "voorbeeld.nl", Attributes: map[string]any{"address": "2001:db8::1"}},
		}
		got := r.Match(findings)
		if got.Score != models.ScoreVoldoende {
			t.Fatalf("score = %s, want voldoende (v6 published in DNS but unreachable)", got.Score)
		}
		if got.Reason != "" {
			t.Errorf("reason = %q, want empty (this is a real dead family, not a scanner limitation)", got.Reason)
		}
	})

	t.Run("v6 unreachable with no AAAA record is not blamed as a dead family", func(t *testing.T) {
		var paths []map[string]any
		for _, host := range []string{"apex", "www"} {
			for _, scheme := range []string{"http", "https"} {
				paths = append(paths, variantPath(host, "v4", scheme, "reachable", "https://www.voorbeeld.nl", ""))
				paths = append(paths, variantPath(host, "v6", scheme, "unreachable", "", ""))
			}
		}
		got := r.Match([]models.Finding{variantsFinding("f1", "voorbeeld.nl", paths)})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein (no AAAA published, v6 was never offered)", got.Score)
		}
	})

	t.Run("not_followed_budget paths never count as dead", func(t *testing.T) {
		paths := allEightConverge("https://www.voorbeeld.nl")
		paths[len(paths)-1] = variantPath("www", "v6", "https", "not_followed_budget", "", "")
		got := r.Match([]models.Finding{variantsFinding("f1", "voorbeeld.nl", paths)})
		if got.Score != models.ScoreSoeverein {
			t.Fatalf("score = %s, want soeverein (not_followed_budget path excluded, not dead)", got.Score)
		}
		if !strings.Contains(got.Verdict, "7 of 8") {
			t.Errorf("verdict = %q, want it to state 7 of 8 paths observed", got.Verdict)
		}
	})

	t.Run("probe unavailable scores onbekend with probe_unavailable", func(t *testing.T) {
		got := r.Match([]models.Finding{
			{ID: "f1", ProbeID: "http.variants.unavailable", Subject: "voorbeeld.nl"},
		})
		if got.Score != models.ScoreOnbekend {
			t.Fatalf("score = %s, want onbekend", got.Score)
		}
		if got.Reason != assessor.ReasonProbeUnavailable {
			t.Fatalf("reason = %q, want %q", got.Reason, assessor.ReasonProbeUnavailable)
		}
	})

	t.Run("no evidence at all scores onbekend with probe_unavailable", func(t *testing.T) {
		got := r.Match(nil)
		if got.Score != models.ScoreOnbekend || got.Reason != assessor.ReasonProbeUnavailable {
			t.Fatalf("score = %s reason = %q, want onbekend/probe_unavailable", got.Score, got.Reason)
		}
	})
}

// ---------- registration wiring ----------

func TestDefaultRules_RegistersAccountabilityRules(t *testing.T) {
	want := map[string]models.DimensionHint{
		"wand.accountability.registrant_identifiable": models.DimensionAccountability,
		"wand.accountability.no_reseller":              models.DimensionAccountability,
		"wand.accountability.soa_rname":                models.DimensionAccountability,
		"wand.accountability.securitytxt":              models.DimensionAccountability,
		"wand.accountability.ns_holder_transparent":    models.DimensionAccountability,
		"wand.operationeel.domain_expiry":              models.DimensionOperationeel,
		"wand.operationeel.variant_convergence":        models.DimensionOperationeel,
	}
	seen := map[string]bool{}
	for _, r := range DefaultRules() {
		if wantDim, ok := want[r.ID]; ok {
			seen[r.ID] = true
			if r.Dimension != wantDim {
				t.Errorf("rule %s: Dimension = %s, want %s", r.ID, r.Dimension, wantDim)
			}
		}
	}
	for id := range want {
		if !seen[id] {
			t.Errorf("accountability-change rule %q not registered in DefaultRules", id)
		}
	}
}

// ---------- rijksoverheid.nl fixture (live RDAP capture, 2026-09-19) ----------

// TestAccountabilityRules_RijksoverheidFixture pins the run-06 fixture
// scenario: registrar "Rijksoverheid" with no reseller at any nesting
// level (no_reseller -> soeverein), and a redacted registrant on a
// .nl TLD, which registry_redaction.yaml turns into onbekend with
// reason registry_redacted rather than a privacy-proxy or unmatched
// verdict.
func TestAccountabilityRules_RijksoverheidFixture(t *testing.T) {
	body, err := os.ReadFile("../../probe/whois/testdata/rdap-rijksoverheid.nl-20260919.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/rdap+json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	p := &whois.Probe{BaseURL: srv.URL + "/domain/"}
	whoisFindings, err := p.Run(context.Background(), models.Target{Domain: "rijksoverheid.nl"}, probe.Config{})
	if err != nil {
		t.Fatalf("whois run: %v", err)
	}
	all := append([]models.Finding{}, whoisFindings...)
	all = append(all, expectedRegistrantFinding("rijksoverheid.nl", nil))

	regRule := ruleByID(t, "wand.accountability.registrant_identifiable")
	got := regRule.Match(all)
	if got.Score != models.ScoreOnbekend {
		t.Fatalf("registrant_identifiable score = %s, want onbekend", got.Score)
	}
	if got.Reason != assessor.ReasonRegistryRedacted {
		t.Fatalf("registrant_identifiable reason = %q, want %q", got.Reason, assessor.ReasonRegistryRedacted)
	}

	noResellerRule := ruleByID(t, "wand.accountability.no_reseller")
	got2 := noResellerRule.Match(all)
	if got2.Score != models.ScoreSoeverein {
		t.Fatalf("no_reseller score = %s, want soeverein (no reseller at any nesting level)", got2.Score)
	}

	expiryRule := ruleByID(t, "wand.operationeel.domain_expiry")
	got3 := expiryRule.Match(all)
	if got3.Score != models.ScoreOnbekend {
		t.Fatalf("domain_expiry score = %s, want onbekend", got3.Score)
	}
	if got3.Reason != assessor.ReasonNotPublishedByRegistry {
		t.Fatalf("domain_expiry reason = %q, want %q (no expiration event in the fixture)", got3.Reason, assessor.ReasonNotPublishedByRegistry)
	}
}

// ---------- name-normalisation unit tests ----------

func TestMatchesExpectedRegistrant(t *testing.T) {
	cases := []struct {
		name     string
		got      string
		expected []string
		want     bool
	}{
		{"exact match", "Gemeente Voorbeeld", []string{"Gemeente Voorbeeld"}, true},
		{"case insensitive", "GEMEENTE VOORBEELD", []string{"gemeente voorbeeld"}, true},
		{"B.V. suffix normalised", "Gemeente Voorbeeld B.V.", []string{"Gemeente Voorbeeld"}, true},
		{"BV suffix normalised", "Gemeente Voorbeeld BV", []string{"Gemeente Voorbeeld"}, true},
		{"N.V. suffix normalised", "Voorbeeld N.V.", []string{"Voorbeeld"}, true},
		{"Stichting prefix normalised", "Stichting Voorbeeld", []string{"Voorbeeld"}, true},
		{"no fuzzy matching", "Gemeente Voorbeld", []string{"Gemeente Voorbeeld"}, false},
		{"name ending in letters bv is not clipped", "Rijksoverheid Archiefbv", []string{"Rijksoverheid Archief"}, false},
		{"empty expected list", "Gemeente Voorbeeld", nil, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesExpectedRegistrant(tc.got, tc.expected); got != tc.want {
				t.Errorf("matchesExpectedRegistrant(%q, %v) = %v, want %v", tc.got, tc.expected, got, tc.want)
			}
		})
	}
}
