---
status: draft
last_reviewed: 2026-09-20
---

# Assessor

The assessor turns the Findings produced by a scan into a structured
`Assessment`: per-dimension score, a completeness flag, and a list of
per-rule rationales that cite the Findings they drew their verdict
from.

This document covers: running the assessor, reading the output, and
how to extend the **wand** rule set.

## Inspired by

Wanderer's first-party rule pack is named **wand** (Wanderer-NL).
Its rule semantics were inspired by the Dutch government's
publicly available *Toetsingsinstrument Soevereiniteit
Clouddiensten*, published by **DICTU** (Dienst ICT Uitvoering, EZK).
The rule authoring, the implementation, and the ongoing
maintenance are Conduction's; DICTU does not endorse, certify,
or otherwise sanction Wanderer or the wand pack. See
[ADR-0011](../explanation/adr/0011-rename-dictu-to-wand.md) for the
rationale behind the rename.

## Running it

From the CLI:

```sh
wanderer assess <scan-id> [--framework wand|eucsf|both] [--format text|markdown|json] [--db wanderer.db] [--persist=false]
```

- `--framework` defaults to `wand`. `eucsf` runs the EU Cybersecurity
  Framework / SEAL pack instead; `both` runs both packs and persists
  one Assessment record per framework, tagged on `Assessment.Framework`.
  `dictu` is accepted as a deprecated alias for `wand` for one
  release (it prints a stderr deprecation warning).
- `--format` defaults to `text`. Markdown is the intended operator-
  facing format; JSON is for tooling.
- `--persist=false` suppresses the side-effect of writing the
  assessment to the store — useful for dry runs.

Over HTTP:

```sh
curl -XPOST http://localhost:8080/scans/<scan-id>/assessments
curl        http://localhost:8080/assessments/<assessment-id>
```

`POST` always persists and always returns the assessment it just
produced. Each POST creates a new record — previous assessments for
the same scan remain retrievable by their own IDs.

## Reading a score

Each wand dimension receives one of:

| Score         | Meaning                                                   |
| ------------- | --------------------------------------------------------- |
| `soeverein`   | The best verdict the rule set issues for this dimension.  |
| `voldoende`   | Adequate. No red flags, but not actively sovereign.       |
| `afhankelijk` | Low sovereignty — dependency on a non-sovereign party.    |
| `onbekend`    | No evidence-backed verdict could be reached.              |

The dimension score is the **worst** score across every
evidence-backed rule in that dimension. One rule scoring `afhankelijk`
drags the whole dimension down even if four others scored `soeverein`
— because one material dependency is one material dependency.

Next to the score, a `Completeness` flag tells you how much of the
rule set was answerable:

| Completeness | Meaning                                                       |
| ------------ | ------------------------------------------------------------- |
| `complete`   | Every rule in this dimension had evidence to work with.       |
| `partial`    | Some rules evaluated; others did not because data was absent. |
| `incomplete` | No rule in this dimension had any evidence.                   |

A dimension with no rules at all (for example, `mens` in the current
rule set) is rendered as `n/a` in the summary table.

## Rule set (MVP)

The wand rule set today is deliberately small. Most rules are Go
functions in `internal/assessor/wand/rules.go`; the accountability
rules live in `internal/assessor/wand/accountability_rules.go`.

| Rule ID                                     | Dimension       |
| -------------------------------------------- | --------------- |
| `wand.juridisch.cert_issuer_eea`            | juridisch       |
| `wand.juridisch.apex_ip_eea`                | juridisch       |
| `wand.juridisch.mx_vendor_jurisdiction`     | juridisch       |
| `wand.juridisch.registrar_jurisdiction`     | juridisch       |
| `wand.operationeel.cert_validity`           | operationeel    |
| `wand.operationeel.dns_redundancy`          | operationeel    |
| `wand.operationeel.caa_restricts_issuance`  | operationeel    |
| `wand.operationeel.domain_expiry`           | operationeel    |
| `wand.operationeel.variant_convergence`     | operationeel    |
| `wand.technologie.third_parties_eea`        | technologie     |
| `wand.technologie.no_us_hyperscaler`        | technologie     |
| `wand.data_ai.mx_present`                   | data_ai         |
| `wand.data_ai.oidc_federation`              | data_ai         |
| `wand.accountability.registrant_identifiable` | accountability |
| `wand.accountability.no_reseller`           | accountability  |
| `wand.accountability.soa_rname`             | accountability  |
| `wand.accountability.securitytxt`           | accountability  |
| `wand.accountability.ns_holder_transparent` | accountability  |
| `wand.standards.dnssec`                     | standards       |
| `wand.standards.mail_auth`                  | standards       |
| `wand.standards.starttls_dane`              | standards       |
| `wand.standards.ipv6`                       | standards       |
| `wand.standards.rpki`                       | standards       |
| `wand.standards.tls_config`                 | standards       |

