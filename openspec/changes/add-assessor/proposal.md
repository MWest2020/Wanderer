# Proposal: Assessor — Findings to DICTU Score + Report

## Intent

The MVP produces `Finding` records. This change turns a collection of
Findings into a structured `Assessment`: per-DICTU-dimension score,
per-criterium rationale, and a human-readable report with evidence
citations. It is what makes Wanderer *complete* for its founding use
case: "how sovereign are we — with evidence?"

The assessor is deterministic. Given the same Findings, it produces
the same Assessment. This is what makes it auditable: two reviewers
running the assessor on the same scan get the same numbers.

## Scope

**In scope:**

- A new data type `Assessment` in `pkg/models` capturing per-dimension
  scores, completeness flags, and per-criterium rationales.
- An `internal/assessor` package containing the rule engine and the
  DICTU rule set.
- CLI: `wanderer assess <scan-id> [--format text|markdown|json]`.
- HTTP API: `POST /scans/{id}/assessments` (run and persist),
  `GET /assessments/{id}` (retrieve).
- A markdown report format with evidence citations pointing back at
  `Finding.ID`.
- Completeness flags per dimension: when a dimension can only be
  informed by `perimeter` probes and nothing else has run, the
  dimension is marked `incomplete` and the score is labelled "based
  on external observation only".

**Out of scope:**

- PDF rendering of the report (markdown → PDF is a one-liner with
  pandoc; we do not ship pandoc in the binary).
- Multi-framework assessors (NIS2, BIO, ISO 27001). The architecture
  supports plugging them in later; this change ships only the DICTU
  rule set.
- Diff between assessments over time — that is `add-scheduling`.
- A web UI.

## DICTU dimensions informed

All five. That is the point — the assessor is the surface on which
the DICTU Toetsingsinstrument becomes addressable from Wanderer's
Findings.

- **Juridisch**: cert issuer country, IP ASN country.
- **Technologie**: third-party dependencies, vendor concentration.
- **Data & AI**: MX target, CNAME flattening, OIDC issuer (once
  `add-egress-probe` lands — flagged as incomplete until then).
- **Operationeel**: DNS delegation, cert validity, CAA posture.
- **Mens**: explicitly not addressed; the report states so.

## Passive/active boundary

N/A — the assessor consumes data, it does not probe.

## Why this ships second (after baseline, before the others)

The assessor is the keystone: it valorises existing Findings
immediately, before any new probes land. Once it exists, each new
probe change can reference which assessor rule it feeds, keeping the
scoring surface and the observation surface coherent.
