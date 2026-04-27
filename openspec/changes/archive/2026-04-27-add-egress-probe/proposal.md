# Proposal: Egress Probe — Waar Gaan Onze Data Heen?

## Intent

The question the MVP cannot answer — "does our data stay in the EEA?"
— requires looking at application configurations from inside: the S3
endpoints our backup jobs target, the SMTP relay our webforms use,
the OIDC issuer our staff federate through, the logging pipeline
shipping to someone else's cluster. External observation sees the
perimeter; inventory sees what runs; **egress** sees where data
*points to when it leaves*.

This change adds `internal/probe/egress/` as the third
observation-modus alongside perimeter and inventory. It runs inside
`wanderer agent` (same binary mode as inventory) and inspects
application configurations for URL-like and host-like references,
classifies them by protocol (object-storage, SMTP, OIDC, DB, log
shipper, webhook target), resolves the host, and emits Findings that
feed the assessor's Data-&-AI and Juridisch rules.

## Scope

**In scope:**

- New probe package `internal/probe/egress/` with inspectors for:
  - **app-config files**: YAML, TOML, INI, `.env`, JSON in
    configurable paths (`/etc/`, `/var/www/*/config/`,
    `/opt/*/config/`). Extract URLs and host references.
  - **environment of running processes**: `/proc/<pid>/environ` for
    each process the agent user can read. Look for `*_URL`, `*_HOST`,
    `DATABASE_URL`, `S3_*`, `SMTP_*`, `OIDC_*` patterns.
  - **systemd unit environments**: `EnvironmentFile=` references and
    `Environment=` directives parsed from unit files.
- Classifier `internal/probe/egress/classify.go`: heuristic mapping
  from URL/host to a category (`object_storage`, `smtp`, `oidc`,
  `database`, `log_shipper`, `webhook`, `unknown`).
- Host resolution: use the IP probe (already present) to resolve each
  discovered host to ASN + country, so the assessor can answer the
  EEA question end-to-end.
- New Finding ProbeIDs: `egress.object_storage`, `egress.smtp`,
  `egress.oidc`, `egress.database`, `egress.log_shipper`,
  `egress.webhook`, `egress.unknown`.
- Opt-in per inspector via `wanderer-agent.yaml`.

**Out of scope:**

- **eBPF / netflow observation** (watching real traffic leave the
  host). Plausible future change `add-egress-flow-probe`; the value
  is high but the kernel-version matrix and operational cost is too
  high to bundle with the simpler config-scanning this change does.
- **Packet inspection** or TLS MITM. Never. That violates Wanderer's
  passive-observation soul.
- **Secrets extraction**. The egress probe encounters secrets (API
  keys, DB passwords) while parsing configs. It MUST redact them and
  MUST NOT write them to Findings or logs. This is a hard requirement,
  not a nice-to-have.
- **Windows hosts** (same as inventory probe; follow-up).

## DICTU dimensions informed

- **Juridisch** (primary): host ASN + country of object-storage, DB,
  SMTP, OIDC endpoints. Directly answers "does data cross a
  non-EEA border."
- **Data & AI** (primary): data residency by service type. An OIDC
  federation to a US IdP is a Data-&-AI finding even if the connection
  itself is inside the EEA.
- **Operationeel**: log-shipper and backup targets reveal who can see
  our operational data.
- **Technologie**: vendor concentration across egress endpoints.

## Passive/active boundary

The egress probe is **passive towards the network**: it never makes
outbound connections itself except to resolve discovered hostnames
through the IP probe (which is a local mmdb lookup, not a network
call) and, for classification, an occasional DNS lookup for MX/A on
discovered hostnames. DNS lookups here are identical in pattern to
the perimeter DNS probe — they are allowed.

It is **active towards the host**: it reads config files (typically
world-readable or group-readable), `/proc/<pid>/environ` (requires the
agent user to be able to read it — usually means same UID or root),
and systemd unit files. It makes **no writes**.

Secrets-redaction is a hard requirement: any value matching known
secret patterns (long random strings in `*_KEY`, `*_SECRET`,
`*_PASSWORD` variables; AWS key regex; PEM blocks) is replaced with
`«redacted»` before storage or logging.

## Parallel-safe

Shares `internal/agent/config.go` and `cmd/wanderer/agent.go` with
`add-inventory-probe`. Implementation order matters here: inventory
lands first, then egress extends the same agent-binary surface. If
implemented in parallel by two agents, they MUST coordinate on the
YAML schema additions — we solve this by making inventory land the
initial schema and egress add keys under a separate `egress:` block.

Otherwise self-contained: new package under `internal/probe/egress/`,
new classifier, reuses the IP probe for ASN/country resolution.
