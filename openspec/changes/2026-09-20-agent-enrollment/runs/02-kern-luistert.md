# Habitat run 02 — de kern luistert echt (tasks 2.1–2.3)

Contract: `openspec/changes/2026-09-20-agent-enrollment/`
(specs/scanner/spec.md, requirement "The served core accepts findings
from enrolled agents").

Waar het om gaat: `cmd/wanderer/serve.go` bouwt de router met
`api.Router(...)`, en die zet `AgentSecrets` op `nil`. De documentatie
bij `FindingsIngestHandler` zegt het zelf — met `nil` is de route
geregistreerd maar wordt élk verzoek geweigerd. Er bestaat al een
`RouterWithSecrets`; niets vult die.

## Scope — ONLY these tasks
- [ ] 2.1 `serve` bouwt de router met een `AgentSecrets` die de
  geheimen uit de store leest (run 01 zette de tabel neer; een
  ingetrokken agent levert geen geheim op). Zijn er geen aangemelde
  agents, dan blijft het gedrag precies zoals nu — weigeren — en zegt
  de opstartregel één keer dat agent-ingest inactief is.
- [ ] 2.2 De agent-kant: bij het starten in `mode=remote` wisselt hij
  een aanmeldtoken in als hij nog geen geheim heeft, schrijft dat 0600
  weg, en meldt zich daarna niet opnieuw aan. Token via configuratie of
  vlag; het token verdwijnt na gebruik uit de configuratie of wordt
  genegeerd.
- [ ] 2.3 Tests: aangemelde agent levert findings af en ze staan op de
  scan; onbekende of ingetrokken agent krijgt 401; zonder agents blijft
  de route dicht.

## Out of scope
De partij-identificatie (run 03) en alles in `internal/ui` — daar loopt
een andere change in dezelfde repo.

## Over bouwen en testen in de kooi
Draai tijdens het werk alleen je eigen pakketten (`go test ./internal/api/...`,
`./internal/agent/...`, `./internal/store/...`) en bewaar één volledige
`go build ./...` + `go vet ./...` + `go test ./...` voor het eind. Zet
GOFLAGS of GOPROXY niet om; die staan goed (offline uit vendor/).

## Done means
Die volledige build/vet/test groen en `openspec validate
2026-09-20-agent-enrollment --strict` groen. Budget is $5.
