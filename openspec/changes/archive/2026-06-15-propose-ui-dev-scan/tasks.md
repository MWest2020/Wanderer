# Tasks: dev-mode scan form
## 1. Implementation
- [x] 1.1 Options.Scanner (ScanTrigger) + POST /ui/scan (scan+assess+redirect).
- [x] 1.2 "Scan a target" form on the dashboard (shown when allow-scan).
- [x] 1.3 serve --ui-allow-scan flag + wiring (+ unauth warning).
- [x] 1.4 Read-only test tightened (PUT/PATCH/DELETE blocked; only /scan POST).
- [x] 1.5 Tests (dev-mode redirect + read-only default) + Playwright; ADR-0016.
- [x] 1.6 Live-smoked (form → POST → overview). CHANGELOG.
## 2. Wrap-up
- [x] 2.1 Commit + push.
- [x] 2.2 Archive.
