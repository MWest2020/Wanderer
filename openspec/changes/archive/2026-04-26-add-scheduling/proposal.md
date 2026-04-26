# Proposal: Scheduling + Posture Drift

## Intent

One scan is a snapshot. A sovereignty observatory becomes useful when
it runs *continuously* and tells you what *changed*. This change adds
two capabilities: cron-like scheduling of scans inside `wanderer serve`,
and diffing between consecutive scans of the same target to surface
"posture drift" as first-class Findings.

Concrete example the MVP cannot answer today:

> "Our MX was `aspmx.l.google.com` last month and today it is
> `outlook.office365.com` — who decided that, when, and does our
> DICTU Data-&-AI score reflect the change?"

Scheduling + drift answer it.

## Scope

**In scope:**

- A schedules file (`wanderer-schedules.yaml`) listing cron
  expressions and target domains, optionally gated by probe sets.
- An in-process scheduler that runs inside `wanderer serve`; no
  separate cron daemon.
- A diff engine that compares a new scan against the previous scan
  for the same target and emits `drift.*` Findings for changes worth
  surfacing.
- Persistence: drift Findings live in the existing `findings` table,
  one `source_modus: drift`.
- API: `GET /targets/{id}/drift?since=<ts>` returns drift Findings
  across all scans of that target.
- CLI: `wanderer diff <scan-a> <scan-b>` prints a human-readable diff
  in markdown.

**Out of scope:**

- Alerting / webhooks. Surfacing drift as Findings is the boring way;
  downstream tools (exporters, MCP, Grafana) already consume Findings
  and can alert from there. A dedicated webhook layer waits.
- Multi-tenant scheduling. MVP is single-tenant (trusted-network).
  Per-org schedules will come when Wanderer grows a tenancy model.
- Retention / pruning. Old scans accumulate forever in the MVP; a
  follow-up change adds retention policies when the operational pain
  becomes real.
- Heroic statistical analysis of trend lines. We emit drift Findings;
  the assessor reads them. Dashboards do the charting.

## DICTU dimensions informed

Drift reveals change in any dimension. Specific drift categories:

- `drift.tls.issuer_changed` → **Juridisch** — CA changed (often
  signals a vendor or renewal-flow change).
- `drift.dns.mx_changed` → **Data & AI** — mail provider moved.
- `drift.dns.ns_changed` → **Operationeel** — DNS delegation moved.
- `drift.ip.country_changed` → **Juridisch** — hosting migrated
  across borders.
- `drift.http.third_party_added` / `_removed` → **Technologie** —
  homepage dependency changed.
- `drift.inventory.package_removed` / `_added` (once inventory
  lands) → **Operationeel**.
- `drift.egress.endpoint_changed` (once egress lands) → **Juridisch**
  or **Data & AI**.

## Passive/active boundary

The scheduler invokes the existing scanner. No new probe, no new
network pattern. Drift analysis is a pure function of the store.

## Parallel-safe

Touches `cmd/wanderer/serve.go` (hook scheduler into the server
lifecycle), adds `internal/scheduler/` and `internal/drift/`, and
extends the API. Zero conflict with assessor, exporters, MCP.
Inventory/egress land the `source_modus` column; scheduling adds a
new value (`drift`) to the set — no schema migration conflict.
