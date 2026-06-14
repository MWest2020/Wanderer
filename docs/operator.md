# Operator Guide

This document describes how to run Wanderer, interpret its output, and
operate the single-tenant HTTP API. It covers the MVP only —
scheduling, authentication, and the wand assessor ship as separate
changes.

## Install

Build from source:

```sh
make build
./bin/wanderer version
```

Or run directly with `go run`:

```sh
go run ./cmd/wanderer scan example.nl
```

## Config file for `wanderer serve`

For long-running deployments — systemd, Compose, anywhere a wall
of CLI flags is awkward — point `wanderer serve` at a YAML config
file:

```sh
wanderer serve --config /etc/wanderer/serve.yaml
```

The same can be set via the `WANDERER_CONFIG` env var. Every
field is optional; an empty file is equivalent to no `--config`
at all.

```yaml
# /etc/wanderer/serve.yaml
listen: "127.0.0.1:8080"
db: "/var/lib/wanderer/wanderer.db"

geoip:
  asn:     "/var/lib/wanderer/geoip/GeoLite2-ASN.mmdb"
  country: "/var/lib/wanderer/geoip/GeoLite2-Country.mmdb"
  optional: false      # equivalent to --no-geoip

ui:
  enabled:  true
  htpasswd: "/etc/wanderer/htpasswd"

oidc:                  # optional — Nextcloud (or any OIDC IdP) login for /ui/
  provider_url:        "https://cloud.example.nl"
  client_id:           "wanderer"
  client_secret_file:  "/etc/wanderer/oidc-secret"
  redirect_url:        "https://wanderer.example.nl/ui/oauth/callback"
  # scopes:            [openid, profile, email]   # default; override if needed
  # session_ttl:       12h                         # hard session lifetime
  # revalidate_interval: 0s                        # 0 = check the IdP every request
  # cookie_secure:     true                         # set false only for local http

schedules: "/etc/wanderer/schedules.yaml"

scan:
  per_probe_timeout:     30s
  budget:                2m
  user_agent:            "Wanderer/1.0"
  allow_private_targets: false
```

The parser is **strict**: a typo like `htpasswrd` for `htpasswd`
fails the process at startup with an error naming the bad field,
never silently dropped to a default.

### Setting precedence

When the same setting is provided in more than one place, the
highest-precedence layer wins:

1. **CLI flag** explicitly passed (`--addr :9090`)
2. **Environment variable** explicitly set (`WANDERER_LISTEN`)
3. **YAML config** value
4. **Hard-coded default**

So an operator can lay down `serve.yaml` as the durable source of
truth, then override one knob from a one-off invocation without
editing the file:

```sh
wanderer serve --config /etc/wanderer/serve.yaml --addr :7070
# YAML says :8080, flag wins → server listens on :7070
```

A flag explicitly set to its default value (e.g.
`--ui=false`) still counts as "explicitly set" and overrides a
YAML `ui.enabled: true`.

### Sample systemd unit

```ini
# /etc/systemd/system/wanderer.service
[Unit]
Description=Wanderer sovereignty monitor
After=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/wanderer serve --config /etc/wanderer/serve.yaml
Restart=on-failure
User=wanderer
Group=wanderer

[Install]
WantedBy=multi-user.target
```

## Organisations

Wanderer groups Targets — perimeter domains and agent hosts — under
**organisations**. Every Target belongs to exactly one organisation.
A fresh database starts with one seeded organisation (slug
`default`); operators with one customer rename it to their real
handle, operators with many add organisations as needed.

### First-run

`wanderer scan example.nl` without `--organisation` attaches the
scan to the seeded `default` organisation and emits a one-line
stderr nudge pointing at the flag and the rename command.

```sh
wanderer org rename --slug default --new-slug acme --name "ACME B.V."
```

That converts the seed in place — every existing Target keeps the
same ID, just under the new slug + name.

### Day-to-day

```sh
wanderer org add --slug customer1 --name "Customer One"
wanderer org list
wanderer org show acme
wanderer scan --organisation acme example.nl
```

Bulk seeding from a YAML file (idempotent — running it twice
updates names but does not duplicate):

