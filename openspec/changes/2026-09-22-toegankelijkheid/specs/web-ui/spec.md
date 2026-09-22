## ADDED Requirements

### Requirement: Oordelen zijn leesbaar zonder kleur en met genoeg contrast

Elk oordeel dat de UI toont SHALL bij zijn werkelijke tekstgrootte een
contrastverhouding van ten minste 4,5:1 halen. Kleur SHALL nooit het
enige onderscheid zijn tussen twee oordelen: er SHALL ook tekst of vorm
verschillen, zodat het zonder kleurwaarneming leesbaar blijft.

#### Scenario: Oordeelbadge in grijstinten

- **GIVEN** een pagina met de oordelen ja, nee, onbekend en n.v.t.
- **WHEN** die zonder kleur wordt bekeken
- **THEN** is elk oordeel nog te onderscheiden aan zijn tekst

#### Scenario: Contrast gemeten

- **WHEN** axe-core over een scherm draait
- **THEN** meldt het geen contrastfout op een oordeel

---

### Requirement: De brede schermen passen op een telefoon

Een pagina SHALL op 390px breed niet horizontaal scrollen. Een tabel die
breder is dan het scherm SHALL binnen zijn eigen kader scrollen, zodat
de rest van de pagina op zijn plaats blijft.

#### Scenario: Onderbouwing op een telefoon

- **GIVEN** een scherm van 390px
- **WHEN** de onderbouwingspagina rendert
- **THEN** scrollt de pagina niet horizontaal
