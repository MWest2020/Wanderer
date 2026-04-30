## ADDED Requirements

### Requirement: UI dashboard surfaces aggregate posture at /ui/

The read-only UI's `/ui/` route SHALL render a dashboard view
containing three sections: a posture summary (counts of targets
per worst-dimension score per framework), top concerns (rule
rationales most commonly scored `afhankelijk` across targets,
target-counted), and recent activity (the five most recent scans
across the estate).

#### Scenario: Posture summary counts every target's worst dimension

- **GIVEN** three targets each with one persisted DICTU
  Assessment whose worst populated dimension scores are
  `soeverein`, `afhankelijk`, and `onbekend` respectively
- **WHEN** an operator opens `/ui/`
- **THEN** the response status is 200
- **AND** the DICTU posture summary block reports `1 soeverein`,
  `1 afhankelijk`, `1 onbekend`

#### Scenario: Top concerns counts each rule once per target

- **GIVEN** an estate where the rule
  `dictu.juridisch.cert_issuer_eea` fired `afhankelijk` on 50
  rationales spread across 5 distinct targets
- **WHEN** the operator opens the dashboard
- **THEN** the "top concerns" section lists
  `dictu.juridisch.cert_issuer_eea` with a target count of 5

#### Scenario: Recent activity is chronological across the estate

- **GIVEN** seven persisted scans across multiple targets
- **WHEN** the operator opens the dashboard
- **THEN** the recent activity section lists exactly five entries
- **AND** they are ordered newest first by `started_at`
- **AND** each entry links to the per-scan Analysis page at
  `/ui/scans/{id}/assessment` when an Assessment exists, or to
  the scan-detail page at `/ui/scans/{id}` otherwise

#### Scenario: Empty store renders empty-state copy

- **GIVEN** a fresh store with zero scans and zero assessments
- **WHEN** the operator opens `/ui/`
- **THEN** the dashboard renders without panicking
- **AND** each section displays empty-state copy explaining the
  next action ("run `wanderer scan <domain>` to populate")

---

### Requirement: Worst-dimension score excludes onbekend dimensions

The dashboard's per-target "worst dimension score" SHALL ignore
dimensions whose Score is `onbekend`, treating them as
not-evaluated rather than worst. If every dimension on a
target's most recent Assessment is `onbekend`, the target's
worst-dimension score SHALL be `onbekend`.

#### Scenario: One onbekend dimension does not drag the rest down

- **GIVEN** a target whose latest DICTU Assessment has
  `juridisch: afhankelijk`, `operationeel: soeverein`, and
  `data_ai: onbekend`
- **WHEN** the dashboard computes the target's worst score
- **THEN** the result is `afhankelijk`
- **AND** `onbekend` is not counted as worse than `afhankelijk`

#### Scenario: All-onbekend target is reported as onbekend

- **GIVEN** a target whose latest DICTU Assessment has every
  dimension at `onbekend` (e.g. only the perimeter probes ran and
  GeoLite2 was unavailable)
- **WHEN** the dashboard computes the worst score
- **THEN** the result is `onbekend`

---

### Requirement: Flat targets table relocates to /ui/targets

The previous `/ui/` flat targets table SHALL be served at
`/ui/targets` after this change, with no behavioural change
beyond the URL.

#### Scenario: Auditor view preserved at new URL

- **GIVEN** a store with three targets and persisted scans /
  assessments per target
- **WHEN** an operator opens `/ui/targets`
- **THEN** the response status is 200
- **AND** the page renders the same per-target table the previous
  `/ui/` index rendered before this change (one row per target,
  last-scan status, framework score badges)

#### Scenario: Dashboard links to the targets table

- **GIVEN** the dashboard at `/ui/`
- **WHEN** the operator looks at the posture summary section
- **THEN** an "All targets" link points at `/ui/targets`

---

### Requirement: Read-only contract holds for the dashboard

The dashboard handler and the new `/ui/targets` handler SHALL be
GET-only. The existing static-analysis test in `internal/ui` that
fails the build on `r.Post|Patch|Put|Delete` in the package SHALL
continue to pass.

#### Scenario: Mutation handler still fails the build

- **GIVEN** a contributor adds `r.Post("/", ...)` to
  `internal/ui/ui.go`
- **WHEN** `go test ./internal/ui/...` runs
- **THEN** the static-analysis test fails with a clear message
  naming the offending file
