# Design: Assessor

## Component placement

```
┌────────────┐     ┌──────────────┐     ┌──────────────┐
│   store    │───► │   assessor   │───► │  Assessment  │
│ (findings) │     │ (rule engine)│     │   (persist)  │
└────────────┘     └──────────────┘     └──────────────┘
                         ▲
                         │
                   DICTU rule set
                   (internal/assessor/dictu)
```

The assessor is a pure function of Findings. It never reads or opens
network connections. It does not know how Findings were produced.

## Data model

```go
// pkg/models/assessment.go

type Assessment struct {
    ID         string
    ScanID     string
    Framework  string       // "dictu" for now
    CreatedAt  time.Time
    Dimensions []DimensionScore
    Report     string       // markdown
}

type DimensionScore struct {
    Dimension    DimensionHint  // juridisch, technologie, ...
    Score        Score          // enum
    Completeness Completeness   // complete | partial | incomplete
    Rationale    []Rationale
}

type Score string
const (
    ScoreOnbekend       Score = "onbekend"
    ScoreAfhankelijk    Score = "afhankelijk"        // low sovereignty
    ScoreVoldoende      Score = "voldoende"
    ScoreSoeverein      Score = "soeverein"
)

type Completeness string
const (
    CompletenessComplete   Completeness = "complete"
    CompletenessPartial    Completeness = "partial"
    CompletenessIncomplete Completeness = "incomplete"
)

type Rationale struct {
    CriteriumID string     // DICTU criterium reference ("1.2", "3.4", ...)
    Verdict     string     // short human-readable line
    Evidence    []string   // Finding.ID values — the citations
}
```

## Rule engine

`internal/assessor/rule.go` defines:

```go
type Rule struct {
    ID          string                       // "dictu.1.1"
    Dimension   models.DimensionHint
    Criterium   string
    Description string
    Match       func([]models.Finding) RuleResult
}

type RuleResult struct {
    Score    models.Score
    Verdict  string
    Evidence []string  // finding IDs
}
```

Rules live in `internal/assessor/dictu/rules.go` as a static slice. Each
rule is independently testable. The assessor runs every rule across
the full Finding set and aggregates per dimension.

### Score aggregation per dimension

Simple, auditable: the dimension score is the **worst** score across
all rules in that dimension whose `Evidence` is non-empty. Rules that
match no evidence are dropped from the rationale and contribute to
the completeness calculation instead.

### Completeness calculation

Per dimension:

- **Complete**: every rule contributing to this dimension returned
  evidence-backed verdicts.
- **Partial**: ≥1 rule returned evidence, but ≥1 other rule returned
  no evidence because the required probe data was absent (e.g. no
  inventory probe has run yet).
- **Incomplete**: no rule returned evidence — we know nothing about
  this dimension from this scan.

`ip.unavailable` counts as "no probe data" for any rule depending on
IP ASN country.

## DICTU rule set (MVP)

Ships with a minimum viable rule set covering what current probes can
answer. Indicative — final criterium numbers match the official DICTU
document:

| Dimension     | Rule                                              | Inputs                        |
| ------------- | ------------------------------------------------- | ----------------------------- |
| Juridisch     | Cert issuer in EU                                 | `tls.issuer.issuer_country`   |
| Juridisch     | Apex IP hosted in EEA                             | `ip.asn.country`              |
| Juridisch     | Mail routes via EU provider                       | `dns.mx` + `ip.asn`           |
| Operationeel  | Cert valid and not expiring within 30d            | `tls.validity`                |
| Operationeel  | DNS delegated to ≥2 nameservers                   | `dns.ns`                      |
| Operationeel  | CAA records restrict issuers                      | `dns.caa`                     |
| Technologie   | Third-party HTTP dependencies hosted in EEA       | `http.third_party` + `ip.asn` |
| Technologie   | No known US-hyperscaler in apex/CDN path          | `ip.asn.organisation`         |
| Data & AI     | MX vendor jurisdiction                            | `dns.mx` + `ip.asn`           |
| Data & AI     | OIDC / federation endpoints                       | (requires egress probe — marked incomplete until then) |

Each rule is one Go function. Total rule count is intentionally small
for the MVP — ~10 rules. Adding rules later is a one-file edit.

## External systems

None. The assessor is a pure consumer of the store.

## Failure modes

- **Finding with unknown ProbeID.** The rule set may not cover it.
  This is not an error: unknown findings are ignored for scoring
  purposes, listed at the end of the report as "evidence without
  verdict".
- **Attribute missing or wrong type.** A rule that expects
  `issuer_country` as `[]string` must handle `nil` and `string`
  gracefully. Rules defensively type-assert and skip on mismatch,
  emitting a warning to `slog` but not failing the assessment.
- **Finding evidence bytes corrupt.** Report rendering tolerates this
  (skips the citation), the rule itself does not depend on `Evidence`
  — only `Attributes`.

## Markdown report format

```markdown
# Wanderer Assessment — example.nl

Scan: s_abc123 (2026-04-23T19:50:11Z)
Framework: DICTU Toetsingsinstrument v1

## Samenvatting

| Dimensie     | Score        | Volledigheid |
| ------------ | ------------ | ------------ |
| Juridisch    | afhankelijk  | complete     |
| Technologie  | afhankelijk  | partial      |
| Data & AI    | onbekend     | incomplete   |
| Operationeel | voldoende    | complete     |
| Mens         | onbekend     | n/a          |

## Juridisch — afhankelijk (complete)

### 1.1 Cert issuer in EU — afhankelijk
Verdict: Cert uitgegeven door Google Trust Services (US).
Evidence: `tls.issuer` (finding f_xyz)

...

## Evidence without verdict

- `dns.txt.other` (finding f_aaa) — not mapped to any rule
```

## Clever valkuil

Tempting to build a DSL for rules — YAML with jsonpath-like
expressions, hot-reloadable rule files, an admin UI. All of that is
wrong for now. Rules are Go functions because:

- Go functions compile. A typo fails at build time, not at runtime
  against a production scan.
- Debugging is `go test`, not a YAML validator.
- The DICTU rule set is ~10 rules. A DSL has higher per-rule cost than
  a function for that scale.
- External contributors can add rules via PR with tests; they do not
  need a DSL for that.

When we ship the *second* framework (NIS2 or BIO), revisit. Until then,
two identical Go function implementations of overlapping rules is
cheaper than one DSL with a plugin system.

## Relation to other changes

- **`add-scheduling`** will store Assessments over time and surface a
  diff ("drifted from `voldoende` to `afhankelijk` on Juridisch").
- **`add-exporters`** will emit Assessments as CSV/JSONL next to
  Findings.
- **`add-mcp-server`** will expose `assess_scan` as an MCP tool.
- **`add-inventory-probe` + `add-egress-probe`** will turn Data & AI
  and some Technologie rules from `incomplete` to `complete`.

## Stability + test coverage

- `pkg/models.Assessment` is stable public surface. Added fields only
  in backwards-compatible ways.
- `internal/assessor` target coverage: 85% (higher than baseline
  because rules are individually cheap to test).
- Golden-file tests for the markdown report format.