The `oidc_federation` rule always returns no evidence until the
egress probe lands — it is listed so the reader sees the future
coverage the dimension is waiting on.

The `mens` dimension has no rules. It appears in the output as
`onbekend (n/a)`. This is explicit, not an omission: the MVP scanner
observes perimeter posture, not human processes.

## The `accountability` dimension

`accountability` is a **wand-native** dimension: it has no DICTU
counterpart in the *Toetsingsinstrument Soevereiniteit
Clouddiensten*, and none is expected — accountability (who is
answerable for a piece of infrastructure, and can you actually reach
them) is a gap in the DICTU instrument itself, not an oversight in
wand's coverage of it. `models.DimensionAccountability` and the
dimension list `assessor.WandDimensions` (the DICTU five plus
`accountability`, replacing the earlier `DICTUDimensions` name) make
this explicit. Callers iterate `WandDimensions`; none may assume a
fixed length or that every entry maps to a DICTU criterium.

The five `wand.accountability.*` rules answer, in plain language:

1. **`registrant_identifiable`** — is the RDAP registrant identifiable,
   and does it match a name the organisation declared
   (`organisations.expected_registrant`, set with `wanderer org add
   --expected-registrant NAME`, repeatable)?
2. **`no_reseller`** — is the registrar relationship direct, with no
   reseller layer in between?
3. **`soa_rname`** — does the zone's SOA RNAME mailbox domain exist and
   accept mail (existence and MX presence only — delivery itself is
   never probed; SMTP verification would cross the passive/abuse
   boundary)?
4. **`securitytxt`** — does the target publish a parseable,
   RFC 9116 `/.well-known/security.txt` with a current `Contact` and
   `Expires`?
5. **`ns_holder_transparent`** — is the organisation behind each
   authoritative nameserver identifiable via RDAP?

Two related rules score under `operationeel` instead, because they
are continuity properties, not accountability ones: `domain_expiry`
(will the registration lapse unexpectedly) and `variant_convergence`
(do all eight apex/www × IPv4/IPv6 × http/https paths converge on one
canonical HTTPS origin).

All seven rules' answer-sheet copy — question, verdict text per
outcome, remediation line — lives in one Dutch string table,
`internal/assessor/wand/accountability_nl.yaml`, keyed by rule ID. A
load-time test fails when a rule lacks an entry or a template names a
parameter the rule does not supply. There is no i18n machinery: a
second language is a second table.

Two YAML lists, embedded and loaded the same way as
`package_vendors.yaml`, back `registrant_identifiable`:

- `internal/assessor/wand/registry_redaction.yaml` — per-TLD
  registries that redact registrant data for *every* domain under
  that TLD as blanket policy (seeded with `nl: SIDN`). Checked
  **first**, independent of the RDAP vcard content, because on such a
  TLD a "REDACTED FOR PRIVACY" vcard is the registry's policy, not the
  registrant's choice.
- `internal/assessor/wand/privacy_proxies.yaml` — commercial proxy
  services ("Domains By Proxy", "Whois Privacy Protection Service", …)
  that publish themselves as the registrant. A miss on this list is
  acceptable: an undetected proxy still surfaces as
  identifiable-but-unmatched via the expected-registrant name
  comparison.

For `.nl` targets, `registrant_identifiable` and `domain_expiry`
render **n.v.t.** — SIDN publishes neither registrant data nor an
expiration event for any `.nl` domain, so two of the five
accountability questions are structurally unanswerable there, not a
probe failure. See
[Accountability: the passive ceiling](../explanation/accountability-boundaries.md)
for why Wanderer does not work around this.

