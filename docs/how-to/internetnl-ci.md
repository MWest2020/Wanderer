---
status: draft
last_reviewed: 2026-09-22
---

# Feed Internet.nl results from CI

Wanderer's `standards` dimension (DNSSEC, SPF/DKIM/DMARC,
STARTTLS/DANE, IPv6, RPKI, TLS configuration) never talks to
Internet.nl itself — it only reads a file. The measurement is
**netnl**'s job (internetnl-cli, a separate tool), already running in
GitHub Actions for most operators. This page is the three-command
route from "a domain measured in CI" to "a scored `standards`
dimension in Wanderer", and the one exit-code detail a CI step must
check.

See [assessor.md](../reference/assessor.md#the-standards-dimension)
for how the six rules score, and
[Permanent non-goals](../explanation/internetnl-non-goals.md) for why
Wanderer does not measure any of this itself.

## The three commands

Run on the host (or CI job) with `netnl` installed, then on the host
(or CI job) with `wanderer` installed — they do not need to be the
same machine, since the coupling point is the exported file, not a
network call between the two tools.

```sh
# 1. Ask netnl's configured Internet.nl (batch API v2) instance to
#    measure a target. --no-poll returns immediately with a request ID
#    instead of blocking until the batch finishes.
internetnl submit example.nl --type web --no-poll

# 2. Once the batch has finished (poll internetnl's own status command,
#    or re-run this on a later CI tick), export it in the shape
#    Wanderer's importer understands.
internetnl results <request-id> --format findings \
  --findings-out findings.json

# 3. Hand the file to Wanderer. The target domain must already exist
#    in Wanderer (`wanderer org add` / an existing scanned target) —
#    an unknown domain is skipped with a WARN, not an error, so the
#    rest of a multi-domain batch still imports.
wanderer import internetnl --db wanderer.db findings.json
```

Repeat step 1–3 per measurement type (`--type web`, `--type mail`) —
they are two independent batches with two independent request IDs.
Wanderer correlates a target's latest web and latest mail import
automatically (`Store.FindingsForAssessment`); neither import needs to
wait for the other.

After both imports, score the target the same way any other scan is
scored — the documented route this file contract is built around:

```sh
wanderer assess --db wanderer.db <scan-id>
```

Any scan ID for the target works, including one from a plain perimeter
scan that never touched Internet.nl — the CLI's `assess` command
correlates the latest imported standards evidence onto whichever scan
you point it at, so you do not need to assess the import-kind scan
itself.

## The exit code a CI step must trust

`internetnl results <id> --json` (netnl's original, general-purpose
export) intentionally exits **0** even when the batch is still
running — it writes `"domains": null` and `request.status: "running"`,
because polling an in-progress batch and getting a partial answer is
normal, expected behaviour for that command.

**`--format findings` does not inherit that.** Export of an
incomplete or failed batch through `--format findings` exits
**non-zero and writes nothing** — no `findings.json` is created at
all. This is the one thing a CI step feeding Wanderer must actually
check: gate the next pipeline step (`wanderer import internetnl ...`)
on this command's exit code, not merely on the file's existence.

```sh
if ! internetnl results "$REQUEST_ID" --format findings \
    --findings-out findings.json; then
  echo "internetnl: batch $REQUEST_ID not ready yet — retry next run" >&2
  exit 1
fi
```

Trusting a `0` exit from the wrong command here is exactly the failure
this guards against: a CI step that calls `--json` (or ignores exit
codes altogether) can archive an empty-looking artifact, report
success, and leave Wanderer importing a file with zero domains — which
renders as "not measured" for every rule, silently, with no error
anywhere in the chain.

## What "not measured" means downstream

A target that has never been imported (or whose import predates
`standards.max_age`, default 30 days) scores `onbekend` on every
`wand.standards.*` rule, with the reason `not_measured` or
`measurement_stale`. That is a **measurement gap**, not a failing
score — the fleet screen and the per-target worst-score pill both
already exclude `onbekend` dimensions from dragging a target down (see
[assessor.md](../reference/assessor.md#reason-codes)). Re-running
`import internetnl` with a fresh export is enough to pick a target back
up; nothing needs to be re-scanned on Wanderer's side.
