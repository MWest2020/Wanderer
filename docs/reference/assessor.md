---
status: draft
last_reviewed: 2026-07-12
---

# Assessor

The assessor turns the Findings produced by a scan into a structured
`Assessment`: per-dimension score, a completeness flag, and a list of
per-rule rationales that cite the Findings they drew their verdict
from.

This document covers: running the assessor, reading the output, and
how to extend the **wand** rule set.

## Inspired by

Wanderer's first-party rule pack is named **wand** (Wanderer-NL).
Its rule semantics were inspired by the Dutch government's
publicly available *Toetsingsinstrument Soevereiniteit
Clouddiensten*, published by **DICTU** (Dienst ICT Uitvoering, EZK).
The rule authoring, the implementation, and the ongoing
maintenance are Conduction's; DICTU does not endorse, certify,
or otherwise sanction Wanderer or the wand pack. See
[ADR-0011](../explanation/adr/0011-rename-dictu-to-wand.md) for the
rationale behind the rename.

## Running it

From the CLI:

```sh
wanderer assess <scan-id> [--framework wand|eucsf|both] [--format text|markdown|json] [--db wanderer.db] [--persist=false]
```

- `--framework` defaults to `wand`. `eucsf` runs the EU Cybersecurity
  Framework / SEAL pack instead; `both` runs both packs and persists
  one Assessment record per framework, tagged on `Assessment.Framework`.
  `dictu` is accepted as a deprecated alias for `wand` for one
  release (it prints a stderr deprecation warning).
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

Each wand dimension receives one of:

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

The wand rule set today is deliberately small. Each rule is a Go
function in `internal/assessor/wand/rules.go`.

| Rule ID                                     | Dimension    |
| ------------------------------------------- | ------------ |
| `wand.juridisch.cert_issuer_eea`            | juridisch    |
| `wand.juridisch.apex_ip_eea`                | juridisch    |
| `wand.juridisch.mx_vendor_jurisdiction`     | juridisch    |
| `wand.juridisch.registrar_jurisdiction`     | juridisch    |
| `wand.operationeel.cert_validity`           | operationeel |
| `wand.operationeel.dns_redundancy`          | operationeel |
| `wand.operationeel.caa_restricts_issuance`  | operationeel |
| `wand.technologie.third_parties_eea`        | technologie  |
| `wand.technologie.no_us_hyperscaler`        | technologie  |
| `wand.data_ai.mx_present`                   | data_ai      |
| `wand.data_ai.oidc_federation`              | data_ai      |

The `oidc_federation` rule always returns no evidence until the
egress probe lands — it is listed so the reader sees the future
coverage the dimension is waiting on.

The `mens` dimension has no rules. It appears in the output as
`onbekend (n/a)`. This is explicit, not an omission: the MVP scanner
observes perimeter posture, not human processes.

## EU CSF / SEAL framework

`internal/assessor/eucsf` ships the SEAL pack — a five-level
sovereignty scale (SEAL0–SEAL4) over the same Findings the wand pack
consumes. The two packs share no rule code; `--framework both` runs
each independently and persists one Assessment per framework.

| Level   | Meaning                                                            |
| ------- | ------------------------------------------------------------------ |
| SEAL0   | No evidence-backed verdict could be produced (no relevant Findings). |
| SEAL1   | Verdict that fails the framework outright — clear non-EU exposure. |
| SEAL2   | Verdict with notable dependence on a non-EU party.                 |
| SEAL3   | Verdict that is adequate — minor or low-impact dependencies only.  |
| SEAL4   | Full sovereignty — every checked surface is EU-resident.           |

The MVP rules are intentionally narrow:

| Rule ID                              | What it checks                                                                                       |
| ------------------------------------ | ---------------------------------------------------------------------------------------------------- |
| `eucsf.sov2.cert_issuer_eu`          | The leaf certificate's issuer country is in the EEA (uses `tls.issuer.issuer_country`).              |
| `eucsf.sov2.apex_jurisdiction`       | The apex domain's resolved IPs sit in EEA jurisdictions (uses `ip.asn.country` joined to `dns.a/aaaa`). |
| `eucsf.sov3.mx_jurisdiction`         | All MX hosts resolve to EEA jurisdictions (uses `dns.mx` + `ip.asn`).                                |
| `eucsf.sov4.operational_eu`          | DNS authority and TLS termination are operated from EEA-resident infrastructure.                     |
| `eucsf.sov6.no_us_hyperscaler`       | Apex and discovered third parties are not hosted on a US hyperscaler (AWS / GCP / Azure / Cloudflare). |

The SEAL level for a dimension is the **worst** rule outcome that
contributed to it; absent evidence collapses to SEAL0 rather than
silently inflating the score. This mirrors the wand "worst wins"
collapse — a reader who sees SEAL3 knows every rule is at SEAL3 or
better, not "on average".

A rule that consults Findings the scanner did not produce returns
SEAL0 with a `Verdict` naming what was missing. Operators who want
to lift a SEAL0 verdict to a real one need to look at the missing
ProbeID rather than re-running the assessor.

[ADR-0009](../explanation/adr/0009-dual-framework-assessor.md) records the
wand/SEAL dual-framework choice; the rationale is that the two
stakeholder groups (Dutch sovereignty reviewers and EU CSF
reviewers) ask the same evidence questions but want different
answer shapes, and we prefer to ship the second shape rather than
translate at read time.

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

1. Add a function to `internal/assessor/wand/rules.go` that returns
   an `assessor.Rule` with a stable `ID` (prefix `wand.<dimension>.`),
   a `Dimension`, a one-line `Description`, a `Rationale`, and a
   `Match` function.
2. Wire it into `DefaultRules`.
3. Add a test in `rules_test.go`. The pattern is: fabricate the minimal
   set of Findings the rule needs, call `r.Match`, assert on
   `Score` and `Evidence`.

Rules MUST be total. On missing attributes, return a no-evidence
result (`Score: onbekend`, empty `Evidence`, `Verdict` explaining
what was missing) rather than panicking. The engine's panic recovery
is a safety net, not a design.

See [ADR-0004](../explanation/adr/0004-assessor-rule-engine.md) for why rules
are Go functions and not a hot-reloadable DSL.
