# Habitat run 02 — de GitHub Action (tasks 2.1–3.1)

Contract: `openspec/changes/2026-09-21-distributie/`
(specs/project-hygiene/spec.md, requirement "De action scant bij de
gebruiker en stuurt niets door").

## Scope — ONLY these tasks
- [ ] 2.1 `action.yml` in de repo-root: een **composite** action die de
  gepubliceerde binary ophaalt (v0.7.0 hangt vol met
  `wanderer_<os>_<arch>.tar.gz` + `checksums.txt` — controleer de
  checksum), het opgegeven domein scant, beoordeelt, en het resultaat
  naar `$GITHUB_STEP_SUMMARY` schrijft: de score, wat het oordeel
  bepaalde, en de handeling per falend punt.
- [ ] 2.2 Invoer: `domain` (verplicht), `fail-on` (leeg = nooit falen;
  waarden: `afhankelijk`, `onbekend`), `geoip` (optioneel pad naar een
  mmdb), `version` (standaard de nieuwste release). Uitvoer: `score`,
  `verdict`, `report-path`.
- [ ] 2.3 Voorbeeldworkflow in `docs/` (NIET in `.github/workflows/` —
  dat mag jouw rol niet aanraken en je token weigert het) plus een blok
  in de README dat die workflow toont.
- [ ] 3.1 CHANGELOG + een korte pagina "in je pijplijn": hoe je hem
  gebruikt, en de waarschuwing dat een soevereiniteitsoordeel geen
  bouwfout is — vandaar dat `fail-on` standaard leeg is.

## Let op
- De action draait bij de gebruiker en stuurt NIETS naar ons: geen
  telemetrie, geen "fonetisch naar huis bellen", geen standaard
  GeoIP-download van een server van ons. Schrijf dat ook zo in de
  documentatie.
- Zonder GeoIP-database zijn de jurisdictie-antwoorden `onbekend`. Zeg
  dat eerlijk in de samenvatting in plaats van het weg te laten.

## Over bouwen en testen in de kooi
Je kunt hier geen GitHub Action draaien. Test wat je kunt: het script
dat de binary ophaalt en de samenvatting bouwt, met een lokaal
gebouwde binary (`go build ./cmd/wanderer`) in plaats van een download.
Zeg in je run-rapport wat je NIET hebt kunnen draaien.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-21-distributie --strict` groen. Budget is $5.
