# Habitat run 07b — tijdzone in datum-verdicts (task 7.5)

Gevonden door `go test ./...` buiten de kooi te draaien, in
Europe/Amsterdam. In UTC slaagt de test, daarbuiten niet — dus de kooi
kan dit niet zelf vangen.

```
--- FAIL: TestDomainExpiry/within_30_days_scores_afhankelijk_and_names_the_date
    verdict = "domain registration expires 2026-10-01 — renewal due within
    30 days or already past", want it to name the date
```

## Scope — ONLY this task
- [ ] 7.5 Eén zone voor datums in verdicts: UTC, want RDAP (`events`) en
  RFC 9116 (`Expires`) publiceren in UTC. Zowel
  `wand.operationeel.domain_expiry` als `wand.accountability.securitytxt`
  formatteren de datum in UTC, en de tests rekenen hun verwachte datum in
  dezelfde zone uit — een test mag niet van de zone van de machine
  afhangen. Voeg één test toe die met een vaste, niet-UTC zone
  (`time.FixedZone`) aantoont dat het oordeel niet verschuift.

Done = het vinkje hierboven, 7.5 afgevinkt in tasks.md.

## Out of scope
Alles anders: geen UI, geen probes, geen nieuwe regels.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen (offline uit
vendor/); `openspec validate 2026-09-19-propose-accountability-dimension
--strict` groen. Budget is $3.
