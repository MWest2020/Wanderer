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

### Nog te bevestigen

Deze meting was `--type web`. Of een **mail**-batch dezelfde platte
`{status, verdict}`-vorm heeft, is niet gemeten. Taak 1.2 bevestigt
dat met één echte mail-batch vóór de fixtures vastgelegd worden.
