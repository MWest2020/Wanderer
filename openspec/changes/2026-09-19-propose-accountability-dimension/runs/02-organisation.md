# Habitat run 02 — organisation (tasks 3.1–3.3)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(design.md "Expected registrant matching", specs/scanner/spec.md
"Organisations declare expected registrant names").

## Scope — ONLY these tasks
- [ ] 3.1 New store migration (append after `add_ui_sessions` in
  `internal/store/migrations.go`): `organisations.expected_registrant`,
  JSON array of names, default `'[]'`, NOT NULL. `models.Organisation`
  gains `ExpectedRegistrant []string` (`json:"expected_registrant"`);
  `UpsertOrganisation` / `GetOrganisation*` / `ListOrganisations` read and
  write it. An upsert without names SHALL NOT wipe names already stored
  unless the caller sets them explicitly — decide and test this.
- [ ] 3.2 `wanderer org add <slug> --expected-registrant NAME` (repeatable
  flag; `org add` already upserts). `wanderer org show` prints the list.
- [ ] 3.3 At scan time the scanner (`internal/scanner/scanner.go`, it holds
  the Store) records one finding `config.expected_registrant` for the
  scan's organisation: attribute with the list of names, also when the
  list is empty (empty list, not omitted). Resolve the organisation the
  same way the scan is already attached to one (target → organisation,
  default org as fallback). The assessor stays a pure function of
  findings; no assessor changes in this run.

Tests: migration on an existing DB (old rows get `[]`), store round-trip,
CLI flag parsing (repeatable), scanner emits the finding with names and
with an empty list, re-assessing an old scan uses the names recorded in
that scan (the scenario "Changing the list does not rewrite history").

Done = these three checkboxes checked, and 3.1–3.3 ticked in tasks.md.

## Out of scope for this run
No rules, no probes, no UI, no whois changes. Follow-up 7.4 is not yours.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/, see openspec/config.yaml);
`openspec validate 2026-09-19-propose-accountability-dimension --strict`
green.
