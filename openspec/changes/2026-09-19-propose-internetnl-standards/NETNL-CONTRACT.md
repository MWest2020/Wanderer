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
   own entry with the API's verdict verbatim: `passed` / `failed` /
   `warning` / `info` / `not_tested` / **`error`**. No aggregation, no
   re-weighting, no omissions — the consumer decides what to score.
   (`error` is a test that broke on the instance, not a bad result;
   netnl already handles it — see `render.py` `_TEST_STATUS_ORDER`.)
4. **`detail` is optional and usually `null`.** Measured against batch
   API v2.7.0 on 2026-09-22: every test entry is exactly
   `{"status", "verdict"}` — including `web_ipv6_ws_reach`, where
   per-variant detail would be expected. Those variant results live in
   the HTML report, **not** in the API. The field stays in the schema
   so a future API version can fill it without a schema bump, but
   producers MUST NOT scrape the report to populate it, and consumers
   MUST NOT depend on it.
5. **Traceability.** Each domain block carries `measured_at`
   (RFC 3339), the batch `request_id`, and the `report_url`.
   `measured_at` is the batch's `request.finished_date` — the API
   publishes no per-domain timestamp, so every domain in one file
   carries the same value. `report_url` comes from the API's
   per-domain `report.url` and is **opaque**: on a self-hosted
   instance it points at that instance
   (`https://netnl.example.org/site/...`), not at internet.nl.
   Consumers display it and never construct it. Wanderer surfaces the report URL as evidence; the
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
          "category": "web_dnssec",
          "verdict": "passed",
          "detail": null
        },
        {
          "test": "web_ipv6_ws_reach",
          "category": "web_ipv6",
          "verdict": "passed",
          "detail": null
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

`category` carries the API's own category name **verbatim**, with its
`web_`/`mail_` prefix. Measured 2026-09-22: web gives
`web_appsecpriv`, `web_dnssec`, `web_https`, `web_ipv6`, `web_rpki`;
mail gives `mail_auth`, `mail_dnssec`, `mail_ipv6`, `mail_rpki`,
`mail_starttls`. (The contract previously named `starttls_dane` — no
such category exists.) A test entry in the
API carries no category of its own; the producer derives it by taking
the longest key in `results.categories` that prefixes the test name
(`web_dnssec_exist` → `web_dnssec`). A test matching no category keeps
`category: null` rather than being dropped.

Renaming these to a tidier vocabulary (`dnssec`, `tls_config`, …) was
considered and rejected: a private word list drifts from the API the
first time Internet.nl adds a category, and the mismatch surfaces as a
silently unscored rule. Mapping API category → wand rule is the
**consumer's** job and lives in Wanderer. Unknown categories MUST be
preserved by producers and ignored by consumers.

## v2 (named follow-up, not in this contract)

`netnl-serve` gains a completion webhook that POSTs the same v1 file
to a configured URL. The file shape does not change; only the
delivery does. Wanderer's scheduler-side ingestion is a separate
proposal on its side.
