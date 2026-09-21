## ADDED Requirements

### Requirement: CAA wordt echt opgezocht, inclusief de boom omhoog

De dns-probe SHALL CAA-records daadwerkelijk opvragen. Vindt zij op de
gevraagde naam niets, dan SHALL zij de boom omhoog lopen tot het
registreerbare domein, zoals RFC 8659 voorschrijft, en SHALL de finding
vastleggen op welke naam de records gevonden zijn. Een resolver die
stilzwijgend niets teruggeeft SHALL de tests laten falen. "Geen CAA"
SHALL alleen gemeld worden wanneer de hele keten leeg is.

#### Scenario: Records staan op het subdomein zelf

- **GIVEN** een naam met eigen CAA-records
- **WHEN** de dns-probe draait
- **THEN** staan die records in de finding, met die naam als herkomst

#### Scenario: Records staan op de apex

- **GIVEN** `iam.voorbeeld.nl` zonder eigen CAA en `voorbeeld.nl` met
  CAA-records
- **WHEN** de dns-probe draait
- **THEN** staan de records van de apex in de finding, met de
  vermelding dat ze van `voorbeeld.nl` komen

#### Scenario: Nergens CAA

- **GIVEN** een zone zonder CAA op enig niveau
- **WHEN** de dns-probe draait
- **THEN** meldt zij "geen CAA", en oordeelt de regel zoals nu

#### Scenario: Resolver geeft stilzwijgend niets

- **GIVEN** een resolver-implementatie die altijd een lege lijst zonder
  fout teruggeeft
- **WHEN** de tests draaien
- **THEN** falen ze
