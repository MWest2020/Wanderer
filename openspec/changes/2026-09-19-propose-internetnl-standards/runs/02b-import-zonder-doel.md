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
- [x] 1.1 Maak de idempotentiesleutel per **(bestandshash, domein)** in
  plaats van per bestand. Een domein dat al uit dít bestand is
  ingelezen, wordt overgeslagen; een domein dat de vorige keer werd
  overgeslagen omdat het doel ontbrak, komt er alsnog in zodra het
  doel bestaat. Dat dekt beide gevallen zonder uitzonderingen. —
  gedaan 2026-09-22: `netnl_imports` heeft nu de samengestelde sleutel
  `(file_hash, domain)`; `Store.NetnlImportRecorded`/`RecordNetnlImport`
  nemen beide een domein, en `importNetnlDomains` checkt per domein in
  plaats van één keer voor het hele bestand.
- [x] 1.2 De slotmelding vertelt wat er gebeurde, ook bij nul:
  hoeveel domeinen ingelezen, hoeveel overgeslagen omdat het doel
  onbekend is, en hoeveel overgeslagen omdat ze al uit dit bestand
  kwamen. "nothing to do" mag alleen als er ook werkelijk niets te
  doen was. — gedaan 2026-09-22: standaardmelding toont alle drie de
  tellers; "already imported (file unchanged), nothing to do" komt
  alleen nog terug wanneer elk domein in het bestand al eerder uit
  precies dit bestand kwam (0 nieuw, 0 onbekend).
- [x] 1.3 Tests: (a) importeren zonder doel, doel aanmaken, opnieuw
  importeren → de bevindingen staan er; (b) twee keer achter elkaar
  met bestaand doel → geen dubbele bevindingen; (c) een bestand met
  twee domeinen waarvan er één bekend is → het bekende komt binnen,
  het andere pas na het aanmaken ervan. Controleer (a) één keer mét
  de reparatie eruit en zeg in je rapport dat je dat deed. — gedaan
  2026-09-22: `TestRunImportInternetnl_TargetAddedAfterUnknownDomainImportsOnRetry`
  (a), bestaande `TestRunImportInternetnl_ReimportSameFileIsNoOp` (b),
  `TestRunImportInternetnl_MixedKnownAndUnknownDomains` (c), plus
  `TestNetnlImportIdempotency` uitgebreid met het cross-domain geval op
  storeniveau. (a) is met de reparatie tijdelijk teruggedraaid (via
  `git checkout` op de vier productiebestanden, test eraan gelaten) —
  faalde toen op `LatestScanByModus: store: not found`, precies de
  gemeten bug; daarna de reparatie teruggezet en alles weer groen.
- [x] 1.4 Migratie: de bestaande tabel `netnl_imports` heeft alleen
  `file_hash`. Voeg het domein toe aan de sleutel. Bestaande rijen
  (er zijn er in de praktijk nog geen buiten testdatabases) mogen
  vervallen; kies de eenvoudigste migratie die klopt en zeg welke. —
  gedaan 2026-09-22: migratie 12 (`netnl_imports_per_domain`) doet
  `DROP TABLE` + `CREATE TABLE` met `PRIMARY KEY (file_hash, domain)`.
  Geen data-migratie, zoals de opdracht toestond.

## Terzijde, niet in deze run
De comment bij migratie 11 zegt dat het request-id "niet op de draad
bestaat". Dat klopt niet — het staat in het API-antwoord, het werd
alleen niet geëxporteerd. Dat wordt aan de netnl-kant gerepareerd
(run 04 daar); als de kop er is, kun je hem hier gaan vastleggen. Pas
in deze run alleen de comment aan zodat hij niet iets onwaars beweert.
— gedaan 2026-09-22: de comment bij migratie 11 zegt nu dat het
request-id wél op de draad zit (netnl's batch-antwoord, top-level
`request_id`), maar nog niet in de netnl-findings/v1-export terechtkomt.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen. Budget is $6.
