# Habitat run 04 — UI, documentatie, afronden (taken 4.1–4.4)

Contract: `openspec/changes/2026-09-19-propose-internetnl-standards/`.
Lees design.md "Design gate outcome" en "Verwachte uitkomst voor
westerweel.work" — daar staat wat er gemeten is.

## Scope — ONLY these tasks

- [ ] 4.1 UI. De standards-dimensie moet via de bestaande
  regelweergave renderen; bouw geen aparte pagina. Drie dingen
  nakijken en repareren waar nodig:
  - de rapport-URL uit het bewijs is aanklikbaar (hij is
    ondoorzichtig — toon hem, bouw hem nooit zelf op);
  - een doel zonder import toont "niet gemeten", niet een leeg of
    goed ogend vakje;
  - de dimensie verschijnt in de vlootscore zonder de bestaande
    telling te verstoren. Let op: als "niet gemeten" als
    onbeantwoorde vraag meetelt, zakt elk domein zonder
    internet.nl-import in de vlootscore. Kies bewust en zeg in je
    rapport wat je koos en waarom.
- [ ] 4.2 Documentatie:
  - `docs/reference/assessor.md` en `docs/reference/findings.md`: de
    nieuwe dimensie, de `internetnl.*`-bevindingen, de
    `SourceModusImport`.
  - Nieuwe how-to "Internet.nl-resultaten uit CI voeden". Gebruik de
    echte commando's; ze zijn allebei gedraaid:

        internetnl submit example.nl --type web --no-poll
        internetnl results <request-id> --format findings \
          --findings-out findings.json
        wanderer import internetnl --db wanderer.db findings.json

    Noem erbij dat de export niet-nul eindigt en niets schrijft als de
    batch nog loopt — dát is wat een CI-stap moet laten falen.
  - `docs/explanation/`: de permanente niet-doelen uit proposal.md
    ("Stripped from Wanderer"), met de reden: wij bouwen geen eigen
    DNSSEC/SPF/DMARC/STARTTLS/DANE/RPKI-probe.
- [ ] 4.3 CHANGELOG onder `[Unreleased]`.
- [ ] 4.4 Een korte opvolgnotitie voor v2 (de facade die hetzelfde
  bestand via een webhook aflevert). Eén bestand in
  `openspec/changes/` met proposal-kop en waarom het wacht: v1 moet
  eerst een paar weken in CI gedraaid hebben. Géén specs, géén taken —
  het is een stub, geen change.

## Niet doen
De oordeelslogica aanpassen (dat is run 03), of het contract
aanpassen.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen, `openspec validate
2026-09-19-propose-internetnl-standards --strict` groen. Playwright kun
je niet draaien; zeg dat, ik meet na. Budget is $7.
