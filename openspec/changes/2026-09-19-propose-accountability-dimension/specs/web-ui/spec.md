# Delta for web-ui

> Accountability is the most human-facing dimension in the pack: its
> findings name organisations, mailboxes, and contact routes, and its
> natural reader is the Tourist persona (ADR-0017) — a bestuurder or
> CISO asking "if this breaks tonight, who do I call, and does that
> route work?". The interface is therefore a first-class requirement
> of this change, not a rendering afterthought. Implementers SHALL
> study Wordsworth's UI for tone and interaction patterns
> (plain-language headline → expandable evidence → one concrete next
> action) before building. Scenarios are in English (repo convention);
> the quoted copy is the shipped Dutch.

## ADDED Requirements

### Requirement: Accountability renders as an answer sheet, not a rule dump

The assessment report page (`/ui/scans/{id}/assessment`) SHALL render
the accountability dimension as a list of plain-language questions,
one per rule, each answered ja / nee / onbekend / n.v.t. with a
one-sentence verdict in the same register. Rule IDs, RDAP field
names, and RFC numbers SHALL appear only inside the expandable
evidence block, never in the headline. Each failing rule SHALL show
exactly one concrete remediation line. n.v.t., onbekend, and nee
SHALL be visually distinct from one another.

#### Scenario: Tourist reads a failing registrant check

- **GIVEN** a `.com` scan where registrant_identifiable scored
  afhankelijk on a commercial privacy proxy
- **WHEN** an operator opens the assessment report
- **THEN** the accountability section shows the question "Is de
  registrant herkenbaar als uw organisatie?" answered "Nee" with the
  proxy named and a single remediation line
- **AND** `wand.accountability.registrant_identifiable` and RDAP
  jargon appear only after expanding the evidence

#### Scenario: Unknown is presented honestly

- **GIVEN** a scan where soa_rname scored onbekend with reason
  `probe_unavailable` after a timeout
- **WHEN** the operator opens the report
- **THEN** the answer reads "Onbekend — de DNS-SOA-opvraging kreeg
  geen antwoord", visually distinct from a failing "Nee", so absence
  of evidence is never dressed up as a defect

#### Scenario: Not applicable is not a gap

- **GIVEN** a `.nl` scan where registrant_identifiable scored onbekend
  with reason `registry_redacted`
- **WHEN** the operator opens the report
- **THEN** the answer reads "n.v.t. — het register publiceert
  registrantgegevens niet voor .nl", styled differently from
  "Onbekend", and shows no remediation line

---

### Requirement: Answer-sheet copy lives in one Dutch string table

All answer-sheet copy — per rule the question, the verdict text per
outcome, and the remediation line — SHALL ship in Dutch from one
per-rule string table (`internal/assessor/wand/accountability_nl.yaml`).
Templates and rule code SHALL NOT contain answer-sheet copy; rules
supply named parameters only. A test SHALL fail when a rule has no
entry or an entry uses a parameter the rule does not supply. There
SHALL be no i18n machinery in v1: a second language is a second
table.

#### Scenario: Every rule has complete copy

- **WHEN** the string table is loaded in tests
- **THEN** each of the seven rules has a question, a verdict for
  every outcome it can produce, and a remediation line for every
  failing outcome

#### Scenario: No copy in templates

- **WHEN** the report template is rendered for an accountability
  dimension
- **THEN** every visible question, verdict, and remediation string
  originates from the string table

---

### Requirement: Evidence is expandable and self-explaining

Each accountability question SHALL expand in place to the raw
findings that produced the verdict (registrant string, RNAME, fetched
security.txt fields, per-path variant table), each labelled in plain
language with the technical key alongside. The collapsed state SHALL
be the default; expanding one question SHALL NOT navigate away from
the report.

#### Scenario: Explorer drills into variant convergence

- **GIVEN** a variant_convergence verdict of voldoende
- **WHEN** the operator expands the question
- **THEN** an 8-row table shows each path, its status, redirect
  chain, and final origin, with the diverging path highlighted and
  budget-skipped or untested paths marked as such

#### Scenario: Evidence absent

- **GIVEN** an onbekend verdict backed only by an unavailable finding
- **WHEN** the operator expands the question
- **THEN** the block shows the failure (probe, error class, time)
  instead of an empty table

---

### Requirement: Overview surfaces accountability without adding a tab

The Overview (Tourist) page SHALL include the accountability verdict
in the existing per-target headline row (alongside the current
framework score), without introducing a new navigation tab, keeping
the two-tab IA of ADR-0017 intact.

#### Scenario: Fleet row shows the dimension

- **GIVEN** three targets with mixed accountability outcomes
- **WHEN** the operator opens `/ui/`
- **THEN** each target row carries an accountability pill whose
  colour matches the dimension verdict and links to that target's
  report section

#### Scenario: No assessment yet

- **GIVEN** a target scanned before this change (no accountability
  findings)
- **WHEN** the operator opens `/ui/`
- **THEN** the pill renders as "niet beoordeeld", not as a failure

---

### Requirement: Scanner limitations render as operator warnings

A rationale whose reason code has subject `scanner` SHALL NOT be
rendered as a property of the target. The report SHALL show it as an
operator environment warning next to the dimension, and the answer
itself SHALL be based on what was observed.

#### Scenario: Scanner without IPv6

- **GIVEN** a variant_convergence rationale with reason
  `scanner_no_ipv6`
- **WHEN** the operator opens the report
- **THEN** a warning reads "scanner heeft geen IPv6 — v6-paden niet
  gemeten", styled as an environment notice
- **AND** the question's answer reflects only the observed v4 paths

## MODIFIED Requirements

### Requirement: Worst-dimension score excludes onbekend dimensions

The dashboard's per-target "worst dimension score" SHALL ignore
dimensions whose Score is `onbekend`, dimensions reported as not
applicable (every rationale structural), and dimensions absent from
the Assessment (scans made before the dimension existed), treating
them as not-evaluated rather than worst. The score SHALL be shown
together with the dimensions it was computed over. If no dimension on
a target's most recent Assessment is evaluated, the target's
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

#### Scenario: Old scan without the new dimension

- **GIVEN** a target whose latest Assessment predates the
  accountability dimension
- **WHEN** the dashboard computes the worst score
- **THEN** the score is computed over the dimensions present
- **AND** the row states which dimensions it covers, and does not
  count accountability as failed or silently average it in
