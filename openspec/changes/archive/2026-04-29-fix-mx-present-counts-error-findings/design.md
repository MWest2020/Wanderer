# Design: Rules ignore meta Findings

## Where the rule reads attributes today

```go
func mxPresent() assessor.Rule {
    return assessor.Rule{
        ID:          "dictu.data_ai.mx_present",
        Match: func(findings []models.Finding) assessor.RuleResult {
            var evidence []string
            for _, f := range findings {
                if f.ProbeID == "dns.mx" {
                    evidence = append(evidence, f.ID)
                }
            }
            // ...
        },
    }
}
```

A `lookupError` Finding has `ProbeID: "dns.mx"` and an `error`
attribute. A `noAnswer` Finding has `no_answer: true`. The rule
does not look at attributes — it just counts.

## What the fix looks like

A small helper in `internal/assessor/rule.go`:

```go
// IsEvidenceLike reports whether a Finding can serve as evidence
// for a rule's verdict. Meta-findings (errors, no-answer, probe
// unavailability) return false.
func IsEvidenceLike(f models.Finding) bool {
    if _, ok := f.Attributes["error"]; ok { return false }
    if v, ok := f.Attributes["no_answer"].(bool); ok && v { return false }
    if v, ok := f.Attributes["unavailable"].(bool); ok && v { return false }
    return true
}
```

`mxPresent` and any other rule that filters by ProbeID alone calls
`IsEvidenceLike(f)` before counting. The dictu package gets one
test fixture per rule that verifies the rule does not produce a
verdict when only meta-findings are present.

## Failure modes

- A future probe emits a meta-finding under a *new* attribute key
  (e.g. `timeout: true`). The helper does not catch it. Mitigation:
  document the convention (`error`, `no_answer`, `unavailable`) in
  `docs/findings.md` and add to the helper as the corpus grows.

## Clever valkuil

Tempting: introduce a `Severity` filter (e.g. ignore SeverityInfo).
Wrong — observation-grade evidence often sits at SeverityInfo and
must contribute. Severity is presentation; meta-status is content.

## Tests

- A regression test in `dictu/rules_test.go` reproducing the smoke
  scenario: a Finding set with only `dns.mx` `lookupError` and
  `noAnswer` produces `Score: onbekend` from `mx_present`.
- A mirror test for any other rule that does bare-ProbeID counting.