```yaml
# /etc/wanderer/orgs.yaml
organisations:
  - slug: customer1
    name: Customer One
  - slug: customer2
    name: Customer Two
    description: Pilot deployment
```

```sh
wanderer org add --from-yaml /etc/wanderer/orgs.yaml
```

### Organisation precedence in `wanderer scan`

The flag `--organisation <slug>` wins over `WANDERER_ORGANISATION`
in the env, which wins over `scan.organisation` in `serve.yaml`,
which falls back to `default`. So an operator can lay down the
config once and only override per-invocation when they need to.

### Schedules

Each entry in the schedules YAML can carry `organisation: <slug>`.
Entries without a slug fall back to `scan.organisation` in
`serve.yaml`, then to `default`. The scheduler resolves the slug at
each tick — `wanderer org rename` takes effect without restarting
the server.

```yaml
# /etc/wanderer/schedules.yaml
schedules:
  - name: customer1-daily
    cron: "0 3 * * *"
    target:
      domain: customer1.example
    organisation: customer1   # explicit override

  - name: acme-internal
    cron: "0 4 * * *"
    target:
      domain: app.acme.internal
    # uses scan.organisation from serve.yaml
```

### Agent

Hosts running `wanderer agent` declare their organisation in the
agent YAML:

```yaml
core:
  mode: local
  db: /var/lib/wanderer/wanderer.db
  organisation: acme
```

An empty `organisation` falls back to `default`; an unknown slug
fails fast at agent startup, before any Findings land.

### UI

`/ui/` lists every registered organisation with a target count and
a drill-in link. Per-organisation dashboards live at
`/ui/orgs/{slug}` — same DAR shape (Dashboard / Analysis /
Reporting) but filtered to that organisation. The Reporting page
takes an optional `?org=<slug>` query parameter to filter the
cross-target view.

### Nextcloud login (OIDC)

By default `/ui/` is protected by HTTP Basic against the `htpasswd`
file. Organisations that already run a self-hosted Nextcloud can
instead accept Nextcloud as the login provider, so operators sign
in once at their Nextcloud and a disable there cuts off Wanderer
access. Configure the `oidc:` block (above) and install the
Nextcloud **OpenID Connect** (`user_oidc`) app's *provider* side, or
register Wanderer as an OIDC client in any standards-compliant IdP.

How it behaves:

- **Lazy discovery.** Wanderer does **not** contact the provider at
  startup — it discovers `.well-known/openid-configuration` on the
  first login. So `wanderer serve` boots even when Nextcloud is
  down, and OIDC starts working the moment the provider answers.
- **Server-side sessions.** A successful login sets an opaque,
  `HttpOnly` cookie backed by a row in the `ui_sessions` SQLite
  table. There is no JWT in the cookie and nothing to revoke-list.
- **Revocation.** On each request past `revalidate_interval`,
  Wanderer re-checks the session against the provider's `userinfo`
  endpoint (refreshing the access token first). A Nextcloud-side
  disable makes that check fail and the session is deleted. The
  default `0s` revalidates on **every** request — a disable then
  cuts access on the very next page load, at the cost of one
  userinfo call per request. Raise it (e.g. `60s`) to trade
  immediacy for fewer IdP round-trips. Note that `0s` also couples
  session *survival* to IdP uptime: while Nextcloud is unreachable,
  revalidation fails closed and live sessions are dropped — by
  design (fail-closed revocation), with the htpasswd break-glass as
  the escape hatch. A small non-zero interval rides out brief IdP
  blips without logging everyone out.
- **Break-glass with htpasswd.** If both `oidc:` and `htpasswd:`
  are set, valid HTTP Basic credentials are still accepted. This is
  the escape hatch for an OIDC outage: when Nextcloud is
  unreachable, an operator with an htpasswd entry can still reach
  `/ui/`. Keep at least one htpasswd account for this reason.
- **The client secret never lives in YAML.** `client_secret_file`
  points at a file (mirroring `hmac_secret_file` for the agent),
  readable only by the Wanderer process user.

Scope of the first wave: authentication only (any user the IdP
knows can browse the read-only UI; group-based authorisation is a
later wave), and a single OIDC provider per Wanderer instance.

