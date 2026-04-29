# Delta for egress-probe

## ADDED Requirements

### Requirement: Flow probe captures runtime egress

The egress flow probe SHALL record outbound `connect()`
destinations during its configured sampling window when enabled
and supported by the kernel, and SHALL emit one
`egress.flow.<category>` Finding per unique
`(destination_ip, destination_port)` pair, reusing the existing
classifier and redactor so the wire format stays consistent with
the static egress probe.

#### Scenario: Unique destination produces a Finding

- **Given** a flow probe with a 30-second window and a process
  that calls `connect()` to `203.0.113.5:443` once
- **When** the window closes
- **Then** one `egress.flow.<category>` Finding is produced
- **And** Attributes contain `destination_ip`, `destination_port`,
  `runtime: true`, and `classifier_rule`

#### Scenario: Privilege missing surfaces gracefully

- **Given** a host without `CAP_BPF` or `CAP_PERFMON`
- **When** the flow probe is enabled and the agent starts
- **Then** an `egress.flow.unavailable` Finding is emitted exactly
  once
- **And** the agent process exits 0 (other inspectors continue)

### Requirement: Flow probe is opt-in

The flow probe SHALL be disabled by default and SHALL only run
when `egress.flow.enabled: true` is set in the agent config, so
operators must consciously accept the kernel-level capability cost
before any kernel attach happens.

#### Scenario: Default config does not load the program

- **Given** an agent with no `egress.flow` block in its config
- **When** the agent starts
- **Then** no eBPF program is loaded
- **And** no `egress.flow.*` Finding (including unavailable) is
  emitted
