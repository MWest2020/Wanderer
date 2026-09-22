# Run report — 09 wrap-up (tasks 9.1–9.2)

## What changed
- `docs/reference/assessor.md`: new "The `accountability` dimension"
  section (the five wand.accountability rules + the two operationeel
  rules this change adds, the two YAML lists, the `.nl` n.v.t. case)
  and a new "Reason codes" section (class `structural`/`gap`, subject
  `target`/`scanner`, the four seeded codes, `NotApplicable`
  aggregation). Rule-set table extended with all seven new rule IDs.
  "Extending the rule set" now mentions `RuleResult.Reason` /
  `reasonRegistry`.
- `docs/reference/findings.md`: `accountability` added to the
  dimension-hint table; HTTP probe table + prose gained
  `http.securitytxt` / `http.securitytxt.unavailable`; WHOIS/RDAP
  table + prose gained `whois.registrant_identity`, `whois.reseller`,
  `whois.status`, `whois.expiry`, and the "entities parsed
  recursively" note; three new sections — SOA probe (`dns.soa`), the
  variants probe (`http.variants`), NS-holder findings
  (`whois.ns_holder`, scanner-level, not a probe), and organisation
  config findings (`config.expected_registrant`).
- `docs/explanation/accountability-boundaries.md` (new): the RDAP
  fields wish-list (actor/escalation relationships, ISO2-prefixed
  subject identifiers) and why they're IETF/ICANN work, not Wanderer
  code; the `.nl` passive ceiling (SIDN redacts registrant data and
  publishes no expiry for every `.nl` domain) and the three
  work-arounds considered and rejected (registrar APIs, scraping,
  live KVK lookups). Linked from `docs/index.md` under Explanation.
- `CHANGELOG.md`: one `### Added` entry for the accountability
  dimension + reason-code mechanism, one `### Changed` entry for the
  `DICTUDimensions` → `WandDimensions` rename and the
  `organisations.expected_registrant` column, both under
  `[Unreleased]`.
- `tasks.md`: ticked 9.1 and 9.2.

No code, spec deltas, or task-ref-adjacent files touched. Task 9.3
(attribution wording) and 9.4 (archiving) untouched, per scope.

## Verification
- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./...` — all packages `ok`.
- `openspec validate 2026-09-19-propose-accountability-dimension
  --strict`: **fails**, on the same two pre-existing, unrelated
  requirements already flagged in
  `runs/08c-anchor-ids.report.md` — `assessor/spec.md: "Rule results
  carry generic reason codes"` and `web-ui/spec.md: "Answer-sheet
  copy lives in one Dutch string table"`, both reported as missing a
  literal SHALL/MUST although each contains several ("Every code
  SHALL be registered…", "SHALL ship in Dutch from one per-rule
  string table…") — an openspec-CLI parsing quirk on those two
  requirement bodies specifically, not a regression from this run.
  Confirmed via `git log` that neither `specs/assessor/spec.md` nor
  `specs/web-ui/spec.md` has changed since commits
  `5e5fa41`/`dc0f38c`, well before this run, and this run made no
  edits under `openspec/specs/` or the change's `specs/` deltas.
  Not in task 9.1/9.2's scope (docs + CHANGELOG only) — noted here,
  not fixed.