### MCP

Three new methods cover the AI-agent surface:

- `org_list` — every organisation with metadata
- `org_show` — one organisation by slug
- `org_targets` — Targets attached to an organisation

## GeoLite2 setup

The IP probe resolves observed IPs to ASN + country via MaxMind's
GeoLite2 database. This is the input that unblocks the wand
**technologie** dimension and most of the **juridisch** dimension —
without it, every rule that depends on `ip.asn` returns `onbekend`,
and an operator looking at the assessment sees a half-blank picture.

GeoLite2 is **free** but the licence forbids redistribution, so
Wanderer cannot ship it; you fetch it once and keep it fresh.

### One-time setup

1. Sign up at <https://www.maxmind.com/en/geolite2/signup>
   (free MaxMind account).
2. From your account portal, generate a **license key**.
3. Download `GeoLite2-ASN.mmdb` (and optionally
   `GeoLite2-Country.mmdb`) — either via the portal's manual
   download, or via `geoipupdate` (recommended for hosts that
   should stay current).

### Paths Wanderer expects

| Env var                     | CLI flag         | Required for     |
| --------------------------- | ---------------- | ---------------- |
| `WANDERER_GEOIP_ASN`        | `--geoip`        | ASN + country    |
| `WANDERER_GEOIP_COUNTRY`    | `--geoip-country`| Country only (optional; defaults to the ASN file) |

The flags apply to `wanderer scan`, `wanderer serve`, and the
agent (where wired into the egress probe). When neither is set
and the opt-out below is also absent, both `wanderer scan` and
`wanderer serve` emit one warning to stderr at startup naming
the missing flag and pointing at this guide. The scan still
completes; the warning is informational.

```
warning: GeoLite2 ASN database is not configured — scan will
continue with reduced assessment coverage. Pass --geoip <path>
(or set WANDERER_GEOIP_ASN), or pass --no-geoip to silence
this warning. See docs/operator.md for setup.
```

If the database file is configured but missing or corrupt at
runtime, Wanderer fails **fast at startup**, never mid-scan.

### Opt-out for offline labs and CI

Hosts that consciously run without ASN annotation (offline audit
labs, CI smoke tests, demos) silence the warning with either:

- the `--no-geoip` CLI flag, or
- the `WANDERER_GEOIP_OPTIONAL=1` environment variable.

Neither changes the runtime behaviour — the IP probe still emits
its single `ip.unavailable` info Finding and the rest of the scan
continues — the opt-out only suppresses the startup warning.

### Recommended: keep GeoLite2 fresh with `geoipupdate`

