# Habitat run 04 — taal en de achtergebleven specs (tasks 4.1–4.2, 6.0)

Contract: `openspec/changes/2026-09-22-drie-lagen-ciso/`
(specs/web-ui/spec.md, requirement "Eén taal per laag").

## Scope — ONLY these tasks
- [ ] 6.0 Twee specs in `tests/playwright/specs/answer-first-flow.spec.ts`
  zoeken `form.door-form`; het scanformulier heet sinds run 02
  `class="scan-form"` en staat onderaan de vlootpagina. Breng de specs in
  lijn met de nieuwe indeling. Verander de UI NIET om een oude spec te
  plezieren.
- [ ] 4.1 De oordelen van de wand-regels in het Nederlands, uit de
  bestaande tekstentabel (`accountability_nl.yaml` heeft al de
  vraag/verdict/handeling-structuur — breid die uit in plaats van een
  tweede tabel te maken). Nu staat er bijvoorbeeld "apex IPs in NL
  (EEA)" naast de Nederlandse vraag "Waar staat de hosting?", en
  "no ip.asn finding for apex — IP probe did not run" naast "niet
  gemeten".
- [ ] 4.2 Een test die een Engelstalig oordeel buiten het bewijs vangt.
  Kies een aanpak die niet op woordenlijsten leunt maar op het feit dat
  elk getoond oordeel uit de tabel moet komen — een oordeel dat niet uit
  de tabel komt, faalt. Motiveer je aanpak in het run-rapport.

Engels blijft staan in het uitgeklapte bewijs (veldnamen, RFC's, ruwe
probe-uitvoer) — daar hoort het.

## Out of scope
De onderbouwingspagina herindelen (run 05).

## Over bouwen en testen in de kooi
Je kunt Playwright hier niet draaien (geen browser-egress). Pas de specs
aan, zeg dat je ze niet gedraaid hebt, en ik meet na. Draai wel
`go test ./internal/ui/... ./internal/assessor/...`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-22-drie-lagen-ciso --strict` groen.
Budget is $5.
