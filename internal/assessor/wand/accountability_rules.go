// accountability_rules.go implements the five wand-native
// `accountability` rules (run 06 of the accountability-dimension
// change): who is legally and operationally behind a target domain,
// and whether that party is reachable. See design.md "Reason codes",
// "Privacy proxy vs registry redaction", and "Expected registrant
// matching" for the contract these rules implement.
package wand

import (
	"fmt"
	"strings"
	"time"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// accountabilityRules returns the five accountability-dimension rules
// plus the two operationeel rules that share this change's RDAP and
// variants-probe evidence (domain_expiry, variant_convergence), wired
// into DefaultRules().
func accountabilityRules() []assessor.Rule {
	return []assessor.Rule{
		registrantIdentifiable(),
		noReseller(),
		soaRname(),
		securitytxt(),
		nsHolderTransparent(),
		domainExpiry(),
		variantConvergence(),
	}
}

// registrantIdentifiable scores whether the target's RDAP registrant
// is identifiable and matches a name the organisation declared. Order
// matters (design.md "Privacy proxy vs registry redaction"): the
// target's TLD is checked against registry_redaction.yaml first,
// before the vcard content is even considered, because on such a TLD
// a redacted vcard is registry policy, not the registrant's choice.
func registrantIdentifiable() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.accountability.registrant_identifiable",
		Dimension:   models.DimensionAccountability,
		Description: "The RDAP registrant is identifiable and matches a name the organisation declared.",
		Observation: "The RDAP registrant identity lookup (`whois.registrant_identity`), compared against the organisation's declared names (`config.expected_registrant`).",
		Rationale: "When a service breaks or is misused, the RDAP registrant is who a " +
			"court order, an abuse report, or a curious citizen finds first. A " +
			"registrant that matches the organisation's own declared name is a " +
			"closed loop of accountability; a registrant hidden behind a commercial " +
			"proxy or an unmatched name breaks that loop and pushes the question " +
			"of 'who is actually responsible here' onto whoever comes looking.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			domain := accountabilityDomain(findings)
			tld := tldOf(domain)

			var identity models.Finding
			haveIdentity := false
			var unavailableID string
			for _, fnd := range findings {
				switch fnd.ProbeID {
				case "whois.registrant_identity":
					identity = fnd
					haveIdentity = true
				case "whois.unavailable":
					unavailableID = fnd.ID
				}
			}

			// TLD-list check first, independent of the vcard: on a
			// redacted TLD the outcome does not hinge on RDAP having
			// succeeded or on what the vcard says.
			if tld != "" {
				if registry, ok := RegistryRedactionFor(tld); ok {
					var evidence []string
					if haveIdentity {
						evidence = append(evidence, identity.ID)
					}
					return assessor.RuleResult{
						Score: models.ScoreOnbekend,
						Verdict: fmt.Sprintf(
							"the %s registry does not publish registrant data for .%s domains — registry policy, not the registrant's choice",
							registry, tld,
						),
						Evidence: evidence,
						Reason:   assessor.ReasonRegistryRedacted,
					}
				}
			}

			if !haveIdentity {
				var evidence []string
				if unavailableID != "" {
					evidence = []string{unavailableID}
				}
				return assessor.RuleResult{
					Score:    models.ScoreOnbekend,
					Verdict:  "no whois.registrant_identity finding — RDAP lookup unavailable",
					Evidence: evidence,
					Reason:   assessor.ReasonProbeUnavailable,
				}
			}

			name := strings.TrimSpace(stringFromAttr(identity.Attributes, "name"))

			if proxy, ok := MatchPrivacyProxy(name); ok {
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  fmt.Sprintf("registrant hidden behind commercial privacy proxy %q", proxy),
					Evidence: []string{identity.ID},
				}
			}

			if name == "" || name == "absent" {
				// RDAP succeeded but published no registrant entity at
				// all — weaker than an unmatched-but-present name, so
				// it does not share the voldoende bucket below.
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  "RDAP published no registrant entity — registrant not identifiable",
					Evidence: []string{identity.ID},
				}
			}

			if matchesExpectedRegistrant(name, expectedRegistrantNames(findings)) {
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("registrant %q matches the organisation's declared name", name),
					Evidence: []string{identity.ID},
				}
			}

			return assessor.RuleResult{
				Score:    models.ScoreVoldoende,
				Verdict:  fmt.Sprintf("registrant %q is identifiable but does not match any declared name", name),
				Evidence: []string{identity.ID},
			}
		},
	}
}

