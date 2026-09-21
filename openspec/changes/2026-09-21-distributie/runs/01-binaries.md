# Habitat run 01 — binaries bij elke release (tasks 1.1–1.4)

Contract: `openspec/changes/2026-09-21-distributie/`
(specs/project-hygiene/spec.md, requirement "Elke release draagt
bruikbare binaries").

## Scope — ONLY these tasks
- [ ] 1.1 `.goreleaser.yaml` in de repo-root: linux/darwin/windows ×
  amd64/arm64, `CGO_ENABLED=0` (mag hier: de database is
  `modernc.org/sqlite`, pure Go), versie via dezelfde ldflags als de
  Makefile (`-X main.Version=...`), checksums, archief per platform.
  Let op: de repo bouwt offline uit `vendor/` — zorg dat de configuratie
  dat niet doorbreekt.
- [ ] 1.2 `.github/workflows/release.yml`: draait goreleaser bij een tag
  `v*` en hangt de bestanden aan de release. Gebruik de bestaande
  `GITHUB_TOKEN`; voeg geen nieuwe secrets toe.
- [ ] 1.3 README: bovenaan één `curl`-regel die de binary voor het
  huidige platform haalt en één `docker run`-regel (het bestaande image
  `ghcr.io/mwest2020/wanderer-exapp` bevat de binary op
  `/usr/local/bin/wanderer` — gebruik dat, bouw geen tweede image).
- [ ] 1.4 Laat CI het README-commando echt draaien (een klein script of
  een stap die de regel uit de README leest en uitvoert), zodat een
  verouderde quickstart de bouw laat falen.

## Let op
Geen nieuwe afhankelijkheden in de Go-module. goreleaser draait als
GitHub Action, niet als library.

## Over bouwen en testen in de kooi
Je kunt hier geen release maken en geen Docker draaien; controleer wat
je kunt (`goreleaser check` als het beschikbaar is, anders de YAML
zorgvuldig nalopen) en zeg in je run-rapport wat je NIET hebt kunnen
draaien.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-21-distributie --strict` groen. Budget is $4.
