# Tasks: Sovereignty overview

## 1. Decisions
- [x] 1.1 Q1 — textual flow list first (graph deferred).
- [x] 1.2 Q2 — six flows: Hosting, Mail, DNS, Transit, CDN/hyperscaler, Third parties.
- [x] 1.3 Q3 — render on the scan's assessment page.

## 2. Implementation
- [x] 2.1 `internal/ui/SovereigntyFlows` — pure synthesis from the
  scored assessment rationales (no EEA logic in the UI).
- [x] 2.2 "Sovereignty overview" panel on the assessment view.
- [x] 2.3 Tests (flows ordering/labels/omission) + Playwright spec
  (`sovereignty-overview.spec.ts`); ADR-0015.
- [x] 2.4 Live-smoked: panel renders all six flows on the baseline
  fixture's assessment page. Reviewed.

## 3. Wrap-up
- [x] 3.1 Commit + push.
- [x] 3.2 Archive.

## Follow-ups
- [ ] Interactive node-graph ("spider in the web") once the flow model proves out.
- [ ] Org-level roll-up across an organisation's scans.
