# Design notes: accountability dimension

## External systems and failure modes

| System | Used for | Failure mode | Handling |
| --- | --- | --- | --- |
| rdap.org (or configured RDAP base) | registrant identity, reseller, status, expiry; NS holder | 404 (no RDAP for TLD), 429 rate-limit, redacted vcards, schema drift per registry, reseller nested under the registrar entity | every gap emits `*.unavailable` / `redacted` findings; rules score **onbekend**, never guess. Entities are parsed recursively (`entities[].entities[]`). Cache per registrable domain within a scan so N nameservers on one provider cost one lookup. |
| Authoritative DNS | SOA (MNAME/RNAME) | NXDOMAIN, timeout, lame delegation | `dns.soa.unavailable`; soa_rname scores onbekend (reason class `gap`) |
| Target webserver | `/.well-known/security.txt`, variant probes | timeout, 4xx/5xx, HTML-instead-of-text, redirect loops | absent ≠ error: 404 is a *valid observation* (scores afhankelijk); parse failure emits `http.securitytxt` with `parseable=false`. Variants: see Politeness. |

## Reason codes (generic mechanism)

A rule result may carry a `reason` code. The mechanism is shared by
every pack and dimension — accountability is its first user, the
standards change its second. One registry (a single Go table in
`internal/assessor`) maps each code to exactly one class:

| class | renders as | dimension aggregation |
| --- | --- | --- |
| `structural` | "n.v.t." + the code's sentence | excluded: neither worst-score nor completeness denominator |
| `gap` | "onbekend" + the code's sentence | counts as a hole: rationale onbekend, completeness drops |

The score stays on the four-value scale; a `structural` rationale
carries score `onbekend` with its reason, so no reader has to learn a
fifth score. Codes seeded by this change:

| code | class | used by |
| --- | --- | --- |
| `registry_redacted` | structural | registrant_identifiable on a TLD in `registry_redaction.yaml` |
| `not_published_by_registry` | structural | domain_expiry when the registry publishes no expiration event |
| `probe_unavailable` | gap | any rule backed only by a `*.unavailable` finding |
| `scanner_no_ipv6` | structural (subject: scanner) | variant_convergence paths the scanner could not test because it has no IPv6 route |

