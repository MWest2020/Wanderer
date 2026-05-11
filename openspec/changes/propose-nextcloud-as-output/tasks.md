# Tasks: Nextcloud as output (design pending)

Every task is a design checkpoint until Mark picks Q1.

## 1. Direction decisions

- [ ] 1.1 Q1 — publication surface (file drop / Talk / Deck /
  combo). Recommendation: file drop first.
- [ ] 1.2 Q2 — push vs pull. Recommendation: push.
- [ ] 1.3 Q3 — output format (JSON-LD + Markdown / HTML).
  Recommendation: JSON-LD + Markdown.
- [ ] 1.4 Q4 — auth model. Recommendation: app password.

## 2. Implementation skeleton (after sign-off)

- [ ] 2.1 New `internal/export/nextcloud/` package
  alongside csv + jsonl. Implements WebDAV PUT against the
  Nextcloud API.
- [ ] 2.2 Configuration block in `serve.yaml`:
  ```yaml
  nextcloud:
    enabled: false
    url: https://cloud.example.nl
    username: wanderer-bot
    app_password_file: /etc/wanderer/nc.token
    target_dir: /Files/Wanderer
  ```
- [ ] 2.3 Post-scan hook wiring — every persisted scan +
  Assessment publishes if the block is enabled. Failures emit
  a `wanderer.nextcloud.publish.error` operator log + a
  bounded retry; never block scan completion.
- [ ] 2.4 Redaction pass before publish — ADR-0008 contract
  on every Finding's Attributes before serialisation.
- [ ] 2.5 Tests with a httptest stub mimicking the Nextcloud
  WebDAV endpoints.
- [ ] 2.6 docs/operator.md walkthrough.

## 3. Wrap-up

- [ ] 3.1 Commit + push.
- [ ] 3.2 Archive.
