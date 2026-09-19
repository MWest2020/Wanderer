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

// accountabilityRules returns the five accountability rules, wired
// into DefaultRules().
func accountabilityRules() []assessor.Rule {
	return []assessor.Rule{
		registrantIdentifiable(),
		noReseller(),
		soaRname(),
		securitytxt(),
		nsHolderTransparent(),
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
// the file to exist).
func securitytxt() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.accountability.securitytxt",
		Dimension:   models.DimensionAccountability,
		Description: "RFC 9116 security.txt is present, parseable, and carries a current Contact.",
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
						Verdict:  fmt.Sprintf("security.txt expired on %s", expiresAt.Format("2006-01-02")),
						Evidence: []string{fnd.ID},
					}
				}
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("security.txt present with Contact and Expires %s", expiresAt.Format("2006-01-02")),
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
