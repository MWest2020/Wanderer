# Design: Organisation pivot

> Design notes. Implementation deferred until Mark signs off on
> shape and migration plan. See proposal.md for context and the
> open questions list.

## Entity shape

```go
// pkg/models/organisation.go
type Organisation struct {
    ID          string    `json:"id"`            // o_<rand>
    Slug        string    `json:"slug"`          // human handle, unique
    Name        string    `json:"name"`          // display name
    Description string    `json:"description,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// Slug rules: lowercase letters, digits, hyphens; 2..40 chars;
// MUST not start/end with hyphen. Validation lives on the model
// so both the CLI seed and the YAML import use the same rules.
```

## Target relationship

`Target` gains one column, `organisation_id`, with a foreign-key
to `organisations(id)`. Migration 004 backfills every existing
Target to a seeded `default` organisation. After backfill the
column is NOT NULL.

```sql
-- migration 004_organisations.sql

CREATE TABLE organisations (
    id          TEXT PRIMARY KEY,
    slug        TEXT NOT NULL UNIQUE,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL
);

INSERT INTO organisations (id, slug, name, created_at)
    VALUES ('o_default', 'default', 'Default organisation', CURRENT_TIMESTAMP);

ALTER TABLE targets ADD COLUMN organisation_id TEXT
    REFERENCES organisations(id);

UPDATE targets SET organisation_id = 'o_default'
    WHERE organisation_id IS NULL;

-- SQLite cannot ALTER COLUMN NOT NULL retroactively, so we
-- enforce the invariant via a CHECK constraint added by
-- recreating the table. Standard SQLite pattern; details in the
-- migration file when implemented.
```

## Store surface

```go
// internal/store/sqlite.go (additions)
type OrgRow struct {
    ID, Slug, Name, Description string
    CreatedAt                   time.Time
}

func (s *Store) UpsertOrganisation(ctx context.Context, o *models.Organisation) error
func (s *Store) GetOrganisationBySlug(ctx context.Context, slug string) (*models.Organisation, error)
func (s *Store) ListOrganisations(ctx context.Context) ([]OrgRow, error)
func (s *Store) ListTargetsByOrganisation(ctx context.Context, orgID string) ([]models.Target, error)
```

`UpsertTarget` gains an `OrganisationID` field on the input. Empty =
seed `default`. Existing call sites compile against the new
field via Go's zero-value default.

## Selectors

`store.Selectors` (the existing scan filter) gains an
`OrganisationID` field. The dashboard's per-org page uses it to
restrict scan listing. Backwards compatible — empty
`OrganisationID` returns scans across every organisation, matching
today's behaviour.

## CLI surface

```
wanderer scan --organisation acme example.nl
wanderer scan example.nl                 # → attaches to "default"

wanderer org list
wanderer org show acme
wanderer org add --slug acme --name "ACME B.V."
wanderer org add --from-yaml orgs.yaml   # bulk seed
```

`org add` is the only mutation path. No remove, no rename. If
that turns out to be a real need, future proposal.

## YAML seed file shape

```yaml
# orgs.yaml — `wanderer org add --from-yaml orgs.yaml`
organisations:
  - slug: acme
    name: ACME B.V.
    description: Customer of Conduction.
  - slug: example-gov
    name: Example Government
    description: Public-sector pilot.
```

Idempotent: running it twice does not duplicate. Slug is the
unique key; name + description are updated on second run.

## Agent config

```yaml
# wanderer-agent.yaml
core:
  organisation: acme   # default: "default"
  ...
```

When the agent registers itself as a Target (Kind=host) with
the API, it includes its organisation. The control-plane validates
the slug exists; unknown slug → 4xx, agent surfaces the error.

## Dashboard pivot

`/ui/orgs/{slug}` renders the same Dashboard template the
instance-wide `/ui/` uses, with the per-target snapshot list
filtered to that organisation's Targets. The headline labels
gain an organisation prefix:

```
Wanderer · ACME B.V. · sovereignty observation
```

The instance-wide `/ui/` keeps working but its headline now reads:

```
Wanderer · all organisations · sovereignty observation
```

…with a list of registered organisations as the first
sub-section, each linking out to its `/ui/orgs/{slug}` page.

The Reporting page gets a `?org=<slug>` query param to filter
the cross-target view to one organisation. No new route — query
param is the boring choice.

## Migration plan

The migration is split into two binaries to keep each step
reversible until the previous one is verified in production:

**Step 1 (binary 1.x):**
- Migration 004 lands.
- New table + column + backfill + constraints in place.
- Old binary keeps reading; the column it doesn't know about
  doesn't break anything (SQLite is permissive).

**Step 2 (binary 2.x):**
- CLI / agent / UI start reading and writing the column.
- `default` organisation is the implicit attach point for any
  call that doesn't specify a slug.

**Step 3 (binary 2.1+):**
- Deprecation warning when an operator uses the implicit
  default. After two releases, require explicit
  `--organisation` (release-time decision; not in this
  proposal's scope).

## Things to inventory before implementing

These are the spots that today assume Targets are
organisation-agnostic. Each needs a code change in the impl PR:

- [ ] `cmd/wanderer/scan.go::buildProbes` — accept org context
- [ ] `cmd/wanderer/serve.go` — schedule-config gains org per
  schedule
- [ ] `cmd/wanderer/agent.go` — read `core.organisation`
- [ ] `internal/scanner/scanner.go::Scan` — propagate org on
  Target lookup
- [ ] `internal/api/...` — REST endpoints that list scans /
  targets accept an `?organisation=<slug>` filter
- [ ] `internal/mcp/...` — three new methods (`org.list`,
  `org.show`, `org.targets`) — pending answer to Open Question 6
- [ ] `internal/export/...` — exports gain an `organisation`
  column / field
- [ ] `internal/ui/...` — new `/ui/orgs/{slug}` route +
  Reporting query param
- [ ] `docs/architecture.md`, `docs/operator.md`,
  `docs/findings.md` — add organisation framing throughout

## Test strategy (for the future implementation)

- Migration test: seed a pre-migration database, run migration
  004, assert every Target has `organisation_id = 'o_default'`,
  the constraint is enforced, the unique slug invariant holds.
- CLI test: `wanderer org add` produces an organisation; `wanderer
  scan --organisation <slug>` attaches to it.
- Filter test: `/ui/orgs/<slug>` returns only that organisation's
  Targets in the headline counts.
- Reporting filter test: `/ui/reporting?org=<slug>` filters
  rule-summary rows to that organisation's targets.

## Estimated implementation size

A focused PR is on the order of:

- ~150 lines of migration + store
- ~80 lines of CLI subcommands
- ~50 lines of agent config wiring
- ~120 lines of UI handlers + templates
- ~200 lines of tests
- Docs

Roughly the size of `add-flow-reverse-dns` plus the dashboard
redesign combined. Not enormous, but wide enough that a careful
review is worth more than implementation speed.
