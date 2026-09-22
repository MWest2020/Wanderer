# Habitat run 04b — Nederlands oordeel MET het waargenomen feit

Run 04 (20260922-104432-15557) liep uit zijn budget en liet vijf rode
tests achter. Zijn branch is NIET samengevoegd. Bouw voort op `main`.

## Wat run 04 goed deed (overnemen uit die branch)
- `tests/playwright/specs/answer-first-flow.spec.ts` in lijn met de
  vlootpagina (`form.scan-form`, vlootscore, domeinenlijst) — die
  aanpassing is goed en verzwakt niets. Neem hem over.
- De zeven Nederlandse zinnen in `accountability_nl.yaml`.
- `dutchFlowVerdict` in `internal/ui/flows.go`.

Die branch heet
`origin/habitat/builder/2026-09-22-drie-lagen-ciso-20260922-104432-15557`.

## Wat er mis is
Run 04 verving het oordeel volledig. Waar eerst stond

    Hosting: apex IPs in CA, CA, CA, CA (outside EEA)

staat nu

    De hosting staat buiten de EER.

Het waargenomen feit — wélk land, wélke partij — is van de pagina
verdwenen; het staat alleen nog in de ingeklapte bewijsattributen. Dat
is precies wat `TestBuildFlowAnswers_MapsScoreToAnswerAndKeepsObservedFact`
bewaakte, en die test is rood. Vier andere tests vielen mee om
dezelfde reden. Repareer NIET door die tests te verzwakken of te
schrappen — zij hebben gelijk.

## Scope — ONLY these tasks
- [ ] 4.1 Het zichtbare oordeel is Nederlands **en** noemt het
  waargenomen feit. Bijvoorbeeld: "De hosting staat buiten de EER —
  apex-adressen in CA (Cloudflare)." De Nederlandse zin komt uit de
  tabel, het feit uit de rationale/bevinding. Hoe je ze samenvoegt is
  aan jou; kies de vorm die het minste dubbel werk oplevert.
  Let op: de Engelse rationale-zin bevat zelf ook woorden als "hosted
  at" en "outside EEA". Als je hem letterlijk aanplakt, staat er half
  Engels. Haal het feit eruit (landen, AS, partijnaam) in plaats van de
  hele zin over te nemen — of, als dat te veel afleiding is, gebruik de
  bevindingsattributen die al in `Evidence` zitten.
- [ ] 4.2 De bestaande tests groen krijgen door ze te laten passen bij
  de nieuwe zin (niet door de eis weg te halen): in `answer_test.go`,
  `fleet_score_test.go`, `flow_answers_test.go`, `flows_test.go`,
  `demo_test.go`.
- [ ] 4.3 Een test die een oordeel vangt dat níét uit de tabel komt.

## Out of scope
De onderbouwingspagina herindelen (run 05), nieuwe metingen.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` groen en
`openspec validate 2026-09-22-drie-lagen-ciso --strict` groen. Playwright
kun je niet draaien (geen egress) — zeg dat in je rapport, ik meet na.
Budget is $9. Werk zuinig: lees eerst de vijf falende tests, dan pas de
code.
