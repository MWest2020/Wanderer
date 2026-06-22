// Package dictu ships the MVP rule set for the DICTU Toetsingsinstrument
// Soevereiniteit Clouddiensten. Each rule is a pure Go function of the
// finding set; adding a rule means adding a function here and wiring it
// into DefaultRules. Rules that need data a current probe does not
// produce return RuleResult{} (no evidence) so the assessor can report
// them as Incomplete rather than absent.
package wand

import (
	"fmt"
	"strings"

	"github.com/MWest2020/wanderer/internal/assessor"
	"github.com/MWest2020/wanderer/pkg/models"
)

// eeaCountries is the EU-27 plus Iceland, Liechtenstein, Norway.
// Kept as an explicit set rather than a lookup function so two
// reviewers can eyeball what counts as "in the EEA".
var eeaCountries = map[string]bool{
	"AT": true, "BE": true, "BG": true, "HR": true, "CY": true,
	"CZ": true, "DK": true, "EE": true, "FI": true, "FR": true,
	"DE": true, "GR": true, "HU": true, "IE": true, "IT": true,
	"LV": true, "LT": true, "LU": true, "MT": true, "NL": true,
	"PL": true, "PT": true, "RO": true, "SK": true, "SI": true,
	"ES": true, "SE": true,
	"IS": true, "LI": true, "NO": true,
}

// knownUSHyperscalerOrgs are substrings (case-insensitive) that
// identify a handful of large US-headquartered hyperscalers in
// MaxMind's AS organisation string. The list is intentionally short:
// the rule's job is to flag obvious concentration, not to be a
// comprehensive vendor catalogue.
var knownUSHyperscalerOrgs = []string{
	"amazon", "google", "microsoft",
	"cloudflare", "akamai", "fastly",
}

// usHyperscalerOrg reports whether an AS organisation string belongs
// to a known US-headquartered hyperscaler, returning the canonical
// org name when it matches. Used where an IP has an attributed AS
// organisation but no resolved country (anycast networks like
// Cloudflare publish no per-PoP country): the operator itself is the
// jurisdiction signal even when the geo lookup yields nothing.
func usHyperscalerOrg(org string) (string, bool) {
	low := strings.ToLower(org)
	for _, needle := range knownUSHyperscalerOrgs {
		if strings.Contains(low, needle) {
			return org, true
		}
	}
	return org, false
}

// jurisdictionTally accumulates the per-IP country evidence the four
// EEA-jurisdiction rules share (apex, MX, NS, third parties). Each
// rule correlates a set of hosts with their ip.asn lookups and feeds
// the matched findings to add(). When no finding carried a country —
// the anycast case, where networks like Cloudflare publish no per-PoP
// geo — noCountryResult() names the AS operator instead of pretending
// the IP probe never ran.
type jurisdictionTally struct {
	seen, inEEA          int
	countries, evidence  []string
	noCountryHyperscaler string
	noCountryEvidence    []string
	noCountrySeen        bool
}

func (jt *jurisdictionTally) add(f models.Finding) {
	country := strings.ToUpper(stringFromAttr(f.Attributes, "country"))
	if country == "" {
		if org := stringFromAttr(f.Attributes, "organisation"); org != "" {
			jt.noCountrySeen = true
			jt.noCountryEvidence = append(jt.noCountryEvidence, f.ID)
			if name, ok := usHyperscalerOrg(org); ok && jt.noCountryHyperscaler == "" {
				jt.noCountryHyperscaler = name
			}
		}
		return
	}
	jt.seen++
	jt.countries = append(jt.countries, country)
	jt.evidence = append(jt.evidence, f.ID)
	if eeaCountries[country] {
		jt.inEEA++
	}
}

