# Habitat run — bewijs onder de gosec-onderdrukking van linkifyVerdict

`golangci-lint` meldt op `internal/ui/linkify.go:48` één ding:

    G203: The used method does not auto-escape HTML ... (gosec)

Dat is inherent aan elke linkifier: hij geeft `template.HTML` terug.
De code is zorgvuldig — hij eist schema `http`/`https` én een host, en
escapet zowel de href als de linktekst — maar dat staat nu alleen in
een comment. Ik heb het met de hand nagemeten en het klopt:

    in : see https://evil.example/"onmouseover="alert(1)
    uit: <a href="https://evil.example/&#34;onmouseover=&#34;alert(1)" ...>

    in : see javascript:alert(1)
    uit: see javascript:alert(1)          (niet gelinkt)

Ik ga die melding in `.golangci.yml` onderdrukken. Een onderdrukking
zonder test eronder is een belofte: de volgende lezer stopt met kijken.
Deze run levert het bewijs.

## Scope — ONLY this
- [ ] 1.1 Eén test in `internal/ui/linkify_test.go` die de vier
  uitbraakpogingen afdekt, elk met een eigen verwachting:
  - `https://evil.example/"onmouseover="alert(1)` — het aanhalingsteken
    staat geëscaped in de href, dus er ontstaat geen extra attribuut;
  - `https://evil.example/'><script>alert(1)</script>` — geen ruwe
    `<script>` in de uitvoer;
  - `javascript:alert(1)` — komt niet in een `<a href>` terecht;
  - `https://evil.example/</a><img src=x onerror=alert(1)>` — geen ruwe
    `<img` in de uitvoer.
  Assert op de afwezigheid van de gevaarlijke vorm, niet op de exacte
  uitvoerstring — anders breekt de test op elke cosmetische wijziging
  en wordt hij weggeklikt.
- [ ] 1.2 Controleer de test één keer mét de escaping eruit (vervang
  `template.HTMLEscapeString(raw)` in de href tijdelijk door `raw`) en
  bevestig in je rapport dat hij dan faalt. Zet de code daarna terug.

## Niet doen
`.golangci.yml` aanpassen — dat doe ik, CI-config valt buiten de
builder-rol.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen. Budget is $4.
