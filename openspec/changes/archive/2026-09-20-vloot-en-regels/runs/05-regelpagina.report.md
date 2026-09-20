# Run report — 05 de regelpagina met advies (tasks 5.1–5.3)

## 5.1 — waarneming en drempels op de regelpagina
`Description`/`Rationale`/`RuleTargetRows` already existed. Two things
did not: "welke waarneming hij gebruikt" and the Thresholds display.

- Added `Observation string` to `assessor.Rule` (`internal/assessor/rule.go`),
  same pattern as `Rationale`: one hand-written line naming the
  finding(s)/probe(s) `Match` reads (e.g. "The RDAP registrant lookup
  (`whois.registrant`), read for its reported country."). Filled it on
  all 28 wand rules. `TestEveryRuleHasObservation` (wand/rules_test.go)
  enforces it, mirroring `TestEveryRuleHasRationale`.
- `reportingRuleView` (ui.go) gained `Observation` and `Thresholds
  []reportingThresholdView`, populated from `rule.Observation` and
  `rule.Thresholds` (run 04). `reporting_rule.tmpl` renders both; a
  rule with no Thresholds shows "Deze regel kijkt of iets aanwezig is
  — er is geen numerieke grens." instead of an empty heading.

## 5.2 / 5.3 — de handeling
Followed the accountability_nl.yaml pattern literally: a new top-level
`handelingen:` map in that same YAML file (not a new file — the
proposal says "in dezelfde ene tekstentabel"), one Dutch sentence per
rule ID with a single `{domein}` parameter. `HandelingFor(ruleID)`
(accountability_nl.go) reads it; `TestEveryRuleHasHandeling`
(wand/rules_test.go) fails the build if any of `DefaultRules()` is
missing an entry — same shape as `TestEveryRuleHasRationale`.

This table has entries for all 28 wand rules, including the seven that
already had per-outcome `remediation` copy in `rules:` (five
accountability rules + domain_expiry + variant_convergence). Those two
sources serve different renderers and were kept separate rather than
merged: the existing `rules[...].remediation` map is evidence-rich
(uses `{proxy}`, `{reseller}`, `{domains}`, `{path}` — params only
available where a Finding's attributes are already in scope, i.e. the
accountability answer sheet and the flow answers) and keeps driving
those two views unchanged ("de accountability-regels hebben dat al").
The new `handelingen` map is the single-param, always-available
fallback used by contexts that only have a domain string to work
with — the rule page's per-target rows and the reasoning page's
generic per-dimension table. Rendering both sourced from the same rule
ID would have meant either downgrading the richer answer-sheet text or
leaving `{reseller}`/`{path}`-style placeholders unresolved on the
rule page (fillParams renders an unmatched placeholder literally,
by design — silent-drop would be worse, but visible garbage in
production copy is worse still), so I kept the two intentionally
separate rather than paper over the mismatch.

Wired the handeling into every place a falling ("afhankelijk") verdict
renders:
- Rule page (`reportingRuleHandler`): per row, when `Score ==
  afhankelijk`, filled with that row's `Domain`.
- Reasoning page's generic per-dimension table (`assessmentHandler`):
  same, filled with the scan's subject domain.
- Reasoning page's sovereignty-flow answers (`BuildFlowAnswers`): this
  previously showed **no** remediation at all for any of its seven
  rules (`AccountabilityAnswer.Remediation` was simply never set there)
  even though the template already supported it. Gave the function a
  `domain` parameter and filled it from the new table — a gap 5.2
  exposed, not a regression from this run.
- Accountability answer sheet (`BuildAccountabilityAnswers`): untouched,
  it already had this.

Every one of the 28 rules got an honest, operator-actionable
handeling — switch registrar/CA/host/DNS-vendor/registry/telemetry
agent/IdP, add a missing record, enable a header, renew something.
None needed the "buiten de macht van de operator" carve-out the
task-ref allows for: even the most drastic ones (move hosting off a US
hyperscaler, move away from a redacting TLD's registry) are decisions
an operator can make, just not cheap ones. Flagging this explicitly
since the task-ref asked me to call it out either way.

## Tests
- `wand/rules_test.go`: `TestEveryRuleHasObservation`,
  `TestEveryRuleHasHandeling` (new).
- `ui/flow_answers_test.go`: extended the existing afhankelijk-scoring
  case to assert a non-empty `Remediation` with `{domein}` filled in;
  updated all three `BuildFlowAnswers` call sites for the new `domain`
  parameter.
- Existing `ui/ui_test.go` reporting-rule and assessment-page tests
  still pass unmodified (they assert on soeverein-scored fixtures, so
  the new remediation column adds nothing for them to break).

## Verification
- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./...` — all packages `ok`.
- `openspec validate 2026-09-20-vloot-en-regels --strict` — valid.

## Tasks.md
Ticked 5.1, 5.2, 5.3.
