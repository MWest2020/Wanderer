# Delta for project-hygiene

## ADDED Requirements

### Requirement: Schema changes ship as numbered up-only migrations

The store SHALL track schema state in a `schema_migrations` table
holding `(version, name, applied_at)` rows, apply migrations
strictly in numeric order in a single transaction per migration,
and refuse the string-matched `ALTER TABLE ... DEFAULT ...` /
`duplicate column name` tolerance pattern for new schema changes,
so an auditor can read one table to know which schema version is
in production.

#### Scenario: Fresh database applies every migration

- **Given** an empty SQLite file
- **When** `store.Open` runs
- **Then** every entry in the `migrations` slice runs in order
- **And** `schema_migrations` lists every applied version with its
  `applied_at` timestamp

#### Scenario: Database at version N applies only N+1..M

- **Given** a database whose `schema_migrations` table contains
  versions 1, 2, 3
- **And** the source ships migrations 1..5
- **When** `store.Open` runs
- **Then** only migrations 4 and 5 execute
- **And** migrations 1..3 are not re-run

#### Scenario: Failed migration rolls back

- **Given** migration 7 contains an invalid SQL statement
- **When** `store.Open` runs against a database at version 6
- **Then** the transaction for migration 7 rolls back
- **And** `schema_migrations` continues to show version 6 as the
  highest applied version
- **And** `store.Open` returns the underlying SQL error