// noCountryResult builds the verdict for the seen==0 case. operator is
// a verb-phrase placed before the org name ("apex fronted by",
// "authoritative DNS run by", "mail routed via", "third parties served
// by"); undetermined and probeAbsent are the full verdict strings for
// the non-hyperscaler and no-finding fallbacks.
func (jt *jurisdictionTally) noCountryResult(operator, undetermined, probeAbsent string) assessor.RuleResult {
	if jt.noCountryHyperscaler != "" {
		return assessor.RuleResult{
			Score:    models.ScoreAfhankelijk,
			Verdict:  fmt.Sprintf("%s %s (US hyperscaler); anycast IPs carry no country", operator, jt.noCountryHyperscaler),
			Evidence: jt.noCountryEvidence,
		}
	}
	if jt.noCountrySeen {
		return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: undetermined, Evidence: jt.noCountryEvidence}
	}
	return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: probeAbsent}
}

// DefaultRules returns the MVP DICTU rule set. Rules are independently
// testable — this function exists only as the default wiring.
func DefaultRules() []assessor.Rule {
	return []assessor.Rule{
		certIssuerEEA(),
		apexIPInEEA(),
		mxVendorJurisdiction(),
		certValidity(),
		dnsRedundancy(),
		caaRestricts(),
		thirdPartiesEEA(),
		noUSHyperscaler(),
		mxPresent(),
		oidcFederation(),
		registrarJurisdiction(),
		hostNoUSTelemetryPackages(),
		hostNoUSTelemetryServices(),
		hostEUPackageOrigin(),
		nextcloudObjectstoreEU(),
		nextcloudOIDCProviderEU(),
		dockerImagesUSRegistry(),
		dockerContainersUSRegistry(),
		transitEUPath(),
		nsVendorJurisdiction(),
		httpExposure(),
	}
}

// registrarJurisdiction scores the registrar / registrant country
// reported by the WHOIS / RDAP probe. EEA countries score
// soeverein; outside-EEA scores afhankelijk.
func registrarJurisdiction() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.juridisch.registrar_jurisdiction",
		Dimension:   models.DimensionJuridisch,
		Description: "Domain registrant registered in an EEA jurisdiction.",
		Rationale: "The party legally registered as the domain holder is the entity " +
			"a court order or law-enforcement request would address first. A registrant " +
			"in the EEA falls under EU jurisdiction (GDPR, Schrems II); a registrant " +
			"outside it depends on the cooperation of foreign authorities for any " +
			"sovereignty-sensitive intervention.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, f := range findings {
				if f.ProbeID != "whois.registrant" {
					continue
				}
				country := strings.ToUpper(stringFromAttr(f.Attributes, "country"))
				if country == "" {
					continue
				}
				if eeaCountries[country] {
					return assessor.RuleResult{
						Score:    models.ScoreSoeverein,
						Verdict:  fmt.Sprintf("registrant in %s (EEA)", country),
						Evidence: []string{f.ID},
					}
				}
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  fmt.Sprintf("registrant in %s (outside EEA)", country),
					Evidence: []string{f.ID},
				}
			}
			return assessor.RuleResult{
				Score:   models.ScoreOnbekend,
				Verdict: "no whois.registrant finding — RDAP probe did not run or returned no registrant",
			}
		},
	}
}

// ---------- Juridisch ----------

