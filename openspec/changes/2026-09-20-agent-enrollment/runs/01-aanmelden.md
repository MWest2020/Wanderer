# Habitat run 01 — aanmelden en intrekken (tasks 1.1–1.4)

Contract: `openspec/changes/2026-09-20-agent-enrollment/`
(specs/scanner/spec.md, requirement "An agent enrols itself once and
can be revoked").

## Scope — ONLY these tasks
- [ ] 1.1 Migratie achteraan in `internal/store/migrations.go` plus
  model/store-functies: tabel voor agents (hostnaam uniek, hash van het
  geheim, aangemeld op, ingetrokken op) en voor aanmeldtokens (hash,
  geldig tot, gebruikt op). Sla NOOIT het kale geheim of het kale token
  op — alleen een hash.
- [ ] 1.2 `POST /agents/enrol` in `internal/api`: neemt een token en een
  hostnaam, geeft één keer een nieuw geheim terug, markeert het token
  als gebruikt. Verlopen, al gebruikt, of onbekend token → geweigerd,
  zonder verschil in de foutmelding waaruit je kunt afleiden wélke van
  de drie het was.
- [ ] 1.3 CLI in `cmd/wanderer`: een token aanmaken (met geldigheidsduur),
  agents tonen (hostnaam, aangemeld op, ingetrokken op), en een agent
  intrekken. Het token wordt één keer getoond en daarna nooit meer.
- [ ] 1.4 Table-driven tests: token twee keer inwisselen faalt, verlopen
  token faalt, intrekken weigert daarna de findings van die agent en
  laat een andere agent ongemoeid.

## Out of scope voor deze run
`cmd/wanderer/serve.go` aansluiten (run 02) en de partij-identificatie
(run 03). Raak die bestanden niet aan — er loopt een andere run in deze
repo aan de UI-kant.

## Over bouwen en testen in de kooi — lees dit eerst
De eerste `go build ./...` in de kooi duurt minuten: `modernc.org/sqlite`
is getranspileerde C en staat gevendord in de repo. Draai daarom tijdens
het werk alleen je eigen pakket:

    go test ./internal/ui/...

en bewaar één volledige `go build ./...` + `go vet ./...` + `go test ./...`
voor het eind. Ga NIET zoeken naar een snellere manier, en zet GOFLAGS of
GOPROXY niet om — die staan goed (offline uit vendor/).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-20-agent-enrollment --strict`
groen. Budget is $5.
