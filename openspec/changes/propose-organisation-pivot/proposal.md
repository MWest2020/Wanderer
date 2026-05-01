# Proposal: Organisation as a first-class concept

> **Status:** Design proposal only. No implementation in this
> change. Schema migrations need a human read-through before code.
> See `tasks.md` — every task is a `[ ]` design checkpoint, not a
> code item.

## Intent

Mark named the gap directly: "liefste een organisatie kunnen
aanwijzen, de scans (nu van buiten getest, maar ook van binnen
naar buiten)". Today's data model is **Target-first**: each
domain or host stands alone, and the dashboard rolls up across
*all* targets indiscriminately. There is no concept of "all the
assets that belong to *this* organisation, both external (the
domains they operate) and internal (the hosts running their
agents)".

This proposal designs a first-class `Organisation` entity that
groups Targets — perimeter and agent — into a single posture
view, so an operator can pick one organisation and see *its*
sovereignty story, not the instance-wide soup.

This is a design proposal. It captures the surface, the
migration plan, and the open questions, so a future change can
ship the implementation in a focused PR after Mark signs off on
the shape.

## Why design-only

A schema migration that introduces a new entity and rewires every
Target through it touches:

- `pkg/models/` — new `Organisation` type, modified `Target`
- `internal/store/sqlite.go` — new table, foreign key, migration
  number `004_organisations`
- `cmd/wanderer/scan.go` — `--organisation <slug>` flag, default
  value handling
- `cmd/wanderer/agent.go` — `core.organisation` config field
- `internal/ui/` — new `/ui/orgs/{slug}` route, dashboard pivot
- Every export, every assessor scenario that summarises by Target,
  every drift query
- Every existing scan + assessment in production stores — they
  need a default organisation to belong to

That is a wide change, and irreversible (you can't drop a NOT
NULL foreign key once data depends on it). Boring/auditable says:
write the design, get explicit OK, then ship the code in a
focused PR.

## Scope of the future implementation

**In scope (when implemented):**

- New `Organisation` struct in `pkg/models/`: `ID`, `Slug`,
  `Name`, `Description`, `CreatedAt`. Slug is the user-typed
  handle; Name is the human-friendly label.
- New `organisations` table in SQLite. Migration 004.
- `Target.OrganisationID` column added to `targets`. Existing
  rows backfilled to a single seed organisation (slug
  `default`, name `"Default organisation"`) so no NULL values
  appear; the migration is deterministic.
- `wanderer scan --organisation <slug>` flag. When unset, the
  scan attaches to the `default` organisation. Unknown slug
  fails fast at startup.
- Agent: `core.organisation: <slug>` field in
  `wanderer-agent.yaml`. Same default behaviour.
- New UI route `/ui/orgs/{slug}` rendering the same Dashboard
  shape (headline + external/internal posture + concerns +
  activity) but **scoped to that organisation**.
- The instance-wide `/ui/` keeps working — it is the rolled-up
  view across every organisation, and explicitly says so.
- Org admin endpoints? **No.** Organisation creation happens
  via `wanderer org add <slug> <name>` CLI subcommand or a YAML
  seed file. The UI stays read-only.
- `wanderer org list` / `wanderer org show <slug>` for read-side
  CLI access mirroring the UI.

**Out of scope (even when implemented):**

- Multi-tenant isolation. An operator with htpasswd access to
  `/ui/` can read every organisation's data. If we ever need
  per-org access control that is a separate (large) proposal.
- Organisation hierarchies / parent-child relationships.
- API endpoints for mutating organisations from the wire. CLI
  + YAML seed only.
- Cross-organisation rollups beyond the existing instance-wide
  `/ui/`.
- Renaming or deleting organisations. v1 is add-only; remove or
  rename is a future migration if the need is real.

## Open questions for review

These are the points where Mark's input shapes the design:

1. **Default organisation handling.** Is a magic `default` slug
   the right migration target for existing rows, or should the
   migration prompt the operator to provide a slug at upgrade
   time? The former is boring; the latter is more correct for an
   operator who has been running multiple-customer scans on one
   instance.

2. **Should agent-hosts be Targets, or a separate `Host` entity
   under Organisation?** Today's model is "Target with
   Kind=host"; this proposal continues that. An alternative is a
   separate `Host` entity. The current shape is simpler and the
   dashboard's external/internal split already works — but the
   rule pack assumes Targets are domains in a few places. We
   need to inventory those before settling.

3. **`wanderer scan` ergonomics.** Should the slug be inferred
   from the scanned domain's WHOIS / SOA record (organisation
   name → slug), or always require explicit `--organisation`?
   Inference is convenient but flaky. Explicit is boring.
   Recommendation: explicit, with a startup hint when
   inference would have produced a confident match.

4. **Existing scheduled-scan YAML.** `WANDERER_SCHEDULES` defines
   recurring scans by domain. After this change, every
   scheduled entry needs an organisation. Migration: existing
   schedules attach to `default`; new schedules require the org
   field.

5. **Asset discovery integration.** The `--amass-enum-json` flag
   on `wanderer scan` already lets an operator hand in
   Amass-discovered FQDNs. Should `wanderer org seed
   <slug> --from-amass <file>` be a thing? It generates the
   per-domain Targets for an organisation in one go. This is a
   nice ergonomics gain but pure operator-side scripting can do
   it too. Probably leave for v2.

6. **MCP exposure.** The `wanderer mcp` JSON-RPC surface
   currently exposes scans, findings, assessments. Adding
   organisations means three new MCP methods (`org.list`,
   `org.show`, `org.targets`). Worth doing in the same change?
   Or follow-up?

## Wand / EUCSF dimensions informed

Indirect — the assessor will continue to score Findings the
same way. The organisation pivot only changes how scores are
*aggregated and displayed*, not how they are computed.

## Passive / active boundary

No new outbound calls; no new data ingested. The change is
schema + presentation.

## Parallel-safe (when implemented)

The implementation will need a careful migration order:

1. Add `organisations` table.
2. Insert `default` organisation row.
3. Add `targets.organisation_id` column with backfill to default.
4. Add NOT NULL constraint after backfill.
5. Update CLI / agent / UI to read the column.
6. Update tests.

That sequence keeps the binary compatible at every intermediate
step (each step's binary boots cleanly against either the
pre-step or post-step schema). Concretely: deploy the migration
script, then deploy the new binary, then enable the new
behaviour. Standard expand-then-contract.

## Why now (well, not yet)

The dashboard redesign and the per-check reporting page now
land an operator on a *generic* sovereignty view. The next
logical step is "and now show me only *my* organisation". The
data model is the right point to formalise that. Doing it as a
written-out proposal first means the implementation PR can be
small and reviewable.
