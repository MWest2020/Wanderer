## ADDED Requirements

### Requirement: Domeinen zijn bij te houden als vloot

De UI SHALL een ingelogde gebruiker domeinen laten toevoegen aan en
verwijderen uit de vloot van een organisatie, los van het doen van een
scan. Per domein SHALL zichtbaar zijn wanneer het voor het laatst
gescand is en volgens welk schema. Een domein verwijderen SHALL de
eerdere scans en oordelen niet weggooien.

#### Scenario: Domein toevoegen zonder te scannen

- **GIVEN** een ingelogde gebruiker op het vlootscherm
- **WHEN** die `voorbeeld.nl` toevoegt
- **THEN** staat het domein in de vloot, met "nog niet gescand" als
  laatste scan

#### Scenario: Verwijderen laat de geschiedenis staan

- **GIVEN** een domein met drie eerdere scans
- **WHEN** het uit de vloot wordt gehaald
- **THEN** verdwijnt het uit het overzicht en blijven de scans
  raadpleegbaar

---

### Requirement: Het vlootscherm scoort x van n, niet ja of nee

Het vlootscherm SHALL per domein tonen hoeveel vragen soeverein zijn
beantwoord van het aantal dat beantwoord kón worden (`x/n`), met het
aantal onbeantwoorde vragen apart ernaast. Een onbeantwoorde vraag
SHALL niet meetellen in n en SHALL nooit als geslaagd gelden. Naast de
score SHALL de zwaarste openstaande bevinding staan, zodat een hoge
score geen makkelijke gaten verbergt. Een percentage MAY getoond worden
om op te sorteren; `x/n` SHALL de getoonde waarde zijn.

#### Scenario: Twee domeinen naast elkaar

- **GIVEN** een domein met 5 van 7 soeverein en 2 onbekend, en een
  domein met 6 van 7 en 0 onbekend
- **WHEN** het vlootscherm rendert
- **THEN** staat er "5/7 · 2 onbekend" respectievelijk "6/7", en niet
  twee keer "nee"

#### Scenario: Zwaarste bevinding staat erbij

- **GIVEN** een domein dat 6 van 7 scoort maar waarvan de mail via een
  Amerikaanse aanbieder loopt
- **WHEN** de rij rendert
- **THEN** staat die bevinding naast de score genoemd

---

### Requirement: Een regel legt zichzelf uit, inclusief de drempel

De regelpagina SHALL per regel tonen: wat hij controleert, waarom het
uitmaakt, welke waarneming hij gebruikt, en welke drempel het oordeel
bepaalt — als waarde, in mensentaal ("verloopt binnen 30 dagen",
"meer dan de helft van de nameservers"). De pagina SHALL tonen welke
domeinen nu aan welke kant van die drempel staan. Een drempel SHALL
niet alleen in de vergelijking in de code bestaan.

#### Scenario: Drempel is zichtbaar

- **GIVEN** de regel `wand.operationeel.domain_expiry`
- **WHEN** een gebruiker de regelpagina opent
- **THEN** staat er dat het oordeel omslaat bij 90 en bij 30 dagen

#### Scenario: Wie staat waar

- **GIVEN** vijf domeinen waarvan twee binnen 30 dagen verlopen
- **WHEN** de regelpagina rendert
- **THEN** staan die twee apart van de andere drie

---

### Requirement: Bij elk oordeel "nee" staat wat je eraan doet

Elke regel SHALL één concrete handeling kennen voor het geval hij
afhankelijk scoort, in dezelfde Nederlandse tekstentabel als de rest
van de copy. Die handeling SHALL benoemen wat er moet gebeuren en waar
("vraag je registrar de privacyproxy op voorbeeld.nl te verwijderen"),
niet een doel herhalen ("zorg voor een herkenbare registrant"). De
regelpagina en de onderbouwing SHALL die handeling tonen bij een
falend oordeel. Een regel zonder handeling SHALL de tests laten falen,
net zoals een regel zonder `Rationale` dat nu doet.

#### Scenario: Falende regel toont de handeling

- **GIVEN** een domein waarvan de mail buiten de EER landt
- **WHEN** de gebruiker de onderbouwing of de regelpagina opent
- **THEN** staat er één concrete handeling, met het domein erin genoemd

#### Scenario: Regel zonder handeling

- **GIVEN** een nieuwe regel zonder tekst voor de handeling
- **WHEN** de tests draaien
- **THEN** falen ze met een melding die de regel noemt