// noReseller scores whether the registrar relationship is direct or
// sits behind a reseller.
func noReseller() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.accountability.no_reseller",
		Dimension:   models.DimensionAccountability,
		Description: "The domain has a direct registrar relationship, with no reseller layer.",
		Observation: "The RDAP reseller entity lookup (`whois.reseller`).",
		Rationale: "A reseller is an extra party between the organisation and the " +
			"registrar of record — another business relationship that can lapse, " +
			"get acquired, or simply stop responding, with the domain caught in " +
			"the middle. A direct registrar relationship is one fewer party that " +
			"has to keep functioning for the domain to stay under control.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, fnd := range findings {
				if fnd.ProbeID != "whois.reseller" {
					continue
				}
				present, _ := fnd.Attributes["present"].(bool)
				name := stringFromAttr(fnd.Attributes, "name")
				if present {
					return assessor.RuleResult{
						Score:    models.ScoreAfhankelijk,
						Verdict:  fmt.Sprintf("reseller %q sits between registrant and registrar", name),
						Evidence: []string{fnd.ID},
					}
				}
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  "no reseller — direct registrar relationship",
					Evidence: []string{fnd.ID},
				}
			}
			return assessor.RuleResult{
				Score:   models.ScoreOnbekend,
				Verdict: "no whois.reseller finding — RDAP entities unavailable or unparseable",
			}
		},
	}
}

// soaRname scores the RFC 2142 / RFC 1035 contactability of the
// zone's SOA RNAME mailbox. It never claims delivery was verified —
// only existence and MX presence, the honest passive ceiling
// (design.md "The clever valkuil").
func soaRname() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.accountability.soa_rname",
		Dimension:   models.DimensionAccountability,
		Description: "The zone's SOA RNAME mailbox domain is contactable.",
		Observation: "The zone's SOA record (`dns.soa`), read for its RNAME mailbox domain and whether that domain resolves and has MX.",
		Rationale: "The SOA record's RNAME is the zone's own designated contact mailbox " +
			"— the address DNS itself points to when something is wrong with the " +
			"zone. A mailbox domain that does not resolve, or resolves with no MX, " +
			"is a contact channel that looks configured but silently swallows " +
			"mail. The rule checks existence and MX presence only; it never sends " +
			"mail, so delivery itself is never verified.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, fnd := range findings {
				if fnd.ProbeID != "dns.soa" {
					continue
				}
				if perr := stringFromAttr(fnd.Attributes, "rname_parse_error"); perr != "" {
					return assessor.RuleResult{
						Score:    models.ScoreOnbekend,
						Verdict:  fmt.Sprintf("SOA RNAME could not be parsed as a mailbox (%s)", perr),
						Evidence: []string{fnd.ID},
					}
				}
				mailboxDomain := stringFromAttr(fnd.Attributes, "mailbox_domain")
				resolves, _ := fnd.Attributes["mailbox_domain_resolves"].(bool)
				hasMX, _ := fnd.Attributes["mailbox_domain_has_mx"].(bool)
				switch {
				case resolves && hasMX:
					return assessor.RuleResult{
						Score: models.ScoreSoeverein,
						Verdict: fmt.Sprintf(
							"SOA mailbox domain %s resolves and publishes MX — delivery itself was not verified",
							mailboxDomain,
						),
						Evidence: []string{fnd.ID},
					}
				case resolves:
					return assessor.RuleResult{
						Score: models.ScoreVoldoende,
						Verdict: fmt.Sprintf(
							"SOA mailbox domain %s resolves but publishes no MX — delivery itself was not verified",
							mailboxDomain,
						),
						Evidence: []string{fnd.ID},
					}
				default:
					return assessor.RuleResult{
						Score:    models.ScoreAfhankelijk,
						Verdict:  fmt.Sprintf("SOA mailbox domain %s does not resolve — dangling contact address", mailboxDomain),
						Evidence: []string{fnd.ID},
					}
				}
			}
			return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.soa finding — SOA query failed"}
		},
	}
}

