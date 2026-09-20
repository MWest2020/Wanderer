## ADDED Requirements

### Requirement: An agent enrols itself once and can be revoked

The core SHALL issue short-lived enrolment tokens and SHALL let an
agent exchange one for its own HMAC secret, recording the agent by
hostname with the moment it enrolled. A token SHALL be usable once and
SHALL expire. An operator SHALL be able to revoke one agent, after
which that agent's findings are refused while every other agent
continues. Secrets SHALL be stored so that the plain secret cannot be
read back from the core.

#### Scenario: First start of a new agent

- **GIVEN** an operator who created an enrolment token
- **WHEN** the agent starts with that token
- **THEN** it receives its own secret, writes it 0600, and the core
  lists the agent as enrolled

#### Scenario: Token used twice

- **GIVEN** a token that an agent already exchanged
- **WHEN** a second agent tries the same token
- **THEN** the exchange is refused and nothing is enrolled

#### Scenario: One agent revoked

- **GIVEN** two enrolled agents
- **WHEN** one is revoked and both send findings
- **THEN** the revoked agent is refused and the other is accepted

---

### Requirement: The served core accepts findings from enrolled agents

`wanderer serve` SHALL build its router with the agent secrets from the
store, so the findings route works for enrolled agents. With no agent
enrolled the route SHALL keep refusing every request, and the startup
log SHALL say that agent ingestion is inactive.

#### Scenario: Enrolled agent delivers

- **GIVEN** an enrolled agent and a running `wanderer serve`
- **WHEN** the agent posts findings signed with its secret
- **THEN** the core accepts them and they appear on the scan

#### Scenario: No agents enrolled

- **GIVEN** a core with no enrolled agents
- **WHEN** anything posts to the findings route
- **THEN** it is refused, and the startup log stated that agent
  ingestion is inactive

---

### Requirement: A replayed batch is stored once

Each batch of findings SHALL carry an identifier. The core SHALL store
a batch once; a repeat SHALL be answered as already received, without
storing the findings again. The agent's outbox SHALL keep the
identifier across a restart, so a batch that was spooled during an
outage is recognised when it is finally delivered.

#### Scenario: Outage and replay

- **GIVEN** an agent that spooled three batches while the core was down
- **WHEN** the core returns and the outbox drains, and the same drain
  runs twice
- **THEN** each batch's findings appear exactly once

#### Scenario: Restart keeps the identifier

- **GIVEN** a spooled batch and an agent that restarts before delivery
- **WHEN** the agent delivers it after the restart
- **THEN** the batch carries the identifier it was spooled with
