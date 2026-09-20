# Habitat run 01 — geplande scans beoordelen (tasks 1.1–2.1)

Contract: `openspec/changes/2026-09-20-assess-scheduled-scans/`.

## Scope — ONLY these tasks
- [x] 1.1 `internal/scheduler`: na een geslaagde scan een Assessment
  maken met BEIDE rule packs en opslaan. Hergebruik wat `wanderer
  assess --framework both` doet (zie `cmd/wanderer/assess.go` en
  `internal/api` route `POST /scans/{id}/assessments`) — geen tweede
  implementatie van dezelfde logica.
- [x] 1.2 Faalt het beoordelen, dan blijft de scan staan, wordt de fout
  gelogd en gaat de volgende tick door.
- [x] 1.3 `assess: false` per schedule; ontbreekt het veld, dan staat het
  aan. Let op: de configuratie wordt met `UnmarshalStrict` geparsed, dus
  bestaande bestanden moeten geldig blijven.
- [x] 1.4 Tests: scan + assessment; `assess: false` → alleen scan;
  falende beoordeling → scan blijft staan.
- [x] 2.1 Scheduling-documentatie + CHANGELOG onder `[Unreleased]`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-20-assess-scheduled-scans --strict`
groen. Budget is $4.