## The `standards` dimension

`standards` is a second **wand-native** dimension — no DICTU
counterpart, same as `accountability` — that scores the Dutch
comply-or-explain standards list (Forum Standaardisatie) as measured
by **Internet.nl**: DNSSEC, SPF/DKIM/DMARC, STARTTLS/DANE, IPv6, RPKI,
and TLS configuration. Wanderer does **not** reimplement any of these
probes itself; see
[Permanent non-goals: what Wanderer will never probe itself](../explanation/internetnl-non-goals.md)
for why, and
[Feed Internet.nl results from CI](../how-to/internetnl-ci.md) for how
to get a measurement into Wanderer.

The dimension has no evidence at all until an operator runs
`wanderer import internetnl <findings-file>` — a **file import**, not
a probe. `internal/scanner/netnlimport.go` parses a `netnl-findings/v1`
document (produced by the standalone **netnl** tool) and persists
`internetnl.<type>.<test>` Findings under a scan of kind `import`,
tagged `SourceModus = "import"` (`models.SourceModusImport`) — see
[findings.md](findings.md#internetnl-import-findings--internalscannernetnlimportgo)
for the Finding shape. No network call happens on Wanderer's side; all
probing is Internet.nl's own, under its own measurement policy.

The six `wand.standards.*` rules
(`internal/assessor/wand/standards_rules.go`) each map one or two
Internet.nl API categories onto a verdict, verbatim — never
recomputing Internet.nl's own percentage score (the "clever valkuil"
design.md warns against: weights change between Internet.nl releases,
and a home-grown score would disagree with the public report the
target's owner can open in a browser):

| Rule ID                          | Internet.nl categor(y/ies)         |
| --------------------------------- | ----------------------------------- |
| `wand.standards.dnssec`          | `web_dnssec`, `mail_dnssec`         |
| `wand.standards.mail_auth`       | `mail_auth`                         |
| `wand.standards.starttls_dane`   | `mail_starttls`                     |
| `wand.standards.ipv6`            | `web_ipv6`, `mail_ipv6`             |
| `wand.standards.rpki`            | `web_rpki`, `mail_rpki` (includes the nameserver-RPKI subtests the API itself groups under the same category) |
| `wand.standards.tls_config`      | `web_https`                         |

Verdict mapping, per rule, from the relevant category's per-test
`status` values:

- every relevant test `passed` → **soeverein**
- a genuine mix of passed and failed/warning → **voldoende**
- tested and none passed → **afhankelijk**
- no relevant Finding, or every relevant test `not_tested`/`error` →
  **onbekend**, reason `not_measured`
- every relevant Finding older than `standards.max_age` (default 30
  days) → **onbekend**, reason `measurement_stale`

`not_tested` and `error` never count as measured evidence and never
score positively: a domain with no mail server configured at all
returns mostly `not_tested` for its mail-security tests, and scoring
that as sovereign would be the single most dangerous mistake this
dimension could make — "never measured" must never read as "measured
and fine". `error` (the measurement itself broke, e.g. an Internet.nl
worker crashed on that test) is likewise never treated as a failure of
the target.

Web and mail measurements for the same target import as two separate
`import`-kind scans (one `type: "web"`, one `type: "mail"`); a
perimeter scan carries neither on its own. `Store.FindingsForAssessment`
(`internal/store/netnlimport.go`) correlates a target's latest web and
latest mail import alongside whichever scan is actually being
assessed, so `wand.assess`/`wanderer assess`/the API/the MCP tools all
see both halves of the evidence regardless of which scan kind they
were called against — a fresh import of one type supersedes only that
type's findings, never the other's.

Every standards rule declares `standards.max_age` as a `Threshold`
(`StandardsMaxAgeDays` / `StandardsMaxAge`,
`internal/assessor/wand/standards_rules.go`) so the staleness boundary
is visible on the rule page, not only in code.

## Reason codes

A `RuleResult` (and the `models.Rationale` it becomes) may carry an
optional `Reason` — a machine-readable code explaining *why* a rule
could not produce an ordinary evidence-backed verdict. The mechanism
is generic: any pack, any dimension, may use it. Accountability is
its first user.