// securitytxt scores RFC 9116 security.txt presence and freshness. A
// 404 is a valid observation, not an error (RFC 9116 does not require
// the file to exist). Named dates are formatted in UTC (RFC 9116
// `Expires` is published in UTC), so the verdict does not depend on
// the scanning machine's local zone.
func securitytxt() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.accountability.securitytxt",
		Dimension:   models.DimensionAccountability,
		Description: "RFC 9116 security.txt is present, parseable, and carries a current Contact.",
		Observation: "The /.well-known/security.txt fetch (`http.securitytxt`).",
		Rationale: "security.txt is the standard, machine-readable answer to 'who do I " +
			"tell if I find a security problem here'. Without it a researcher has " +
			"no clear channel and either gives up or improvises one (a public " +
			"tweet, a random inbox) — worse outcomes than a quiet, direct report. " +
			"An expired Expires date is nearly as bad as absent: it signals the " +
			"file is not maintained and the listed contact may be stale.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, fnd := range findings {
				if fnd.ProbeID != "http.securitytxt" {
					continue
				}
				present, _ := fnd.Attributes["present"].(bool)
				if !present {
					return assessor.RuleResult{
						Score:    models.ScoreAfhankelijk,
						Verdict:  "no security.txt published at /.well-known/security.txt (404 is a valid observation)",
						Evidence: []string{fnd.ID},
					}
				}
				parseable, _ := fnd.Attributes["parseable"].(bool)
				contacts := attrStrings(fnd.Attributes["contact"])
				expiresRaw := stringFromAttr(fnd.Attributes, "expires")

				if !parseable || len(contacts) == 0 || expiresRaw == "" {
					return assessor.RuleResult{
						Score:    models.ScoreVoldoende,
						Verdict:  "security.txt present but missing required fields (Contact/Expires) or unparseable",
						Evidence: []string{fnd.ID},
					}
				}
				expiresAt, err := time.Parse(time.RFC3339, expiresRaw)
				if err != nil {
					return assessor.RuleResult{
						Score:    models.ScoreVoldoende,
						Verdict:  fmt.Sprintf("security.txt Expires value %q could not be parsed", expiresRaw),
						Evidence: []string{fnd.ID},
					}
				}
				if !expiresAt.After(time.Now()) {
					return assessor.RuleResult{
						Score:    models.ScoreVoldoende,
						Verdict:  fmt.Sprintf("security.txt expired on %s", expiresAt.UTC().Format("2006-01-02")),
						Evidence: []string{fnd.ID},
					}
				}
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("security.txt present with Contact and Expires %s", expiresAt.UTC().Format("2006-01-02")),
					Evidence: []string{fnd.ID},
				}
			}
			return assessor.RuleResult{
				Score:   models.ScoreOnbekend,
				Verdict: "no http.securitytxt finding — transport failure fetching security.txt",
			}
		},
	}
}

// nsHolderTransparent scores whether the registrable domain behind
// each authoritative nameserver has an identifiable RDAP registrant.
func nsHolderTransparent() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.accountability.ns_holder_transparent",
		Dimension:   models.DimensionAccountability,
		Description: "The organisation(s) behind the authoritative nameservers are identifiable.",
		Observation: "The RDAP lookup on each nameserver holder domain (`whois.ns_holder`).",
		Rationale: "Authoritative DNS is the control plane for the domain: whoever holds " +
			"the nameserver provider's account can redirect or withhold every name " +
			"the organisation publishes. A nameserver holder hidden behind a proxy " +
			"is one more opaque party with that level of control.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var evidence []string
			var opaque []string
			total := 0
			identifiable := 0
			for _, fnd := range findings {
				if fnd.ProbeID != "whois.ns_holder" {
					continue
				}
				total++
				evidence = append(evidence, fnd.ID)
				if stringFromAttr(fnd.Attributes, "registrant") == "present" {
					identifiable++
				} else {
					opaque = append(opaque, fnd.Subject)
				}
			}
			if total == 0 {
				return assessor.RuleResult{
					Score:   models.ScoreOnbekend,
					Verdict: "no whois.ns_holder finding — no NS holder lookup succeeded",
				}
			}
			switch {
			case identifiable == total:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("all %d nameserver holder(s) are identifiable", total),
					Evidence: evidence,
				}
			case identifiable > 0:
				return assessor.RuleResult{
					Score: models.ScoreVoldoende,
					Verdict: fmt.Sprintf(
						"%d of %d nameserver holders are identifiable; opaque: %s",
						identifiable, total, strings.Join(dedupe(opaque), ", "),
					),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  fmt.Sprintf("no nameserver holder is identifiable (%s)", strings.Join(dedupe(opaque), ", ")),
					Evidence: evidence,
				}
			}
		},
	}
}