func certIssuerEEA() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.juridisch.cert_issuer_eea",
		Dimension:   models.DimensionJuridisch,
		Description: "TLS certificate issued by an authority in the EEA.",
		Rationale: "Certificate Authorities can revoke or refuse to renew certificates. " +
			"When the issuer is incorporated outside the EEA, the authority that " +
			"controls the cryptographic identity of the site sits under foreign " +
			"jurisdiction — a sanctions regime or court order in that jurisdiction " +
			"can pressure the issuer in ways an EEA regulator cannot reach.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, f := range findings {
				if f.ProbeID != "tls.issuer" {
					continue
				}
				countries := stringsFromAttr(f.Attributes, "issuer_country")
				if len(countries) == 0 {
					continue
				}
				inEEA := 0
				for _, c := range countries {
					if eeaCountries[strings.ToUpper(c)] {
						inEEA++
					}
				}
				switch {
				case inEEA == len(countries):
					return assessor.RuleResult{
						Score:    models.ScoreSoeverein,
						Verdict:  fmt.Sprintf("cert issued in %s (EEA)", strings.Join(countries, ",")),
						Evidence: []string{f.ID},
					}
				case inEEA > 0:
					return assessor.RuleResult{
						Score:    models.ScoreVoldoende,
						Verdict:  fmt.Sprintf("cert issuer jurisdictions %s (mixed EEA)", strings.Join(countries, ",")),
						Evidence: []string{f.ID},
					}
				default:
					return assessor.RuleResult{
						Score:    models.ScoreAfhankelijk,
						Verdict:  fmt.Sprintf("cert issued in %s (outside EEA)", strings.Join(countries, ",")),
						Evidence: []string{f.ID},
					}
				}
			}
			return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no tls.issuer finding — TLS probe did not run or failed"}
		},
	}
}

// observedHostingOperators returns the recognisable host names the
// scanner's ip.hosting synthesis observed (de-duplicated), so the rule can
// lead its verdict with the "who" — "hosted at Hetzner …" rather than a
// bare country. Tolerant of the routes attribute's in-memory
// ([]map[string]any) and JSON-reloaded ([]any) shapes. Empty when the
// synthesis named no operator, in which case the rule keeps its
// country-only verdict.
func observedHostingOperators(findings []models.Finding) []string {
	var ops []string
	seen := map[string]bool{}
	add := func(m map[string]any) {
		op, _ := m["operator"].(string)
		if op == "" || seen[op] {
			return
		}
		seen[op] = true
		ops = append(ops, op)
	}
	for _, f := range findings {
		if f.ProbeID != "ip.hosting" {
			continue
		}
		switch routes := f.Attributes["routes"].(type) {
		case []map[string]any:
			for _, m := range routes {
				add(m)
			}
		case []any:
			for _, r := range routes {
				if m, ok := r.(map[string]any); ok {
					add(m)
				}
			}
		}
		break
	}
	return ops
}

func apexIPInEEA() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.juridisch.apex_ip_eea",
		Dimension:   models.DimensionJuridisch,
		Description: "Apex IP addresses resolve to AS registered in the EEA.",
		Rationale: "The IP address that serves the apex domain belongs to an Autonomous " +
			"System operated by a specific organisation in a specific country. Where " +
			"that AS is registered determines which legal regime governs the data " +
			"in transit and the host's day-to-day operation. Apex traffic landing " +
			"outside the EEA means the front door of the service depends on a " +
			"non-EEA operator.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var apex string
			for _, f := range findings {
				// DNS probe emits lowercase ProbeIDs ("dns.a", "dns.aaaa");
				// matching uppercase here used to silently skip every real
				// scan and return Onbekend.
				if f.ProbeID == "dns.a" || f.ProbeID == "dns.aaaa" {
					apex = f.Subject
					break
				}
			}
			if apex == "" {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no apex DNS finding — cannot identify apex host"}
			}
			var jt jurisdictionTally
			for _, f := range findings {
				if f.ProbeID != "ip.asn" || f.Subject != apex {
					continue
				}
				jt.add(f)
			}
			if jt.seen == 0 {
				return jt.noCountryResult(
					"apex fronted by",
					"apex IP has an AS operator but no country (anycast?) — jurisdiction undetermined",
					"no ip.asn finding for apex — IP probe did not run or no --geoip database",
				)
			}
			// Lead with the observed operator ("hosted at Hetzner — …")
			// when the scanner named one; otherwise keep the country-only
			// verdict.
			lead := func(detail string) string {
				if who := joinAnd(observedHostingOperators(findings)); who != "" {
					return fmt.Sprintf("hosted at %s — %s", who, detail)
				}
				return detail
			}
			switch {
			case jt.inEEA == jt.seen:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  lead(fmt.Sprintf("apex IPs in %s (EEA)", strings.Join(jt.countries, ","))),
					Evidence: jt.evidence,
				}
			case jt.inEEA > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  lead(fmt.Sprintf("apex IPs split across %s", strings.Join(jt.countries, ","))),
					Evidence: jt.evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  lead(fmt.Sprintf("apex IPs in %s (outside EEA)", strings.Join(jt.countries, ","))),
					Evidence: jt.evidence,
				}
			}
		},
	}
}

