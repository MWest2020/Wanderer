# Design: Egress Probe

## Component placement

```
internal/probe/egress/
  egress.go              # Probe implementation — runs enabled scanners
  classify.go            # URL/host → category heuristics
  redact.go              # secret redaction
  scanners/
    configfiles.go       # walk configurable paths, parse known formats
    procenv.go           # /proc/<pid>/environ for readable processes
    systemd.go           # unit file EnvironmentFile= + Environment=
```

Like inventory inspectors, egress scanners implement a minimal
interface:

```go
type Scanner interface {
    ID() string
    Available() bool
    Scan(ctx context.Context) ([]Candidate, error)
}

type Candidate struct {
    Source   string  // path or pid that yielded this value
    Key      string  // env var name, config key path
    Value    string  // raw value (BEFORE redaction)
}
```

The probe then runs `classify.Classify(candidate)` on each raw
candidate, drops unknowns below a confidence threshold, redacts
secrets, and emits Findings.

## Classifier

A lookup table + regex matcher. Deliberately boring.

| Pattern                                                | Category        | Confidence |
| ------------------------------------------------------ | --------------- | ---------- |
| `s3://...`, `s3.<region>.amazonaws.com`, MinIO hosts, GCS `storage.googleapis.com` | `object_storage` | high |
| `smtp://`, `*:25/465/587`, `smtp_host=`, `MAIL_HOST=`  | `smtp`          | high       |
| `postgres://`, `mysql://`, `mongodb://`, `redis://`    | `database`      | high       |
| `issuer=https://...` in OIDC configs, `openid-configuration` URLs | `oidc` | high |
| `rsyslog`, `fluentd`, `elasticsearch.*:9200`           | `log_shipper`   | medium     |
| `*_WEBHOOK_URL=https://...`, Slack/Teams webhook hosts | `webhook`       | medium     |
| (no pattern match)                                     | `unknown`       | low        |

Vendor-specific host lists (AWS regions, Google, Azure, Scaleway,
Hetzner, Microsoft Graph, Auth0, Google Workspace) ship as a YAML
data file at `internal/probe/egress/vendors.yaml` — updatable without
code changes.

## Redaction

Before any Candidate reaches a Finding or a log line:

```go
redact.Apply(raw) returns redacted string, bool secret
```

Patterns:

- URLs with credentials: `postgres://user:pw@host` → `postgres://user:«redacted»@host`.
- Env values whose key contains `KEY`, `SECRET`, `TOKEN`, `PASSWORD`, `PW`, `API[_-]?KEY`, matched case-insensitively — value replaced with `«redacted»`.
- Values that look like AWS keys (`AKIA[0-9A-Z]{16}`), Google API keys, Slack tokens (`xox[baprs]-...`), PEM blocks, JWT-shaped strings — redacted.
- Bare numeric database passwords shorter than 4 chars are kept (they are not secrets; if an operator uses "1234" as a password, the finding should surface it).

Tests: a golden file of 30 realistic config snippets with their
expected pre/post-redaction form.

## ProbeIDs + attributes

| ProbeID                   | Subject (the external host) | Dimension    | Attributes                                                                  |
| ------------------------- | --------------------------- | ------------ | --------------------------------------------------------------------------- |
| `egress.object_storage`   | bucket host                 | `juridisch`  | `provider` (aws\|gcs\|azure\|minio\|…), `region`, `config_source`, `config_key` |
| `egress.smtp`             | relay host                  | `data_ai`    | `port`, `starttls_hint`, `config_source`, `config_key`                      |
| `egress.database`         | DB host                     | `juridisch`  | `engine` (postgres\|mysql\|…), `port`, `config_source`, `config_key`        |
| `egress.oidc`             | issuer host                 | `data_ai`    | `issuer_url` (redacted), `config_source`                                    |
| `egress.log_shipper`      | log target                  | `operationeel`| `kind` (rsyslog\|fluentd\|es), `config_source`                             |
| `egress.webhook`          | webhook host                | `technologie`| `kind` (slack\|teams\|generic), `config_source`                            |
| `egress.unknown`          | host                        | —            | `raw_context`, `confidence` — emitted only when a URL was found but not classifiable |

Every `egress.*` Finding's `Evidence` contains the redacted source
line (e.g. `S3_ENDPOINT=s3.eu-west-1.amazonaws.com`). The operator can
audit without re-scanning.

## Host resolution

After all egress candidates are collected, the probe collects unique
subject hosts and calls the IP probe once per host to annotate each
Finding with ASN + country. This reuses the existing
`internal/probe/ip` code — no new GeoLite2 dependency surface.

If the IP probe is unavailable (no GeoLite2 DB), host resolution is
skipped and Findings carry the host name without ASN/country; the
assessor correctly marks the Juridisch dimension as `partial` in
that case.

## External systems + failure modes

| System                 | Failure mode                                          |
| ---------------------- | ----------------------------------------------------- |
| Filesystem config path | Missing → inspector emits `egress.configfiles.path_missing` info finding and continues |
| `/proc/<pid>/environ`  | Permission denied on a specific PID → skip that PID, no Finding emitted for the skip (would create noise) |
| DNS (for classifying some candidates via MX/A lookup) | Timeout → candidate kept, classification lowered to `medium` or `unknown` |
| IP probe (GeoLite2)    | Unavailable → egress findings still emitted, no ASN annotation |

## Clever valkuil

Two temptations:

1. **"Let's train a model on config files."** No. Heuristics get us
   to ~90% correct on Dutch public-sector Conduction-stack configs.
   The remaining 10% are `egress.unknown` findings, which a human
   categorises — and that human decision informs a new regex, not a
   retrain cycle. A model is a black box; a regex is a PR diff.

2. **"Let's read process memory to catch runtime-computed URLs."**
   That way lies debugger-territory, ptrace, kernel capability
   surface, and a trust boundary nobody wants. Runtime-only egress
   (URLs assembled from multiple env vars at runtime) is a known
   limitation — documented in `docs/egress.md` as a gap that
   `add-egress-flow-probe` (eBPF-based) would close.

## Relation to other changes

- **`add-inventory-probe`** — shares the agent binary and
  `wanderer-agent.yaml` schema. Egress scanner section nests under
  `egress:` so it is orthogonal to inventory's `inspectors:`.
- **`add-assessor`** — Data-&-AI rules become `complete` for orgs
  running the egress probe. `docs/assessor.md` must call this out.
- **`add-exporters`** — egress Findings are a natural CSV consumer
  ("show me every external host we depend on"). No code changes
  needed there.

## Coverage

- `internal/probe/egress/classify.go` target: 90% (classification is
  pure logic, easy to test).
- `internal/probe/egress/redact.go` target: 95% — security-sensitive.
- `internal/probe/egress/scanners/*` target: 75%.

## Security review note

Required before merge. Focus: redaction correctness (false-negatives
are the danger), `/proc/<pid>/environ` access control, treatment of
root-owned config files accessible to the agent user.
