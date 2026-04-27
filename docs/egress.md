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
