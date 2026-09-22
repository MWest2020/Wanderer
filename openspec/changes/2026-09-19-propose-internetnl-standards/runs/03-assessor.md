# Habitat run 03 — de standards-regels (taken 3.1, 3.2)

Contract: `openspec/changes/2026-09-19-propose-internetnl-standards/`
(specs/assessor/spec.md). **Lees eerst design.md "Design gate
outcome"** — het contract is op zeven punten gecorrigeerd tegen echte
metingen.

## De mapping van API-categorie naar regel

Gemeten op 2026-09-22 (beide fixtures staan in de change). De API
levert deze categorieën:

| API-categorie   | regel                          |
| --------------- | ------------------------------ |
| `web_dnssec`    | `wand.standards.dnssec`        |
| `mail_dnssec`   | `wand.standards.dnssec`        |
| `mail_auth`     | `wand.standards.mail_auth`     |
| `mail_starttls` | `wand.standards.starttls_dane` |
| `web_ipv6`      | `wand.standards.ipv6`          |
| `mail_ipv6`     | `wand.standards.ipv6`          |
| `web_rpki`      | `wand.standards.rpki`          |
| `mail_rpki`     | `wand.standards.rpki`          |
| `web_https`     | `wand.standards.tls_config`    |
| `web_appsecpriv`| GEEN regel — zie hieronder     |

Let op: de regel heet `starttls_dane`, de categorie heet
`mail_starttls`. Die naam staat al in de spec; hernoem hem niet, maar
zet de mapping expliciet in de code met een comment, anders zoekt de
volgende lezer een categorie die niet bestaat.

`web_appsecpriv` wordt **niet** gescoord. Daar zitten
`web_appsecpriv_securitytxt` (die Wanderer zelf meet, met de ruwe
Contact/Expires-velden als bewijs) en de header-subtests. Dat is de
eis "never double-score first-party ground" uit de spec. Zet een test
die faalt als er ooit een regel bijkomt die op `web_appsecpriv` scoort.

## Scope — ONLY these tasks
- [ ] 3.1 De `standards`-dimensie registreren en de zes regels
  bouwen. Alleen verdictmapping: alle relevante tests `passed` →
  soeverein; `warning` of gemengd → voldoende; substantieel `failed` →
  afhankelijk; `not_tested`, afwezig, of ouder dan `standards.max_age`
  → onbekend mét reden. Tabelgedreven tests inclusief de
  niet-gemeten-, verouderde-, gemengde- en niet-dubbel-scoren-paden.
- [ ] 3.2 `standards.max_age` in de configuratie, standaard 30 dagen.

## Wat de gemeten werkelijkheid oplegt
- **`error` is geen `failed`.** Eén test in de mail-fixture kwam terug
  als `{"status": "error", "verdict": "other"}`
  (`mail_starttls_tls_available`). Een meting die stukliep is
  **onbekend** met reden "meting mislukt" — niet afhankelijk. Wie hem
  als failed telt, rekent onze eigen instantie-storing de gemeten
  partij aan.
- **`not_tested` is de normale toestand, niet de uitzondering.** In de
  mail-fixture staan 18 van de 38 tests op `not_tested`. Een domein
  zonder volwaardige mailopstelling mag daar niet goed van scoren:
  een categorie waarvan élke test `not_tested` is, is onbekend, nooit
  soeverein.
- **Nooit de percentage-score narekenen.** De API geeft er zelf een
  (`scoring.percentage`, 95 resp. 70). Draag hem informatief over en
  bouw er geen oordeel op. Een eigen getal dat afwijkt van het
  publieke rapport maakt beide ongeloofwaardig.

## Uitdrukkelijk buiten scope, met reden
Een domein dat helemaal géén mail voert, scoort nu onbekend op de
mailregels — voor altijd. "N.v.t." zou eerlijker zijn, maar dat is uit
het importbestand alleen niet af te leiden (internet.nl meldt
`not_tested`, wat ook "wij konden het niet meten" betekent), en de
spec verbiedt eigen bevindingen als voeding voor deze regels. Laat het
dus op onbekend staan en schrijf de vraag op in design.md; hem half
oplossen met een aanname is erger dan hem open laten.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` en
`golangci-lint run ./...` groen, en `openspec validate
2026-09-19-propose-internetnl-standards --strict` groen. Budget is $8.