// observedMailOperators returns the recognisable mail-operator names the
// scanner's dns.mx_routing synthesis observed (preference-ordered,
// de-duplicated), so the rule can lead its verdict with the "who" —
// "mail lands at Google Workspace …" rather than a bare country. It is
// tolerant of the routes attribute's in-memory ([]map[string]any) and
// JSON-reloaded ([]any) shapes. Empty when the synthesis named no
// operator, in which case the rule keeps its country-only verdict.
func observedMailOperators(findings []models.Finding) []string {
	var ops []string
	seen := map[string]bool{}
	add := func(m map[string]any) {
		op, _ := m["operator"].(string)
		if op == "" || seen[op] {
			return
		}
		seen[op] = true
		ops = append(ops, op)
	}
	for _, f := range findings {
		if f.ProbeID != "dns.mx_routing" {
			continue
		}
		switch routes := f.Attributes["routes"].(type) {
		case []map[string]any:
			for _, m := range routes {
				add(m)
			}
		case []any:
			for _, r := range routes {
				if m, ok := r.(map[string]any); ok {
					add(m)
				}
			}
		}
		break
	}
	return ops
}

// mailLandsAt renders the observed operators as a phrase — "Google
// Workspace" or "Microsoft 365 and Proton". Empty input yields "".
func mailLandsAt(ops []string) string {
	return joinAnd(ops)
}

// joinAnd renders a list as an English "a, b and c" phrase. Empty input
// yields "". Shared by the mail and DNS operator-led verdicts.
func joinAnd(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

func mxVendorJurisdiction() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.juridisch.mx_vendor_jurisdiction",
		Dimension:   models.DimensionJuridisch,
		Description: "MX hosts resolve to AS registered in the EEA.",
		Rationale: "Email is the most common channel for sensitive correspondence. " +
			"The mail-exchange host (`MX`) is where the organisation's inbound " +
			"messages physically arrive; a non-EEA MX vendor processes citizen " +
			"correspondence under foreign jurisdiction, with implications under " +
			"GDPR and the post-Schrems II EU adequacy regime.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			mxHosts := map[string]string{}
			for _, f := range findings {
				if f.ProbeID != "dns.mx" {
					continue
				}
				h := strings.TrimSuffix(stringFromAttr(f.Attributes, "host"), ".")
				if h == "" {
					continue
				}
				mxHosts[strings.ToLower(h)] = f.ID
			}
			if len(mxHosts) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.mx finding — no mail routing to assess"}
			}
			var jt jurisdictionTally
			for _, f := range findings {
				if f.ProbeID != "ip.asn" {
					continue
				}
				host := strings.TrimSuffix(strings.ToLower(f.Subject), ".")
				if _, ok := mxHosts[host]; !ok {
					continue
				}
				jt.add(f)
			}
			if jt.seen == 0 {
				return jt.noCountryResult(
					"mail routed via",
					"mx host has an AS operator but no country (anycast?) — jurisdiction undetermined",
					"mx hosts found but no ip.asn lookup — IP probe did not run or no --geoip database",
				)
			}
			// Merge in the MX finding IDs so the rationale cites both
			// sides of the correlation.
			evidence := jt.evidence
			for _, id := range mxHosts {
				evidence = append(evidence, id)
			}
			// Lead with the observed operator ("mail lands at Google
			// Workspace — …") when the scanner named one; otherwise keep
			// the country-only verdict.
			lead := func(detail string) string {
				if who := mailLandsAt(observedMailOperators(findings)); who != "" {
					return fmt.Sprintf("mail lands at %s — %s", who, detail)
				}
				return detail
			}
			switch {
			case jt.inEEA == jt.seen:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  lead(fmt.Sprintf("mx hosts in %s (EEA)", strings.Join(jt.countries, ","))),
					Evidence: evidence,
				}
			case jt.inEEA > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  lead(fmt.Sprintf("mx hosts split across %s", strings.Join(jt.countries, ","))),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  lead(fmt.Sprintf("mx hosts in %s (outside EEA)", strings.Join(jt.countries, ","))),
					Evidence: evidence,
				}
			}
		},
	}
}

