---
status: draft
last_reviewed: 2026-09-24
---

# 0018 — UI information architecture: vloot / domein / techniek

**Status:** Accepted, 2026-09-22. Implements `2026-09-22-drie-lagen-ciso`.
Supersedes [0017](0017-ui-personas.md).

**Context.** Mark walked the shipped UI with Playwright for the
audience it is actually built for — a CISO or security officer who
wants to know *where does my data live, how is my infrastructure put
together, and what can I do about it* — and found the layer he expects
first does not exist. `/ui/` answered a single domain (an input field
plus "recent beantwoord"); the fleet screen managed domains but never
totalled them; the aggregate fleet numbers a Tourist actually wants
("Sovereignty by flow — Hosting: all 3 in EEA") were buried 6.700
pixels down `/ui/trends`, dressed as a table header between two rule
dumps.

0017 mapped Tourist to `/ui/` (a target list), Explorer to the
per-scan report, and Farmer to Trends (the rule catalogue + matrix,
returned to over time). That mapping assumed `/ui/` was already a
fleet view. It became a single-domain answer page under
`2026-09-20-answer-first-ui`, and no fleet aggregate ever moved back
to replace it — the persona-to-route assignment quietly went stale
without anyone re-deciding it.

**Decision.** Re-derive the three surfaces from what each persona
actually asks for, not from which route happened to hold that
question before:

- **De vloot (Tourist)** — `/ui/`: one score for the whole fleet
  (`x/n`, unanswered apart), the count of domains that are not
  sovereign, the verdict-per-flow breakdown, and the fleet's most
  expensive rules. The scan input field stays, as an action on the
  page, not the page's reason to exist.
- **Het domein (Farmer)** — the answer page
  (`/ui/scans/{id}/answer`): the seven sovereignty questions for one
  domain, each with its `ja`/`nee`/`onbekend` verdict, the `x/n` score
  for that domain, and — for every point that is not sovereign — the
  concrete handeling to fix it. "How am I doing" and "what do I do
  about it" live on the same screen.
- **De techniek (Explorer)** — the onderbouwing
  (`/ui/scans/{id}/assessment`): the same seven questions restated as
  an answer sheet with evidence, and — one click further, not open by
  default — the two raw framework tables (DICTU + EU CSF/SEAL) with
  rule IDs, criteria, and per-rule rationale. The per-rule deep-dive
  (`/ui/reporting/{fw}/{ruleID}`) stays the floor beneath this layer.

The nav keeps its two tabs — **Overzicht** (`/ui/`, `/ui/orgs/{slug}`)
and **Trends** (`/ui/trends`) — but they no longer carry the
Tourist/Farmer labels 0017 gave them: Trends is a fleet-wide rule
catalogue and score matrix an operator returns to, not a persona's
home screen in this mapping. It is not retired — de-scoping Trends is
explicitly out of scope for this change — but it no longer anchors the
IA the way Overview did under 0017.

**Consequences.** No probe/assessor/store change; pure web-ui IA, same
as 0017. `/ui/` gains the fleet aggregate it never had (vlootscore,
distribution, top-3 costly rules). The answer page gains an `x/n`
score and remediation lines it did not carry under 0017. The
onderbouwing page's frameworktabellen move behind a `<details>` so the
seven questions are the page's first impression, not a stepping stone
past two rule dumps. `docs/explanation/adr/0017-ui-personas.md` stays
in the repo, marked superseded, for anyone tracing why an old PR
referenced "Tourist = Overview".

## Rendered surface

- **De vloot** (`/ui/`, `/ui/orgs/{slug}`) opens with the scan-input
  form as a compact bar under the header, then shows the fleet as
  pictures (change `2026-09-24-vloot-in-beeld`, after Mark on
  2026-09-24: "still too much Analysis"): the vlootscore as a ring with
  a legend (soeverein, niet soeverein, onbeantwoord), one stacked bar
  per stroom, the top-3 duurste regels as handelingen with a "x van n
  domeinen" bar, and the domains as a grid of domein × stroom
  (slechtste eerst). No rule rationale on this layer; it lives on the
  rule page, one layer down. Every picture carries its facts as text
  too.
- **Het domein** (`/ui/scans/{id}/answer`) shows the `x/n` score next
  to the oordeelzin, and one handeling per niet-soeverein flow.
- **De techniek** (`/ui/scans/{id}/assessment`) opens with the seven
  flow questions (`section.sovereignty-overview`); the two framework
  tables render inside `<details class="framework-details">`, closed
  by default, with a `<summary>` naming how many regels are underneath
  across how many frameworks.
- **Trends** (`/ui/trends`) is unchanged by this decision: rule
  catalogue + score matrix, still reachable from the nav.

Go tests cover the vlootscore build (`internal/ui/fleet_score_test.go`),
the answer page's score + remediation
(`internal/ui/flow_answers_test.go`, `internal/ui/answer_test.go`),
and the onderbouwing page's collapsed framework tables
(`internal/ui/ui_test.go`,
`TestAssessmentPage_FrameworkTablesCollapseBehindOneClick`). Playwright
coverage for the fleet score and the worst-domain-first ordering is
tracked as run 05 task 6.1, not yet written as of this ADR.
