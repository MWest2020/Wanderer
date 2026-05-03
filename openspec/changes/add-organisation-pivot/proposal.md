# Proposal: Organisation as a first-class concept

> **Status:** Implementation. The design pass landed
> 2026-05-01; Mark approved every recommendation in the
> open-questions table on 2026-05-03 and this change ships the
> code.

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

## Resolved decisions

Mark approved every recommendation on 2026-05-03. The decisions
are summarised here; the full options + pros/cons walk-through
that produced them is preserved verbatim below this section as
the historical design record.

| # | Decision                       | Choice | Reason |
|---|--------------------------------|--------|--------|
| 1 | Default-org migration          | `default` slug + first-run nudge | Smooth upgrade with explicit hint to rename |
| 2 | Agent-host modelling           | Target with Kind=host | Discriminator does the work; promote later if needed |
| 3 | Scan organisation selection    | flag > env > yaml > default | Same precedence as serve-config |
| 4 | Schedule organisation handling | per-schedule with fallback | Same precedence as Q3 |
| 5 | Asset-discovery seed           | generic YAML only | Don't pin third-party output formats |
| 6 | MCP exposure                   | same PR as CLI/UI | Schema-as-documentation for AI agents |

Concrete consequences for the implementation:

- `org rename` ships in v1 (Q1 escape hatch)
- `scan.organisation` is added to `serve.yaml` (Q3 fallback)
- Each schedule entry takes an optional `organisation:` field;
  unset uses `scan.organisation`; both unset → startup error
  naming the schedule (Q4)
- `wanderer org add --from-yaml` ships in v1 (Q5)
- MCP gains `org.list`, `org.show`, `org.targets` (Q6)

---

## Open questions (resolved — preserved as design record)

These are the points where Mark's input shaped the design.
Each question is laid out as: the decision being made, the
realistic options, pros and cons of each, and the chosen path
with reasoning. They are kept here so future readers can see
the trade-offs we walked through.

### Q1. How does the migration treat existing Targets?

**Decision.** Adding an `organisation_id` column means every
existing Target needs to belong somewhere. What's "somewhere"?

**Option A — magic `default` slug.** Migration 004 creates an
organisation with slug `default`, name `"Default organisation"`,
and attaches every pre-existing Target to it. Operator can
later use a future rename command (not in v1) or a manual SQL
update.

- Pros: Migration is fully automatic. Single-customer operators
  don't notice. Matches the boring/auditable default — every
  upgrade path produces a predictable end state.
- Cons: An operator who has been scanning multiple customers on
  one instance gets all their data lumped into one bucket they
  then have to disentangle. The slug `default` is also a
  semantic lie for them — it implies a placeholder, not their
  real organisation.

**Option B — refuse to start until the operator names it.**
Migration 004 lands but the binary refuses to start at v2.x
until the operator runs `wanderer org rename default --slug
<real-slug> --name <real-name>` or sets a `WANDERER_DEFAULT_ORG`
env var.

- Pros: Forces every operator to think about the organisation
  shape before any new data lands. No semantic-lie slug.
- Cons: Breaks the expand-then-contract rollout — the deploy
  pipeline can't apply the migration and bring the binary up in
  one step. Adds a v1 rename command we don't otherwise need.
  The upgrade is a customer-visible event for operators who
  legitimately have only one organisation.

**Option C — `default` slug + first-run nudge.** Same as A,
but on first start of the v2.x binary, emit a one-time stderr
line: *"Existing Targets attached to organisation 'default'.
Set `WANDERER_DEFAULT_ORG` or run `wanderer org rename` to use
your real organisation handle."* Add the rename command to v1.

- Pros: Migration stays automatic. The operator gets a clear
  pointer at how to fix the slug if `default` doesn't fit. No
  blocked upgrades.
- Cons: Adds the `org rename` command to v1 scope (modest
  cost). The nudge can be ignored — operators who want to
  pretend it's not there will keep the slug `default`.