// domainExpirySafeDays and domainExpiryUrgentDays are domainExpiry's
// two decision boundaries, shared between the Match comparison below
// and the rule's Thresholds so the two cannot drift apart.
const (
	// domainExpirySafeDays is the day count beyond which a domain
	// registration's expiry carries no near-term lapse risk.
	domainExpirySafeDays = 90
	// domainExpiryUrgentDays is the day count at or below which
	// renewal has become urgent (or is already past).
	domainExpiryUrgentDays = 30
)

// domainExpiry scores the registration expiry event under the
// operationeel dimension. It never guesses: when RDAP answered but
// the registry publishes no expiration event (as SIDN does not for
// .nl), the rule scores onbekend with reason
// `not_published_by_registry` — structural, so it never lowers the
// operationeel dimension's completeness. The named date is always
// formatted in UTC (RDAP `events` are published in UTC) so the
// verdict text does not depend on the scanning machine's local zone.
func domainExpiry() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.operationeel.domain_expiry",
		Dimension:   models.DimensionOperationeel,
		Description: "The domain registration will not lapse unexpectedly.",
		Observation: "The RDAP expiry event lookup (`whois.expiry`).",
		Rationale: fmt.Sprintf(
			"A domain that lapses because nobody renewed it in time is an "+
				"outage an attacker can turn into a takeover: a lapsed domain can be "+
				"re-registered by anyone, including someone impersonating the "+
				"organisation. A %d-day horizon gives an operator real lead time; "+
				"inside %d days — or already past — the renewal has become urgent.",
			domainExpirySafeDays, domainExpiryUrgentDays,
		),
		Thresholds: []assessor.Threshold{
			{
				Name:  "domain_expiry_safe_days",
				Value: domainExpirySafeDays,
				Unit:  "dagen",
				Explanation: fmt.Sprintf(
					"boven %d dagen is er geen kortetermijnrisico op verlies van de domeinregistratie",
					domainExpirySafeDays,
				),
			},
			{
				Name:  "domain_expiry_urgent_days",
				Value: domainExpiryUrgentDays,
				Unit:  "dagen",
				Explanation: fmt.Sprintf(
					"verloopt binnen %d dagen — of is al verlopen — waardoor verlenging urgent wordt",
					domainExpiryUrgentDays,
				),
			},
		},
		Match: func(findings []models.Finding) assessor.RuleResult {
			var expiry models.Finding
			haveExpiry := false
			var unavailableID string
			for _, fnd := range findings {
				switch fnd.ProbeID {
				case "whois.expiry":
					expiry = fnd
					haveExpiry = true
				case "whois.unavailable":
					unavailableID = fnd.ID
				}
			}
			if !haveExpiry {
				var evidence []string
				if unavailableID != "" {
					evidence = []string{unavailableID}
				}
				return assessor.RuleResult{
					Score:    models.ScoreOnbekend,
					Verdict:  "no whois.expiry finding — RDAP lookup unavailable",
					Evidence: evidence,
					Reason:   assessor.ReasonProbeUnavailable,
				}
			}

			present, _ := expiry.Attributes["present"].(bool)
			if !present {
				return assessor.RuleResult{
					Score:    models.ScoreOnbekend,
					Verdict:  "RDAP answered but the registry publishes no expiration date for this domain",
					Evidence: []string{expiry.ID},
					Reason:   assessor.ReasonNotPublishedByRegistry,
				}
			}

			dateRaw := stringFromAttr(expiry.Attributes, "date")
			expiresAt, err := time.Parse(time.RFC3339, dateRaw)
			if err != nil {
				return assessor.RuleResult{
					Score:    models.ScoreOnbekend,
					Verdict:  fmt.Sprintf("whois.expiry date %q could not be parsed", dateRaw),
					Evidence: []string{expiry.ID},
				}
			}

			daysLeft := int(time.Until(expiresAt).Hours() / 24)
			switch {
			case daysLeft > domainExpirySafeDays:
				return assessor.RuleResult{
					Score: models.ScoreSoeverein,
					Verdict: fmt.Sprintf(
						"domain registration expires %s (%d days out) — no near-term lapse risk",
						expiresAt.UTC().Format("2006-01-02"), daysLeft,
					),
					Evidence: []string{expiry.ID},
				}
			case daysLeft > domainExpiryUrgentDays:
				return assessor.RuleResult{
					Score: models.ScoreVoldoende,
					Verdict: fmt.Sprintf(
						"domain registration expires %s (%d days out) — renewal due within 90 days",
						expiresAt.UTC().Format("2006-01-02"), daysLeft,
					),
					Evidence: []string{expiry.ID},
				}
			default:
				return assessor.RuleResult{
					Score: models.ScoreAfhankelijk,
					Verdict: fmt.Sprintf(
						"domain registration expires %s — renewal due within 30 days or already past",
						expiresAt.UTC().Format("2006-01-02"),
					),
					Evidence: []string{expiry.ID},
				}
			}
		},
	}
}

