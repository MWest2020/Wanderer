---
status: draft
last_reviewed: 2026-07-12
---

# Egress

Wanderer's perimeter probes see what the outside world sees.
`wanderer agent`'s inventory inspectors see what runs on the host.
The **egress probe** sees where data points when it leaves: the S3
bucket the backup job writes to, the SMTP relay the webform calls,
the OIDC issuer staff federate through, the webhook target the
alerting bot fires at.

## What it catches

The probe walks the sources you opt into and classifies what it
finds:

| Source            | Toggled by                        | What it reads                                       |
| ----------------- | --------------------------------- | --------------------------------------------------- |
| Config files      | `egress.configfiles.enabled: true` | Files under `egress.configfiles.paths` — `.env`, `.yaml`, `.yml`, `.toml`, `.ini`, `.conf`, `.json` |
| Process environments | `egress.procenv.enabled: true`  | `/proc/<pid>/environ` for every PID readable by the agent user |
| systemd units     | `egress.systemd.enabled: true`     | `Environment=` directives and `EnvironmentFile=` references in `.service`/`.socket`/`.timer` units |

For each `KEY=VALUE` pair the probe finds, the
[classifier](#classification) decides whether the value is an
egress endpoint and which category it belongs to. Plain values
like `DEBUG=true` are dropped — only URL-shaped or vendor-host
matches surface as Findings.

## What it misses (deliberately)

- **Runtime-computed URLs.** A service that assembles
  `https://${REGION}.${VENDOR}/api` at runtime from multiple env
  vars is invisible to a static config scan. Closing this gap is
  what a future eBPF-based `add-egress-flow-probe` would do — it
  is not part of this change.
- **Packet inspection.** The probe never sniffs traffic.
- **Process memory.** The probe never reads `/proc/<pid>/mem` or
  attaches via ptrace. Static config is the boundary.
- **Windows hosts.** Linux only.

## Redaction guarantee

Every value the probe handles is run through
`internal/probe/egress/redact.Apply` *before* it is stored in a
Finding's `Attributes`, written to `Evidence`, or logged via slog.
The placeholder is the literal string `«redacted»`.

A value is replaced when:

1. Its **key name** matches a known secret pattern: `*KEY*`,
   `*SECRET*`, `*TOKEN*`, `*PASSWORD*`, `*PASSWD*`, `PW`, `PWD`,
   `*ACCESS_KEY*`, `*PRIVATE_KEY*`, `*CLIENT_SECRET*`,
   `*AUTH_TOKEN*`, `BEARER` (case-insensitive).
2. Its **value shape** matches a known token format: AWS access
   key (`AKIA[0-9A-Z]{16}`), Slack tokens (`xox[baprs]-…`), GitHub
   PATs (`gh[opusr]_…`), Google API keys (`AIza…`), PEM
   `-----BEGIN PRIVATE KEY-----` blocks, JWT-shaped strings.
3. The value is a URL with inline credentials
   (`scheme://user:secret@host`) — only the password portion is
   replaced.

A `DEBUG=true` style value is left alone. Plain hostnames are left
alone — they are not secrets.

ADR-0008 records the redaction contract and its testing discipline.

## Example config

```yaml
# /etc/wanderer/agent.yaml
hostname: webapp-01.example.internal

core:
  mode: local
  db: /var/lib/wanderer/wanderer.db

scan:
  interval: 1h
  timeout: 5m

geoip:
  asn: /var/lib/wanderer/GeoLite2-ASN.mmdb
  country: /var/lib/wanderer/GeoLite2-Country.mmdb

egress:
  configfiles:
    enabled: true
    paths:
      - /etc/wanderer-sample
      - /opt/conduction/config
  procenv:
    enabled: true
  systemd:
    enabled: true
```

## Classification

Each match is tagged with a `classifier_rule` attribute on the
emitted Finding so an operator can trace why a host was flagged:

| Rule                       | Probe ID emitted          | Default dimension |
| -------------------------- | ------------------------- | ----------------- |
| `aws_s3_region_host`       | `egress.object_storage`   | `juridisch`       |
| `gcs_storage_host`         | `egress.object_storage`   | `juridisch`       |
| `azure_blob_host`          | `egress.object_storage`   | `juridisch`       |
| `s3_endpoint_keyname`      | `egress.object_storage`   | `juridisch`       |
| `oidc_issuer`              | `egress.oidc`             | `data_ai`         |
| `database_url_scheme`      | `egress.database`         | `juridisch`       |
| `smtp_keyname_or_scheme`   | `egress.smtp`             | `data_ai`         |
| `log_shipper`              | `egress.log_shipper`      | `operationeel`    |
| `webhook`                  | `egress.webhook`          | `technologie`     |
| `no_match`                 | `egress.unknown`          | (none)            |

## Host resolution

When `geoip.asn` is configured the probe uses the same GeoLite2
lookup the perimeter IP probe uses to annotate each Finding with
`asn`, `organisation`, and `country` attributes. Without GeoLite2,
the probe still emits Findings — it just emits a single
`egress.host_resolution.unavailable` info Finding per run so the
assessor can correctly mark the Juridisch dimension as `partial`.

## Operating tips

- Start with a single small `paths:` directory. Tune the noise/value
  ratio before pointing the probe at `/etc/`.
- The `egress.configfiles.unconfigured` info Finding is informational,
  not a warning — it means you enabled the scanner without supplying
  a path. Either add paths or disable the scanner.
- Audit the Evidence field on suspicious findings. Evidence is the
  redacted source line; secrets never leak there.

## Runtime flow probe (eBPF, opt-in)

The static egress probe sees only what is *configured* — env vars,
config files, systemd units. URLs an application assembles at
runtime are invisible until the request leaves the host.

The flow probe attaches a small eBPF tracepoint program to
`sys_enter_connect` for a configured sampling window, deduplicates
the observed destinations by `(ip, port)`, and emits one
`egress.flow.<category>` Finding per unique pair. The classifier
and redactor are the same ones the static probe uses, so the wire
shape is identical.

```yaml
# wanderer-agent.yaml
egress:
  flow:
    enabled: false  # default — must be explicitly opted in
    window: 30s     # optional, default 60s
```

### Capability / kernel matrix

| Host state                                | Behaviour                                                                  |
| ----------------------------------------- | -------------------------------------------------------------------------- |
| Kernel < 5.8 / no BTF                     | Inspector emits `egress.flow.unavailable` with reason "kernel BTF missing" |
| Kernel ≥ 5.8 + BTF + no CAP_BPF / non-root | Inspector emits `egress.flow.unavailable` with reason naming the cap       |
| BPF program load fails (verifier reject, etc.) | Inspector emits `egress.flow.unavailable` with the specific load error |
| All of the above + enabled: false         | Inspector is not constructed; no `egress.flow.*` finding is emitted        |
| Kernel ready + CAP_BPF + program loads    | Inspector attaches `sys_enter_connect`, drains perf events for the configured window, emits one Finding per unique destination |

The kernel attach is wired through `cilium/ebpf`. The compiled BPF
blob ships in the repo (committed by
`./scripts/bpf-build.sh`) so `go build ./...` works without
clang/llvm; only the developer who edits `bpf/connect.bpf.c` needs
the toolchain. See [ADR-0010](../explanation/adr/0010-ebpf-flow-probe.md)
for the kernel-version contract and the build path.

### Rebuilding the BPF object

After editing `internal/probe/egress/flow/bpf/connect.bpf.c`:

```sh
./scripts/bpf-build.sh
git add internal/probe/egress/flow/connect_x86_bpfel.{go,o}
git add internal/probe/egress/flow/connect_arm64_bpfel.{go,o}
git commit -m "bpf: regenerate connect program"
```

The script emits one artefact pair per build target listed in
`gen.go::go:generate` (currently `amd64,arm64`). Each pair carries
its own Go build constraint (`//go:build amd64`, `//go:build
arm64`) so a `wanderer agent` binary built for either GOARCH
embeds the matching `.o`. No runtime selection logic; selection
falls out of `go build`.

The script uses a pinned `build/bpf-builder/Dockerfile` (Fedora 42
+ clang 20 + libbpf-devel 1.5 + bpf2go), so the host needs only
docker / podman.

### Opt-in default — why

Loading a kernel program is a privilege escalation an operator must
consciously accept. The probe defaults to `enabled: false`; with
the default config the inspector is not constructed at all, so the
agent emits no `egress.flow.*` Findings (not even `unavailable`).
That keeps the absence-of-evidence path quiet on hosts where the
operator never asked for runtime flow observation.

### Reverse DNS annotation (opt-in)

The flow probe records destinations as `(IP, port)` pairs. When the
vendor classifier in `vendors.yaml` matches the IP, the Finding
carries `provider`, `region`, and (with GeoLite2 wired) `asn` /
`organisation` / `country`. When the classifier does **not** match
— generic cloud ranges, customer infrastructure, anything not in
the table — the Finding is `egress.flow.unknown` with only the raw
IP. Accurate, but lossy.

Optional reverse-DNS annotation recovers a hostname hint:

```yaml
egress:
  flow:
    enabled: true
    reverse_dns:
      enabled: false   # default — opt-in only
      timeout: 500ms   # per-lookup
```

When enabled, each unique destination IP in the sampling window is
resolved via `net.DefaultResolver.LookupAddr`. On success the
Finding's Attributes gain `reverse_dns: "<hostname>"`. On failure
(NXDOMAIN, timeout, refused, transport error) nothing is added —
no error Finding, no `reverse_dns: null`. Reverse DNS is best-effort
enrichment, not a probe in its own right.

Multiple ports to the same IP within one window produce exactly
one PTR query (per-IP cache). The cache lifetime is one tick.

**Why opt-in.** A PTR query leaks the observation back through the
host's DNS path — the resolver, every cache between, and the
authoritative server for the reverse zone all learn that *this host
saw IP X*. In a sovereignty monitor whose explicit purpose is
reducing data flight, that side-channel is non-trivial. Operators
in tight sovereignty contexts leave the toggle off and accept
IP-only Findings; operators in closed labs flip it on for richer
labels. See [ADR-0010](../explanation/adr/0010-ebpf-flow-probe.md) for the
full tradeoff.

## Customising the vendor list

The classifier's vendor / region / regex table is loaded from
`internal/probe/egress/vendors.yaml` via `//go:embed` at build time.
Operators can replace the embedded defaults with an organisation-
specific file at runtime by either:

1. Passing `--vendors /etc/wanderer/vendors.yaml` to `wanderer agent`
2. Setting `WANDERER_VENDORS=/etc/wanderer/vendors.yaml` in the agent's
   environment (used when `--vendors` is empty).

A loaded override file replaces the defaults wholesale — there is no
merge mode, so the file you ship must include every vendor entry you
want active. Schema:

```yaml
log_shippers:
  - host_contains: vendor.example.nl  # substring match against the
    rule_id: example_logger           #   destination host
log_shipper_key_regex: "(?i)(loki|elastic_host)"

webhooks:
  - host_contains: hooks.example.nl
    rule_id: example_webhook
webhook_key_regex: "(?i)webhook"

object_storage:
  aws_regional_regex: '^s3[.\-]([a-z]{2}-[a-z]+-\d)\.amazonaws\.com$'
  gcs_host_contains: storage.googleapis.com
  azure_host_contains: blob.core.windows.net
```

`object_storage.*` keys are required. `log_shippers` / `webhooks` may
be empty, but every entry must carry both `host_contains` and
`rule_id`. `*_key_regex` fields are optional fallbacks for
configuration keys whose value's host does not match a known vendor
(e.g. self-hosted Loki at an internal hostname).

The agent fails fast on a missing file, malformed YAML, an invalid
regex, or a missing required key: it prints the offending file path
and parse position to stderr and exits non-zero. Operators get a
hard signal rather than a silent fall-back to the embedded list.