// ---------- Operationeel ----------

func certValidity() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.operationeel.cert_validity",
		Dimension:   models.DimensionOperationeel,
		Description: "TLS certificate is valid and not expiring within 30 days.",
		Rationale: "An expired or imminently-expiring TLS certificate is the single most " +
			"common cause of unplanned downtime on public-facing services. The rule " +
			"flags renewals that have not been automated and gives the operator a " +
			"30-day lead-time to fix the underlying process before the outage hits.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			for _, f := range findings {
				if f.ProbeID != "tls.validity" {
					continue
				}
				expired, _ := f.Attributes["expired"].(bool)
				expiringSoon, _ := f.Attributes["expiring_soon"].(bool)
				daysLeft, _ := f.Attributes["days_left"].(int)
				if v, ok := f.Attributes["days_left"].(float64); ok {
					// JSON unmarshal surfaces numbers as float64 if a
					// caller round-trips through JSON before assessing.
					daysLeft = int(v)
				}
				switch {
				case expired:
					return assessor.RuleResult{
						Score:    models.ScoreAfhankelijk,
						Verdict:  "certificate expired",
						Evidence: []string{f.ID},
					}
				case expiringSoon:
					return assessor.RuleResult{
						Score:    models.ScoreVoldoende,
						Verdict:  fmt.Sprintf("certificate expires in %d days", daysLeft),
						Evidence: []string{f.ID},
					}
				default:
					return assessor.RuleResult{
						Score:    models.ScoreSoeverein,
						Verdict:  fmt.Sprintf("certificate valid, %d days remaining", daysLeft),
						Evidence: []string{f.ID},
					}
				}
			}
			return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no tls.validity finding — TLS probe did not run"}
		},
	}
}

func dnsRedundancy() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.operationeel.dns_redundancy",
		Dimension:   models.DimensionOperationeel,
		Description: "At least two authoritative nameservers are delegated.",
		Rationale: "A single authoritative nameserver is a single point of failure for " +
			"the entire domain — every service the organisation runs becomes " +
			"unreachable when that one server is down. RFC 2182 recommends at " +
			"least two; sovereign-grade operators typically run nameservers in " +
			"two distinct ASs and two distinct geographic regions.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			hosts := map[string]bool{}
			var evidence []string
			for _, f := range findings {
				if f.ProbeID != "dns.ns" {
					continue
				}
				h := strings.TrimSuffix(strings.ToLower(stringFromAttr(f.Attributes, "host")), ".")
				if h == "" {
					continue
				}
				hosts[h] = true
				evidence = append(evidence, f.ID)
			}
			if len(hosts) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.ns finding — DNS probe did not return NS records"}
			}
			if len(hosts) < 2 {
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  "only one nameserver delegated; no redundancy",
					Evidence: evidence,
				}
			}
			return assessor.RuleResult{
				Score:    models.ScoreVoldoende,
				Verdict:  fmt.Sprintf("%d nameservers delegated", len(hosts)),
				Evidence: evidence,
			}
		},
	}
}