// Path status values mirror internal/probe/variants.Status*. Kept as
// local string literals — not imported from the probe package — so
// the assessor stays a pure consumer of models.Finding (rule.go: "the
// package is a pure consumer of models.Finding — it does not probe").
const (
	variantStatusReachable         = "reachable"
	variantStatusNotFollowedBudget = "not_followed_budget"
	variantStatusNotTested         = "not_tested"
)

// variantPathInfo is the subset of variants.PathResult the rule reads
// off the http.variants finding's "paths" attribute. Parsed by hand
// (not imported from internal/probe/variants) so the assessor package
// keeps its "pure consumer of Findings" boundary.
type variantPathInfo struct {
	HostPart    string
	Family      string
	Scheme      string
	Status      string
	FinalOrigin string
	Reason      string
}

// variantPaths parses the http.variants finding's "paths" attribute,
// tolerant of both the in-memory ([]variants.PathResult serialised as
// []map[string]any by the store's JSON round trip becomes []any) and
// direct-from-probe shapes tests construct.
func variantPaths(raw any) []variantPathInfo {
	var out []variantPathInfo
	add := func(m map[string]any) {
		out = append(out, variantPathInfo{
			HostPart:    stringFromAttr(m, "host_part"),
			Family:      stringFromAttr(m, "family"),
			Scheme:      stringFromAttr(m, "scheme"),
			Status:      stringFromAttr(m, "status"),
			FinalOrigin: stringFromAttr(m, "final_origin"),
			Reason:      stringFromAttr(m, "reason"),
		})
	}
	switch v := raw.(type) {
	case []map[string]any:
		for _, m := range v {
			add(m)
		}
	case []any:
		for _, r := range v {
			if m, ok := r.(map[string]any); ok {
				add(m)
			}
		}
	}
	return out
}

// familyHasDNSRecord reports whether the scan recorded an apex A
// (family "v4") or AAAA (family "v6") record — the signal that
// distinguishes "this family was never offered" from "this family is
// offered but broken", so a family the domain never published in DNS
// is not blamed as "dead".
func familyHasDNSRecord(findings []models.Finding, family string) bool {
	probeID := "dns.a"
	if family == "v6" {
		probeID = "dns.aaaa"
	}
	for _, f := range findings {
		if f.ProbeID == probeID && assessor.IsEvidenceLike(f) {
			return true
		}
	}
	return false
}

