# Delta for assessor

> Held active until Mark signs off; the impl PR carries the
> requirements when it opens.

## ADDED Requirements

### Requirement: Host-side findings produce a non-onbekend verdict

The assessor SHALL score agent-host scans (Targets with
`Kind=host`) on at least one rule per registered rule pack,
so a host scan produces a non-`onbekend` Assessment whenever
the agent's inspectors land their canonical Findings
(`inventory.packages.*`, `inventory.systemd.service`,
`egress.*` from the static scanner). Rules that target
perimeter ProbeIDs MUST continue to return `onbekend` on host
scans — they describe perimeter behaviour, not host
behaviour — but at least one host-shaped rule per pack must
fire on the agent's canonical findings.

#### Scenario: Agent scan produces a host-side verdict

- **Given** an agent scan with `inventory.packages.rpm` and
  `inventory.systemd.service` findings
- **When** the operator runs `wanderer assess <scan-id>
  --framework both`
- **Then** the resulting Assessment has at least one
  dimension with a `soeverein`, `voldoende`, or `afhankelijk`
  score (not all `onbekend`)
- **And** the host scan's verdict pill on `/ui/orgs/{slug}`
  renders that worst score
