# Delta for web-ui

> Implemented 2026-06-15; ready to merge into the canonical web-ui spec.

## ADDED Requirements

### Requirement: An opt-in dev mode lets the UI trigger a scan

When `wanderer serve --ui-allow-scan` is set, the UI SHALL render a
"Scan a target" form and accept `POST /ui/scan`, which scans the
submitted target, assesses it, and redirects to the target scan's
assessment page. When the flag is not set, the UI SHALL expose neither
the form nor the route and SHALL remain read-only (no mutating
handlers other than the sanctioned scan route exist). The scanner's
private-target guard SHALL continue to apply.

#### Scenario: Dev mode scans from the browser

- **GIVEN** serve is started with `--ui-allow-scan`
- **WHEN** an operator submits a target on the dashboard scan form
- **THEN** a scan runs, an assessment is produced, and the browser is
  redirected to that scan's assessment page

#### Scenario: Read-only by default

- **GIVEN** serve is started without `--ui-allow-scan`
- **WHEN** a client POSTs to `/ui/scan`
- **THEN** the request does not trigger a scan (the route is not mounted)
