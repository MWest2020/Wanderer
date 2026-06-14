# Delta for exporters

> Accepted 2026-06-14 — implemented and ready to archive into the
> canonical exporters spec.

## ADDED Requirements

### Requirement: Wanderer can publish scan artefacts to Nextcloud via WebDAV

When configured, Wanderer SHALL publish each completed scan's
artefacts (JSON-LD + Markdown by default) to a Nextcloud
instance via WebDAV under a configurable target directory.
Publish failures SHALL NOT block scan completion; the local
`/ui/` view remains the authoritative surface.

#### Scenario: A successful scan lands as a JSON-LD file in Nextcloud

- **GIVEN** `serve.yaml`'s `nextcloud.enabled: true` block is
  configured against a reachable Nextcloud
- **WHEN** a scan completes for `conduction.nl`
- **THEN** a file appears at
  `/Files/Wanderer/<org-slug>/<scan-id>.jsonld` on the
  Nextcloud, containing the scan's Findings + Assessment

#### Scenario: Publish failure does not block the scan

- **GIVEN** the configured Nextcloud is unreachable
- **WHEN** a scan completes
- **THEN** the scan is persisted to the local store with
  status `complete`, AND a
  `wanderer.nextcloud.publish.error` log entry is emitted,
  AND the scan does NOT show up as `failed`
