# Tasks: answer-first UI

## 1. Verdict in mensentaal — run 01
- [ ] 1.1 One place that maps an assessment to ja / nee / onbekend
  (soeverein+voldoende → ja, afhankelijk → nee, onbekend stays), with
  the count of unanswered questions and the observation that decided a
  "nee". Never promote onbekend to ja.
- [ ] 1.2 Dutch copy for the headline in one string table, like
  `accountability_nl.yaml`; no copy in templates.
- [ ] 1.3 Table-driven tests incl. all-onbekend, mixed, and the
  "one unknown question" wording.

## 2. De deur — run 02
- [x] 2.1 `/ui/` leads with the domain input plus recently answered
  targets as one-line verdicts; the fleet table and matrix move out of
  the first screen.
- [x] 2.2 The scan route requires a signed-in user (OIDC session or
  htpasswd). No authentication configured → route refused, said once at
  startup. `--ui-allow-scan` is no longer the gate.
- [x] 2.3 Hosts that report through an agent are selectable in the same
  input.

## 3. Het antwoord dat invult — run 03
- [x] 3.1 The answer page renders from the findings so far, per flow:
  nog bezig / answered / niet gemeten.
- [x] 3.2 It refreshes until the scan completes; without JavaScript a
  meta-refresh does the same job.
- [x] 3.3 Tests: partial scan renders, completion stops the refresh.

## 4. De onderbouwing — run 04
- [x] 4.1 The reasoning page: seven flows + accountability as questions
  with answers, observed fact in the verdict, evidence collapsed.
- [x] 4.2 Rule IDs and protocol jargon only inside evidence.
- [x] 4.3 One link from answer to reasoning, and back.

## 5. Bewijs en documentatie — run 05
- [ ] 5.1 Playwright: domain in → answer → reasoning, and the
  onbekend-is-not-ja case. Wired into `playwright.config.ts`.
- [ ] 5.2 `docs/` + CHANGELOG; ADR-0017 gets an addendum describing the
  answer-first layering (it is an IA change, so it belongs there).
