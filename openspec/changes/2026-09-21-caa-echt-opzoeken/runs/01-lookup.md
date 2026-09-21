# Habitat run 01 — CAA echt opzoeken (tasks 1.1–2.2)

Contract: `openspec/changes/2026-09-21-caa-echt-opzoeken/`.

Waargenomen 2026-09-21: `internal/probe/dns/resolver.go` bevat

```go
func (n *netResolver) LookupCAA(_ context.Context, _ string) ([]CAA, error) {
	return nil, nil
}
```

Daardoor meldde de probe voor elk domein "no CAA records". digid.nl,
mijnoverheid.nl en ncsc.nl hebben die records wél (certsign.ro,
digicert.com; ncsc.nl ook een iodef).

## Scope — ONLY these tasks
- [ ] 1.1 `LookupCAA` doet een echte CAA-vraag (DNS-type 257). Go's
  `net.Resolver` kent geen CAA; kies een aanpak die past bij deze repo
  (stdlib-first, ADR-0003) — bijvoorbeeld zelf een DNS-vraag opbouwen,
  of een kleine, gepinde afhankelijkheid als dat aantoonbaar minder
  code is. Motiveer de keuze in je run-rapport; voeg GEEN zware
  DNS-bibliotheek toe zonder die afweging.
- [ ] 1.2 Vindt hij niets, dan omhoog tot het registreerbare domein
  (RFC 8659 §3). De finding legt vast op welke naam de records stonden.
  Er is al logica voor registreerbare domeinen in de scanner
  (NS-holder-lookups); hergebruik die in plaats van een tweede variant.
- [ ] 1.3 Bij erven noemt de verdicttekst de herkomst.
- [ ] 1.4 Tests met een stub-resolver: eigen records, geërfd, nergens
  iets, én een test die faalt op een resolver die altijd leeg
  teruggeeft — die had de huidige fout gevangen.
- [ ] 2.1 Loop de andere opzoekingen in `internal/probe/dns` en de rest
  van `internal/probe` na op hetzelfde patroon (een functie die
  stilzwijgend niets teruggeeft) en zeg in je run-rapport wat je vond,
  ook als dat "niets" is.
- [ ] 2.2 docs + CHANGELOG: wat er mis was en wat eerdere CAA-oordelen
  waard waren.

## Let op
De regel `caa_restricts_issuance` blijft zoals hij is — die oordeelde
correct op verkeerde gegevens. Verander zijn logica niet.

## Over bouwen en testen in de kooi
Tests mogen GEEN echte DNS doen: gebruik een stub-resolver. Draai
tijdens het werk `go test ./internal/probe/dns/...`; bewaar één
volledige build/vet/test voor het eind.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-21-caa-echt-opzoeken --strict` groen.
Budget is $5.
