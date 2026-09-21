## ADDED Requirements

### Requirement: Een oordeelvlag draagt het getal waarop hij berust

Een probe die een finding met een oordeelvlag emitteert (bijvoorbeeld
"verloopt binnenkort") SHALL het getal meeleveren waarop die vlag is
gezet, met zijn eenheid. De vlag en dat getal SHALL uit dezelfde
constante komen, zodat ze niet uiteen kunnen lopen. De UI SHALL die
grens tonen bij de regel die op de vlag oordeelt, met de vermelding dat
de waarneming hem toepaste.

#### Scenario: Certificaat verloopt binnenkort

- **GIVEN** een certificaat dat over 20 dagen verloopt
- **WHEN** de tls-probe zijn finding emitteert
- **THEN** staat er naast de vlag dat de grens 30 dagen is

#### Scenario: Vlag en grens lopen uiteen

- **GIVEN** een probe waarvan de vlag op een andere waarde wordt gezet
  dan het meegeleverde getal
- **WHEN** de tests draaien
- **THEN** falen ze
