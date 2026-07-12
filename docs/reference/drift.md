---
status: draft
last_reviewed: 2026-07-12
---

# Drift

A scan tells you what posture a domain has *now*. A pair of scans
tells you what *changed*. Wanderer surfaces those changes as drift
Findings: they live in the same store, behind the same APIs, and
are consumed by the assessor like any other Finding.

## When drift runs

Drift runs automatically after every scheduled scan finishes (see
[scheduling.md](../how-to/scheduling.md)). It also runs on demand via
`wanderer diff <scan-a> <scan-b>` — that variant prints the would-be
Findings to stdout without persisting anything.

The drift engine compares the new scan against the most recent
previous scan of the same target. The first scan of a target has
nothing to compare against; the engine emits a single
`drift.baseline_established` Finding so an operator can distinguish
"new target, no history" from "no changes since last scan" (which
emits `drift.no_changes`).

## Rules

A drift rule is a Go function `func(prev, curr *models.Scan) []models.Finding`.
It looks at specific Findings in the two scans and decides whether a
change is surface-worthy. The MVP rule set:

| Rule                                    | Triggers on                                                | Severity     | Dimension       |
| --------------------------------------- | ---------------------------------------------------------- | ------------ | --------------- |
| `drift.tls.issuer_changed`              | `tls.issuer.issuer_cn` differs                             | finding      | `juridisch`     |
| `drift.tls.days_left_dropped`           | `tls.validity.days_left` crosses the 30-day boundary down  | concern      | `operationeel`  |
| `drift.dns.mx_set_changed`              | set of `dns.mx.host` values differs                        | observation  | `data_ai`       |
| `drift.dns.ns_set_changed`              | set of `dns.ns.host` values differs                        | observation  | `operationeel`  |
| `drift.ip.country_changed`              | `ip.asn.country` differs for the same host subject         | finding      | `juridisch`     |
| `drift.http.third_party_added`          | new subject appears in `http.third_party` Findings         | observation  | `technologie`   |
| `drift.http.third_party_removed`        | subject disappears from `http.third_party` Findings        | info         | `technologie`   |

Adding a rule: add a function to `internal/drift/rules.go`, append it
to `DefaultRules`, write a test in `rules_test.go` with fabricated
`prev` / `curr` scans. Rules MUST be total: missing or mistyped
attributes return nil rather than panicking. The drift engine itself
does not currently isolate panics — keep your rule code disciplined.

## Attributes carried on every drift Finding

```
source_modus: "drift"
prev_scan_id: "<scan id of the comparison baseline>"
curr_scan_id: "<scan id of the new scan>"
```

Plus any rule-specific keys (`added`, `removed`, `prev_issuer_cn`,
…). See [findings.md](findings.md#drift-findings) for the full
catalogue.

## How the assessor reads drift

The assessor treats drift Findings exactly like probe Findings: a
rule whose `Match` function inspects `Finding.Attributes` works the
same on `tls.issuer` and on `drift.tls.issuer_changed`. There is no
parallel "drift assessment" pipeline.

A future rule may explicitly act on drift, e.g. *"a `juridisch`-
relevant drift Finding in the last 30 days reduces the dimension's
score regardless of current posture"*. That is a one-rule change
when the demand is concrete.

## `wanderer diff`

```sh
wanderer diff <scan-a> <scan-b>
```

Computes the same Findings the scheduler would persist, prints them
as markdown, and **does not touch the store**. Use it to inspect
historical scans without firing fresh ones.

The output is deterministic: same input scans → same markdown bytes,
sorted by ProbeID then Subject. That makes it diff-friendly itself.