func caaRestricts() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.operationeel.caa_restricts_issuance",
		Dimension:   models.DimensionOperationeel,
		Description: "CAA records restrict which CAs may issue certificates.",
		Rationale: "A `CAA` DNS record names the Certificate Authorities allowed to " +
			"issue certificates for the domain. Without one, any CA in the " +
			"world's public trust store can mint a valid certificate — including " +
			"one obtained by an attacker who tricked a CA into issuing it. CAA " +
			"is a low-cost preventive control with no operational downside.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var evidence []string
			var hasCAA bool
			var noAnswer bool
			for _, f := range findings {
				if f.ProbeID != "dns.caa" {
					continue
				}
				if _, isNoAnswer := f.Attributes["no_answer"]; isNoAnswer {
					noAnswer = true
					evidence = append(evidence, f.ID)
					continue
				}
				tag := strings.ToLower(stringFromAttr(f.Attributes, "tag"))
				value := stringFromAttr(f.Attributes, "value")
				if (tag == "issue" || tag == "issuewild") && value != "" {
					hasCAA = true
					evidence = append(evidence, f.ID)
				}
			}
			switch {
			case hasCAA:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  "CAA records restrict certificate issuance",
					Evidence: evidence,
				}
			case noAnswer:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  "no CAA records — any public CA may issue",
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.caa finding — DNS probe did not query CAA"}
			}
		},
	}
}

// ---------- Technologie ----------

// observedNonEEAVendors returns the recognisable vendor names the
// scanner's http.origin_map synthesis observed in a non-EEA country
// (de-duplicated, the actionable export surface), so the rule can lead its
// verdict with the "who" — "loads from Google (US) …" rather than a bare
// count. Vendors with no observed country are not flagged (undetermined,
// not non-EEA). Tolerant of the vendors attribute's in-memory
// ([]map[string]any) and JSON-reloaded ([]any) shapes. Empty when the
// synthesis named no non-EEA vendor, in which case the rule keeps its
// count-only verdict.
func observedNonEEAVendors(findings []models.Finding) []string {
	var vendors []string
	seen := map[string]bool{}
	add := func(m map[string]any) {
		v, _ := m["vendor"].(string)
		country, _ := m["country"].(string)
		if v == "" || country == "" || seen[v] {
			return
		}
		if eeaCountries[strings.ToUpper(country)] {
			return
		}
		seen[v] = true
		vendors = append(vendors, v)
	}
	for _, f := range findings {
		if f.ProbeID != "http.origin_map" {
			continue
		}
		switch vs := f.Attributes["vendors"].(type) {
		case []map[string]any:
			for _, m := range vs {
				add(m)
			}
		case []any:
			for _, r := range vs {
				if m, ok := r.(map[string]any); ok {
					add(m)
				}
			}
		}
		break
	}
	return vendors
}

