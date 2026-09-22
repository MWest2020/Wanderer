## ADDED Requirements

### Requirement: De vloot is de eerste laag

`/ui/` SHALL het vlootoverzicht van de gekozen organisatie tonen: één
score voor het geheel als `x/n` (de som van de soeverein beantwoorde
vragen over de som van de beantwoordbare), het aantal onbeantwoorde
vragen apart, het aantal domeinen dat niet soeverein is, en de
verdeling over de zeven stromen. De lijst SHALL standaard gesorteerd
zijn op de slechtste score eerst. Een invoerveld om een domein te
scannen SHALL aanwezig blijven, maar niet de hoofdzaak van de pagina
zijn.

#### Scenario: CISO opent de tool

- **GIVEN** een organisatie met vijf gescande domeinen
- **WHEN** een ingelogde gebruiker `/ui/` opent
- **THEN** ziet die één score voor de vloot, hoeveel domeinen niet
  soeverein zijn, en de domeinen met de slechtste bovenaan

#### Scenario: Een goede score verbergt geen slecht domein

- **GIVEN** negen domeinen die soeverein scoren en één dat faalt
- **WHEN** de vlootpagina rendert
- **THEN** staat naast de score dat één domein niet soeverein is, en
  staat dat domein bovenaan de lijst

---

### Requirement: Het domein toont zijn score en wat eraan te doen is

De antwoordpagina van één domein SHALL naast de oordeelzin de
`x/n`-score tonen, en SHALL bij elk niet-soeverein punt de handeling
tonen die bij die regel hoort. De zin SHALL blijven zeggen wat het
oordeel bepaalde.

#### Scenario: Domein met twee gebreken

- **GIVEN** een domein dat op mail en certificaat niet soeverein scoort
- **WHEN** de antwoordpagina rendert
- **THEN** staat er een score, de zin die het zwaarste punt noemt, en
  bij beide punten één concrete handeling

---

### Requirement: Eén taal per laag

De oordelen op de vloot-, antwoord- en onderbouwingspagina SHALL in het
Nederlands staan, uit de bestaande tekstentabel. Engelse veldnamen,
RFC-verwijzingen en ruwe probe-uitvoer SHALL alleen binnen het
uitgeklapte bewijs voorkomen.

#### Scenario: Geen half-Engelse regel

- **WHEN** een oordeel op de antwoordpagina rendert
- **THEN** staat er geen Engelstalige zin naast een Nederlandse vraag

#### Scenario: Bewijs blijft technisch

- **WHEN** iemand het bewijs uitklapt
- **THEN** mag daar de ruwe, Engelstalige waarneming staan
