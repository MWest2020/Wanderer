# Proposal: drie lagen voor een CISO — vloot, domein, techniek

## Why

Mark, 2026-09-22, over wie dit gebruikt: een **CISO of security officer**
die wil weten *waar staat mijn data, hoe is mijn infrastructuur
opgebouwd, en wat kan ik eraan doen*. Zijn verwachting, in drie stappen:

1. een **vloot kiezen**, laten scoren, en één cijfer zien voor het
   geheel — de Tourist;
2. **inzoomen op één domein** en de zeven punten zien — de Farmer;
3. **inzoomen op de techniek** voor wie dat wil — de Explorer.

Ik heb de huidige UI met Playwright doorlopen en per scherm bekeken.
De laag die hij als eerste verwacht, bestaat niet.

## Wat de screenshots laten zien

**`/ui/` is een enkel-domein-antwoord, geen vloot.** Er staat een
invoerveld plus "recent beantwoord" — drie regels met ja/nee. Geen
totaal, geen "hoe staat mijn organisatie ervoor". Wie binnenkomt met
veertig domeinen ziet drie willekeurige antwoorden.

**Het vlootscherm is een beheerlijst, geen scorebord.** Domeinen
toevoegen en verwijderen, met per domein x/n — maar geen totaal over de
vloot, geen verdeling, geen "welke regel kost mij de meeste punten".

**`/ui/trends` bevat wél vloot-informatie**, maar begraven: ruim 6.700
pixels hoog, met een regelcatalogus van dertig regels en een
score-matrix eronder. Bovenaan staat "Sovereignty by flow — Hosting:
all 3 in EEA", precies wat een Tourist wil, maar het is opgemaakt als
tabelkop tussen twee datadumps.

**De antwoordpagina (één domein) is het beste scherm dat er staat.**
Zeven regels, per stuk een oordeel en de waargenomen feiten. Dat is de
Farmer-laag, en die klopt al grotendeels — behalve dat er geen
x/n-score op staat en geen "wat kan ik eraan doen".

**De onderbouwing is de Explorer-laag**, en die is compleet maar ruw:
de Nederlandse vragen bovenaan, daaronder twee frameworktabellen met
Engelse oordelen, afkortingen en regel-ID's.

**En overal loopt de taal door elkaar.** "Waar staat de hosting?" met
daarachter "apex IPs in NL (EEA)". "Niet gemeten" naast "no ip.asn
finding for apex — IP probe did not run". Voor de lezer die we op het
oog hebben, leest dat als een half afgemaakt product.

## What Changes

**Laag 1 — de vloot (Tourist).** `/ui/` wordt het vlootoverzicht van de
gekozen organisatie: één score voor het geheel (som van x over som van
n, met de onbeantwoorde vragen apart), de verdeling over de zeven
stromen ("Mail: 3 van 5 buiten de EER"), en de drie regels die over de
hele vloot de meeste punten kosten. Het invoerveld blijft, maar als
actie binnen die pagina, niet als hoofdzaak.

**Laag 2 — het domein (Farmer).** De antwoordpagina krijgt de
x/n-score naast de zin, en per niet-soeverein punt de handeling die er
al is ("vraag je registrar…"). Zodat "wat kan ik eraan doen" op dezelfde
pagina staat als "hoe sta ik ervoor".

**Laag 3 — de techniek (Explorer).** De onderbouwing blijft wat ze is,
maar de twee frameworktabellen gaan achter één klik en de pagina opent
met de zeven vragen. De regelpagina is al de diepste laag en blijft.

**Eén taal per scherm.** De oordelen die uit de regels komen, worden
Nederlands, uit dezelfde tekstentabel als de rest. Engels blijft in het
bewijs (veldnamen, RFC's, probe-uitvoer) — daar hoort het.

## Dit vervangt een deel van ADR-0017

ADR-0017 koppelt Tourist aan `/ui/`, Explorer aan het rapport en Farmer
aan trends. De rolverdeling die Mark nu beschrijft is een andere:
Tourist = de vloot, Farmer = één domein met de zeven punten, Explorer =
de techniek. Dat is geen detail maar de indeling zelf, dus de ADR krijgt
een opvolger in plaats van een voetnoot.

## Scope / Not in scope

**In:** de drie lagen, de vlootscore, de handelingen op de
antwoordpagina, de taal van de oordelen, en de opvolger van ADR-0017.

**Out:** filteren en zoeken over de vloot ("query the data") — dat is
een eigen change zodra de lagen staan. Ook out: nieuwe metingen; dit
gaat over wat we al weten beter tonen.

## Risico dat expliciet benoemd hoort

Eén cijfer voor een hele vloot nodigt uit tot sturen op het cijfer. Een
organisatie die tien domeinen heeft waarvan er negen niets doen, scoort
prachtig terwijl het domein dat ertoe doet faalt. Daarom staat naast de
vlootscore altijd het aantal domeinen dat níét soeverein is, en is de
lijst standaard gesorteerd op de slechtste eerst.
