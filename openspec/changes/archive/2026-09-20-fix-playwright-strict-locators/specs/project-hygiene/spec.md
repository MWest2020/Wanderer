## ADDED Requirements

### Requirement: De Playwright-suite is groen en wijst op één element

De Playwright-suite SHALL groen zijn op `main`. Een locator in een spec
SHALL op precies één element wijzen: scope binnen de tabel, sectie of
kop die de test bedoelt, in plaats van een kale tekst- of regex-match
over de hele pagina. Een spec SHALL bovendien in een `testMatch` van
`tests/playwright/playwright.config.ts` staan, anders draait hij niet en
bewijst hij niets.

Een suite met blijvend rode specs leert lezers de uitslag te negeren,
en dan valt een echte regressie niet meer op.

#### Scenario: Pagina toont hetzelfde regel-ID in twee tabellen

- **GIVEN** `/ui/trends` toont regel-ID's in zowel `table.rule-catalogue`
  als `table.reporting-rules`
- **WHEN** een spec controleert dat een regel in de catalogus staat
- **THEN** scoopt de locator binnen `table.rule-catalogue` en slaagt de
  test, in plaats van te falen op "resolved to 2 elements"

#### Scenario: Nieuwe spec zonder testMatch

- **GIVEN** een spec-bestand dat in geen enkel project van
  `playwright.config.ts` genoemd wordt
- **WHEN** `make playwright` draait
- **THEN** draait die spec niet mee — de spec telt pas als bewijs zodra
  hij in een `testMatch` staat
