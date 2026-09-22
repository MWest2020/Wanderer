# Habitat run 03 — een keer is een keer (tasks 3.1–3.3, 4.1)

Contract: `openspec/changes/2026-09-20-agent-enrollment/`
(specs/scanner/spec.md, requirement "A replayed batch is stored once").

## Waar het om gaat
`internal/api/findings.go` (`FindingsIngestHandler`) slaat elke POST op
met `st.AppendFindings(...)`. Er is geen identificatie per partij, dus
dezelfde partij twee keer afleveren zet de bevindingen er twee keer in.
Dat gebeurt niet bij een aanval maar bij gewoon gebruik: de outbox
(`internal/agent/outbox.go`) hertikt een partij zodra de aflevering
mislukte — en "mislukt" is óók een antwoord dat onderweg verdween nadat
de kern hem al had opgeslagen.

`SpooledBatch` bewaart nu alleen `scan_id` + de ruwe body. De
identificatie moet dáár bij, zodat hij een herstart overleeft: dat is
precies wat het tweede scenario in de spec eist.

## Scope — ONLY these tasks
- [ ] 3.1 Elke partij draagt een identificatie; de kern slaat hem één
  keer op. Een herhaling antwoordt "al ontvangen" (kies de status
  bewust en zeg in je rapport waarom: 200 met `received: 0` en een
  `duplicate: true`, of 409 — niet allebei) en slaat niets opnieuw op.
  Waar de identificatie vandaan komt is jouw keuze; leg hem vast in de
  spec-taal die er al staat. Een kopregel die de agent zet is de
  eenvoudigste; hem uit de body afleiden (een hash over de inhoud)
  scheelt een veld maar maakt twee echt verschillende partijen met
  dezelfde inhoud onzichtbaar. Kies, en schrijf op waarom.
- [ ] 3.2 De outbox bewaart die identificatie over een herstart heen —
  dus in het bestand op schijf, niet alleen in het geheugen.
- [ ] 3.3 Tests: dezelfde partij twee keer (de bevindingen staan er één
  keer), drie gespoolde partijen waarvan de drain twee keer loopt, en
  een outbox die na een herstart aflevert met de identificatie waarmee
  hij gespoold werd. Controleer elke nieuwe test één keer mét de
  reparatie eruit en zeg in je rapport dat je dat deed.
- [ ] 4.1 `docs/explanation/agent.md` + operator-documentatie bijwerken;
  CHANGELOG onder `[Unreleased]`.

## Out of scope
`internal/ui` en `internal/assessor` — daar lopen twee andere changes in
dezelfde repo. Raak ze niet aan.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen, en
`openspec validate 2026-09-20-agent-enrollment --strict` groen.
Budget is $8.
