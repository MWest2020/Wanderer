# Habitat run 02 — de importer (taken 1.3, 2.1, 2.2)

Contract: `openspec/changes/2026-09-19-propose-internetnl-standards/`
(proposal.md, design.md inclusief "Design gate outcome",
specs/scanner/spec.md, NETNL-CONTRACT.md).

**Lees eerst design.md "Design gate outcome".** Het contract is op
zeven punten gecorrigeerd tegen echte metingen. Wat daar staat gaat
vóór wat elders in de change staat.

## Beslissingen die al genomen zijn — niet opnieuw uitvinden

1. **Geen nieuwe scan-kind-kolom.** `models.Finding` heeft al
   `SourceModus` (`perimeter` / `inventory` / `egress` / `drift`, met
   leeg = perimeter). Voeg `SourceModusImport = "import"` toe en
   accepteer hem in `Valid()`. Dat is dezelfde ingreep als waarmee
   `accountability` aan `DimensionHint` werd toegevoegd. Bouw géén
   `Scan.Kind` met migratie; er is één geval, dus er is geen
   abstractie nodig.
2. **Het patroon voor een importeur staat er al:**
   `internal/scanner/amass.go` (`LoadAmassFQDNs`) — misvormde regels
   naar WARN en overslaan, nooit de hele invoer weggooien. Volg dat.
3. **De rapport-URL is ondoorzichtig.** Nooit zelf opbouwen, nooit
   `internet.nl` erin verwachten: op een zelf-gehoste instantie wijst
   hij naar die instantie.

## Scope — ONLY these tasks
- [ ] 1.3 De fixtures overnemen in Wanderer's testdata, byte-voor-byte
  gelijk aan de kant van netnl. Ze staan al klaar in deze change onder
  `fixtures/`: `findings-v1-web-20260922.json` en
  `findings-v1-mail-20260922.json` — dat is wat de importeur leest.
  (De twee `batch-v2-*`-bestanden ernaast zijn de ruwe API-antwoorden
  waar ze uit gemaakt zijn; die leest de importeur niet.)

  Beide bevatten 38 testuitslagen, allemaal met een categorie, geen
  enkele `null`. Web: `web_https` 22, `web_ipv6` 5, `web_appsecpriv` 5,
  `web_rpki` 4, `web_dnssec` 2. Mail: `mail_starttls` 19, `mail_rpki` 6,
  `mail_auth` 5, `mail_dnssec` 4, `mail_ipv6` 4. Wijkt jouw import daar
  vanaf, dan is de import fout, niet de fixture.
- [ ] 2.1 `wanderer import internetnl <bestand>`: inlezen,
  domein-matchen op bestaande targets, wegschrijven onder een scan met
  `SourceModus: import`. Onbekend domein → WARN + overslaan.
  Misvormde entry → WARN + overslaan. Schemaversie die je niet kent →
  afbreken en de verwachte versie noemen. Twee keer dezelfde file
  importeren verandert niets (bestandshash + request-id).
- [ ] 2.2 Store: import-scans staan naast perimeter-scans; niets
  overschrijft elkaar. De assessor leest per soort de nieuwste.

## Valkuilen uit de gemeten werkelijkheid
- Een domein kan in het bestand staan met status `error` en een lege
  `results`. Dat is GEEN misvormde entry en mag niet worden
  overgeslagen — het betekent "gemeten, ging mis", niet "niet
  aangeleverd".
- `detail` is altijd `null`. Bouw er niets op.
- `status` kent zes waarden, inclusief `error`. Een importeur die er
  vijf kent, gooit stilletjes weg.

## Out of scope
De assessor-regels (run 03) en de UI (run 04). Voeg nog geen
`standards`-dimensie toe.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen, en
`openspec validate 2026-09-19-propose-internetnl-standards --strict`
groen. Budget is $8.