MaxMind ships a small daemon
([`geoipupdate`](https://github.com/maxmind/geoipupdate)) that
re-downloads the database on a schedule. Wanderer reads whatever
file is at the configured path, so a periodic `geoipupdate` run
followed by a `wanderer serve` SIGHUP (or process restart) is the
operational shape we recommend.

Minimal `/etc/GeoIP.conf` (the daemon's config, not Wanderer's):

```conf
AccountID YOUR_ACCOUNT_ID
LicenseKey YOUR_LICENSE_KEY
EditionIDs GeoLite2-ASN GeoLite2-Country
```

Systemd timer (preferred over crontab on systemd hosts):

```ini
# /etc/systemd/system/geoipupdate.service
[Unit]
Description=Update MaxMind GeoIP databases
[Service]
Type=oneshot
ExecStart=/usr/bin/geoipupdate
```

```ini
# /etc/systemd/system/geoipupdate.timer
[Unit]
Description=Daily geoipupdate
[Timer]
OnCalendar=daily
Persistent=true
[Install]
WantedBy=timers.target
```

```sh
sudo systemctl daemon-reload
sudo systemctl enable --now geoipupdate.timer
```

Crontab alternative (for non-systemd hosts):

```cron
# 03:00 every day, append output to a log
0 3 * * * /usr/bin/geoipupdate >> /var/log/geoipupdate.log 2>&1
```

### Test stub for offline runs

`scripts/geoip-stub.sh /tmp/stub.mmdb` produces a deterministic
empty-but-valid GeoLite2-shaped mmdb so the test suite (and an
operator running smoke tests) can exercise the populated-but-empty
branch of the IP probe without a real MaxMind license. The stub
opens cleanly via the same reader Wanderer uses; every IP lookup
returns "not found" rather than erroring out.

## Run a single scan

```sh
wanderer scan example.nl \
  --geoip /var/lib/wanderer/GeoLite2-ASN.mmdb \
  --db ./wanderer.db
```

Exit codes:

- `0` — scan completed or completed partially (at least one probe
  produced findings)
- `1` — scan failed (domain did not resolve, every probe failed, or
  the store was unreachable)
- `2` — invalid arguments

Output is grouped by probe. Example:

```
Scan s_1729...  status=complete

== dns ==
  [info] dns.a  subject=example.nl
      address: 93.184.216.34
  [observation] dns.mx  subject=example.nl  dim=data_ai
      host: mail.example.nl.
      preference: 10
...
```

Persisted findings live in the SQLite database (`./wanderer.db` by
default). You can inspect them with any SQLite client:

```sh
sqlite3 wanderer.db "SELECT probe_id, severity, subject FROM findings ORDER BY created_at DESC LIMIT 20;"
```

### Pair with Amass for richer subdomain coverage

Wanderer's built-in subdomain discovery is intentionally light: SAN
mining from the apex certificate plus a fixed prefix sweep. For a
broader picture without bolting an active enumerator into the scanner
itself, run [`amass`](https://github.com/owasp-amass/amass) once and
feed the result in:

```sh
amass enum -passive -d example.nl -json amass.json
wanderer scan example.nl \
  --geoip /var/lib/wanderer/GeoLite2-ASN.mmdb \
  --amass amass.json \
  --db ./wanderer.db
```

`internal/scanner/amass.go` parses the JSONL produced by `amass enum
-json` and merges the FQDNs into `target.Related` so the IP probe
resolves them in pass 2 and the assessor sees them as third parties.
The same mechanism is available over the API: `POST /scans` accepts an
`amass_json` field carrying a server-local file path (the serve
endpoint refuses inline file bodies — keep the file on the box).

CLI failures during Amass parsing are fatal at startup, not silent.
A malformed JSON file aborts before the scan begins; a missing path
errors immediately. This is deliberate — silent fallthrough on a
flag that names a file is the kind of thing that makes a scan look
fine while being effectively unenriched.

## Interpret the output

Every finding carries:

- `ProbeID` — stable identifier, e.g. `tls.issuer`, `http.third_party`.
- `Severity` — `info`, `observation`, `concern`, `finding`. This is
  deliberately coarse; precise scoring is the assessor's job.
- `DimensionHint` — which sovereignty dimension the finding informs, if any.
- `Subject` — the thing being described (a domain, a host, an IP).
- `Attributes` — probe-specific structured data.
- `Evidence` — raw source material (certificate PEM, verbatim DNS
  record) retained so a reviewer can audit without re-scanning.

Interesting starting points when looking at a fresh scan:

| Question                                     | Probe/Finding to check        |
| -------------------------------------------- | ----------------------------- |
| Where does mail land?                        | `dns.mx` hosts + `ip.asn`     |
| Who issued the cert?                         | `tls.issuer` (dim: juridisch) |
| Which non-EU providers serve resources?      | `http.third_party` + `ip.asn` |
| Who controls DNS continuity?                 | `dns.ns`                      |
| Is HTTPS configured correctly?               | `tls.validity`, `tls.verify`  |

## Serve the HTTP API

```sh
wanderer serve \
  --addr :8080 \
  --db /var/lib/wanderer/wanderer.db \
  --geoip /var/lib/wanderer/GeoLite2-ASN.mmdb
```

Endpoints:

- `POST /scans` — body `{"domain":"example.nl","related":["..."]}`.
  Returns the finished Scan record (synchronous in the MVP).
- `GET /scans/{id}` — returns a stored Scan with all findings.
- `GET /healthz` — liveness probe.
- `GET /metrics` — Prometheus counters (see `docs/observability.md`).

All errors use a structured shape:

```json
{"error":{"code":"not_found","message":"scan not found"}}
```

**No authentication**. The MVP is intended for a trusted network (for
example, run inside a Nextcloud stack alongside OpenConnector). Adding
auth is a separate change — do not expose this port to the internet
without a reverse proxy enforcing access control.

## Configuration summary

| Flag                     | Env var                     | Default               |
| ------------------------ | --------------------------- | --------------------- |
| `--addr`                 | `WANDERER_LISTEN`           | `:8080`               |
| `--db`                   | `WANDERER_DB`               | `./wanderer.db`       |
| `--geoip`                | `WANDERER_GEOIP_ASN`        | (empty)               |
| `--geoip-country`        | `WANDERER_GEOIP_COUNTRY`    | (fall back to ASN DB) |
| `--per-probe-timeout`    | —                           | `30s`                 |
| `--budget`               | —                           | `2m`                  |
| `--user-agent`           | —                           | `Wanderer/0.x`        |

## Package vendor jurisdiction

Both package inspectors now emit a vendor / maintainer
attribute:

- `inventory.packages.rpm` includes the `vendor` attribute,
  populated from rpm's `%{VENDOR}` tag (e.g.
  `"Fedora Project"`, `"Red Hat, Inc."`,
  `"Microsoft Corporation"`)
- `inventory.packages.dpkg` includes the `maintainer`
  attribute, populated from dpkg's `${Maintainer}` field (e.g.
  `"Bash Maintainers <bash@packages.debian.org>"`)

One rule classifies the host's package origin:

- `wand.host.eu_package_origin` — reads both probe families,
  classifies each finding's vendor / maintainer against
  `internal/assessor/package_vendors.yaml`. Scores:
  - `afhankelijk` on any US-tied vendor (Red Hat / Fedora,
    Microsoft, Oracle, Canonical-UK, Datadog, etc.)
  - `soeverein` when every classified package resolves to an
    EU-tied vendor (SUSE / openSUSE, …)
  - `voldoende` when no US hits AND not every classified
    package is EU-tied (mixed or unclassified — no red flag,
    no positive call)
  - `onbekend` without inventory.packages.* findings

Bare maintainer values without a parseable email fall through
as unknown jurisdiction. Locally-built RPMs with
`Vendor: (none)` are skipped at the agent.

## Container image sovereignty

When the Docker inspector is enabled the agent emits two finding
families:

- `inventory.docker.image` — every image present on disk, with
  `repo_tags` carrying the human-readable refs
- `inventory.docker.container` — every running container, with
  the `image` attribute naming the ref it was started from

Three rules score this surface against
`internal/assessor/container_registries.yaml`:

- `wand.docker.images_us_registry` — afhankelijk on any image
  whose registry resolves to a US-headquartered registry
  (`docker.io`, `gcr.io`, `ghcr.io`, `mcr.microsoft.com`,
  `public.ecr.aws`, `quay.io`, `registry.access.redhat.com`,
  `registry.suse.com`)
- `wand.docker.containers_us_registry` — same shape, scoped to
  actually-running containers (a stronger live signal)
- `eucsf.sov6.container_supply_chain` — SEAL roll-up combining
  both shapes

A bare image name (`nginx:1.27`) or the library shorthand
(`library/nginx:1.27`) is the Docker Engine's default for
`docker.io/library/...` — the rule classifies these as a
docker.io hit and flags the implicit resolution in the verdict.
EU self-hosted registries (`harbor.example.de/...`) are
**sovereign by default**: they do not appear on the YAML list,
so they do not fire the rule.

## Nextcloud inspector

`wanderer agent` ships a Nextcloud inspector that scores the
sovereignty posture *of* the Nextcloud instance running on the
agent host. Enable it in `wanderer-agent.yaml`:

```yaml
inspectors:
  nextcloud:
    enabled: true
    occ_path: /var/www/nextcloud/occ
    run_as: www-data
```

The inspector shells out to `occ` four times per tick:

| `occ` command                               | ProbeID family                                | What it reads                                  |
| ------------------------------------------- | --------------------------------------------- | ---------------------------------------------- |
| `app:list --output=json`                    | `inventory.nextcloud.app`                     | every enabled / disabled app + version          |
| `status --output=json`                      | `inventory.nextcloud.version`                 | Nextcloud version + `supported` flag           |
| `config:list system --output=json`          | `inventory.nextcloud.trusted_domain` + `.objectstore` | trusted domains, S3 backends             |
| `user_oidc:provider --output=json`          | `inventory.nextcloud.oidc_provider`           | OIDC IdP list, with `issuer_host` + geoip      |

If the `user_oidc` app is absent, the inspector falls back to
detecting alternatives (`oidc_login`, `social_login`,
`user_saml`) and emits one
`inventory.nextcloud.oidc.unavailable` Finding naming the
alternative — so an operator sees "we can't probe because
you're on social_login" rather than "Wanderer doesn't work".

### Sovereignty rules

Three rules score the Nextcloud surface:

- `wand.nextcloud.objectstore_eu` — afhankelijk when any S3
  backend resolves to a non-EEA country (the inspector
  enriches with geoip when configured)
- `wand.nextcloud.oidc_provider_eu` — afhankelijk when any
  IdP's issuer URL resolves to a non-EEA jurisdiction
- `eucsf.sov6.nextcloud_supply_chain` — SEAL analogue rolling
  both signals into one observation

Supported Nextcloud majors: 28, 29, 30. Older versions parse
but emit `supported: false` in the version Finding so an
operator sees the contract mismatch.

## Playwright UI smoke layer

The Playwright suite at `tests/playwright/` covers the read-only
UI. It runs against three hermetic SQLite fixtures (one per
scenario) seeded by `internal/fixtures`, not against an
operator's hand-rolled `/tmp/wanderer-demo.db`. A fresh checkout
runs end-to-end with:

```
make playwright-install     # once: installs node deps + chromium
make playwright             # builds binary, seeds fixtures, runs specs
```

### Scenarios

| Scenario     | DB                                        | Used by                                       |
| ------------ | ----------------------------------------- | --------------------------------------------- |
| `baseline`   | `tests/playwright/fixtures/baseline.db`   | DAR layering + reporting catalogue            |
| `agent-host` | `tests/playwright/fixtures/agent-host.db` | host-rule + nextcloud-as-target deep dives    |
| `empty-org`  | `tests/playwright/fixtures/empty-org.db`  | empty-state copy on every UI surface          |

Each scenario boots its own `wanderer serve` instance on a
different port (8281/8282/8283). The Playwright config pins each
spec file to one scenario via `testMatch`.

### Adding a scenario

1. Add `internal/fixtures/<name>.go` with an exported
   `Build<Name>(ctx, *store.Store) error` function.
2. Register it in `internal/fixtures/seed.go`'s `Scenarios` map.
3. Add a Makefile line (under `playwright-fixture`) that emits
   the DB.
4. Add a project block + a `webServer` entry in
   `tests/playwright/playwright.config.ts`.

The seeder uses the public store API + `wand.DefaultRules()` /
`eucsf.DefaultRules()`, so a schema change or a rule rename that
breaks a fixture surfaces at `make playwright-fixture` build
time — before Playwright ever opens a browser.

### Separate from `tests/playwright/fixtures/agent-host.yaml`

The YAML config file at the same path is **not** part of the
fixture seeder. It is a stop-gap for running
`./bin/wanderer agent --once --config tests/playwright/fixtures/agent-host.yaml`
against your real laptop's `/tmp/wanderer-demo.db` if you want
to smoke test the host rule against actual `occ` /
`/proc` / `/etc/systemd/` state. The hermetic
`agent-host` scenario above is what Playwright reads against.

## Troubleshooting

- **`ip: asn DB: no such file or directory`** — the GeoLite2 path is
  wrong. Fail-fast is intentional; the probe will not start without a
  readable database file.
- **`tls.ct` shows `unavailable`** — crt.sh rate-limited you or was
  unreachable. The rest of the TLS probe is unaffected.
- **`http.robots_blocked`** — the target's `robots.txt` disallows `/`
  for your User-Agent. Wanderer honours it; use a different User-Agent
  if you have authorisation.