A rationale that carries a reason always scores `onbekend` on the
existing four-value scale — a reason never introduces a fifth score —
but the code additionally classifies **how the dimension aggregation
should treat it**:

| Class        | Renders as                       | Dimension aggregation                                                            |
| ------------ | --------------------------------- | ----------------------------------------------------------------------------------- |
| `structural` | "n.v.t." + the code's sentence   | Excluded from both the worst-score computation and the completeness denominator.    |
| `gap`        | "onbekend" + the code's sentence | Counts as a missing observation, same as an evidence-less rule.                     |

A dimension whose every registered rule carries a `structural` reason
is marked `NotApplicable` on its `DimensionScore` — explicit, not
merely derivable by re-scanning the rationale list — and is left out
of the target's overall score, the same way an all-`onbekend`
dimension already was. The overview names how many dimensions the
overall score was computed over (e.g. "over 5 van 6 dimensies") so a
`.nl` fleet with a structurally n.v.t. sub-slice of accountability
never reads as silently averaged in.

Each code also names a **subject**:

- `target` (the default) — the verdict says something about the
  scanned domain.
- `scanner` — the verdict says something about Wanderer's own
  environment, never the target. A scanner-subject reason must never
  render as a property of the target; the UI shows it as an operator
  environment warning next to the report instead (see the
  `scanner_no_ipv6` row below).

The codes seeded by this change:

| Code                        | Class        | Subject   | Used by                                                                    |
| ---------------------------- | ------------ | --------- | ----------------------------------------------------------------------------- |
| `registry_redacted`         | `structural` | `target`  | `registrant_identifiable`, on a TLD listed in `registry_redaction.yaml`.       |
| `not_published_by_registry` | `structural` | `target`  | `domain_expiry`, when RDAP answered but published no expiration event.        |
| `probe_unavailable`         | `gap`        | `target`  | Any accountability/operationeel rule backed only by a `*.unavailable` finding. |
| `scanner_no_ipv6`           | `structural` | `scanner` | `variant_convergence`, for v6 paths the scanner host could not test because it has no working IPv6 route. |
| `not_measured`               | `gap`        | `target`  | Every `wand.standards.*` rule, when no relevant `internetnl.*` Finding exists or every relevant test came back `not_tested`/`error` (no import yet, or Internet.nl never ran that measurement). |
| `measurement_stale`          | `gap`        | `target`  | Every `wand.standards.*` rule, when the relevant `internetnl.*` Finding(s) are all older than `standards.max_age` (default 30 days). |

`internal/assessor/reason.go` holds the single registry mapping every
code to its class and subject, across every pack. Emitting a code
that is not registered there is a programming error: `ReasonInfo`
panics rather than silently defaulting a class, so an unregistered
code fails a test immediately instead of shipping a wrong render.

## EU CSF / SEAL framework

`internal/assessor/eucsf` ships the SEAL pack — a five-level
sovereignty scale (SEAL0–SEAL4) over the same Findings the wand pack
consumes. The two packs share no rule code; `--framework both` runs
each independently and persists one Assessment per framework.

| Level   | Meaning                                                            |
| ------- | ------------------------------------------------------------------ |
| SEAL0   | No evidence-backed verdict could be produced (no relevant Findings). |
| SEAL1   | Verdict that fails the framework outright — clear non-EU exposure. |
| SEAL2   | Verdict with notable dependence on a non-EU party.                 |
| SEAL3   | Verdict that is adequate — minor or low-impact dependencies only.  |
| SEAL4   | Full sovereignty — every checked surface is EU-resident.           |

The MVP rules are intentionally narrow:

| Rule ID                              | What it checks                                                                                       |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------- |
| `eucsf.sov2.cert_issuer_eu`          | The leaf certificate's issuer country is in the EEA (uses `tls.issuer.issuer_country`).              |
| `eucsf.sov2.apex_jurisdiction`       | The apex domain's resolved IPs sit in EEA jurisdictions (uses `ip.asn.country` joined to `dns.a/aaaa`). |
| `eucsf.sov3.mx_jurisdiction`         | All MX hosts resolve to EEA jurisdictions (uses `dns.mx` + `ip.asn`).                                |
| `eucsf.sov4.operational_eu`          | DNS authority and TLS termination are operated from EEA-resident infrastructure.                     |
| `eucsf.sov6.no_us_hyperscaler`       | Apex and discovered third parties are not hosted on a US hyperscaler (AWS / GCP / Azure / Cloudflare). |

