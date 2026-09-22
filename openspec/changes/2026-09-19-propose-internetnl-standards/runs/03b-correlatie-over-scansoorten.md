# Habitat run 03b — de assessor kijkt maar naar één scan tegelijk

Run 03 is binnen: de zes regels bestaan, de tests zijn grondig, de
gate is groen (ik heb golangci-lint zelf gedraaid — nul meldingen — en
het achtergebleven `zzdebug_test.go` verwijderd). Eén ding werkt niet,
en het is precies wat taak 2.2 en design.md "Import semantics" eisen.

## Gemeten, op een verse database

Eén doel, twee imports (web en mail) plus een gewone perimeter-scan.
Daarna per scan beoordeeld:

| regel   | perimeter-scan | web-import              | mail-import                       |
| ------- | -------------- | ----------------------- | --------------------------------- |
| ipv6    | niet gemeten   | **soeverein** (5 tests) | **voldoende** (4 tests, 1 failed) |
| dnssec  | niet gemeten   | soeverein (2)           | soeverein (4)                     |
| rpki    | niet gemeten   | soeverein (4)           | soeverein (6)                     |
| tls_cfg | niet gemeten   | voldoende (20)          | niet gemeten                      |

Drie problemen in één tabel:

1. **De gewone scan toont niets.** Wie een perimeter-scan opent — de
   normale route — ziet zes keer "niet gemeten", ook al staan de
   importbevindingen in dezelfde database, op hetzelfde doel. De hele
   dimensie is onzichtbaar in de normale flow.
2. **Web en mail worden nooit samen gezien.** `dnssec`, `ipv6` en
   `rpki` hebben tests aan beide kanten. Elke beoordeling scoort op de
   helft.
3. **Daardoor spreekt dezelfde regel zichzelf tegen.** `ipv6` is
   soeverein op de web-import en voldoende op de mail-import. Wie het
   webrapport opent leest "alle IPv6-tests geslaagd", terwijl
   `mail_ipv6_mx_reach` faalt. Dat is een zelfverzekerd groen op de
   helft van het bewijs — precies wat deze change wil voorkomen.

design.md zegt het al: "The assessor correlates a target's newest
standards findings across scan kinds; a perimeter scan never erases
imported findings and vice versa." Taak 2.2 stond afgevinkt; die vink
is eraf.

## Scope — ONLY this
- [ ] 2.2 De standards-regels lezen de nieuwste `internetnl.*`-
  bevindingen **van het doel**, niet van de scan die toevallig
  beoordeeld wordt. Web en mail komen uit aparte importbestanden en
  dus aparte scans: beide horen mee te tellen, elk zijn nieuwste.
  Een tweede import van hetzelfde soort vervangt de vorige; een
  perimeter-scan raakt ze niet aan.
- [ ] 2.3 Tests die de tabel hierboven vastleggen: na twee imports
  ziet `ipv6` negen tests en `rpki` er tien, ongeacht welke scan je
  beoordeelt — ook een perimeter-scan. En: een verse mail-import
  vervangt de oude mail-bevindingen zonder de web-kant te raken.
  Controleer er één mét de reparatie eruit en zeg dat in je rapport.

## Let op
- Dit raakt de `standards`-regels, niet de andere dimensies. Een
  perimeter-regel moet de findings van zijn eigen scan blijven lezen.
- De staleness-regel (`standards.max_age`) geldt per bevinding, dus
  na deze wijziging per importsoort. Een oude web-import naast een
  verse mail-import hoort web als verouderd te melden en mail niet.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen (lint kun jij niet draaien — zeg dat,
ik meet na), en `openspec validate
2026-09-19-propose-internetnl-standards --strict` groen. Budget is $8.
