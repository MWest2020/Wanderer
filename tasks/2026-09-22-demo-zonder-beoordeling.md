# Habitat run — een scan zonder beoordeling maakt de demopagina leeg

Gevonden door het live te doen, 2026-09-22: ik startte een scan van
westerweel.work via `POST /scans` op de draaiende instantie. De scan
slaagde. Daarna toonde https://wanderer.westerweel.work/demo:

    westerweel.work
    Nog geen meting voor dit domein.

De pagina ging van 3.216 naar 1.087 bytes. Pas ná een losse
`POST /scans/{id}/assessments` kwam hij terug.

## Waarom
`latestCompletedScan` (`internal/ui/demo.go:117`) kiest de nieuwste
scan met status `complete` of `partial`. Het kijkt niet of die scan een
beoordeling heeft. `POST /scans` beoordeelt niet — dat is een aparte
route. Elke scan die langs die weg binnenkomt, verdringt dus de laatste
scan die wél een verhaal had, en de publieke pagina zegt "nog geen
meting" terwijl er twintig metingen zijn.

Dat is geen randgeval: het is de enige pagina zonder inlog, en de weg
ernaartoe is een gewone API-aanroep.

## Scope — ONLY these tasks
- [x] 1.1 De demopagina kiest de nieuwste scan **die een beoordeling
  heeft**. Een scan zonder beoordeling slaat hij over in plaats van
  erop te stranden. Blijft er dan niets over, dan mag "nog geen
  meting" — dat klopt dan ook.
- [x] 1.2 Een test die dit vastlegt: twee scans voor hetzelfde domein,
  de nieuwste zonder beoordeling, en de pagina toont de oudere mét.
  Controleer hem één keer mét de reparatie eruit en zeg in je rapport
  dat je dat deed.
- [x] 1.3 Kijk of `/ui/scans/{id}/answer` en de vlootpagina dezelfde
  aanname maken. Zo ja, repareer ze mee; zo nee, zeg in je rapport
  waarom ze er geen last van hebben.

## Uitdrukkelijk NIET
`POST /scans` laten beoordelen. Scannen en beoordelen zijn met opzet
gescheiden; de pagina hoort tegen een half afgemaakte keten te kunnen.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen. Budget is $5.
