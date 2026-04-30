## Context

The IP probe (`internal/probe/ip`) reads a MaxMind GeoLite2-ASN
mmdb file at startup. When the file is absent, the probe's
constructor returns `ip.unavailable` semantics — every
subsequent Inspect emits one `ip.unavailable` info Finding and
no `ip.asn` records.

This is the right runtime behaviour (graceful degradation) but
the wrong onboarding behaviour: a fresh-installed operator has
no signal that something is missing. The probe runs, the scan
completes, the assessment is filled with `onbekend` rationales —
all silent, all green at the CLI level.

`cmd/wanderer/scan.go` already exposes `--geoip` and
`--geoip-country` flags with `WANDERER_GEOIP_ASN` and
`WANDERER_GEOIP_COUNTRY` env-var fallbacks. The flag value is
empty by default; the IP probe constructor handles empty as
"unavailable".

## Goals / Non-Goals

**Goals:**

- A new operator running `wanderer scan example.nl` for the first
  time on a host without GeoLite2 sees a single, actionable
  warning to stderr at startup pointing at the documentation.
- An operator who consciously runs without GeoLite2 (an air-
  gapped audit lab; CI) can silence the warning without
  hand-editing source.
- The setup steps are written once and linked from architecture,
  operator, and tutorial docs.
- The test suite can run with a stub mmdb so the
  `internal/probe/ip` tests cover both the populated path and
  the empty-but-configured path without a real MaxMind license.

**Non-Goals:**

- Bundling GeoLite2 with Wanderer. License terms forbid
  redistribution; we cannot ship the file.
- Fetching GeoLite2 automatically. That requires a license key
  in the agent's environment, which is an operator concern.
  We document `geoipupdate` instead.
- Replacing GeoLite2 with a different ASN/country source. That
  is a separate design conversation (RIPE bulk feeds, DB-IP free
  tier, …) and out of scope here.

## Decisions

### Decision 1: One warning at startup, not per-finding

The IP probe already emits an `ip.unavailable` info Finding per
scan when GeoLite2 is missing. Adding a second signal at scan-
time would be redundant. Adding one stderr warning **at process
startup** is the right place — it's the first time the operator
opens the CLI and asks "is this configured?".

### Decision 2: `--no-geoip` is opt-out, not opt-in

The flag default is "GeoLite2 expected → warn if missing" rather
than "GeoLite2 optional → warn if requested but missing". The
asymmetry matches how operators think: GeoLite2 should be there;
its absence is the exception worth flagging. An audit lab that
opts out adds the flag once.

The opt-out flag is `--no-geoip`. The env equivalent is
`WANDERER_GEOIP_OPTIONAL=1`. Both silence the warning; neither
changes the probe's runtime behaviour.

### Decision 3: Stub mmdb via a script, not a vendored binary

A tiny pre-built mmdb in `testdata/` would be the smallest possible
change but obscures how the file was produced. A script + a
`make geoip-stub` target makes the production reproducible and
keeps `testdata/` from becoming a binary blob someone
might one day "improve" by hand.

The stub-builder uses `github.com/maxmind/mmdbwriter` (separate
from the read library we already depend on). It is a build-time
dep only (used in `scripts/geoip-stub.sh`); we could also write
the script in pure shell + `mmdbinspect` if we wanted to skip
the dep entirely, but that complicates the build for marginal
benefit.

If the dep is rejected during review, we fall back to vendoring
a checked-in 64-byte stub file with a README explaining its
provenance.

### Decision 4: Documentation lives in `docs/operator.md`

`docs/operator.md` is already the operator-facing entry point;
GeoLite2 is an operator concern, not an architecture decision
(no ADR needed). A new H2 section "GeoLite2 setup" with
subsections for free-tier, geoipupdate, and licence-key handling
is enough. `docs/architecture.md` and `docs/tutorial.md` link
into the section.

## Risks / Trade-offs

[Risk] An operator interprets the stderr warning as a fatal
error and aborts the scan. → Mitigation: the warning text starts
with "warning:" not "error:" and explicitly states "scan will
continue with reduced assessment coverage; see docs/operator.md
for setup instructions".

[Risk] CI environments that run `wanderer` for smoke tests now
emit a noisy warning. → Mitigation: CI sets
`WANDERER_GEOIP_OPTIONAL=1`; the warning becomes a single
info-level log line that operators do not see.

[Risk] An operator copies the docs' suggested `geoipupdate`
crontab without thinking and the cron user has no shell.
→ Mitigation: docs use a systemd-timer example as the primary
recommendation, with crontab as an alternative for hosts that
don't run systemd.

**Clever valkuil:**

1. **Auto-downloading GeoLite2 on first run.** Tempts because
   it would "just work". Wrong because (a) it requires a
   license key the binary cannot reasonably acquire, and (b) it
   would make the agent network-active at startup, which the
   passive-observation contract explicitly rules out.
2. **Embedding a tiny test mmdb at build-time.** Saves a script
   step but bakes test data into production binaries.
3. **Replacing the warning with a hard failure.** Some operators
   do legitimately run without GeoLite2 (offline labs, CI smoke
   tests). A fatal default would be wrong.

**External systems & failure modes:**

- MaxMind's GeoLite2 download service — operator concern, not
  agent concern. We document the URL but never call it.
- `geoipupdate` — operator-managed daemon; we document its use
  but do not depend on it.
- `mmdbwriter` (test-only) — pure Go, no network, no failure
  modes beyond malformed input. Used only in the stub script.
