# Delta for project-hygiene

> The one durable requirement from the high-signal research: the design
> principle that keeps Wanderer's output concrete.

## ADDED Requirements

### Requirement: Observed signals lead; scores annotate

Wanderer features SHALL surface the concrete observed fact before any
derived score. Each Finding SHALL be readable as a plain statement of
what was observed (e.g. a host, a route, a destination and its
jurisdiction) independent of any framework verdict; rule-pack scores
annotate that fact rather than replace it. Synthesis surfaces SHALL
present "what goes where" as observed data, not solely as a table of
verdicts.

#### Scenario: A new signal is added concrete-first

- **GIVEN** a new observability feature (e.g. email routing)
- **WHEN** it emits a Finding
- **THEN** the Finding states the observed fact in plain terms ("mail
  is routed to <provider> in <country>")
- **AND** any sovereignty score is layered on top of that fact, not
  presented in its place

#### Scenario: A verdict can always be traced to its observation

- **GIVEN** any sovereignty score Wanderer reports
- **WHEN** an operator inspects it
- **THEN** the underlying observed Finding(s) that produced the score
  are available and human-readable
