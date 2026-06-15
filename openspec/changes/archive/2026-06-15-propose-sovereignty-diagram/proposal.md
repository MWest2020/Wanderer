# Proposal: Sovereignty flow diagram (the visual "spider in the web")

> **Status:** Accepted + implemented (2026-06-15). The visual capstone
> of the Sovereignty overview (ADR-0015). No-JS baseline now;
> progressive JS enhancement is a follow-up.

## Why

The textual overview answers "what goes where"; the metaphor Mark keeps
invoking — the org/host as the spider in the web — is inherently
visual. A hub-and-spoke diagram makes the posture graspable at a glance:
the host at the centre, each flow a spoke coloured by its score.

## What Changes

- A server-rendered, **no-JS inline SVG** hub-and-spoke
  (`SovereigntyDiagram`) beside the overview table on the assessment
  page: the target at the centre, each flow laid out evenly around a
  circle, the node coloured by score (reusing the existing score colour
  variables), labels anchored clear of the hub.
- Derived purely from the same Flow model — no new collection.

## Both approaches supported

The SVG is the no-JS baseline (works and is auditable without
JavaScript). A **progressive-enhancement JS layer** can later make it
interactive (hover, drill-in, force layout) without changing the
server-side structure — the agreed "both can be supported" path. The JS
layer + visual polish are a follow-up (they want a human's eye on the
result, which a headless build cannot provide).

## Not in scope

- The interactive JS layer itself (follow-up).
- Org-level diagram (the dashboard roll-up is tabular for now).

## Parallel-safe

`internal/ui/diagram.go` (pure layout) + the assessment template + a
small CSS block reusing the score colour vars + a Playwright assertion.
No schema, probe, or rule change.
