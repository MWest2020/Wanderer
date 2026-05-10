# Delta for project-hygiene

> Held active until Mark signs off; the impl PR carries the
> requirements when it opens.

## ADDED Requirements

### Requirement: Every UI-touching ADR has a Playwright spec

The project SHALL maintain one Playwright spec file per ADR
that describes UI behaviour. A doc-lint check SHALL run as part
of `go test ./...` to grep `docs/decisions/` for `## UI
surface` sections and confirm a matching spec exists at
`tests/playwright/specs/<adr-slug>.spec.ts`. ADRs without UI
claims need no spec.

#### Scenario: ADR without spec fails the lint

- **Given** an ADR `docs/decisions/0042-some-ui-change.md`
  contains a `## UI surface` section
- **And** no file exists at
  `tests/playwright/specs/0042-some-ui-change.spec.ts`
- **When** the doc-lint test runs
- **Then** the test fails and names the missing spec

#### Scenario: ADR with spec passes the lint

- **Given** the ADR's spec file exists
- **When** the doc-lint test runs
- **Then** the test passes
