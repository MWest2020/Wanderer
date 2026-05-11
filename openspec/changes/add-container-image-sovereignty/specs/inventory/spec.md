# Delta for inventory

## ADDED Requirements

### Requirement: Container image sovereignty rules score the Docker inventory

The assessor SHALL register three rules covering the Docker
inventory surface: `wand.docker.images_us_registry`,
`wand.docker.containers_us_registry`, and
`eucsf.sov6.container_supply_chain`. Each rule reads
`inventory.docker.image` and/or `inventory.docker.container`
findings, classifies their image references against an
embedded list of US-headquartered registries, and follows the
soeverein / afhankelijk / onbekend shape established by the
host telemetry rules.

#### Scenario: gcr.io image scores afhankelijk

- **GIVEN** one `inventory.docker.image` Finding whose
  `repo_tags` includes `gcr.io/foo/bar:v1`
- **WHEN** the assessor runs
  `wand.docker.images_us_registry`
- **THEN** the resulting Rationale has Score `afhankelijk`,
  Verdict naming the offending image + Google as the vendor
  of record, and Evidence citing the Finding's ID

#### Scenario: Bare nginx is treated as docker.io

- **GIVEN** one `inventory.docker.image` Finding whose
  `repo_tags` is `["nginx:1.27"]` (no registry prefix)
- **WHEN** the assessor runs
  `wand.docker.images_us_registry`
- **THEN** the Verdict text names docker.io as the implicit
  registry and the Score is `afhankelijk`

#### Scenario: Self-hosted EU registry scores soeverein

- **GIVEN** three `inventory.docker.image` Findings whose
  `repo_tags` all start with `harbor.example.de/`
- **WHEN** the assessor runs
  `wand.docker.images_us_registry`
- **THEN** the Score is `soeverein`, the Verdict text
  includes `"inspected 3 images"`, and Evidence cites at
  least one Finding ID
