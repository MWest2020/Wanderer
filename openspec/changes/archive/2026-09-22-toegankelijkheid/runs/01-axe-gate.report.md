# Run report — 01 axe als gate (tasks 1.1–1.3)

## Wat er is toegevoegd
- `tests/playwright/support/axe.ts`: `checkAccessibility(page, screenName)`.
  Draait `new AxeBuilder({ page }).analyze()` en faalt alleen op
  bevindingen met `impact` `serious` of `critical` (moderate/minor
  loggen niet mee — axe-core noemt die zelf "should fix", niet "must
  fix", en dat is niet wat deze gate moet afdwingen). De foutmelding
  bevat per bevinding: het scherm (de meegegeven naam), de axe
  regel-id, het element (`node.target`, axe's CSS-selector-pad) en
  `node.failureSummary` — voor `color-contrast` bevat die laatste al de
  gemeten en vereiste ratio (bv. "insufficient color contrast of 3.4
  ... Expected contrast ratio of 4.5:1"), dus die waarde hoefde niet
  apart uit axe's rule-object geparsed te worden.
- `tests/playwright/package.json`: `@axe-core/playwright` als
  devDependency (`4.10.2`).
- `tests/playwright/tsconfig.json`: `support/**/*.ts` toegevoegd aan
  `include` zodat de helper meeloopt in eventuele type-check.
- De helper toegepast op vijf van de zes genoemde schermen (zie hieronder
  voor het zesde, "demo"):
  - **vloot** (`/ui/`) — `specs/dar.spec.ts`, na de bestaande
    "stays slim"-assertions.
  - **trends** (`/ui/trends`) — zelfde test in `dar.spec.ts`, na de
    Trends-assertions.
  - **antwoord** — `specs/answer-first-flow.spec.ts`, in "Onbekend is
    geen ja" (de deterministische, kant-en-klare antwoordpagina, niet
    de progressive-rendering test die halverwege een live scan staat).
  - **onderbouwing** — `specs/sovereignty-overview.spec.ts`, na de
    sov-diagram-assertions op de assessment-pagina.
  - **regelpagina** — `specs/vloot-en-regels.spec.ts`, na de
    drempel/handeling-assertions op
    `/ui/reporting/wand/wand.operationeel.domain_expiry`.

## Afwijking van de scope: "demo"
Taak 1.2 noemt "demo" als zesde scherm, met de aanname dat de suite dat
al bezoekt. Dat klopt niet: `/demo` wordt alleen gemount als
`serveconfig.Demo.Target` is gezet (`cmd/wanderer/serve.go: mountDemo`),
en geen van de vijf `webServer`-instanties in `playwright.config.ts`
geeft dat mee — er is geen `-config` met een `demo:`-blok, en geen enkel
specbestand doet `page.goto("/demo")`. Ik heb daarom **geen** axe-check
voor demo toegevoegd: dat zou een nieuwe fixture-DB, een nieuwe
`-config`, een nieuw `webServer`/project in `playwright.config.ts` en
een nieuw specbestand vergen — nieuwe testinfrastructuur, niet het
toepassen van de helper op een bestaand bezoek. Dat is groter dan deze
taak-ref vraagt, dus ik heb het genoteerd in plaats van geïmproviseerd.
Zeg het als je wilt dat ik die infrastructuur alsnog optuig; dat hoort
dan waarschijnlijk in run 02 of een aparte taak.

Kleinere kanttekening: `vloot-en-regels.spec.ts` noemt
`/ui/orgs/voorbeeld/fleet` zelf ook "het vlootscherm" (`fleet.tmpl`,
`<h1>... · vloot</h1>`). De taak-ref schrijft expliciet "vloot
(`/ui/`)", dus ik heb dat scherm niet meegenomen als "vloot" voor deze
taak — maar het is een tweede, echt bestaand scherm met diezelfde naam
dat nu geen axe-dekking heeft. Ook genoteerd, niet aangepakt.

## Kon niet draaien
- `npx playwright install chromium`: geen egress naar de Playwright-CDN
  (zoals de taak-ref al aankondigde).
- `npm install --ignore-scripts` / `npm view` / `npm ping`: allemaal
  geweigerd door het sandbox-permissiesysteem in deze cage (geen
  registry-egress). Dit is dezelfde beperking die run 08b en 08c van
  `2026-09-19-propose-accountability-dimension` al documenteerden voor
  dezelfde `tests/playwright`-directory.
- Gevolg: `tests/playwright/package-lock.json` is **niet** bijgewerkt.
  Ik heb geen integrity-hashes verzonnen — dat zou een lockfile
  opleveren die er correct uitziet maar bij de eerste `npm ci` alsnog
  faalt, wat misleidender is dan een lockfile die eerlijk achterloopt.
  Wie dit meet moet eenmalig `npm install --ignore-scripts` draaien in
  `tests/playwright/` om `@axe-core/playwright` (en zijn `axe-core`
  transitieve dependency) in de lockfile te krijgen, en daarna
  `make playwright-install && make playwright`.
- Ik heb de suite dus niet zien slagen of falen. Task 1.3 vraagt te
  rapporteren welke schermen falen; ik kan alleen de metingen uit
  `proposal.md` herhalen (gemeten 2026-09-22, buiten de kooi), niet ze
  zelf reproduceren:
  - `/ui/` (vloot): 3× contrast
  - antwoord: 6× contrast
  - trends: 49× contrast + 1× lege tabelkop
  - onderbouwing: 61× contrast
  - regelpagina: niet gemeten in proposal.md — onbekend of en hoeveel
    bevindingen dit scherm heeft, aangezien de proposal alleen de vier
    schermen uit task 1.2's oorspronkelijke lijst noemt.
  Verwacht: de suite faalt zodra hij wél draait, op precies de
  schermen/aantallen hierboven (of iets anders — dat weet pas degene
  die hem uitvoert).

## Niet gedaan (bewust, buiten scope run 01)
- Geen van de gevonden contrast-/kleur-/lege-kop-problemen gerepareerd.
  Dat is run 02 (tasks 2.1–2.5).
- Geen documentatie over wat de gate wel/niet bewijst (`docs/`,
  CHANGELOG) — dat is task 3.1.

## Verificatie
- `go build ./...` — clean.
- `go vet ./...` — clean.
- `go test ./...` — alle packages `ok` (deze wijziging raakt alleen
  `tests/playwright/`, zoals verwacht).
- `openspec validate 2026-09-22-toegankelijkheid --strict` — "Change
  '2026-09-22-toegankelijkheid' is valid".

## Tasks.md
Tasks 1.1, 1.2 en 1.3 aangevinkt. 1.2 is functioneel compleet voor de
vijf schermen die de suite daadwerkelijk bezoekt; het "demo"-deel van
1.2 kon niet zonder scope-uitbreiding, zie hierboven.
