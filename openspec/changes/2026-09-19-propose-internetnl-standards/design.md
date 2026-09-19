# Design notes: standards dimension (Internet.nl via netnl)

## External systems and failure modes

| System | Used for | Failure mode | Handling |
| --- | --- | --- | --- |
| netnl-findings file (v1) | all standards findings | absent, schema-version mismatch, malformed entries, unknown domains, stale measured_at | absent → dimension "not measured"; version mismatch → refuse import with the expected version named; malformed entries → WARN + skip (Amass precedent), partial import is first-class; unknown domain → WARN + skip; stale → import fine, rules score onbekend |
| Internet.nl batch API | none directly | — | v1 never touches it; that is netnl's job. Keep it that way. |
| netnl-serve facade (v2 only) | scheduled submit/ingest | auth, timeout, batch pending | out of scope here; named follow-up proposal |

## Verdict mapping

Internet.nl per-test verdicts (`passed` / `failed` / `warning` /
`info` / `not_tested`) map per category:

- all relevant tests `passed` → **soeverein**
- any `warning`, or a mix of passed/failed within the category →
  **voldoende**
- category substantively `failed` → **afhankelijk**
- `not_tested`, no finding, or measurement older than
  `standards.max_age` → **onbekend**, with the reason ("not measured"
  vs "measurement stale (measured 2026-07-01)") in the verdict text.

The wand scale is reused unchanged for consistency across the pack;
verdict *text* speaks compliance language ("DNSSEC signed and
valid"), not sovereignty language.

## Import semantics

- Findings persist under a scan of kind `import` referencing the
  source file hash and the Internet.nl request ID, so evidence stays
  traceable and re-imports of the same file are idempotent.
- The assessor correlates a target's newest standards findings across
  scan kinds; a perimeter scan never erases imported findings and
  vice versa.
- One file may carry many domains (batch output); one import call
  handles the whole fleet.

## The clever valkuil

Recomputing Internet.nl's score. The batch API exposes subtests,
weights change between Internet.nl releases, and a home-grown
percentage will disagree with the public report the bestuurder can
open in a browser — instantly destroying trust in both tools. Map
verdicts, link the report URL, never aggregate. (Second, smaller one:
falling back to scraping the HTML report when the API omits a field.
If the API doesn't expose it, the rule doesn't exist.)

## Why a file, not a library or an RPC

- The schema is testable with fixtures on both sides; neither repo
  imports the other; Go and Python stay decoupled.
- CI today: netnl step produces the artifact, `wanderer import`
  consumes it — works without the facade, without network, in
  air-gapped deployments.
- v2 slots in without changing the consumer: the facade delivers the
  same file over a webhook.
