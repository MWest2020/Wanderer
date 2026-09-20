# Habitat run 04 — drempels uit de code (tasks 4.1–4.2)

Contract: `openspec/changes/2026-09-20-vloot-en-regels/`
(specs/assessor/spec.md, requirement "Een regel draagt zijn drempels
als waarde").

Waarom: een drempel als "90 dagen", "30 dagen" of "24 verbindingen" is
een aanname over één corpus. Zit hij alleen in de vergelijking, dan kan
niemand hem zien, navertellen of weerleggen — en een lezer weet niet
waarom zijn domein omslaat.

## Scope — ONLY these tasks
- [ ] 4.1 `assessor.Rule` krijgt een veld voor zijn beslisgrenzen: per
  grens een naam, een waarde en een eenheid, plus één zin in mensentaal
  ("verloopt binnen 30 dagen"). Regels zonder grens (aanwezig-of-niet)
  houden een lege lijst. Geen gedrag verandert in deze run.
- [ ] 4.2 De bestaande regels van de wand- en eucsf-pakketten vullen hun
  grenzen in, en een test vangt het uiteenlopen: als een regel een grens
  meedraagt, moet zijn vergelijking diezelfde waarde gebruiken. Doe dat
  waar het kan met een gedeelde constante per regel, zodat de test niet
  alleen naar de tekst kijkt.

Begin bij de regels waar de grens hard is en zichtbaar hoort te zijn:
`wand.operationeel.domain_expiry` (90/30 dagen),
`wand.operationeel.variant_convergence` (8 paden, 24 verbindingen),
`wand.accountability.securitytxt` (verlopen ja/nee) en
`wand.operationeel.cert_validity`. Kom je een regel tegen die een grens
gebruikt die nergens is uitgelegd, noteer hem dan in je run-rapport.

## Out of scope
De regelpagina zelf en de handelingen (run 05). Geen scoringsgedrag
wijzigen: dezelfde findings horen dezelfde oordelen te geven.

## Over bouwen en testen in de kooi
Draai tijdens het werk `go test ./internal/assessor/...`; bewaar één
volledige build/vet/test voor het eind (eerste build duurt minuten).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-20-vloot-en-regels --strict` groen.
Budget is $5.
