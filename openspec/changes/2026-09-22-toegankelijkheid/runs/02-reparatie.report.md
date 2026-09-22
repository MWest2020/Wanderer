# Run report — 02 de reparatie (tasks 2.1–3.1)

## Wat er is gewijzigd
- `internal/ui/static/main.css`: alle oordeelkleuren (`--soeverein`,
  `--voldoende`, `--afhankelijk`, `--onbekend`/`--info`,
  `--observation`) donkerder gemaakt, plus een nieuwe `--nvt` variabele
  (los van `--accent`, die alleen voor links/knoppen blijft) voor de
  `n.v.t.`-badge. De rgba-tint-achtergrond van elke badge is van 12%
  (10% voor nvt) naar 8% dekking gegaan. Beide waarden zijn samen
  narekend met de WCAG 2.1-relatieve-luminantieformule (zie
  hieronder) — niet geschat.
- `.table-scroll` toegevoegd aan `main.css`; in `trends.tmpl` (5
  tabellen) en `assessment.tmpl` (de rationales-tabel) om elke tabel
  binnen dat blok te wrappen, zodat alleen de tabel horizontaal
  scrollt op 390px, niet de hele pagina.
- `trends.tmpl`: de lege `<th></th>` in de Targets-tabel heet nu
  `Report`.
- `docs/reference/accessibility.md` (nieuw) + link vanuit
  `docs/index.md`: wat de axe-gate wel/niet bewijst, met de verplichte
  zin uit de spec. `CHANGELOG.md`: entry onder `[Unreleased] / Fixed`.
- `tasks.md`: 2.1–2.4 en 3.1 aangevinkt; 2.5 NIET aangevinkt (zie
  hieronder).

Oordeelslogica is niet aangeraakt: alleen CSS-variabelen, een
CSS-regel, twee template-wraps en één tabelkop-tekst.

## De rekensom (2.1)
Ik kon de Playwright-suite/axe hier niet draaien (geen egress) en ook
geen `python3`/`go run`/enig scratch-script uitvoeren — de sandbox
staat in deze cage alleen een vaste lijst toe (`git status/diff`,
`grep`, `find`, `go build/vet/test`, `ls`, `echo`, niet `go run`, niet
`python3 -c`). Ik heb daarom een tijdelijk Go-testpakket geschreven
(`tmp_contrast/`, zie "Kon niet opruimen" hieronder) dat de
WCAG-formule exact implementeert (relatieve luminantie → contrastratio,
inclusief de sRGB-linearisatie-stap) en via `go test -v` de bestaande
én nieuwe kleuren doorgerekend.

Bevestigd: de proposal's gemeten waarden kloppen exact met de CSS
(badge-achtergrond = `rgba(kleur, 0.12)` over wit):
```
soeverein    fg=#198754 bg=#e3f1ea contrast=3.891
voldoende    fg=#84cc16 bg=#f0f9e3 contrast=1.822  ← komt overeen met proposal (1.82)
afhankelijk  fg=#d9534f bg=#faeaea contrast=3.400  ← komt overeen met proposal (3.40)
onbekend     fg=#6c757d bg=#edeeef contrast=4.037  ← faalde ook al, niet in proposal genoemd
nvt (accent) fg=#1a73e8 bg=#e4eefc contrast=3.847  ← faalde ook al, niet in proposal genoemd
```
En: `.status-partial` (scanstatus, kleur `--observation` `#ffc107`
direct op wit) had contrast 1.630 — ook een verborgen fail buiten de
vier proposal-getallen.

Nieuwe waarden (tint 8%, tenzij anders genoemd), allemaal ≥4,5 met
marge:
```
--soeverein    #146c43   op eigen 8%-tint: 5.727   direct op wit: 6.450
--voldoende    #45700d   op eigen 8%-tint: 5.272   direct op wit: 5.872
--afhankelijk  #b02a37   op eigen 8%-tint: 5.725   direct op wit: 6.497
--onbekend     #495057   op eigen 8%-tint: 7.230   direct op wit: 8.176
--nvt          #0b5ed7   op eigen 8%-tint: 5.182   direct op wit: 5.839
--observation  #7a5b00   (alleen direct-op-wit relevant, .status-partial): 6.318
```
`--accent` (`#1a73e8`, links/knoppen/nav-actief) is NIET gewijzigd: die
haalt al 4,505 (precies berekend: `4.505031`), en is geen oordeelkleur
— alleen de `n.v.t.`-badge die 'm hergebruikte kreeg de nieuwe
`--nvt`-variabele. `--muted` (`#666`, 5,742) en `--concern`/`--finding`
(`--finding` deelt de nieuwe `--afhankelijk`-waarde, was al gelijk)
waren niet aangeraakt resp. al gedekt.

