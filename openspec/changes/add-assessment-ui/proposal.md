## Why

The read-only operator UI today shows raw probe output (per-probe
findings with severity badges) but never renders the assessor's
verdict, even though every scan with `wanderer assess` already has
one persisted. An operator opening the UI sees facts (`tls.issuer
cert issued in US`) without the sovereignty interpretation
(`dictu.juridisch.cert_issuer_eea — afhankelijk: cert issued in US
(outside EEA)`). The DAR (Dashboard / Analysis / Reporting) layering
that the project wants is missing the **Analysis** middle layer —
this proposal adds it.

The Analysis layer is also the only place a non-technical reader
gets to read *why* a rule matters. Today the rule's `Description`
("TLS certificate issued by an authority in the EEA.") is barely
visible and never explains the consequence ("when the issuer's
country is outside the EEA, the certificate's revocation and
issuance authority lives under a foreign jurisdiction").

## What Changes

- Add `/ui/scans/{id}/assessment` (the Analysis page). Renders one
  card per DICTU dimension, plus parallel rendering of any SEAL
  assessment that exists for the same scan. Each card shows the
  dimension's overall score, completeness flag, and one row per
  rule with the rule's score-badge, verdict, evidence finding IDs
  (linking back to the scan-detail page), and a static
  "why this matters" string per rule.
- Add an "Open assessment" link on `/ui/scans/{id}` for every scan
  that has at least one persisted Assessment. Scans without an
  Assessment show a hint that `wanderer assess <scan-id>` will
  produce one.
- Extend `assessor.Rule` with a `Rationale` field — a paragraph
  explaining what the rule observes and why it matters in plain
  language. `internal/assessor/dictu/rules.go` and
  `internal/assessor/eucsf/rules.go` populate the field on every
  registered rule.
- The internal/ui static-analysis test continues to pin the
  read-only contract: every new handler is GET-only.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `web-ui`: adds an Analysis page rendering the persisted
  Assessment(s) for a scan, with rule rationales surfaced from the
  rule registry.
- `assessor`: every Rule SHALL carry a non-empty `Rationale` string
  — a plain-language description of what the rule observes and why
  it matters — populated alongside the existing `Description`.

## Impact

**Code**:
- `internal/ui/`: one new handler (assessment page), one new
  template, links from the existing scan-detail template, possibly
  one new CSS rule for dimension cards. The `ui_test.go`
  static-analysis test stays unchanged but its grep continues to
  pin "no POST/PUT/PATCH/DELETE handlers".
- `internal/assessor/`: `Rule` struct grows a `Rationale string`
  field. Existing call sites that construct Rules must populate it
  (the build breaks loudly if they don't — that is desirable).
- `internal/assessor/dictu/rules.go`,
  `internal/assessor/eucsf/rules.go`: every rule gets a Rationale
  paragraph.

**APIs**: none. The HTTP API already exposes Assessment records
via `GET /assessments/{id}`; the UI consumes the same store path.

**Dependencies**: none. Pure stdlib + html/template.

**Read-only contract**: preserved. All new handlers are GET-only;
the static-analysis test in `internal/ui/ui_test.go` continues to
fail the build if any mutating handler is added.

**DICTU dimensions informed**: every dimension. The Analysis page
is the rendering surface for the DICTU rule set; this change
doesn't add new rules, it surfaces the existing ones.

**Passive/active boundary**: N/A — UI rendering only, no probe
work.
