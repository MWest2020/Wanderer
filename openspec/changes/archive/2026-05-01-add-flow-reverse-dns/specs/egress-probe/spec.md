# Delta for egress-probe

## ADDED Requirements

### Requirement: Flow probe MAY annotate Findings with reverse DNS

The egress flow probe SHALL support optional reverse DNS (PTR)
annotation of its Findings. When enabled, each unique destination
IP observed in the sampling window is resolved to a hostname via
the host's resolver, and the Finding's `Attributes` gain a
`reverse_dns` key carrying that hostname. Resolution failure
(NXDOMAIN, timeout, refused, transport error) SHALL leave the
Finding unchanged: no `reverse_dns` key, no error Finding. Reverse
DNS is enrichment, not a probe in its own right.

#### Scenario: Successful PTR annotates the Finding

- **Given** the flow probe with reverse DNS enabled, observing
  one connect to `203.0.113.5:443`, and a resolver that returns
  `ec2-203-0-113-5.eu-west-1.compute.amazonaws.com.`
- **When** the window closes
- **Then** the resulting `egress.flow.<category>` Finding's
  Attributes contain
  `reverse_dns: "ec2-203-0-113-5.eu-west-1.compute.amazonaws.com"`
  (trailing dot stripped)

#### Scenario: Failed PTR is silent

- **Given** the flow probe with reverse DNS enabled and a resolver
  that returns NXDOMAIN for every IP in the sampling window
- **When** the window closes
- **Then** every Finding's Attributes omit the `reverse_dns` key
- **And** no `egress.flow.reverse_dns.error` Finding is emitted

#### Scenario: Same IP across ports queries once

- **Given** the flow probe observes connects to `203.0.113.5:443`
  and `203.0.113.5:8443` within one window
- **When** the window closes
- **Then** the reverse DNS resolver is called exactly once for
  `203.0.113.5`
- **And** both Findings carry the same `reverse_dns` annotation

---

### Requirement: Reverse DNS is opt-in

The flow probe's reverse DNS annotation SHALL be disabled by
default and SHALL only run when
`egress.flow.reverse_dns.enabled: true` is set in the agent
config. PTR queries leak the observation back through the host's
DNS path — the resolver and every cache between learn that this
host saw the destination IP — which a sovereignty monitor cannot
default-on without consent.

#### Scenario: Default config makes no PTR queries

- **Given** an agent with the flow probe enabled but no
  `egress.flow.reverse_dns` block in its config
- **When** the agent starts and runs one tick
- **Then** the reverse DNS resolver is never constructed
- **And** no Finding's Attributes contain a `reverse_dns` key

#### Scenario: Per-lookup timeout caps blocking

- **Given** the flow probe with
  `egress.flow.reverse_dns: { enabled: true, timeout: 500ms }` and
  a resolver that hangs past the deadline
- **When** the window closes
- **Then** affected Findings have no `reverse_dns` key
- **And** total resolution time per IP does not exceed the
  configured timeout