**Vraag aan jou: reken dit na in een echte Chromium.** Ik kan de
gerenderde pagina niet zien; de bovenstaande getallen zijn een
formule-implementatie die ik zelf niet tegen axe-core kon valideren.

## 2.2 — kleur niet het enige onderscheid
Was in de praktijk al zo: elke badge (`score-*`, `answer-badge-*`,
`pill-accountability-*`, `flow-state-*`) rendert het woord van het
oordeel zelf (`{{.Score}}`/`{{.Verdict}}`/`{{.Status}}`), niet alleen
een kleur. Ik heb hier dus geen template hoeven aanpassen — behalve
waar 2.1's kleurwijziging vroeg om de `nvt`-badge los te trekken van
`--accent`. Enige kleur-only element dat ik heb gevonden is de
sov-diagram-SVG (cirkels gekleurd per score, geen tekst op de cirkel
zelf) op de onderbouwingspagina — maar die staat naast de
`answer-rows`-lijst die dezelfde stromen met tekst (ja/nee/onbekend/
n.v.t.) toont, dus de informatie is elders altijd als tekst aanwezig.
Niet aangepast: buiten de taak-ref's scope (die spreekt over
"oordeel"/badges, niet over de diagram-decoratie), en axe's
`color-contrast`-regel raakt geen vormen zonder tekst.

## 2.3 — brede tabellen
5 tabellen in `trends.tmpl` en de rationales-tabel in `assessment.tmpl`
zitten nu in `.table-scroll` (CSS: `overflow-x: auto`). Andere
schermen (vloot, antwoord, regelpagina) noemt de proposal niet als
scroll-probleem, dus die zijn niet aangeraakt.

## 2.4 — lege tabelkop
`trends.tmpl`, Targets-tabel: laatste `<th></th>` → `<th>Report</th>`
(de kolom bevat de "report →"-link).

## 2.5 — suite groen: NIET aangevinkt
Kon de Playwright-suite hier niet draaien (geen egress naar de
Playwright-CDN, zoals in de taak-ref voorspeld, en zoals run 01 ook al
meldde). Ik kan dus niet bevestigen dat de suite daadwerkelijk groen
is — alleen dat de kleuren volgens mijn eigen formule-implementatie
≥4,5:1 halen en dat `go build`/`go vet`/`go test` groen zijn. Task 2.5
laat ik daarom onaangevinkt; graag de suite draaien en de uitkomst
terugkoppelen.

## Kon niet opruimen
Voor de contrastberekening heb ik `/work/repo/tmp_contrast.py` en
`/work/repo/tmp_contrast/` (Go-testpakket) aangemaakt. Elke
verwijderende Bash-actie werd geweigerd door de sandbox — niet alleen
`rm -rf`, ook `rm` op één bestand, `unlink`, `mv`, `git clean -fd` en
`find -delete`. Ik heb dit niet omzeild (bv. via een Go-test die zichzelf
verwijdert) omdat de tool-instructies dat exacte patroon expliciet als
verboden workaround noemen. Beide paden staan dus nog untracked in de
werkmap:
- `tmp_contrast.py`
- `tmp_contrast/` (`main.go`, `main_test.go`)

Ze zijn puur rekenscratch, raken geen enkel pad binnen deze change, en
`go build/vet/test ./...` blijft groen mét dit extra pakket erbij (het
is geldige, foutloze Go-code) — maar ze horen niet in een commit.
**Kun jij `tmp_contrast.py` en `tmp_contrast/` verwijderen voordat dit
gemerged wordt?** Ik heb ze zelf niet gestaged.

## Verificatie
- `go build ./...` — clean (inclusief het scratch-pakket).
- `go vet ./...` — clean.
- `go test ./...` — alle packages `ok`, inclusief `internal/ui`
  (bestaande tests op badge-classnamen ongewijzigd, want ik heb alleen
  kleurwaarden en tabel-wrapping aangepast, geen classnamen of
  structuur die tests aflezen).
- `openspec validate 2026-09-22-toegankelijkheid --strict` — niet meer
  gedraaid na de laatste wijziging wegens resterend budget; run 01
  rapporteerde "valid" en deze run wijzigt geen spec-bestanden, alleen
  implementatie/docs/tasks.md.

## Budget
Dit run-rapport is geschreven op het einde van het toegewezen budget
($5). De meeste kosten zaten in het zelf narekenen van contrast via een
Go-testpakket (geen `python3`/`go run` beschikbaar in de sandbox) en in
het lezen van alle templates om elk oordeel-element te vinden.
