# Proposal: een vloot die je bijhoudt, en regels die zichzelf uitleggen

## Why

Mark, 2026-09-20, na het zien van de antwoord-eerst UI: *"wat ik op
termijn verwacht is dat je domeinen kunt opslaan en over de vloot kunt
inzien. dit was als analysis geweldig. een drilldown op hoe en wat de
regels is wat ik in mijn hoofd heb."*

Drie gaten, en ze horen bij elkaar.

**1. Een domein bestaat alleen als bijvangst.** Targets ontstaan nu
doordat iemand een scan doet. Je kunt geen vloot samenstellen: geen
"dit zijn mijn 40 domeinen", geen groepering, en het scanschema staat
in een ConfigMap in de homelab-repo — onzichtbaar vanuit de UI, en niet
te wijzigen door wie het aangaat.

**2. De vloot is niet te overzien.** `/ui/trends` heeft een
regelcatalogus en een score-matrix, maar geen scherm dat zegt: dit zijn
de domeinen, zo staan ze ervoor, hier is het slechter geworden.

**3. Een regel legt zichzelf niet uit.** Een `Rule` heeft een
`Description` en een `Rationale` — een alinea over wat hij waarneemt en
waarom het uitmaakt — maar die staan niet op de plek waar iemand een
oordeel leest. En de drempel die het oordeel bepaalt (90 dagen tot
verloop, 24 verbindingen, "meer dan de helft") staat nergens: die zit
in de code. Een drempel is een aanname, en een aanname die je niet kunt
zien, kun je niet weerleggen.

## Niet ja/nee, maar x/n

Mark: *"ipv een ja/nee wil je een % of een x/n"*. Terecht: voor één
domein is ja/nee de juiste kop, maar zodra je veertig domeinen naast
elkaar zet, zegt een binair oordeel niets — alles staat op "nee" zodra
er één ding misgaat, en je ziet niet wie er beter voor staat.

Dus: **x van n**, waarbij n de vragen zijn die beantwoord kónden
worden. Een onbeantwoorde vraag telt niet mee in n en wordt apart
getoond ("5/7, 2 onbekend"). Nooit meetellen als geslaagd, nooit
stilzwijgend wegrekenen.

Een percentage mag ernaast staan om op te sorteren, maar x/n is wat er
staat: 5/7 is navertelbaar, 71% niet.

## What Changes

**De vloot bijhouden.** Domeinen toevoegen, hernoemen en verwijderen
vanuit de UI, gegroepeerd per organisatie; per domein zichtbaar wanneer
het voor het laatst gescand is en volgens welk schema. Het schema
verhuist van "alleen een ConfigMap" naar iets dat de UI toont en kan
zetten.

**De vloot overzien.** Eén scherm: alle domeinen met hun x/n, het
verschil sinds de vorige scan, en welke regel de meeste "nee" oplevert
over de hele vloot. Sorteren op score, op verandering, op laatste scan.

**De regel als eindpunt van de drilldown.** Per regel: wat hij
controleert (`Description`), waarom het uitmaakt (`Rationale`), wélke
waarneming hij gebruikt, wélke drempel het oordeel bepaalt, en welke
domeinen nu aan welke kant staan. De drempels worden expliciet: ze
staan als waarde bij de regel in plaats van verstopt in een
vergelijking.

**En erbij: hoe het beter kan.** Mark: *"liefst natuurlijk in de
drilldown naast de regels advies hoe beter, zoals op internet.nl"*.
Elke regel krijgt één concrete handeling voor het geval hij "nee"
zegt — niet "zorg voor soevereine hosting", maar "vraag je registrar de
privacyproxy op example.nl te verwijderen". De accountability-regels
hebben dat al (`accountability_nl.yaml`); dit trekt het door naar alle
regels, in dezelfde ene tekstentabel. Een regel zonder advies faalt de
test, net zoals een regel zonder `Rationale` dat nu al doet.

## Scope / Not in scope

**In:** het beheren van domeinen, het vlootscherm, het x/n-oordeel, en
de regelpagina met drempels.

**Out:** de probes en de scoring zelf (er komt geen nieuwe waarneming
bij), authenticatie (dat staat), en rapportage-export. Ook out: het
verwijderen van de bestaande analyse-schermen — die blijven, dit is
wat eromheen ontbreekt.

## Risico dat expliciet benoemd hoort

x/n verleidt tot optellen van ongelijke dingen. "Mail loopt via een
Amerikaanse aanbieder" en "geen security.txt" zijn niet even zwaar, en
een score die ze gelijk telt, nodigt uit om het makkelijke gat te
dichten. Daarom blijft op de antwoordpagina de zin staan die zegt wát
het oordeel bepaalde, en toont het vlootscherm naast de score altijd de
zwaarste openstaande bevinding.