func thirdPartiesEEA() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.technologie.third_parties_eea",
		Dimension:   models.DimensionTechnologie,
		Description: "HTTP third-party dependencies resolve to AS registered in the EEA.",
		Rationale: "Modern web pages load fonts, analytics, scripts, and assets from " +
			"third-party hosts. Each third party that runs on a non-EEA AS adds " +
			"another foreign-jurisdiction dependency to every page load — the " +
			"third party sees the visitor's IP, their browsing context, and " +
			"often a session cookie. Reducing non-EEA third parties shrinks the " +
			"export surface citizen data crosses on its way to the screen.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			thirdParties := map[string]string{} // host -> finding id
			for _, f := range findings {
				if f.ProbeID != "http.third_party" {
					continue
				}
				thirdParties[strings.ToLower(f.Subject)] = f.ID
			}
			if len(thirdParties) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no http.third_party finding — HTTP probe found no third parties or did not run"}
			}
			var jt jurisdictionTally
			for _, f := range findings {
				if f.ProbeID != "ip.asn" {
					continue
				}
				host := strings.ToLower(f.Subject)
				if _, ok := thirdParties[host]; !ok {
					continue
				}
				jt.add(f)
			}
			if jt.seen == 0 {
				return jt.noCountryResult(
					"third parties served by",
					"third parties on an AS operator with no country (anycast?) — jurisdiction undetermined",
					"third parties found but no ip.asn lookup — IP probe did not run or no --geoip database",
				)
			}
			seen, inEEA := jt.seen, jt.inEEA
			evidence := jt.evidence
			for _, id := range thirdParties {
				evidence = append(evidence, id)
			}
			// Lead with the observed non-EEA vendors ("loads from Google,
			// Cloudflare (non-EEA) — …") when the scanner named any; the
			// all-EEA case keeps its clean count with no scary lead.
			lead := func(detail string) string {
				if who := joinAnd(observedNonEEAVendors(findings)); who != "" {
					return fmt.Sprintf("loads from %s (non-EEA) — %s", who, detail)
				}
				return detail
			}
			switch {
			case inEEA == seen:
				return assessor.RuleResult{
					Score:    models.ScoreSoeverein,
					Verdict:  fmt.Sprintf("all %d third-party hosts resolve in the EEA", seen),
					Evidence: evidence,
				}
			case inEEA > 0:
				return assessor.RuleResult{
					Score:    models.ScoreVoldoende,
					Verdict:  lead(fmt.Sprintf("%d of %d third-party hosts resolve in the EEA", inEEA, seen)),
					Evidence: evidence,
				}
			default:
				return assessor.RuleResult{
					Score:    models.ScoreAfhankelijk,
					Verdict:  lead(fmt.Sprintf("all %d third-party hosts resolve outside the EEA", seen)),
					Evidence: evidence,
				}
			}
		},
	}
}

// observedApexFront returns the edge name and country the scanner's
// http.cdn_front synthesis observed when the apex is fronted, so the
// hyperscaler rule can lead its verdict with the named front ("apex
// fronted by Cloudflare (US) — …"). Tolerant of the attribute shapes;
// returns empty edge when the apex is not fronted or no map was produced.
func observedApexFront(findings []models.Finding) (edge, country string) {
	for _, f := range findings {
		if f.ProbeID != "http.cdn_front" {
			continue
		}
		if fronted, _ := f.Attributes["fronted"].(bool); !fronted {
			return "", ""
		}
		edge, _ = f.Attributes["edge"].(string)
		country, _ = f.Attributes["country"].(string)
		return edge, country
	}
	return "", ""
}

func noUSHyperscaler() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.technologie.no_us_hyperscaler",
		Dimension:   models.DimensionTechnologie,
		Description: "Apex and third-party hosts are not routed via known US hyperscalers.",
		Rationale: "AWS, Google, Microsoft, Cloudflare, Akamai, and Fastly are US-" +
			"headquartered providers subject to the CLOUD Act, which obliges them " +
			"to surface customer data on a US warrant regardless of where the " +
			"data physically sits. A site whose apex or third-party traffic " +
			"flows through one of these providers operates inside that " +
			"jurisdictional reach, which the DICTU framework treats as a " +
			"sovereignty dependency even when individual data centres are in EU " +
			"countries.",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var matched []string
			var evidence []string
			for _, f := range findings {
				if f.ProbeID != "ip.asn" {
					continue
				}
				org := strings.ToLower(stringFromAttr(f.Attributes, "organisation"))
				if org == "" {
					continue
				}
				for _, needle := range knownUSHyperscalerOrgs {
					if strings.Contains(org, needle) {
						matched = append(matched, stringFromAttr(f.Attributes, "organisation"))
						evidence = append(evidence, f.ID)
						break
					}
				}
			}
			if len(evidence) == 0 {
				// No ip.asn at all? Return no-evidence. Any ip.asn
				// seen without a hyperscaler match means we have
				// evidence of absence.
				for _, f := range findings {
					if f.ProbeID == "ip.asn" {
						return assessor.RuleResult{
							Score:    models.ScoreVoldoende,
							Verdict:  "no known US hyperscaler in the apex or third-party path",
							Evidence: []string{f.ID},
						}
					}
				}
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no ip.asn finding — IP probe did not run"}
			}
			// Lead with the observed apex front when the scanner detected
			// one ("apex fronted by Cloudflare (US); …"), so the verdict
			// names the edge and says it is a front, not just an org in the
			// path. The US-hyperscaler-in-path detail is kept behind it.
			detail := fmt.Sprintf("US hyperscaler in path: %s", strings.Join(dedupe(matched), ", "))
			if edge, country := observedApexFront(findings); edge != "" {
				loc := country
				if loc == "" {
					loc = "country undetermined"
				}
				detail = fmt.Sprintf("apex fronted by %s (%s); %s", edge, loc, detail)
			}
			return assessor.RuleResult{
				Score:    models.ScoreAfhankelijk,
				Verdict:  detail,
				Evidence: evidence,
			}
		},
	}
}

