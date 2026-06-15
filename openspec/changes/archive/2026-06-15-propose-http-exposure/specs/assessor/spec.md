# Delta for assessor

> Implemented 2026-06-15; ready to merge into the canonical assessor spec.

## ADDED Requirements

### Requirement: Passive HTTP exposure is scored

The wand rule pack SHALL score a target's passive HTTP exposure from
the observed security-header set and response banner. When HSTS is
absent it SHALL score afhankelijk; when HSTS is present but other
baseline headers are missing it SHALL score voldoende; when all
baseline headers are present it SHALL score soeverein; and when no
security-header observation exists it SHALL score onbekend. A Server /
X-Powered-By stack disclosure SHALL be named in the verdict. The rule
SHALL NOT perform any active or intrusive probing.

#### Scenario: Missing HSTS scores afhankelijk

- **GIVEN** a scan whose http.security_headers finding lists
  Strict-Transport-Security as missing
- **WHEN** the assessor runs
- **THEN** wand.operationeel.http_exposure scores afhankelijk and names
  the missing headers
