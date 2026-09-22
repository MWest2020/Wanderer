---
status: draft
last_reviewed: 2026-09-22
---

# Wanderer in your pipeline

`action.yml` at the repository root is a composite GitHub Action that
runs a Wanderer scan against one domain and writes the verdict to the
job summary. It exists so a sovereignty check can sit next to your
linters and tests, instead of being a website someone has to remember
to visit.

## What it does

1. Downloads the requested (or latest) published release archive for
   the runner's OS/arch, verifies it against the release's
   `checksums.txt`, and extracts the `wanderer` binary.
2. Runs `wanderer scan <domain>` into a throwaway SQLite database on
   the runner, then `wanderer assess --format json` against the
   resulting scan.
3. Collapses the assessment into an overall verdict — the same "worst
   wins" rule documented in [the assessor reference](../reference/assessor.md#reading-a-score),
   applied across dimensions instead of within one — and writes a
   summary to `$GITHUB_STEP_SUMMARY`: the score, which rule(s) decided
   the verdict, and one remediation sentence per rule that scored
   `afhankelijk`.

## What it does not do

The action runs entirely in the runner it was invoked from. It talks
to the domain you asked it to scan, and to this repository's GitHub
releases to fetch the binary — nothing else. No telemetry, no
"phoning home", and no default GeoIP database download from a server
of ours: bring your own `mmdb` via `geoip`, or accept that
jurisdiction-dependent answers come back `onbekend`. See [Not in
scope](../../openspec/changes/2026-09-21-distributie/proposal.md) for
the reasoning.

## Usage

```yaml
- uses: MWest2020/wanderer@v0.7.0
  id: wanderer
  with:
    domain: onze.gemeente.nl
    geoip: ./GeoLite2-ASN.mmdb # optional
    fail-on: "" # "afhankelijk" | "onbekend" | "" (default)
    version: "" # a release tag, e.g. "v0.7.0"; empty = latest

- run: echo "${{ steps.wanderer.outputs.verdict }}"
```

See [`docs/examples/wanderer-scan.yml`](../examples/wanderer-scan.yml)
for a complete workflow, including a scheduled weekly scan.

### Inputs

| Input     | Required | Default  | Meaning                                                              |
| --------- | -------- | -------- | ---------------------------------------------------------------------- |
| `domain`  | yes      | —        | Domain to scan.                                                      |
| `fail-on` | no       | `""`     | Verdict that fails the step: `afhankelijk` or `onbekend`.            |
| `geoip`   | no       | `""`     | Path to a GeoLite2-compatible `mmdb` for IP/ASN jurisdiction lookups. |
| `version` | no       | `""`     | Release tag to install. Empty installs the latest release.           |

### Outputs

| Output        | Meaning                                                          |
| ------------- | ----------------------------------------------------------------- |
| `score`       | Questions answered with evidence vs. total, e.g. `"12/18"`.       |
| `verdict`     | Overall verdict: `soeverein` \| `voldoende` \| `afhankelijk` \| `onbekend`. |
| `report-path` | Path to the full JSON assessment report, on the runner.           |

## Why `fail-on` defaults to empty

A sovereignty verdict is a judgement about where an organisation's
infrastructure lives and who is accountable for it — it is not the
same kind of fact as "the tests pass" or "the code compiles". Turning
someone's pipeline red on a `afhankelijk` verdict is a policy decision
their organisation gets to make, not a default this action should
impose. Set `fail-on: afhankelijk` (or `onbekend`, if you want
incomplete evidence to block too) once you have made that call.

## Without a GeoIP database

Several wand rules (hosting jurisdiction, mail jurisdiction, DNS
jurisdiction, hyperscaler detection) read IP→ASN→country data from a
local GeoLite2-compatible database. Without `geoip` set, those
questions score `onbekend` — not `voldoende`, and never silently
skipped. The job summary says so explicitly rather than quietly
shipping a shorter list of questions.

## Retrieving the full report

`report-path` points at a JSON file on the runner's filesystem, which
disappears when the job ends. Upload it as an artifact if you want to
keep it:

```yaml
- uses: actions/upload-artifact@v4
  with:
    name: wanderer-report
    path: ${{ steps.wanderer.outputs.report-path }}
```
