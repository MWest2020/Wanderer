---
status: draft
last_reviewed: 2026-07-12
---

# 0015 — Sovereignty overview (the synthesis view)

**Status:** Accepted, 2026-06-15. Implements
`propose-sovereignty-overview` — the Wave-3 synthesis from
`research-high-signal-observability`.

**Context.** Field feedback was that Wanderer felt *weinigzeggend*:
abstract framework scoring, not concrete observed reality. A diligence
pass while building the high-signal leads found that Wanderer *already
collects* the signals that matter — apex hosting jurisdiction
(`apex_ip_eea`), mail routing (`mx_vendor_jurisdiction`), authoritative
DNS (`ns_vendor_jurisdiction`), the transit path (`transit.eu_path`),
US hyperscaler/CDN fronting (`no_us_hyperscaler`), and web third
parties (`third_parties_eea`). The problem was **presentation**: those
signals are scattered across dozens of findings and per-rule verdicts,
never assembled into the "your service lives here, its mail goes there,
its DNS is run by …" picture an operator needs.

**Decision.** Add a **Sovereignty overview** — a synthesis panel that
re-groups the already-scored rule rationales into an ordered set of
*flows* (Hosting, Mail, DNS, Transit path, CDN / hyperscaler, Third
parties), each shown with its observed verdict and score. It is pure
presentation: `internal/ui/SovereigntyFlows` reads the scan's
assessment rationales and maps the six flow rule IDs to labels; **no
EEA logic and no new collection live in the UI** — the overview mirrors
what the rule pack already observed, honouring the project-hygiene
principle "observed signals lead, scores annotate" (the verdict text,
e.g. "mx hosts in US (outside EEA)", is itself the concrete statement).

The first cut is a **textual flow list** on the assessment page. An
interactive node-graph ("spider in the web") is a deliberate follow-up
once the flow model proves out — the list answers the feedback at a
fraction of the cost, and the data is already in the store.

## UI surface

- The assessment page (`/ui/scans/{id}/assessment`) renders a
  "Sovereignty overview" section above the per-framework cards when the
  scan's assessment contains at least one flow rule.
- Each flow row shows the category (Hosting, Mail, DNS, …), a score
  pill, and the rule's observed verdict.
- When no flow rule fired, the section is omitted (no empty panel).

The Playwright spec `tests/playwright/specs/sovereignty-overview.spec.ts`
asserts the overview renders on a fixture scan's assessment page.

**Consequences.**

- The overview adds no probe, rule, or schema — it is a view over
  existing Assessment data, so it cannot drift from the scores.
- It depends on the flow rule IDs; if a rule is renamed, its flow row
  silently disappears until the map in `flows.go` is updated. Acceptable
  (the assessment cards still show every rule); a missing flow is a
  visible gap, not a wrong statement.

**Alternatives considered.**

- *Re-derive flows from raw findings in the UI* — rejected: would
  duplicate the EEA/jurisdiction logic the rule pack already owns and
  risk the overview disagreeing with the scores. Reading the scored
  rationales keeps a single source of truth.
- *A graphical map first* — deferred: higher cost, and the textual list
  already delivers the "what goes where" synthesis.

**See also.**

- `research-high-signal-observability` — the umbrella this completes the
  first synthesis slice of
- `openspec/specs/project-hygiene` — the "observed signals lead" principle
