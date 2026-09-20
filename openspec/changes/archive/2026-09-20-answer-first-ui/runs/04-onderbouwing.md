# Habitat run 04 — de onderbouwing (tasks 4.1–4.3)

Contract: `openspec/changes/2026-09-20-answer-first-ui/` (requirement
"The reasoning is one click from the answer").

## Scope — ONLY these tasks
- [ ] 4.1 De onderbouwingspagina toont de zeven stromen uit
  `SovereigntyFlows` (hosting, mail, DNS, transit, CDN/hyperscaler,
  derde partijen, certificaat) plus de accountability-antwoordlijst,
  elk als vraag met ja / nee / onbekend / n.v.t., met het waargenomen
  feit in het oordeel en het bewijs ingeklapt eronder. Hergebruik wat
  de accountability-sectie al doet; bouw geen tweede weergave.
- [ ] 4.2 Regel-ID's, RDAP-velden en RFC-nummers staan alleen binnen
  het uitgeklapte bewijs.
- [ ] 4.3 Eén link van antwoord naar onderbouwing en terug. De
  analistenlaag (`/ui/trends`, de catalogus) blijft ongewijzigd en
  blijft bereikbaar.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-20-answer-first-ui --strict` groen.
Budget is $5.
