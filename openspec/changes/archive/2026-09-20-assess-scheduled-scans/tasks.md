# Tasks: assess-scheduled-scans

## 1. Scheduler — run 01
- [x] 1.1 `internal/scheduler`: na een geslaagde scan een `Assessment`
  maken met beide rule packs (wand + eucsf), net als
  `wanderer assess --framework both`, en opslaan via de store.
- [x] 1.2 Mislukt het beoordelen, dan blijft de scan staan en wordt de
  fout gelogd (`scan.assess_failed`), zonder de volgende tick te raken.
- [x] 1.3 `assess: false` per schedule zet het uit; ontbreekt het veld,
  dan staat het aan. Bestaande configuraties blijven geldig.
- [x] 1.4 Tests: schedule levert scan én assessment; `assess: false`
  levert alleen een scan; een falende beoordeling laat de scan staan.

## 2. Documentatie
- [x] 2.1 `docs/reference/` (de scheduling-pagina) en CHANGELOG.
