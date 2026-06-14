# Tasks: Nextcloud as output

## 1. Direction decisions

- [x] 1.1 Q1 — publication surface. **Decided: WebDAV file drop**
  (Talk/Deck deferred as later opt-in adapters).
- [x] 1.2 Q2 — push vs pull. **Decided: push** (post-scan hook).
- [x] 1.3 Q3 — output format. **Decided: JSON-LD + Markdown.**
- [x] 1.4 Q4 — auth model. **Decided: app password** via
  `app_password_file` (HTTP Basic).

## 2. Implementation

- [x] 2.1 New `internal/export/nextcloud/` package — WebDAV
  MKCOL + PUT client (`webdav.go`) + publisher (`publisher.go`).
- [x] 2.2 `serve.yaml` `nextcloud:` block + validation (partial
  block + non-http(s) url rejected at startup).
- [x] 2.3 Post-scan hook — `scanner.Publisher` seam; runs after the
  scan is persisted, bounded timeout + retry, logs
  `wanderer.nextcloud.publish.error`, never fails the scan.
- [x] 2.4 Redaction pass — ADR-0008 `egress.Apply` on every Finding
  attribute (incl. nested maps + slice strings); raw Evidence
  dropped before serialisation.
- [x] 2.5 Tests with an httptest WebDAV stub (drop, redaction,
  retry-success, bounded-retry-give-up) + serveconfig tests.
- [x] 2.6 docs/operator.md walkthrough ("Publish scans into
  Nextcloud (WebDAV)").

## 3. Wrap-up

- [x] 3.1 Commit.
- [x] 3.2 Archive.
