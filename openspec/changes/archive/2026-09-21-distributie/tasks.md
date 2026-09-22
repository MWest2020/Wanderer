# Tasks: distributie

## 1. Binaries — run 01
- [x] 1.1 `.goreleaser.yaml`: linux/darwin/windows × amd64/arm64,
  `CGO_ENABLED=0`, versie via ldflags zoals de Makefile dat doet,
  checksums, changelog uit de tag.
- [x] 1.2 Release-workflow die bij een tag `v*` goreleaser draait.
- [x] 1.3 README: één curl-regel en één docker-regel bovenaan.
- [x] 1.4 CI draait het README-commando, zodat het niet kan verouderen.

## 2. De action — run 02
- [x] 2.1 `action.yml` (composite): haalt de binary op, scant het
  opgegeven domein, beoordeelt, schrijft de job-samenvatting.
- [x] 2.2 Invoer: `domain` (verplicht), `fail-on` (leeg = nooit falen),
  `geoip` (optioneel pad). Uitvoer: score, oordeel, en het pad naar het
  rapport.
- [x] 2.3 Voorbeeldworkflow in `docs/` plus een regel in de README.
- [x] 2.4 De action draait in haar eigen repo-CI op één echt domein
  (`.github/workflows/action-smoke.yml`, westerweel.work). De workflow
  faalt niet op het oordeel — wel op een lege score, een onbekend
  oordeel, een leeg rapport, of een falend punt zonder handeling. Die
  laatste controle is één keer nagemeten met de handeling eruit
  gesloopt: 0 met, 3 zonder.

## 3. De handeling bereikbaar maken — run 03
- [x] 3.1 Gevonden bij het uitproberen van de Action (2026-09-22): de
  handelingen bestaan wél (28 in `accountability_nl.yaml`, test dekt
  alle regels), maar staan niet in de JSON-uitvoer van `wanderer assess`.
  De Action meldt daarom "Geen kant-en-klare handeling beschikbaar" bij
  elk falend punt. Zet de handeling bij elke rationale die niet
  soeverein scoort in de JSON-uitvoer, zodat elke machinale lezer erbij
  kan — de UI leest hem al rechtstreeks uit de tabel.
- [x] 3.2 De Action toont die handeling in plaats van de
  plaatsvervangende zin.

## 4. Documentatie
- [x] 4.1 CHANGELOG + een korte pagina "in je pijplijn" met de
  waarschuwing dat een soevereiniteitsoordeel geen bouwfout is.