**Recommendation: C.** It's A's automatic migration with an
explicit hint at how to make the data right. Adding `org rename`
to v1 is the cheapest way to remove A's "you're stuck with
`default` forever" problem. B is the most correct but breaks
the deploy story for operators who legitimately only have one
organisation.

---

### Q2. How are agent-hosts modelled?

**Decision.** Today's data model says "agent-hosts are Targets
with `Kind=host`". The dashboard already splits posture by Kind.
But organisations conceptually own *both* domains and hosts —
should they be one entity (Target) with a discriminator, or two
sibling entities (Target and Host) under Organisation?

**Option A — keep Target with `Kind=host`.** Status quo. The
discriminator continues to do the work.

- Pros: Zero schema change beyond the `organisation_id` column.
  The dashboard's external/internal split already works.
  Existing rule packs that branch on `Kind` keep working
  unchanged. The MCP and export surfaces don't grow.
- Cons: A few rule packs hard-code "Target = public domain"
  assumptions in their meta-finding logic. Those need a code
  audit and possibly small fixes. The semantic muddle ("a
  Target is sometimes a domain, sometimes a host") doesn't
  fully go away.

**Option B — split into `Target` + `Host` siblings.**
`Organisation` has `Targets []Target` and `Hosts []Host`.
Different tables, different IDs, different validation rules
(`Target.Domain` requires a TLD; `Host.Hostname` allows bare
names).

- Pros: Cleaner conceptual model. Each rule pack only
  encounters the entity shape it cares about. New asset types
  later (e.g. cloud accounts) plug in as siblings.
- Cons: Wider migration — `findings.target_id` and
  `assessments.scan_id`'s upstream chain all need to know
  whether they belong to a Target or Host. Either two columns
  (one always NULL) or a polymorphic foreign key (uglier in
  SQLite). The dashboard split has to be rewritten. Doubles
  the store surface.

**Option C — `Asset` parent + `Domain`/`Host` child tables.**
Polymorphic 3rd-normal-form: `Asset` carries the FK to
Organisation; `Domain` and `Host` join on `asset_id`.

- Pros: Theoretically the cleanest data model. Easy to add new
  asset types.
- Cons: Every query joins through `Asset`. Every test needs to
  set up the join chain. Over-engineered for two
  asset-types-and-counting. Strictly worse ergonomics than B
  for marginal modelling gain.

**Recommendation: A.** The Kind discriminator is doing real
work and the dashboard already proves the split is meaningful.
Promote to B if and when a third asset kind shows up. C is
out — it's normalisation theatre. The "rule packs assume
Target=domain" risk is bounded: the rules are in
`internal/assessor/wand/` and `internal/assessor/eucsf/`, both
small, both unit-tested, and a one-hour grep + fix is on the
critical path of the impl PR anyway.

---

### Q3. How does `wanderer scan` know which organisation?

**Decision.** When an operator runs `wanderer scan example.nl`,
how do we pick the organisation?

**Option A — explicit `--organisation <slug>` only.** Required
flag (or environment variable). No flag → command refuses to
start.

- Pros: Always unambiguous. The audit trail (logs, persisted
  Scan rows) carries an explicit organisation, not an inferred
  one. Cross-customer mistakes are impossible.
- Cons: One more thing to type for every invocation. Operators
  with one organisation feel the friction.

**Option B — explicit flag, defaulting to a configured value.**
Required as a logical concept, but with two precedence layers:
the `--organisation` flag overrides; otherwise the
`WANDERER_ORGANISATION` env var; otherwise the
`scan.organisation` field in `serve.yaml` (which we just
shipped); otherwise hard-fail.

- Pros: Operators with one org set it once in YAML and never
  type it again. Operators with many orgs use the flag per
  invocation. Same precedence story as the serve-config we
  just landed.
- Cons: A misconfigured default silently attaches scans to the
  wrong org. The audit trail still records the right
  organisation, but the operator has to read the log to
  realise their default was wrong.

**Option C — WHOIS inference with confirmation.** The scanner
inspects the domain's WHOIS / SOA record, extracts the
registrant org name, fuzzy-matches it against registered
organisation Names, and uses the match if confidence is high.
Otherwise fails.

- Pros: Zero typing. Looks magical when it works.
- Cons: WHOIS is flaky (rate limits, redacted records, GDPR
  blanking). Org names rarely match slugs cleanly. The
  "confidence" threshold is arbitrary. The audit trail records
  an *inferred* attachment, which weakens the
  evidence-of-decision.

**Option D — implicit `default` with stderr warning.** No flag
required. Scans without a flag attach to `default` and Wanderer
emits one stderr line per invocation pointing at the
`--organisation` flag.

- Pros: Zero-friction defaults. Discovery via the warning.
- Cons: Quiet drift — an operator scanning for ACME but never
  reading stderr ends up with ACME's data in `default` until
  they notice. Hard to undo.

**Recommendation: B.** The serve-config-file change we just
shipped already has the `flag > env > yaml > default`
precedence. Reusing that pattern for organisation selection
keeps the operator's mental model consistent. A is purer but
costs friction every invocation; D is friendly but creates
recovery work later; C is too clever.

---

### Q4. What happens to existing scheduled scans?

**Decision.** `WANDERER_SCHEDULES` (the YAML file the scheduler
reads) lists recurring scans by domain. Every scheduled entry
needs to attach to an organisation. Three migration shapes:

**Option A — every existing entry attaches to `default`.**
Migration is silent. The operator updates the YAML when it's
convenient.

- Pros: Smooth migration. No upgrade-time toil.
- Cons: A multi-customer instance gets every scheduled scan
  shoved into `default` until the operator notices and edits
  the YAML.

**Option B — refuse to load a schedule without an
organisation.** The binary fails to start until every entry in
the YAML has `organisation: <slug>`.

- Pros: Forces correctness at upgrade time. No drift.
- Cons: The upgrade requires a YAML edit before the binary
  comes back up. Coordinating that with a `default`-org-only
  operator's deploy pipeline is friction for no value.

**Option C — per-schedule default with optional override.**
The serve config (or `WANDERER_DEFAULT_ORG` env var) sets the
fallback slug. Schedules without an explicit `organisation:`
use the fallback. Schedules with an explicit one override.

```yaml
# /etc/wanderer/serve.yaml
scan:
  organisation: acme   # the fallback for any schedule that doesn't override

# /etc/wanderer/schedules.yaml
schedules:
  - cron: "0 3 * * *"
    domain: customer1.example
    organisation: customer1   # explicit override
  - cron: "0 4 * * *"
    domain: example.acme.tld   # uses scan.organisation = acme
```

- Pros: Same precedence model as Q3 — one mental model
  everywhere. One-customer operators set it once. Multi-
  customer operators add the per-schedule key only where it
  matters.
- Cons: Two layers to remember. Slight ambiguity if both are
  unset (we'd error at startup, naming the schedule).

**Recommendation: C.** It's the same model as Q3. An operator
who picks B in Q3 also picks B here; an operator who picks A in
Q3 also picks A here. Make the precedence rule the same
everywhere and the operator's mental load drops.

---

### Q5. Should asset discovery integrate?

**Decision.** Operators identify all the assets of an
organisation via tools like Amass (subdomain enumeration),
SecurityTrails, or hand-curated lists. The
`--amass-enum-json` flag on `wanderer scan` already accepts an
Amass output file. Should `wanderer org seed <slug> --from-amass
<file>` be a thing? It would generate per-domain Targets for an
organisation in one go.

**Option A — ship `--from-amass` (and possibly other formats)
in v1.** First-class Amass integration: read the file, dedup
FQDNs against existing Targets in the org, create new ones,
report what it did.

- Pros: Day-1 polished experience. Amass is the de facto
  open-source tool in this space; supporting it directly
  signals "we expect operators to do asset discovery first".
- Cons: We pin the Amass output format. Updates to Amass risk
  breaking us. We grow a maintenance surface for what is
  essentially a boilerplate transformation. Adds parser code.

**Option B — accept generic YAML / JSON only; operator scripts
the conversion.** `wanderer org seed --from-yaml <file>` reads
a generic shape (just a list of domains under an org). The
operator pipes Amass through `jq`/`yq` to produce the YAML.

- Pros: Wanderer doesn't pin any third-party tool's format.
  The conversion is a one-liner shell script. Operators who
  use SecurityTrails / Shodan / etc. follow the same pattern.
- Cons: Slightly more setup work for operators using Amass
  specifically. They have to learn the conversion line.

**Option C — out of scope entirely.** Operators add domains
one at a time via `wanderer org add` plus per-domain `wanderer
scan`. No bulk seed at all.

- Pros: Smallest surface. Asset discovery stays operationally
  separate from sovereignty observation, which is what
  Wanderer actually does.
- Cons: 50-domain orgs require 50 invocations. Painful.

**Recommendation: B.** Generic YAML seed (already in the
design) handles every asset-discovery workflow without locking
us to one tool's output format. The operator's conversion
script is one line of `jq`. Skip the `--from-amass` parser; it
adds a dep we don't need.

---

### Q6. Do organisations show up over MCP?

**Decision.** `wanderer mcp` exposes scans, findings,
assessments to AI agents over JSON-RPC stdio. Adding
organisations means three new MCP methods (`org.list`,
`org.show`, `org.targets`). Same PR or follow-up?

**Option A — same PR as the implementation.** Ship the full
surface atomically: CLI, UI, MCP all together.

- Pros: A reviewer sees the whole picture. No drift between
  the CLI/UI and MCP — they go live together. Consistent
  semantics: `wanderer org list` and `org.list` are the same
  query.
- Cons: PR is wider. ~30 extra lines of MCP method wiring +
  tests.

**Option B — follow-up PR.** Ship CLI + UI first; MCP later.

- Pros: Smaller PR, easier review.
- Cons: There's a window where AI agents consuming MCP can't
  see organisations. The follow-up PR can slip; in this
  codebase that's a real risk because nothing forces it to
  ship.

**Option C — skip MCP entirely.** Don't expose organisations
over MCP. AI agents that need org-filtered scans can use
existing methods with an `?organisation=<slug>` query parameter
on the existing `scan.list` / `target.list` methods.

- Pros: No new MCP surface to maintain. The query-param
  approach is technically sufficient.
- Cons: An AI agent inspecting the MCP tool catalogue can't
  *discover* that organisations exist as a concept. They have
  to know the query parameter is there. Documentation
  carries the load instead of the schema.

**Recommendation: A.** The cost is low — three thin methods
that wrap the same store helpers the CLI uses — and the
benefit is structural. MCP is the schema-as-documentation
surface for AI agents; an org-aware Wanderer should advertise
that fact in the tool catalogue, not bury it in a query
parameter.

---

## Summary of recommendations

| # | Question                       | Recommended option | One-line reason |
|---|--------------------------------|--------------------|-----------------|
| 1 | Default-org migration          | C — `default` + nudge | Smooth upgrade, escape hatch via `org rename` |
| 2 | Agent-host modelling           | A — Target with Kind=host | Discriminator does the work; promote later if needed |
| 3 | Scan organisation selection    | B — flag/env/yaml/default | Same precedence as serve-config |
| 4 | Schedule organisation handling | C — per-schedule with fallback | Same precedence as Q3 |
| 5 | Asset-discovery seed           | B — generic YAML | Don't pin third-party output formats |
| 6 | MCP exposure                   | A — same PR | Schema-as-documentation for AI agents |

If Mark agrees with the recommendation column wholesale, the
impl PR can open immediately. If any line gets a "no, do
something else", that single decision propagates through the
design and the impl PR opens after the design rev.

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