Each code also names its **subject**: `target` (default — the verdict
says something about the scanned domain) or `scanner` (the verdict
says something about Wanderer's own environment). A scanner-subject
code never renders as a property of the target: the UI shows it as an
operator environment warning ("scanner heeft geen IPv6 — v6-paden
niet gemeten") beside the report, not as an answer.

The standards change adds `not_measured` and `measurement_stale`
(both `gap`) to the same table. An unknown code is a test failure,
not a runtime default.

A dimension whose every rule is `structural` renders "n.v.t." and is
left out of the target's overall score. A rationale's `reason` is
additive JSON (`omitempty`); assessments stored before this change
load unchanged.

## Politeness (variants probe)

8 paths (apex/www × v4/v6 × http/https), redirect depth ≤ 5 per path,
**hard budget of 24 connections per target** across all paths,
sequential, one User-Agent identifying Wanderer, no retries, inside
the scan's global timeout. Every hop passes the existing SSRF guard
(`internal/probe/ssrf.go`); a refused hop is recorded as `refused`,
never followed. When the budget runs out the remaining paths are
recorded as `not_followed_budget` and the rule scores on what was
observed. Allowed optimisation: a chain stops as soon as it lands on
an origin already verified in this run. If the scanner host has no
IPv6 route, v6 paths are recorded as `not_tested` with reason
`scanner_no_ipv6` (structural, subject scanner) — a dead v6 family on
the target is only claimed when the scanner could have reached it, and
the missing measurement is reported to the operator as an environment
warning, not charged to the target.

At 24 connections this is still less traffic than one browser page
load. If an operator scans third-party domains they do not own, the
existing scan-consent posture applies unchanged.

## Privacy proxy vs registry redaction

Two separate YAML lists next to the rule (pattern:
`package_vendors.yaml`), two separate outcomes:

- `registry_redaction.yaml` — per TLD, the registry that redacts
  registrant data for everyone (seed: `nl: SIDN`). Target TLD on this
  list → onbekend, reason `registry_redacted`, verdict "het register
  publiceert registrantgegevens niet voor .nl". Checked **first**: on
  such a TLD a "REDACTED FOR PRIVACY" vcard is the registry's policy,
  not the organisation's choice.
- `privacy_proxies.yaml` — commercial proxy services ("Domains By
  Proxy", "Whois Privacy", …). Match → afhankelijk.

Where the response carries an RFC 9537 `redacted` array (SIDN's does,
see `fixtures/rdap-rijksoverheid.nl-20260919.json`), the probe records
it as evidence of registry-level redaction; the TLD list stays the
deciding input so the outcome does not hinge on one registry's
response shape.

Misses on the proxy list are acceptable: an undetected proxy scores as
identifiable-but-unmatched, which still surfaces via the name
comparison.

## Expected registrant matching

`expected_registrant` is a list of names on the **organisation**
(`organisations.expected_registrant`, JSON array; set with
`wanderer org add <slug> --expected-registrant NAME` — `org add` already upserts; flag repeatable). The
scheduler config is unchanged: it already names the organisation by
slug.

The assessor stays a pure function of findings (assessor spec:
"the same Findings produce the same Assessment"). The scanner
therefore records the organisation's list at scan time as a
`config.expected_registrant` finding. That also makes the report
auditable: it shows which names were expected *when the scan ran*,
not what the organisation says today.

Comparison is case-insensitive with legal-form suffixes (B.V., N.V.,
Stichting) normalised away. No fuzzy matching in v1 — a miss must be
explainable in the UI in one sentence.

## Copy: one Dutch string table

All answer-sheet copy — question, verdict text per outcome
(ja/nee/onbekend/n.v.t.), one remediation line per failing outcome —
lives in one embedded file, `internal/assessor/wand/accountability_nl.yaml`,
keyed by rule ID. Rules fill named parameters (`{registrant}`,
`{date}`, `{path}`); templates never hard-code copy. A load-time test
fails when a rule lacks an entry or an entry names a parameter the
rule does not supply. Spec scenarios stay in English (repo
convention); shipped copy is Dutch.

## Overall score over assessed dimensions only

Old scans have no `accountability` entry; a dimension that is entirely
`structural` is n.v.t. Both are left out of the target's overall
score, and the overview names the dimensions it was computed over
("over 5 van 6 dimensies"). Never counted as failure, never averaged
in silently. This extends the existing "worst-dimension score
excludes onbekend dimensions" requirement.

## The clever valkuil

Two, both tempting:

1. **Live KVK lookups** (or registrar APIs) to "prove" the
   registrant. Adds an authenticated external dependency, rate
   limits, and a data-quality rabbit hole. Operator-declared names are
   boring, auditable, and correct for the deployment model
   (organisations scan themselves — they *know* their own name). For
   `.nl` the honest answer is "n.v.t.", not a work-around.
2. **SMTP RCPT probing** to verify RNAME/security.txt mailboxes
   "without sending mail". It shows up in abuse logs and turns a
   sovereignty scanner into something that looks like recon. Existence
   + MX presence is the honest passive ceiling; say so in the verdict
   text ("het maildomein bestaat en ontvangt mail — aflevering niet
   gecontroleerd").

## UI direction (binding for the web-ui delta)

Accountability findings name organisations, mailboxes, and people —
this dimension will be read by exactly the "Tourist" persona
(ADR-0017): a bestuurder or CISO asking *"if this breaks tonight, who
do I call, and does that route work?"* Therefore:

- The report page renders accountability as **five plain-language
  questions with ja/nee/onbekend/n.v.t. answers**, evidence collapsed
  beneath, one concrete remediation line per failing rule.
- Rule IDs, RDAP jargon, and RFC numbers live in the expanded
  evidence, never in the headline.
- n.v.t. and onbekend look different from each other and from "nee".
- For `.nl` fleets two of five questions are structurally n.v.t.; the
  UI review (task 4.3) checks that the other four still carry the
  story, or the dimension looks empty to exactly its audience.
- Wordsworth is the in-house reference for this register: study its
  tone and interaction patterns (headline → expandable evidence →
  next action) before building. Intuitive beats complete: if a
  question can't be phrased so a non-specialist understands the
  answer, the rule's verdict text is wrong, not the reader.

## Design gate outcome (task 1.1, 2026-09-19)

| Question | Decision |
| --- | --- |
| Variants politeness | 8 paths, ≤ 5 hops/path, 24 connections/target, stop on already-verified origin, SSRF guard on every hop, `scanner_no_ipv6` so the scanner's own blind spot is never blamed on the target |
| SSRF guard reuse | reuse `internal/probe/ssrf.go` as-is per hop; no second guard |
| YAML list location | `internal/assessor/wand/{privacy_proxies,registry_redaction}.yaml`, embedded, loader tested like `package_vendors.yaml` |
| expected_registrant shape | organisation-level list, recorded per scan as a finding; no per-target override |
| Unknown vs not applicable | generic reason codes with class `structural`/`gap`; four-value scale intact |
| Dimension list | `WandDimensions`, iterated, never counted |
| Copy | Dutch, one string table, no i18n machinery |

Checked against the code at `682807f`: `DimensionHint`
(`pkg/models/finding.go`), `DICTUDimensions` + `scoreDimension`
(`internal/assessor/engine.go`), `rdapEntity` without nested entities
(`internal/probe/whois/whois.go`), `organisations` table (migration
`add_organisations`), `WorstScore` (`internal/ui/aggregate.go`).