// variantConvergence scores, under the operationeel dimension,
// whether all observed apex/www x IPv4/IPv6 x http/https paths
// converge on one canonical HTTPS origin (design.md "Politeness
// (variants probe)"). Paths recorded `not_followed_budget` or
// `not_tested` never count as dead — a scanner-side limitation
// (`scanner_no_ipv6`) is reported as a reason on the result, never
// charged to the target as a missing family.
func variantConvergence() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.operationeel.variant_convergence",
		Dimension:   models.DimensionOperationeel,
		Description: "All apex/www x IPv4/IPv6 x http/https paths converge on one canonical HTTPS origin.",
		Observation: "The site variants probe (`http.variants`), read for each apex/www x IPv4/IPv6 x http/https path's reachability and final origin.",
		Rationale: "A visitor, a bookmark, or a stale link can land on any of eight " +
			"apex/www x IPv4/IPv6 x http/https combinations. When they don't all " +
			"funnel to the same secure origin, some of those entry points serve " +
			"unencrypted content or a different, possibly unmaintained site — a " +
			"gap an attacker can sit in front of. Convergence is the operational " +
			"discipline of making every door lead to the same, secured room.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var variantsFinding models.Finding
			haveVariants := false
			unavailableID := ""
			for _, fnd := range findings {
				switch fnd.ProbeID {
				case "http.variants":
					variantsFinding = fnd
					haveVariants = true
				case "http.variants.unavailable":
					unavailableID = fnd.ID
				}
			}
			if !haveVariants {
				var evidence []string
				if unavailableID != "" {
					evidence = []string{unavailableID}
				}
				return assessor.RuleResult{
					Score:    models.ScoreOnbekend,
					Verdict:  "no http.variants finding — variants probe unavailable",
					Evidence: evidence,
					Reason:   assessor.ReasonProbeUnavailable,
				}
			}

			paths := variantPaths(variantsFinding.Attributes["paths"])
			total := len(paths)
			if total == 0 {
				return assessor.RuleResult{
					Score:    models.ScoreOnbekend,
					Verdict:  "http.variants finding carries no path results",
					Evidence: []string{variantsFinding.ID},
					Reason:   assessor.ReasonProbeUnavailable,
				}
			}

			var reachable, plainHTTP []variantPathInfo
			observed := 0
			scannerBlind := false
			familyAttempted := map[string]bool{}
			familyReachable := map[string]bool{}
			for _, p := range paths {
				if p.Status == variantStatusNotTested {
					if p.Reason == assessor.ReasonScannerNoIPv6 {
						scannerBlind = true
					}
					continue
				}
				if p.Status == variantStatusNotFollowedBudget {
					continue
				}
				observed++
				familyAttempted[p.Family] = true
				if p.Status == variantStatusReachable {
					familyReachable[p.Family] = true
					reachable = append(reachable, p)
					if p.Scheme == "http" && strings.HasPrefix(p.FinalOrigin, "http://") {
						plainHTTP = append(plainHTTP, p)
					}
				}
			}

			if len(plainHTTP) > 0 {
				names := make([]string, 0, len(plainHTTP))
				for _, p := range plainHTTP {
					names = append(names, fmt.Sprintf("%s/%s/%s", p.HostPart, p.Family, p.Scheme))
				}
				return assessor.RuleResult{
					Score: models.ScoreAfhankelijk,
					Verdict: fmt.Sprintf(
						"%s serves content over plain HTTP without redirecting to HTTPS",
						strings.Join(names, ", "),
					),
					Evidence: []string{variantsFinding.ID},
				}
			}

			if len(reachable) == 0 {
				return assessor.RuleResult{
					Score: models.ScoreAfhankelijk,
					Verdict: fmt.Sprintf(
						"no path reached a final destination (%d of %d paths observed)",
						observed, total,
					),
					Evidence: []string{variantsFinding.ID},
				}
			}

			origins := map[string]bool{}
			for _, p := range reachable {
				origins[p.FinalOrigin] = true
			}

			var deadFamilies []string
			for _, fam := range []string{"v4", "v6"} {
				if !familyAttempted[fam] || familyReachable[fam] {
					continue
				}
				if familyHasDNSRecord(findings, fam) {
					deadFamilies = append(deadFamilies, fam)
				}
			}

			if len(origins) == 1 && len(deadFamilies) == 0 {
				var origin string
				for o := range origins {
					origin = o
				}
				res := assessor.RuleResult{
					Score: models.ScoreSoeverein,
					Verdict: fmt.Sprintf(
						"all %d of %d observed paths converge on %s",
						observed, total, origin,
					),
					Evidence: []string{variantsFinding.ID},
				}
				if scannerBlind {
					res.Reason = assessor.ReasonScannerNoIPv6
				}
				return res
			}

			detail := fmt.Sprintf("%d of %d observed paths reach %d distinct origins", observed, total, len(origins))
			if len(deadFamilies) > 0 {
				detail = fmt.Sprintf(
					"%s family unreachable despite a published DNS record; %s",
					strings.Join(deadFamilies, ", "), detail,
				)
			}
			res := assessor.RuleResult{
				Score:    models.ScoreVoldoende,
				Verdict:  detail,
				Evidence: []string{variantsFinding.ID},
			}
			if scannerBlind {
				res.Reason = assessor.ReasonScannerNoIPv6
			}
			return res
		},
	}
}

