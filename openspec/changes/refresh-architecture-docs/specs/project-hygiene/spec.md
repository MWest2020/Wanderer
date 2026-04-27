# Delta for project-hygiene

## ADDED Requirements

### Requirement: Architecture page covers every shipped modus

The top-level `docs/architecture.md` SHALL describe every
operating modus the binary supports today (perimeter, inventory,
egress) and SHALL link the per-capability documentation page for
each, so a new contributor reading the document gets an accurate
mental model of where each ProbeID prefix originates.

#### Scenario: Modus coverage

- **Given** the current main branch with all capabilities landed
- **When** a contributor reads `docs/architecture.md`
- **Then** the document references the perimeter modus
  (`wanderer scan` / `serve`), the inventory modus
  (`wanderer agent` inspectors), and the egress modus
  (`wanderer agent` egress probe)
- **And** every per-capability doc page (`assessor.md`,
  `agent.md`, `egress.md`, `mcp.md`, `scheduling.md`,
  `drift.md`, `exporters.md`) is linked at least once

#### Scenario: How-to-add-a-probe stays current

- **Given** the architecture page's "How to add a perimeter
  probe" section
- **When** a contributor follows the steps against the current
  `internal/probe` layout
- **Then** every file path the section names exists
- **And** every interface name the section names matches the
  symbol in the codebase
