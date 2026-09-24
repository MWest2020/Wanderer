## MODIFIED Requirements

### Requirement: De vloot is de eerste laag

`/ui/` SHALL het vlootoverzicht van de gekozen organisatie tonen: één
score voor het geheel als `x/n` (de som van de soeverein beantwoorde
vragen over de som van de beantwoordbare), het aantal onbeantwoorde
vragen apart, het aantal domeinen dat niet soeverein is, en de
verdeling over de zeven stromen. De lijst SHALL standaard gesorteerd
zijn op de slechtste score eerst. Een invoerveld om een domein te
scannen SHALL aanwezig blijven, maar niet de hoofdzaak van de pagina
zijn; het SHALL bovenaan de pagina staan, direct onder de kop, waar een
mens het zoekt.

De vlootlaag is voor wie in één oogopslag wil zien waar de vloot staat
(ADR-0018). Daarom SHALL de vlootscore als ring getoond worden, de
verdeling per stroom als balk per stroom, en de domeinen als raster met
één gekleurde cel per stroom. Elk beeld SHALL dezelfde feiten ook als
tekst dragen (label of titel), zodat niets alleen aan kleur hangt. De
top-3 van de duurste regels SHALL in de taal van deze laag staan: de
stroom en de handeling, met hoeveel domeinen het raakt; de
onderbouwing van een regel hoort op de regelpagina, niet hier.

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

#### Scenario: Het invoerveld staat bovenaan

- **GIVEN** een ingelogde gebruiker op `/ui/`
- **WHEN** de pagina rendert
- **THEN** staat het scan-invoerveld vóór de vlootscore in de pagina

#### Scenario: Elk beeld heeft zijn woorden

- **GIVEN** een vloot met 24 van 46 vragen soeverein en 31 onbeantwoord
- **WHEN** de vlootpagina rendert
- **THEN** draagt de ring het label "24 van 46 vragen soeverein, 31
  onbeantwoord"
- **AND** draagt elke rastercel de stroom en het oordeel als titel

#### Scenario: De top-3 zonder onderbouwing

- **GIVEN** een regel die op zeven van elf domeinen faalt
- **WHEN** de top-3 rendert
- **THEN** staat er de stroom en de handeling met "7 van 11 domeinen"
- **AND** staat de Engelse onderbouwing van die regel niet op de pagina