// accountabilityDomain returns the target domain for a scan's
// findings, used to derive the TLD for the registry-redaction check
// independent of whether the RDAP lookup itself succeeded.
// config.expected_registrant is always emitted (even with an empty
// list) so it is the primary source; the whois findings are a
// fallback for tests that construct a narrower finding set directly.
func accountabilityDomain(findings []models.Finding) string {
	for _, fnd := range findings {
		if fnd.ProbeID == "config.expected_registrant" {
			return fnd.Subject
		}
	}
	for _, fnd := range findings {
		if fnd.ProbeID == "whois.registrant_identity" {
			return fnd.Subject
		}
	}
	for _, fnd := range findings {
		if fnd.ProbeID == "whois.unavailable" {
			return fnd.Subject
		}
	}
	return ""
}

// tldOf returns the last label of domain, lower-cased, or "" when
// domain has no dot.
func tldOf(domain string) string {
	domain = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(domain)), ".")
	i := strings.LastIndex(domain, ".")
	if i < 0 || i == len(domain)-1 {
		return ""
	}
	return domain[i+1:]
}

// expectedRegistrantNames extracts the organisation's declared names
// from the scan's config.expected_registrant finding.
func expectedRegistrantNames(findings []models.Finding) []string {
	for _, fnd := range findings {
		if fnd.ProbeID != "config.expected_registrant" {
			continue
		}
		return attrStrings(fnd.Attributes["expected_registrant"])
	}
	return nil
}

// legalFormSuffixTokens are the trailing-word legal-form markers
// normalised away when comparing registrant names. Matched as a whole
// token (after splitting on whitespace) so a name that merely ends in
// the letters "bv" is not clipped.
var legalFormSuffixTokens = map[string]bool{
	"B.V.": true,
	"BV":   true,
	"N.V.": true,
	"NV":   true,
}

// matchesExpectedRegistrant compares name against each of expected
// case-insensitively, with legal-form suffixes (B.V., N.V.) and the
// "Stichting" prefix normalised away on both sides. No fuzzy matching:
// anything short of an exact match on the normalised strings is a
// miss.
func matchesExpectedRegistrant(name string, expected []string) bool {
	got := normaliseRegistrantName(name)
	if got == "" {
		return false
	}
	for _, e := range expected {
		if normaliseRegistrantName(e) == got {
			return true
		}
	}
	return false
}

// normaliseRegistrantName upper-cases name, drops a leading
// "Stichting" token and a trailing B.V./N.V. token, and rejoins the
// remaining words. Word-boundary based (not a substring trim) so a
// name that happens to end in the letters "bv" is left intact.
func normaliseRegistrantName(name string) string {
	fields := strings.Fields(strings.ToUpper(strings.TrimSpace(name)))
	if len(fields) == 0 {
		return ""
	}
	if fields[0] == "STICHTING" {
		fields = fields[1:]
	}
	if len(fields) > 0 {
		last := strings.TrimSuffix(fields[len(fields)-1], ",")
		if legalFormSuffixTokens[last] {
			fields = fields[:len(fields)-1]
		}
	}
	return strings.Join(fields, " ")
}
