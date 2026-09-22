# Design notes: standards dimension (Internet.nl via netnl)

## External systems and failure modes

| System | Used for | Failure mode | Handling |
| --- | --- | --- | --- |
| netnl-findings file (v1) | all standards findings | absent, schema-version mismatch, malformed entries, unknown domains, stale measured_at | absent → dimension "not measured"; version mismatch → refuse import with the expected version named; malformed entries → WARN + skip (Amass precedent), partial import is first-class; unknown domain → WARN + skip; stale → import fine, rules score onbekend |
| Internet.nl batch API | none directly | — | v1 never touches it; that is netnl's job. Keep it that way. |
| netnl-serve facade (v2 only) | scheduled submit/ingest | auth, timeout, batch pending | out of scope here; named follow-up proposal |

## Verdict mapping

Internet.nl per-test verdicts (`passed` / `failed` / `warning` /
`info` / `not_tested`) map per category:

- all relevant tests `passed` → **soeverein**
- any `warning`, or a mix of passed/failed within the category →
  **voldoende**
- category substantively `failed` → **afhankelijk**
- `not_tested`, no finding, or measurement older than
  `standards.max_age` → **onbekend**, with the reason ("not measured"
  vs "measurement stale (measured 2026-07-01)") in the verdict text.

The wand scale is reused unchanged for consistency across the pack;
verdict *text* speaks compliance language ("DNSSEC signed and
valid"), not sovereignty language.

## Import semantics

- Findings persist under a scan of kind `import` referencing the
  source file hash and the Internet.nl request ID, so evidence stays
  traceable and re-imports of the same file are idempotent.
- The assessor correlates a target's newest standards findings across
  scan kinds; a perimeter scan never erases imported findings and
  vice versa.
- One file may carry many domains (batch output); one import call
  handles the whole fleet.

## The clever valkuil

Recomputing Internet.nl's score. The batch API exposes subtests,
weights change between Internet.nl releases, and a home-grown
percentage will disagree with the public report the bestuurder can
open in a browser — instantly destroying trust in both tools. Map
verdicts, link the report URL, never aggregate. (Second, smaller one:
falling back to scraping the HTML report when the API omits a field.
If the API doesn't expose it, the rule doesn't exist.)

## Why a file, not a library or an RPC

- The schema is testable with fixtures on both sides; neither repo
  imports the other; Go and Python stay decoupled.
- CI today: netnl step produces the artifact, `wanderer import`
  consumes it — works without the facade, without network, in
  air-gapped deployments.
- v2 slots in without changing the consumer: the facade delivers the
  same file over a webhook.

## Design gate outcome (2026-09-22)

Het contract is getoetst aan een **echte meting**, niet aan de
documentatie: één web-batch op Marks eigen instantie
(`api.westerweel.work`, batch API v2.7.0, request
`b2dda607433522760b51faecbcb23c87`, westerweel.work, score 95%). Het
antwoord staat als fixture in
`fixtures/batch-v2-web-westerweel.work-20260922.json`. Vijf dingen
klopten niet.

### 1. `detail` per variant bestaat niet in de batch-API

Contract-eis 4 zegt dat variantresultaten (www/non-www, IPv4/IPv6)
worden meegedragen onder `detail`. In het echte antwoord is **elke**
testuitslag exact `{"status": ..., "verdict": ...}` — 38 van de 38,
`results.custom` is `null`. Ook `web_ipv6_ws_reach`, de test waar het
contractvoorbeeld `{"www": "passed", "apex": "failed"}` bij verzint,
geeft alleen een status.

Die per-variantdetails staan in het **HTML-rapport**, niet in de API.
Eis 4 duwt de producent dus precies de kant op die design.md zelf
verbiedt ("falling back to scraping the HTML report... If the API
doesn't expose it, the rule doesn't exist").

**Besluit:** eis 4 vervalt. `detail` blijft in het schema als optioneel
veld met `null` als normale waarde, zodat een latere API-versie die
het wél levert er zonder schemabump in past. Geen enkele regel mag op
`detail` leunen.

### 2. De verdictverzameling mist `error`

Contract-eis 3 noemt `passed` / `failed` / `warning` / `info` /
`not_tested`. De echte meting leverde `passed`, `failed`, `info` en
`not_tested` — maar netnl's eigen code kent er zes: `render.py`
(`_TEST_STATUS_ORDER`) en `cli.py` behandelen ook `error`. Een test
die op de instantie zelf stukloopt, komt als `error` binnen.

**Besluit:** `error` hoort in het schema. De assessor behandelt hem als
`onbekend` met reden "meting mislukt", niet als `afhankelijk` — een
kapotte meting is geen slecht resultaat.

### 3. De categorieënlijst komt niet overeen

Het contract noemt `dnssec`, `ipv6`, `mail_auth`, `starttls_dane`,
`rpki`, `tls_config`, `web_security`. De API geeft
`web_appsecpriv`, `web_dnssec`, `web_https`, `web_ipv6`, `web_rpki` —
met `web_`-voorvoegsel, en `web_https` dekt zowel TLS-configuratie als
HSTS (het contract splitst die in `tls_config` en `web_security`).
Bovendien draagt een testuitslag zélf geen categorie: die staat apart
in `results.categories`, en de koppeling zit in de testnaam
(`web_dnssec_exist` → `web_dnssec`).

**Besluit:** de producent leidt `category` af uit de testnaam door het
langste voorvoegsel te nemen dat in `results.categories` voorkomt, en
neemt de API-naam ongewijzigd over (`web_dnssec`, niet `dnssec`). Geen
hernoeming: een eigen woordenlijst die naast de API gaat lopen is
precies de fout die dit contract wil vermijden. De assessor mapt van
API-categorie naar regel, en die mapping staat in Wanderer.

### 4. `measured_at` bestaat niet per domein

Het contract zet `measured_at` in elk domeinblok. Een domeinblok heeft
exact vier sleutels: `report`, `results`, `scoring`, `status`. De enige
tijd in het antwoord is `request.finished_date`, voor de hele batch.

**Besluit:** `measured_at` blijft per domein staan (de consument wil
het daar), maar het contract zegt erbij dat het de `finished_date` van
de batch is en dus voor alle domeinen in één bestand gelijk. Anders
gaat een lezer denken dat er per domein een eigen meetmoment is.

### 5. De rapport-URL wijst niet naar internet.nl

`report.url` is echt en zit per domein — maar op deze zelf-gehoste
instantie luidt hij
`https://netnl.westerweel.work/site/westerweel.work/485/`. Het
contractvoorbeeld toont `https://internet.nl/site/...`.

**Besluit:** Wanderer behandelt de rapport-URL als ondoorzichtig: hij
toont hem als bewijs en bouwt hem nooit zelf op uit domein + id. Een
regel die `internet.nl` in die URL verwacht, breekt op elke
zelf-gehoste instantie — en zelf-hosten is precies wat wij doen.

### Wat wél klopte

- `report.url` bestaat per domein (eis 5).
- `scoring.percentage` bestaat en blijft informatief (de valkuil uit
  design.md is terecht: niet zelf aggregeren).
- `web_appsecpriv_securitytxt` zit echt in de uitslag, dus de
  "niet dubbel scoren"-eis uit de assessor-delta is geen theorie.
- De testnamen zijn stabiel en machineleesbaar.

### 6. `results` op een onafgeronde batch slaagt stil

Gemeten terwijl de mail-batch nog liep: `internetnl results <id>
--json` eindigt met **exitcode 0** en schrijft een document met
`"domains": null` en `request.status: "running"`. Geen foutmelding,
geen waarschuwing.

Contract-eis 8 zegt dat export van een onvolledige batch niet-nul moet
eindigen en niets moet schrijven. Dat is dus nog niet zo — en het is
geen theoretisch risico: een CI-stap die `results` aanroept en de
exitcode gelooft, archiveert een leeg bestand en meldt succes. De
importkant aan Wanderer-zijde zou dan een geldig ogend bestand met nul
domeinen inlezen en "niet gemeten" tonen, precies zoals de demopagina
vanmiddag deed.

**Besluit:** eis 8 geldt voor de nieuwe `--format findings`-uitvoer, en
taak 1.2 legt hem vast met een test die een lopende batch aanbiedt en
een niet-nul exit plus een afwezig bestand verwacht. Het bestaande
`results`-gedrag blijft zoals het is (dat is netnl's eigen contract
met zijn gebruikers), maar de findings-export erft het niet.

### Bevestigd met een mail-batch (2026-09-22)

Request `abf44e2a8f4d00b3b40447d6d11583ca`, westerweel.work, score 70%.
Fixture: `fixtures/batch-v2-mail-westerweel.work-20260922.json`.

- **Zelfde platte vorm.** 0 van de 38 testuitslagen draagt meer dan
  `{status, verdict}`. Bevinding 1 geldt dus voor beide testsoorten;
  `detail` is geen web-eigenaardigheid.
- **`error` is echt.** `mail_starttls_tls_available` kwam terug als
  `{"status": "error", "verdict": "other"}`. Bevinding 2 is daarmee
  niet langer afgeleid uit netnl's code maar gemeten. Let op de
  tweede waarde: `verdict` is hier `other`, een woord dat in geen
  enkele lijst in het contract stond.
- **Categorieën dragen een `mail_`-voorvoegsel:** `mail_auth`,
  `mail_dnssec`, `mail_ipv6`, `mail_rpki`, `mail_starttls`. Het
  contract noemde `starttls_dane`; die categorie bestaat niet, het is
  `mail_starttls`. Bevinding 3 bevestigd, inclusief het gevaar van een
  eigen woordenlijst.
- **18 van de 38 tests staan op `not_tested`.** Een mail-meting van een
  domein zonder volwaardige mailopstelling levert dus veel lege
  uitslagen. De assessor moet `not_tested` als `onbekend` behandelen
  en niet als "goed" — anders scoort een domein zonder mail
  uitstekend op mailbeveiliging.

### 7. De voorvoegselregel laat RPKI stilletjes vallen

Mijn eigen correctie bij bevinding 3 — "leid de categorie af uit het
langste voorvoegsel dat in `results.categories` voorkomt" — is
uitgeprobeerd op de twee fixtures en **klopt niet**. Zes tests vallen
buiten de boot:

    web_ns_rpki_exists      web_ns_rpki_valid
    mail_ns_rpki_exists     mail_ns_rpki_valid
    mail_mx_ns_rpki_exists  mail_mx_ns_rpki_valid

De categorie heet `web_rpki`, de test heet `web_ns_rpki_exists`: het
`ns` zit ertussen, dus geen enkele categorienaam is een voorvoegsel.
Met de voorvoegselregel scoort `wand.standards.rpki` op 2 van de 4
web-tests en 2 van de 6 mail-tests, en blijft nameserver-RPKI
ongemeten. Een domein met ongeldige RPKI op zijn nameservers zou
gewoon soeverein scoren. Precies de stille niet-gescoorde regel waar
bevinding 3 voor waarschuwde — veroorzaakt door de oplossing van
bevinding 3.

**De instantie publiceert de juiste koppeling zelf.**
`GET /metadata/report` geeft `report.hierarchy.<web|mail>`: een lijst
van categorieën met hun subtestgroepen. Opgevraagd op
api.westerweel.work:

    web_rpki -> web_rpki
    web_rpki -> web_ns_rpki

netnl haalt dat document al op en ontleedt het al
(`client.metadata_report`, `gating.reference_from_metadata`).

**Besluit:** de categorie komt uit de metadata-hiërarchie, niet uit
een voorvoegselregel op `results.categories`. Een test hoort bij de
categorie waarvan een groepsnaam het langste voorvoegsel van de
testnaam is. Is de metadata niet op te halen, dan valt de export terug
op de voorvoegselregel én zet hij dat als waarschuwing op stderr — een
stille degradatie is hier erger dan geen export, want de consument ziet
alleen een regel die "soeverein" zegt.

Een test die ook dan nergens bij hoort, krijgt `category: null` en
blijft staan.

### Wat het contract nog meer verzweeg: `verdict`

Elke testuitslag heeft náást `status` een `verdict`-woord: gemeten
waarden zijn `good`, `bad`, `warning`, `not-tested`,
`recommendations`, `other`. Het contract noemt alleen `status` (en
noemt dát verwarrend genoeg "verdict"). De producent draagt beide
over: `status` is waar de assessor op scoort, `verdict` is
verklarende tekst die netnl niet hoort te interpreteren.

## Verwachte uitkomst voor westerweel.work (om de assessor tegen te ijken)

Onafhankelijk uitgerekend uit de twee fixtures, vóór de
assessor-run binnenkwam, zodat "de regel doet wat hij zegt" niet uit
diezelfde run komt:

| regel                          | tests | statussen                                  | verwacht    |
| ------------------------------ | ----- | ------------------------------------------ | ----------- |
| `wand.standards.dnssec`        | 6     | 6 passed                                   | soeverein   |
| `wand.standards.mail_auth`     | 5     | 5 passed                                   | soeverein   |
| `wand.standards.rpki`          | 10    | 10 passed                                  | soeverein   |
| `wand.standards.ipv6`          | 9     | 8 passed, 1 failed                         | afhankelijk |
| `wand.standards.tls_config`    | 22    | 15 passed, 3 failed, 2 info, 2 not_tested  | afhankelijk |
| `wand.standards.starttls_dane` | 19    | 18 not_tested, 1 error                      | onbekend    |
| (`web_appsecpriv`, 5 tests)    | —     | 3 passed, 2 info                           | geen regel  |

`starttls_dane` is het geval dat de meting oplevert en een verzonnen
fixture nooit had gegeven: geen enkele test geslaagd, achttien niet
uitgevoerd en één mislukt. Dat MOET onbekend worden. Wordt het
soeverein, dan leest een domein zonder meetbare STARTTLS als een
domein dat het goed heeft — en dat is de gevaarlijkste fout die deze
dimensie kan maken.

De RPKI-regel ziet hier 10 tests; met de voorvoegselregel uit
bevinding 3 waren dat er 4 geweest, en dan was het oordeel nog steeds
"soeverein" — even groen, op minder dan de helft van het bewijs.