The SEAL level for a dimension is the **worst** rule outcome that
contributed to it; absent evidence collapses to SEAL0 rather than
silently inflating the score. This mirrors the wand "worst wins"
collapse — a reader who sees SEAL3 knows every rule is at SEAL3 or
better, not "on average".

A rule that consults Findings the scanner did not produce returns
SEAL0 with a `Verdict` naming what was missing. Operators who want
to lift a SEAL0 verdict to a real one need to look at the missing
ProbeID rather than re-running the assessor.

[ADR-0009](../explanation/adr/0009-dual-framework-assessor.md) records the
wand/SEAL dual-framework choice; the rationale is that the two
stakeholder groups (Dutch sovereignty reviewers and EU CSF
reviewers) ask the same evidence questions but want different
answer shapes, and we prefer to ship the second shape rather than
translate at read time.

## Evidence and auditability

Every rationale entry cites one or more `Finding.ID` values in its
`Evidence` field. The markdown report renders these as:

```
Evidence: f_abc, f_def
```

A reader who wants to verify the verdict can pull those findings out
of the scan and inspect their `Attributes` and `Evidence` fields — the
raw source material the probe captured. Two reviewers running the
assessor on the same stored scan will get the same verdicts,
rationales, and evidence lists, modulo assessment ID and timestamp.

## Who owns a threshold

A `Threshold` on a `Rule` is for a number the rule's own `Match`
compares against (`domain_expiry`, `dns_redundancy`): the same named
constant feeds both the comparison and the declared `Threshold`, so a
test can catch the two drifting apart.

Some rules only read a judgement flag a probe already set —
`cert_validity`'s `expiring_soon`, `variant_convergence`'s per-path
`status`. There the probe owns the boundary: the 30-day window lives
in `internal/probe/tls`, the 8 paths and 24-connection budget in
`internal/probe/variants`. The probe carries that number on the
Finding alongside the flag (`expiring_soon_threshold`, `path_count`,
`connection_budget`), so a reader can trace the verdict to a concrete
number without the assessor importing the probe package — it stays a
pure consumer of `models.Finding`. The rule still declares a
`Threshold` for the rule page, reusing the same rendering as a
rule-owned one, but its `Explanation` names the observation as the one
applying it, and the value is a hand-synced mirror of the probe's
constant (kept in sync by hand, same as `variantStatusReachable` and
`reasonScannerNoIPv6` already are across this boundary) — `Match`
never compares against it, so it cannot be verified the way
`domain_expiry`'s can.

The rejected alternative was having the probe report only raw data
(`days_left`) and moving the 30-day decision into each rule's `Match`.
That is a cleaner separation on paper, but it pushes the same decision
into every future rule that reads a probe's judgement, and it breaks
any already-stored Finding that assumed the probe had already decided.

## Extending the rule set

Rules are Go functions, not a DSL. To add one:

1. Add a function to `internal/assessor/wand/rules.go` that returns
   an `assessor.Rule` with a stable `ID` (prefix `wand.<dimension>.`),
   a `Dimension`, a one-line `Description`, a `Rationale`, and a
   `Match` function.
2. Wire it into `DefaultRules`.
3. Add a test in `rules_test.go`. The pattern is: fabricate the minimal
   set of Findings the rule needs, call `r.Match`, assert on
   `Score` and `Evidence`.

Rules MUST be total. On missing attributes, return a no-evidence
result (`Score: onbekend`, empty `Evidence`, `Verdict` explaining
what was missing) rather than panicking. The engine's panic recovery
is a safety net, not a design.

When a rule genuinely does not apply to the target (not merely "no
evidence yet"), set `RuleResult.Reason` to a code registered in
`internal/assessor/reason.go` instead of inventing a fifth score —
see "Reason codes" above. Adding a new reason code means adding it to
`reasonRegistry` with its class and subject; an unregistered code
panics at rule-evaluation time, which the accompanying test must
catch.

See [ADR-0004](../explanation/adr/0004-assessor-rule-engine.md) for why rules
are Go functions and not a hot-reloadable DSL.
