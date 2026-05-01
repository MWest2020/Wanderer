# Delta for scanner

> **Note:** This delta documents the *target end-state* of the
> Organisation pivot. The implementation PR (`add-organisation-pivot`)
> will land the requirements; archiving this change before that PR
> exists would create a spec the code does not satisfy. Hold this
> change as **active** (not archived) until the implementation
> change opens.

## ADDED Requirements

### Requirement: Organisations group Targets

Wanderer SHALL model an `Organisation` as a first-class entity
that groups one or more Targets (perimeter `Kind=domain`,
agent-host `Kind=host`, or a mix). Every Target MUST belong to
exactly one Organisation. A seeded `default` Organisation
SHALL exist on every freshly-migrated store, and any Target that
predates the migration MUST be attached to `default` by the
backfill step.

#### Scenario: New Target without explicit organisation

- **Given** an instance with the seed `default` Organisation
- **When** an operator runs `wanderer scan example.nl` without
  `--organisation`
- **Then** the resulting Target is attached to `default`

#### Scenario: Existing Target after migration

- **Given** a pre-migration store with N Targets and zero
  Organisations
- **When** migration 004 runs
- **Then** the `default` Organisation is created
- **And** every existing Target's `organisation_id` is set to
  the `default` Organisation's ID
- **And** the column is NOT NULL after the backfill

---

### Requirement: Organisation slugs are unique and validated

Organisation slugs SHALL be 2–40 characters, lowercase letters,
digits, and hyphens only, MUST NOT start or end with a hyphen,
and MUST be unique across the store. The slug is the operator-
facing handle (used in `--organisation <slug>` and the URL
`/ui/orgs/{slug}`); the Name is the display label.

#### Scenario: Invalid slug rejected

- **Given** an operator runs `wanderer org add --slug -bad
  --name "Bad"`
- **When** the command processes the slug
- **Then** the command exits non-zero
- **And** the error names the slug rule that failed
