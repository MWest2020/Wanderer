## Why

Without a GeoLite2 ASN database the IP probe emits a single
`ip.unavailable` info Finding and skips the per-host
ASN/country lookup. That cascades: every DICTU rule that depends
on `ip.asn` (the entire `technologie` dimension and most of the
`juridisch` dimension) returns `onbekend`. A new operator
running their first `wanderer scan` against `conduction.nl`
sees a half-blank assessment and reasonably concludes "this
tool is broken", when in fact one missing input file would have
populated the whole picture.

The MaxMind GeoLite2 download story is awkward (free account,
license key, periodic re-download, redistribution forbidden),
which is why the file is **not** vendored. But that means
documentation is the only signpost — and the current
`docs/operator.md` does not even mention GeoLite2 by name.

## What Changes

- Document the GeoLite2 setup explicitly in
  `docs/operator.md` (or a new `docs/geoip.md` referenced from
  there): MaxMind account, license key, the `geoipupdate`
  approach vs the manual download, where Wanderer expects the
  file (`--geoip` flag, `WANDERER_GEOIP_ASN` env), and the same
  for the optional country DB (`--geoip-country`,
  `WANDERER_GEOIP_COUNTRY`).
- Add a `--geoip` requirement check at the beginning of every
  CLI entry point that uses the IP probe (`scan`, `serve`).
  Today a missing GeoLite2 silently degrades to `ip.unavailable`;
  after this change the CLI emits one warning to stderr at
  startup naming the missing file and pointing at the docs.
  An explicit `--no-geoip` flag (or `WANDERER_GEOIP_OPTIONAL=1`)
  silences the warning for environments that consciously run
  without ASN annotation.
- Wire a `make geoip-stub` (or a small `scripts/geoip-stub.sh`)
  that produces an empty-but-valid mmdb file for offline test
  runs and CI — so the test suite can opt into "GeoLite2 is
  configured but returns no matches" without a real download.

## Capabilities

### New Capabilities

(none — purely operational/documentation hygiene plus a one-line
warning in the CLI)

### Modified Capabilities

- `project-hygiene`: `docs/operator.md` SHALL include explicit
  GeoLite2 setup instructions; the architecture doc and tutorial
  SHALL link to them.

## Impact

**Code**:
- `cmd/wanderer/scan.go` and `cmd/wanderer/serve.go`: add a
  startup check that warns when no `--geoip` / env is set and
  `--no-geoip` is not present. The check is one log.Warn line;
  existing `ip.unavailable` Finding behaviour is unchanged.
- `cmd/wanderer/version.go` (or a similar shared helper): a
  `func geoipPathFromFlags(...)` so the warning text is identical
  across commands.
- `scripts/geoip-stub.sh`: produces a minimal valid GeoLite2 mmdb
  file via `mmdbwriter` (we already depend on
  `oschwald/maxminddb-golang` for reading; the writer is a
  separate small dep and only used from this script).

**APIs**: none.

**Dependencies**: optional new go.mod entry for
`github.com/maxmind/mmdbwriter` — used only by the stub-builder
script. Could also be omitted by checking in a tiny pre-built
empty mmdb under `testdata/`. Decision deferred to design.md.

**Read-only contract**: N/A — CLI/docs only.

**DICTU dimensions informed**: every dimension benefits, since
GeoLite2 unblocks the `ip.asn` ProbeID that gates many rules.

**Passive/active boundary**: N/A — local file lookup only.
