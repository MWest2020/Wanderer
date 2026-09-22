# Habitat run 02b — een import die niets importeerde, telt als gedaan

Run 02 is binnen en goed; dit is één gat dat pas zichtbaar werd door
het commando echt te draaien. Bouw voort op `main`.

## Wat er mis is — gemeten, niet geredeneerd

```
$ wanderer import internetnl --db nieuw.db findings-v1-web-20260922.json
WARN import.internetnl.unknown_domain domain=westerweel.work
wanderer: import: 0 target(s) imported, 1 domain(s) skipped (unknown target)

$ wanderer scan westerweel.work --db nieuw.db     # doel bestaat nu wel
$ wanderer import internetnl --db nieuw.db findings-v1-web-20260922.json
wanderer: import: ... already imported (file unchanged), nothing to do
```

In de database: `netnl_imports` heeft 1 rij, `findings` met
`source_modus='import'` heeft er 0. Het bestand staat als verwerkt
genoteerd terwijl er niets in kwam, en de enige uitweg is een rij uit
de database verwijderen.

Dit is precies de volgorde die een nieuwe gebruiker aanhoudt: eerst
het exportbestand proberen, de waarschuwing lezen, het domein
toevoegen, opnieuw proberen. Die tweede poging hoort te werken.

## Scope — ONLY this
- [ ] 1.1 Maak de idempotentiesleutel per **(bestandshash, domein)** in
  plaats van per bestand. Een domein dat al uit dít bestand is
  ingelezen, wordt overgeslagen; een domein dat de vorige keer werd
  overgeslagen omdat het doel ontbrak, komt er alsnog in zodra het
  doel bestaat. Dat dekt beide gevallen zonder uitzonderingen.
- [ ] 1.2 De slotmelding vertelt wat er gebeurde, ook bij nul:
  hoeveel domeinen ingelezen, hoeveel overgeslagen omdat het doel
  onbekend is, en hoeveel overgeslagen omdat ze al uit dit bestand
  kwamen. "nothing to do" mag alleen als er ook werkelijk niets te
  doen was.
- [ ] 1.3 Tests: (a) importeren zonder doel, doel aanmaken, opnieuw
  importeren → de bevindingen staan er; (b) twee keer achter elkaar
  met bestaand doel → geen dubbele bevindingen; (c) een bestand met
  twee domeinen waarvan er één bekend is → het bekende komt binnen,
  het andere pas na het aanmaken ervan. Controleer (a) één keer mét
  de reparatie eruit en zeg in je rapport dat je dat deed.
- [ ] 1.4 Migratie: de bestaande tabel `netnl_imports` heeft alleen
  `file_hash`. Voeg het domein toe aan de sleutel. Bestaande rijen
  (er zijn er in de praktijk nog geen buiten testdatabases) mogen
  vervallen; kies de eenvoudigste migratie die klopt en zeg welke.

## Terzijde, niet in deze run
De comment bij migratie 11 zegt dat het request-id "niet op de draad
bestaat". Dat klopt niet — het staat in het API-antwoord, het werd
alleen niet geëxporteerd. Dat wordt aan de netnl-kant gerepareerd
(run 04 daar); als de kop er is, kun je hem hier gaan vastleggen. Pas
in deze run alleen de comment aan zodat hij niet iets onwaars beweert.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen. Budget is $6.
