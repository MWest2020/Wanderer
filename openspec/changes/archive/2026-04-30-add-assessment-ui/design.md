## Context

`internal/ui` already wraps `internal/store` in three GET-only
handlers backed by Go `html/template`:

- `indexHandler` → targets list (`/ui/`)
- `scanHandler` → per-scan findings page (`/ui/scans/{id}`)
- `driftHandler` → per-target drift (`/ui/targets/{id}/drift`)

`internal/assessor` already produces `models.Assessment` records,
persisted with `Framework`, `ScanID`, `Dimensions[]`, and inside
each Dimension a `Rationales[]` slice carrying the rule's
`CriteriumID`, `Score`, `Verdict`, `Completeness`, and `Evidence`
finding-IDs. `internal/store` exposes `ListAssessmentsForScan`
already (used by the targets index to show last-known scores).

The persisted `CriteriumID` on each Rationale (e.g.
`dictu.juridisch.cert_issuer_eea`) is exactly the `Rule.ID` used
in `internal/assessor/dictu/rules.go`. So a renderer can resolve
"which rule fired" by looking up the rule registry — that is how
we attach the rule's `Description` and the new `Rationale`
("why this matters") string to the report row.

The CLI `wanderer assess --format markdown` already produces a
near-identical layout via `internal/assessor/report.go`. The HTML
view is essentially the same content with a different template;
no new query path is needed.

## Goals / Non-Goals

**Goals:**

- An operator opening `/ui/scans/{id}/assessment` sees, per
  framework available, a card per dimension with the score,
  completeness flag, and one row per fired rule.
- Each rule row makes the cited Findings clickable (links to
  `/ui/scans/{id}#f_<finding-id>` on the existing scan-detail
  page).
- Each rule row carries a "why this matters" expandable detail
  populated from the new `Rule.Rationale` field.
- The scan-detail page (`/ui/scans/{id}`) gains an "Open
  assessment" link when at least one Assessment exists for the
  scan.

**Non-Goals:**

- Comparison view across two assessments. That's a separate
  proposal — could fold into the existing drift page later.
- Rule-level remediation suggestions ("how to fix this"). Out of
  scope here; assessor produces a verdict, not a runbook.
- New aggregate Dashboard at `/ui/`. That's its own proposal
  (`add-posture-dashboard`).
- Editing or re-running assessments from the UI. Read-only
  contract holds.

## Decisions

### Decision 1: Resolve rule metadata at render time, not store time

The persisted Assessment carries `CriteriumID` strings; it does
not carry `Description` or `Rationale`. Two options:

1. Bake `Description` + `Rationale` into the Assessment row at
   write time (one column per Rule run).
2. Look them up at render time from `dictu.DefaultRules()` /
   `eucsf.DefaultRules()` keyed by `CriteriumID`.

We pick **(2)**. Storing the strings denormalises the rule
registry into the DB and makes a Description edit either silently
ineffective for past Assessments, or a backfill chore. Looking up
at render time means a rule-text update lands the moment the new
binary serves the page. The trade-off is that rendering an old
Assessment whose CriteriumID has been retired falls back to a
"Rule retired" placeholder; we accept that — retiring a rule is
already a versioned change, and the historical evidence is the
Findings, not the rule text.

### Decision 2: Render every framework that has an Assessment for the scan

A scan run with `wanderer assess --framework both` produces two
Assessment rows; the page renders both side-by-side (or stacked
on narrow screens) so an operator can compare DICTU vs SEAL
verdicts on the same evidence. Sites that only run one framework
get a single column. No extra navigation needed.

### Decision 3: `Rule.Rationale` becomes a required field

Adding the field as required (build breaks if unset) forces every
rule author to write a one-paragraph "why this matters" string
when they add a rule. Optional fields with empty strings rot in
practice. Existing rules each get a hand-written paragraph in
this same change — the assessor package author is in the best
position to write them.

### Decision 4: No client-side JavaScript

The expandable "why this matters" detail uses
`<details><summary>` (native HTML, no JS). The links to findings
are plain anchor tags. The UI stays as static HTML rendered
server-side; the read-only static-analysis test continues to pin
the contract.

## Risks / Trade-offs

[Risk] An operator rendering a scan with a framework whose rule
registry no longer contains the persisted CriteriumID (rule
retired between assess-time and view-time) sees the verdict but
no rule context. → Mitigation: render the verdict + evidence
unchanged, prefix the row with "rule retired" so the operator
knows why the description is missing. The Assessment's verdict
itself is unchanged — rule retirement is a real event the UI
should disclose, not paper over.

[Risk] `Rule.Rationale` becomes required and breaks downstream
forks of `internal/assessor` that defined their own rules. →
Mitigation: `Rule.Rationale` going from absent to required is a
breaking change to the assessor package. CHANGELOG entry under
`### Changed (breaking)` calls it out, and an ADR is not required
because the rule is internal-only (no `pkg/` re-export). The
project-hygiene spec already requires the CHANGELOG signal.

[Risk] Template authors are tempted to embed user-controlled
strings (CriteriumID) into HTML directly. → Mitigation: Go
`html/template` auto-escapes by default. The existing scan-detail
template is the reference; copy that pattern.

**Clever valkuil — would-be solutions to avoid:**

1. **Adding a JSON endpoint that returns the rendered HTML.**
   Tempting because it would let a future SPA front-end consume
   it. Wrong: the existing API already returns
   `models.Assessment` — that's the structured surface. The UI
   server-renders for read-only operators; mixing a new "HTML
   over JSON" endpoint pollutes the API contract.
2. **Rendering both DICTU and SEAL into one merged table.** They
   use different score scales (DICTU's `onbekend / afhankelijk /
   gedeeld / soeverein` vs SEAL's `seal_0..seal_4`) and merging
   them into one column flattens information. Side-by-side keeps
   the contracts distinct.
3. **Persisting `Rationale` strings into the assessment row.**
   Already discussed above (Decision 1) — denormalisation chore
   masquerading as "self-contained Assessments". The store stays
   the audit trail of what the probes saw and how the rules
   scored; the rendering layer is separate.

**External systems & failure modes:**

- `internal/store.ListAssessmentsForScan(scanID)` — returns an
  empty slice when no Assessment exists for a scan; the handler
  renders "no assessment yet — run `wanderer assess <scan-id>`"
  rather than 404.
- `dictu.DefaultRules()` / `eucsf.DefaultRules()` — pure Go calls,
  no failure mode. A nil/empty result would mean a zero-rule
  build, which the existing `TestDefaultRules_HasTenRules` already
  pins.
- The browser — auto-escaping via `html/template` covers XSS;
  there is no client-side rendering.
