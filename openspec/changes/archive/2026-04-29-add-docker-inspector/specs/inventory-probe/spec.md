# Delta for inventory-probe

## ADDED Requirements

### Requirement: Docker inspector reports containers and images

When the Docker socket is reachable, the inventory agent SHALL emit
one `inventory.docker.container` Finding per container and one
`inventory.docker.image` Finding per image, populating the
attributes documented in `docs/findings.md`, and SHALL continue to
honour the read-only contract by issuing only GET calls against
the Docker socket.

#### Scenario: Containers listed

- **Given** a host with the Docker daemon running and one
  container present
- **When** the agent's Docker inspector runs
- **Then** exactly one `inventory.docker.container` Finding is
  produced
- **And** its Subject is the container name
- **And** Attributes contain `image`, `image_digest`, `state`

#### Scenario: Read-only contract holds

- **Given** the inspector is running
- **When** it inspects the daemon
- **Then** every HTTP request issued against the socket has method
  GET
- **And** no request URL contains `/exec`, `/wait`, `/start`,
  `/stop`, `/kill`, `/pause`, or `/unpause`

#### Scenario: API error surfaces gracefully

- **Given** the daemon returns an HTTP 500 to `/containers/json`
- **When** the inspector runs
- **Then** an `inventory.docker.error` Finding is produced with a
  `status_code` attribute
- **And** the agent process exits 0 (other inspectors continue)
