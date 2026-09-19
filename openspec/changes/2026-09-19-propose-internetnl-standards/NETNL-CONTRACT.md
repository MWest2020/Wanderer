# NETNL-CONTRACT — the `netnl-findings/v1` schema

> Lives in the **internetnl-cli** repo (this copy travels with the
> Wanderer proposal for review). It is the *only* coupling point
> between netnl and Wanderer: netnl produces this file, Wanderer
> imports it. Neither project depends on the other's code, runtime,
> or availability.

## Requirements for netnl

1. **New output format, additive.** `netnl export --format findings`
   (or `--findings-out <file>`) writes the schema below from a
   completed batch request. Existing output formats are untouched;
   CI users notice nothing unless they opt in.
2. **Versioned schema.** Top-level `"schema": "netnl-findings/v1"`.
   Breaking changes bump the version; Wanderer refuses versions it
   does not know. Additive fields are allowed within v1.
3. **Per-test granularity.** Every Internet.nl subtest appears as its
   own entry with the API's verdict verbatim (`passed` / `failed` /
   `warning` / `info` / `not_tested`). No aggregation, no
   re-weighting, no omissions — the consumer decides what to score.
4. **Variant results included.** Where the batch API reports
   per-variant detail (www/non-www, IPv4/IPv6), it is carried through
   under the test's `detail`, never flattened away.
5. **Traceability.** Each domain block carries `measured_at`
   (RFC 3339, from the API), the batch `request_id`, and the public
   `report_url`. Wanderer surfaces the report URL as evidence; the
   percentage score is carried as informational only.
6. **Determinism.** The same completed batch exports byte-identical
   files (stable ordering), so re-imports are idempotent and the file
   diffs cleanly in git-based CI archives.
7. **Fixtures.** The netnl repo ships at least one golden file per
   test type (web, mail) as fixtures; Wanderer's importer tests
   consume copies of the same fixtures, so schema drift breaks a test
   on both sides before it breaks an operator.
8. **Exit codes.** Export of an incomplete/failed batch exits
   non-zero and writes nothing — a partial file must never exist.
   (Partial *content* — e.g. `not_tested` subtests — is fine; a
   partially *written* file is not.)

## Schema draft (v1)

```json
{
  "schema": "netnl-findings/v1",
  "generated_at": "2026-09-19T10:00:00Z",
  "source": {
    "api": "internet.nl batch v2",
    "endpoint": "https://batch.internet.nl",
    "request_id": "abc123",
    "report_url": "https://internet.nl/site/voorbeeld.nl/123/"
  },
  "domains": [
    {
      "domain": "voorbeeld.nl",
      "type": "web",
      "measured_at": "2026-09-19T09:41:03Z",
      "score_percent": 87,
      "report_url": "https://internet.nl/site/voorbeeld.nl/123/",
      "results": [
        {
          "test": "web_dnssec_exist",
          "category": "dnssec",
          "verdict": "passed",
          "detail": null
        },
        {
          "test": "web_ipv6_ws_reach",
          "category": "ipv6",
          "verdict": "warning",
          "detail": {"www": "passed", "apex": "failed"}
        }
      ]
    },
    {
      "domain": "voorbeeld.nl",
      "type": "mail",
      "measured_at": "2026-09-19T09:44:17Z",
      "score_percent": 92,
      "report_url": "https://internet.nl/mail/voorbeeld.nl/456/",
      "results": [
        {
          "test": "mail_auth_dmarc_policy",
          "category": "mail_auth",
          "verdict": "failed",
          "detail": {"policy": "none"}
        }
      ]
    }
  ]
}
```

Category values v1 consumers may rely on: `dnssec`, `ipv6`,
`mail_auth`, `starttls_dane`, `rpki`, `tls_config`, `web_security`
(headers/security.txt subtests — carried for completeness; Wanderer
deliberately does not score these, see the assessor delta). Unknown
categories MUST be preserved by producers and ignored by consumers.

## v2 (named follow-up, not in this contract)

`netnl-serve` gains a completion webhook that POSTs the same v1 file
to a configured URL. The file shape does not change; only the
delivery does. Wanderer's scheduler-side ingestion is a separate
proposal on its side.
