## ADDED Requirements

### Requirement: Elke release draagt bruikbare binaries

Een release SHALL binaries bevatten voor Linux, macOS en Windows op
amd64 en arm64, met checksums, gebouwd zonder CGo. De README SHALL één
commando tonen waarmee iemand de tool draait zonder Go te installeren,
en dat commando SHALL in CI gedraaid worden zodat het niet stilletjes
kan verouderen.

#### Scenario: Nieuwe release

- **WHEN** een tag `v*` wordt gepubliceerd
- **THEN** hangen er binaries en checksums aan de release

#### Scenario: Quickstart blijft werken

- **WHEN** CI draait
- **THEN** wordt het commando uit de README uitgevoerd en faalt de
  bouw als het niet meer werkt

---

### Requirement: De action scant bij de gebruiker en stuurt niets door

De GitHub Action SHALL de scan in de runner van de gebruiker draaien en
SHALL geen gegevens naar een instantie van ons sturen. Zij SHALL het
oordeel als job-samenvatting tonen, met het aantal beantwoorde vragen,
wat het oordeel bepaalde, en de handeling. Zij SHALL standaard niet
falen op een negatief oordeel; falen SHALL een expliciete keuze van de
gebruiker zijn.

#### Scenario: Scan in een workflow

- **GIVEN** een workflow die de action aanroept op `voorbeeld.nl`
- **WHEN** de workflow draait
- **THEN** staat het oordeel in de job-samenvatting en is de stap
  geslaagd, ook bij een negatief oordeel

#### Scenario: Gebruiker wil wél falen

- **GIVEN** `fail-on: afhankelijk`
- **WHEN** het oordeel afhankelijk is
- **THEN** faalt de stap

#### Scenario: Niets verlaat de runner

- **WHEN** de action draait
- **THEN** praat zij alleen met het gemeten domein en met niets van ons
