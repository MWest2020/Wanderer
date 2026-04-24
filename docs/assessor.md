# Assessor

The assessor turns the Findings produced by a scan into a structured
`Assessment`: per-dimension score, a completeness flag, and a list of
per-rule rationales that cite the Findings they drew their verdict
from.

This document covers: running the assessor, reading the output, and
how to extend the DICTU rule set.

## Running it

From the CLI:

```sh
wanderer assess <scan-id> [--format text|markdown|json] [--db wanderer.db] [--persist=false]
```

- `--format` defaults to `text`. Markdown is the intended operator-
  facing format; JSON is for tooling.
- `--persist=false` suppresses the side-effect of writing the
  assessment to the store — useful for dry runs.

Over HTTP:

```sh
curl -XPOST http://localhost:8080/scans/<scan-id>/assessments
curl        http://localhost:8080/assessments/<assessment-id>
```

`POST` always persists and always returns the assessment it just
produced. Each POST creates a new record — previous assessments for
the same scan remain retrievable by their own IDs.

## Reading a score

Each DICTU dimension receives one of:

| Score         | Meaning                                                   |
| ------------- | --------------------------------------------------------- |
| `soeverein`   | The best verdict the rule set issues for this dimension.  |
| `voldoende`   | Adequate. No red flags, but not actively sovereign.       |
| `afhankelijk` | Low sovereignty — dependency on a non-sovereign party.    |
| `onbekend`    | No evidence-backed verdict could be reached.              |

The dimension score is the **worst** score across every
evidence-backed rule in that dimension. One rule scoring `afhankelijk`
drags the whole dimension down even if four others scored `soeverein`
— because one material dependency is one material dependency.

Next to the score, a `Completeness` flag tells you how much of the
rule set was answerable:

| Completeness | Meaning                                                       |
| ------------ | ------------------------------------------------------------- |
| `complete`   | Every rule in this dimension had evidence to work with.       |
| `partial`    | Some rules evaluated; others did not because data was absent. |
| `incomplete` | No rule in this dimension had any evidence.                   |

A dimension with no rules at all (for example, `mens` in the current
rule set) is rendered as `n/a` in the summary table.

## Rule set (MVP)

The DICTU rule set today is deliberately small. Each rule is a Go
function in `internal/assessor/dictu/rules.go`.

| Rule ID                                      | Dimension    |
| -------------------------------------------- | ------------ |
| `dictu.juridisch.cert_issuer_eea`            | juridisch    |
| `dictu.juridisch.apex_ip_eea`                | juridisch    |
| `dictu.juridisch.mx_vendor_jurisdiction`     | juridisch    |
| `dictu.operationeel.cert_validity`           | operationeel |
| `dictu.operationeel.dns_redundancy`          | operationeel |
| `dictu.operationeel.caa_restricts_issuance`  | operationeel |
| `dictu.technologie.third_parties_eea`        | technologie  |
| `dictu.technologie.no_us_hyperscaler`        | technologie  |
| `dictu.data_ai.mx_present`                   | data_ai      |
| `dictu.data_ai.oidc_federation`              | data_ai      |

The `oidc_federation` rule always returns no evidence until the
egress probe lands — it is listed so the reader sees the future
coverage the dimension is waiting on.

The `mens` dimension has no rules. It appears in the output as
`onbekend (n/a)`. This is explicit, not an omission: the MVP scanner
observes perimeter posture, not human processes.

## Evidence and auditability

Every rationale entry cites one or more `Finding.ID` values in its
`Evidence` field. The markdown report renders these as:

```
Evidence: f_abc, f_def
```

A reader who wants to verify the verdict can pull those findings out
of the scan and inspect their `Attributes` and `Evidence` fields — the
raw source material the probe captured. Two reviewers running the
assessor on the same stored scan will get the same verdicts,
rationales, and evidence lists, modulo assessment ID and timestamp.

## Extending the rule set

Rules are Go functions, not a DSL. To add one:

1. Add a function to `internal/assessor/dictu/rules.go` that returns
   an `assessor.Rule` with a stable `ID` (prefix `dictu.<dimension>.`),
   a `Dimension`, a one-line `Description`, and a `Match` function.
2. Wire it into `DefaultRules`.
3. Add a test in `rules_test.go`. The pattern is: fabricate the minimal
   set of Findings the rule needs, call `r.Match`, assert on
   `Score` and `Evidence`.

Rules MUST be total. On missing attributes, return a no-evidence
result (`Score: onbekend`, empty `Evidence`, `Verdict` explaining
what was missing) rather than panicking. The engine's panic recovery
is a safety net, not a design.

See [ADR-0004](decisions/0004-assessor-rule-engine.md) for why rules
are Go functions and not a hot-reloadable DSL.
