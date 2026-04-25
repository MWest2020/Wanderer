# Delta for exporters

## ADDED Requirements

### Requirement: CSV is deterministic and diffable

The CSV exporter SHALL produce byte-stable output for a stable input
set: running the same export twice yields identical bytes.

#### Scenario: Stable ordering

- **Given** a scan `s_abc` with N findings
- **When** `wanderer export findings --scan s_abc --format csv` runs
  twice
- **Then** both outputs are byte-identical
- **And** the header row comes first
- **And** data rows are ordered by `created_at`, then `id`

#### Scenario: Empty set yields header only

- **Given** a scan ID that matches no findings
- **When** the export runs
- **Then** stdout contains exactly one line: the CSV header
- **And** the exit code is 0

---

### Requirement: JSONL is streaming-safe

The JSONL exporter SHALL emit one complete JSON object per line,
with no trailing comma, no wrapping array, and a newline after the
last line.

#### Scenario: Tail-pipe friendly

- **Given** a running export to stdout
- **When** the consumer reads line-by-line with a buffered reader
- **Then** each non-empty line `json.Unmarshal`'s cleanly into the
  target struct

#### Scenario: Evidence encoding

- **Given** a finding with non-empty `Evidence` bytes
- **When** JSONL export runs with default flags
- **Then** the line's `evidence` field is a base64-encoded string
- **When** JSONL export runs with `--include-evidence=false`
- **Then** the `evidence` field is absent from every line

---

### Requirement: Selectors compose

The exporter SHALL combine selectors (`--scan`, `--since`, `--until`,
`--probe`, `--dimension`) with AND semantics, and SHALL push them
down to the store query where possible rather than filter in memory.

#### Scenario: Combined selectors

- **Given** a store containing findings for multiple scans and probes
- **When** `wanderer export findings --probe tls --dimension juridisch`
  runs
- **Then** every output row has a `probe_id` starting with `tls.` and
  a `dimension_hint` of `juridisch`

#### Scenario: Unrecognised selector value

- **Given** a selector with a typo (e.g. `--dimension juridiscxh`)
- **When** the export runs
- **Then** the exit code is 0 (selector is valid, no rows match)
- **And** stderr contains no error — the header-only output is the
  signal

---

### Requirement: Output destinations

The exporter SHALL write to stdout by default and to a file path
supplied via `-o <path>`, with an error exit when the path is
not writable.

#### Scenario: File output

- **Given** `-o /tmp/findings.csv`
- **When** the export runs
- **Then** `/tmp/findings.csv` contains the output
- **And** stdout contains nothing

#### Scenario: Unwritable path

- **Given** `-o /root/protected/findings.csv` (path user cannot write)
- **When** the export runs
- **Then** the exit code is 1
- **And** stderr contains the OS error message
- **And** no partial file is left at the target path
