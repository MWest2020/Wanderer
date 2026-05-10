# Delta for web-ui

## ADDED Requirements

### Requirement: Analysis layer renders the steering matrix

The UI SHALL expose `/ui/analysis` (and
`/ui/analysis?org=<slug>`) as the central Analysis page. The
view MUST render a per-rule cross-target table with one row
per rule that has produced at least one Rationale, and columns
for distinct-target counts per `soeverein`, `voldoende`,
`afhankelijk`, and `onbekend` score. Stable row order is
framework first (`wand` > `eucsf` > alphabetical), then ruleID
alphabetical within a framework. The page MUST honour the
`?org=<slug>` filter and render a "Scope: {orgName}" pill in
the header when active.

#### Scenario: Analysis matrix renders rules and counts

- **Given** rule `wand.juridisch.cert_issuer_eea` has fired on
  three targets — one soeverein, two afhankelijk — across the
  current organisation's scans
- **When** the operator opens `/ui/analysis`
- **Then** the row for that rule shows soeverein=1,
  afhankelijk=2, voldoende=0, onbekend=0
- **And** the row's rule cell links to
  `/ui/reporting/wand/wand.juridisch.cert_issuer_eea`

---

### Requirement: Reporting layer is the rule catalogue

The UI SHALL render `/ui/reporting` as a rule catalogue:
**every** registered rule from every framework, with the
framework, dimension, rule ID, human-readable description, and
rationale text. The catalogue MUST NOT carry scoring data —
it answers "what is being measured" rather than "how am I
doing". Each row links to the existing per-rule deep-dive page
at `/ui/reporting/{framework}/{ruleID}`.

#### Scenario: Catalogue lists rules with descriptions

- **Given** the wand pack registers
  `wand.juridisch.cert_issuer_eea` and the eucsf pack registers
  `eucsf.sov2.cert_issuer_eu`
- **When** the operator opens `/ui/reporting`
- **Then** both rules appear in the catalogue with their
  description text
- **And** neither row contains a numeric score column

---

### Requirement: Dashboard is "is dit goed of niet"

The Dashboard at `/ui/` and `/ui/orgs/{slug}` SHALL render
high-level health information only: the headline-stats strip,
one verdict pill per framework (worst score reached across all
targets in scope), and the organisations list (instance-wide
view only). The page MUST NOT carry the rule × score-counts
matrix, the Top concerns table, or the External / Internal
posture distribution blocks — those answer steering questions
that belong on Analysis.

#### Scenario: Dashboard renders verdict pills, not the matrix

- **Given** an operator opens `/ui/`
- **When** the page renders
- **Then** the body contains a per-framework verdict pill
  (e.g. "wand · afhankelijk · 3 of 4 targets")
- **And** the body does NOT contain a "Top concerns" heading
- **And** the body does NOT contain a "Recent activity" heading
- **And** the body does NOT contain "External posture" or
  "Internal posture" headings
