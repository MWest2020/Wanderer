# Delta for inventory-probe

## ADDED Requirements

### Requirement: Agent registers host-only Targets

The agent SHALL be able to write Findings under a Target whose
`Domain` is a bare hostname (no TLD) by tagging the Target with
`Kind: host`, and the model layer's TLD requirement SHALL apply
only to Targets whose Kind is `domain` (the default).

#### Scenario: Bare hostname accepted

- **Given** a host with `hostname` set to `webapp-01` (no dot)
- **When** `wanderer agent --once` writes its Findings
- **Then** the Target row in the store has `domain = "webapp-01"`
- **And** `kind = "host"`
- **And** no validation error is raised

#### Scenario: Public-domain validation still firm

- **Given** a perimeter scan invoked via `POST /scans` with body
  `{"domain": "no-tld-here"}`
- **When** the scanner validates the Target
- **Then** the request is rejected with the existing TLD error
- **And** no scan row is created
