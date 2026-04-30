## ADDED Requirements

### Requirement: UI renders the persisted Assessment per scan as the Analysis layer

The read-only operator UI SHALL mount a `/ui/scans/{id}/assessment`
GET handler that renders every persisted `models.Assessment` for
the scan as one card per dimension, with the dimension's score
badge, completeness flag, and one row per fired Rationale showing
the rule's `CriteriumID`, `Score`, `Verdict`, the linked Evidence
Finding IDs, and the rule's `Description` plus `Rationale` looked
up from the live rule registry (`dictu.DefaultRules` /
`eucsf.DefaultRules`) at render time.

#### Scenario: Assessment renders for a scored scan

- **GIVEN** a scan with one persisted DICTU Assessment that
  contains rationales for the `cert_issuer_eea` and
  `caa_restricts_issuance` rules
- **WHEN** an operator opens `/ui/scans/{scan-id}/assessment`
- **THEN** the response status is 200
- **AND** the page renders one card per DICTU dimension that has
  at least one Rationale
- **AND** each rationale row shows the rule's score badge, the
  Verdict text, the rule's `Description`, and the rule's
  `Rationale` paragraph in an expandable detail block

#### Scenario: Both frameworks render side by side

- **GIVEN** a scan with two persisted Assessments (one
  `Framework: dictu`, one `Framework: eucsf`)
- **WHEN** the operator opens the assessment page
- **THEN** the page renders one column per framework, each with
  its own dimension cards and rule rows

#### Scenario: Scan without an Assessment shows a hint

- **GIVEN** a scan with zero persisted Assessments
- **WHEN** the operator opens `/ui/scans/{scan-id}/assessment`
- **THEN** the response status is 200
- **AND** the page renders a one-line hint that no Assessment has
  been produced and shows the `wanderer assess <scan-id>` command
  the operator should run

#### Scenario: Retired-rule rationale degrades gracefully

- **GIVEN** an Assessment containing a rationale whose
  `CriteriumID` is no longer present in the live rule registry
- **WHEN** the operator opens the assessment page
- **THEN** the row renders the persisted score, verdict, and
  evidence unchanged
- **AND** the description column reads "rule retired" instead of
  the live rule description

---

### Requirement: Scan-detail page links to the Analysis page when an Assessment exists

The existing `/ui/scans/{id}` template SHALL include a prominent
link to `/ui/scans/{id}/assessment` when at least one persisted
Assessment exists for the scan, and SHALL omit the link
otherwise.

#### Scenario: Link present when scored

- **GIVEN** a scan with at least one persisted Assessment
- **WHEN** the operator opens `/ui/scans/{scan-id}`
- **THEN** the page renders an "Open assessment" link pointing at
  `/ui/scans/{scan-id}/assessment`

#### Scenario: Link absent when not scored

- **GIVEN** a scan with zero persisted Assessments
- **WHEN** the operator opens `/ui/scans/{scan-id}`
- **THEN** the page does not render an "Open assessment" link

---

### Requirement: Read-only contract holds for the assessment page

The new assessment handler SHALL be GET-only, and the existing
static-analysis test in `internal/ui` that fails the build on the
appearance of `r.Post|Patch|Put|Delete` in the package SHALL
continue to pass after this change.

#### Scenario: Mutation handler still fails the build

- **GIVEN** a contributor adds `r.Post("/ui/scans/{id}/assessment", ...)` to `internal/ui/ui.go`
- **WHEN** `go test ./internal/ui/...` runs
- **THEN** the static-analysis test fails with a clear message
  naming the offending file
