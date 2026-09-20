# Habitat run 03 — het antwoord dat zich vult (tasks 3.1–3.3)

Contract: `openspec/changes/2026-09-20-answer-first-ui/`
(requirements "The answer is one sentence with its reason" and "The
answer fills in while the scan runs").

Waarom dit moet: een scan duurt 30–60 seconden, vrijwel helemaal door
de traceroute. Wachten op alles betekent een lege pagina terwijl DNS en
TLS al binnen een seconde klaar zijn.

## Scope — ONLY these tasks
- [ ] 3.1 De antwoordpagina rendert uit de findings die er op dat
  moment zijn: per stroom `nog bezig`, een antwoord, of `niet gemeten`.
  De kop komt uit run 01's functie en zegt ja / nee / onbekend plus het
  aantal onbeantwoorde vragen. De scanner schrijft findings per probe
  weg (`AppendFindings`), dus dit is lezen wat er al staat — geen nieuw
  opslagmechanisme.
- [ ] 3.2 De pagina ververst zichzelf tot de scan klaar is en stopt
  daarna. Zonder JavaScript doet een `meta refresh` hetzelfde werk; de
  pagina blijft dus werken zonder scripts (dat is de bestaande lijn van
  deze UI — inline SVG, geen bouwstap).
- [ ] 3.3 Tests: een halve scan rendert (DNS klaar, transit nog bezig),
  een afgeronde scan ververst niet meer, en een scan zonder enige
  finding zegt dat hij net begonnen is in plaats van "nee".

## Out of scope
De onderbouwingspagina (run 04). Eén link ernaartoe volstaat.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-20-answer-first-ui --strict` groen.
Budget is $5.