// ---------- Data & AI ----------

func mxPresent() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.data_ai.mx_present",
		Dimension:   models.DimensionDataAI,
		Description: "Domain has configured mail exchangers (routing is knowable).",
		Rationale: "A domain without `MX` records cannot route inbound mail. For an " +
			"organisation that publishes contact addresses on the apex domain, " +
			"missing MX records mean that mail addressed to the organisation " +
			"silently fails — a posture issue (the organisation publishes an " +
			"unreachable contact path) and a data-flow issue (citizen messages " +
			"vanish without a clear delivery report).",
		Match: func(findings []models.Finding) assessor.RuleResult {
			var evidence []string
			for _, f := range findings {
				if f.ProbeID != "dns.mx" {
					continue
				}
				// Skip lookup-error / no-answer rows so a non-resolvable
				// domain does not score voldoende just because the probe
				// emitted meta findings.
				if !assessor.IsEvidenceLike(f) {
					continue
				}
				evidence = append(evidence, f.ID)
			}
			if len(evidence) == 0 {
				return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no dns.mx finding — domain may not accept mail"}
			}
			return assessor.RuleResult{
				Score:    models.ScoreVoldoende,
				Verdict:  fmt.Sprintf("%d MX records published", len(evidence)),
				Evidence: evidence,
			}
		},
	}
}

func oidcFederation() assessor.Rule {
	return assessor.Rule{
		ID:          "wand.data_ai.oidc_federation",
		Dimension:   models.DimensionDataAI,
		Description: "Identity federation endpoints are sovereign. Requires the egress probe (not yet landed).",
		Rationale: "OpenID Connect (OIDC) federation lets users sign in with credentials " +
			"managed by an external identity provider. Each authentication round-trip " +
			"sends the user's identity, organisation, and login context through that " +
			"provider's infrastructure; a non-sovereign IdP becomes the single most " +
			"sensitive third-party dependency on every protected endpoint. The rule " +
			"will fire once the egress probe surfaces the configured `OIDC_ISSUER` " +
			"endpoints; today it returns no evidence as a placeholder.",
		Match: func(_ []models.Finding) assessor.RuleResult {
			// Intentional no-evidence until add-egress-probe ships.
			return assessor.RuleResult{Score: models.ScoreOnbekend, Verdict: "no evidence — requires add-egress-probe"}
		},
	}
}

// ---------- helpers ----------

func stringFromAttr(attrs map[string]any, key string) string {
	if attrs == nil {
		return ""
	}
	if s, ok := attrs[key].(string); ok {
		return s
	}
	return ""
}

func stringsFromAttr(attrs map[string]any, key string) []string {
	if attrs == nil {
		return nil
	}
	switch v := attrs[key].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	}
	return nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
