## ADDED Requirements

### Requirement: The server imports netnl findings over HTTP, with a token

`wanderer serve` SHALL offer `POST /imports/internetnl`, taking a
`netnl-findings/v1` document as its body and importing it exactly as
`wanderer import internetnl` does: the same matching, the same idempotency, the
same refusal of an unknown schema version. It SHALL answer with the counts of
domains imported, skipped as unknown target, and skipped as already imported
from that file.

This route exists because the server is the only writer of its SQLite
database; an import from a second process would be a second writer.

The route SHALL require `Authorization: Bearer <token>` matching
`WANDERER_IMPORT_TOKEN`. When that variable is unset, the route SHALL refuse
every request, and the server SHALL say once at startup that the import route
is inactive. The comparison SHALL be constant-time. An import writes verdicts
that appear in reports as evidence; the REST API is reachable on the tailnet
without authentication, so this route may not be.

#### Scenario: A valid file with the token is imported

- **GIVEN** `WANDERER_IMPORT_TOKEN` is set and a target `westerweel.work`
  exists
- **WHEN** a v1 findings file for `westerweel.work` is posted with the token
- **THEN** the response reports one domain imported and the findings are stored
  under an import scan

#### Scenario: No token, no import

- **WHEN** the same file is posted without a token, or with a wrong one
- **THEN** the response is 401 and nothing is stored

#### Scenario: An unconfigured server refuses everything

- **GIVEN** `WANDERER_IMPORT_TOKEN` is unset
- **WHEN** any request reaches the route, with or without a token
- **THEN** the response is 401, and the startup log said the route is inactive

#### Scenario: The same file twice is imported once

- **WHEN** the same file is posted twice with the token
- **THEN** the second response reports it as already imported and no findings
  are stored twice

#### Scenario: A wrong schema version is refused whole

- **WHEN** a file declaring `netnl-findings/v2` is posted with the token
- **THEN** the response is 400, names the supported version, and nothing is
  stored
