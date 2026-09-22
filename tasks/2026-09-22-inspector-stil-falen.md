# Habitat run — de Nextcloud-inspecteur zwijgt als hij niets begrijpt

## Wat er mis is

`ParseSystemConfig` (`internal/probe/inventory/nextcloud/nextcloud.go:379`)
doet dit bij onleesbare invoer:

```go
if err := json.Unmarshal([]byte(raw), &doc); err != nil {
    return nil, nil
}
```

De fout verdwijnt. `Inspect` (regel 118) voegt dan niets toe, logt
niets en geeft geen fout terug. De eucsf-regel ziet nul bevindingen en
zegt:

    "no inventory.nextcloud.objectstore / .oidc_provider findings —
     Nextcloud inspector did not run or no relevant configuration"

Drie situaties worden daar één: de inspecteur draaide niet, er ís niets
geconfigureerd, of **wij konden de uitvoer niet lezen**. Alleen de
derde is een fout van ons, en juist die leest als "bij die klant staat
niets ingesteld".

Geen theoretisch geval: `occ config:list system` eindigt met exitcode 0
en drukt bij sommige Nextcloud-versies een deprecation-regel vóór de
JSON. Geldig voor de shell, onleesbaar voor ons.

Regel 110–111 heeft dezelfde vorm: de fout van `ParseStatus` wordt
weggegooid.

## Scope
- [ ] 1.1 Beide plekken loggen op WARN wat er misging, met genoeg
  context om het terug te vinden (welk commando, welke fout, hoeveel
  bytes). Volg `internal/scanner/amass.go` (`amass.malformed_line`).
- [ ] 1.2 Onleesbare uitvoer levert een bevinding met een reden die
  zegt dat de scanner het niet kon lezen — niet stilte. Het mechanisme
  bestaat al: `internal/assessor/reason.go` kent
  `ReasonSubjectScanner`, en zo'n reden rendert als
  operatorwaarschuwing in plaats van als antwoord over het doel.
  Voeg een code toe in de stijl van `scanner_no_ipv6`, bijvoorbeeld
  `scanner_unreadable_output`.
- [ ] 1.3 De eucsf-regel onderscheidt dan "niets gevonden" van "niet
  te lezen". Splits de verdict-tekst; de score blijft onbekend.
- [ ] 1.4 Tests: geldige JSON (ongewijzigd gedrag), JSON met een regel
  ervoor, afgekapte JSON, lege uitvoer. Controleer er één mét de
  reparatie eruit en zeg in je rapport dat je dat deed.

## Out of scope
De andere inspecteurs, en de `url.Parse`-gevallen op regel 444/487 —
die vallen terug op een lege waarde, en dat is daar juist.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen. Budget is $6.
