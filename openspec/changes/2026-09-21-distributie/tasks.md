# Tasks: distributie

## 1. Binaries — run 01
- [ ] 1.1 `.goreleaser.yaml`: linux/darwin/windows × amd64/arm64,
  `CGO_ENABLED=0`, versie via ldflags zoals de Makefile dat doet,
  checksums, changelog uit de tag.
- [ ] 1.2 Release-workflow die bij een tag `v*` goreleaser draait.
- [ ] 1.3 README: één curl-regel en één docker-regel bovenaan.
- [ ] 1.4 CI draait het README-commando, zodat het niet kan verouderen.

## 2. De action — run 02
- [ ] 2.1 `action.yml` (composite): haalt de binary op, scant het
  opgegeven domein, beoordeelt, schrijft de job-samenvatting.
- [ ] 2.2 Invoer: `domain` (verplicht), `fail-on` (leeg = nooit falen),
  `geoip` (optioneel pad). Uitvoer: score, oordeel, en het pad naar het
  rapport.
- [ ] 2.3 Voorbeeldworkflow in `docs/` plus een regel in de README.
- [ ] 2.4 De action draait in haar eigen repo-CI op één echt domein.

## 3. Documentatie
- [ ] 3.1 CHANGELOG + een korte pagina "in je pijplijn" met de
  waarschuwing dat een soevereiniteitsoordeel geen bouwfout is.
