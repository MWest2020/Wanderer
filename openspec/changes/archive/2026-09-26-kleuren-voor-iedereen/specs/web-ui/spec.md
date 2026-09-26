## ADDED Requirements

### Requirement: Oordeelkleuren zijn te onderscheiden zonder kleurzien

De vier oordeelkleuren (soeverein, voldoende, afhankelijk, onbekend) SHALL
ook zonder rood-groen- of blauw-geelzien van elkaar te onderscheiden zijn: het
kleinste CIEDE2000-verschil tussen twee oordeelkleuren SHALL minstens 20 zijn,
bij normaal zien én bij gesimuleerde protanopie, deuteranopie en tritanopie
(Machado 2009, ernst 1,0). Elke oordeelkleur SHALL minstens 3:1 contrast
hebben met de paginaachtergrond. Tekst op een oordeelkleur SHALL minstens 4,5:1
contrast hebben. Een oordeel SHALL nooit alleen aan kleur hangen: er staat
altijd een letter, woord of label bij.

#### Scenario: Deuteranopie

- **GIVEN** de oordeelkleuren uit `main.css`
- **WHEN** ze door de deuteranopie-simulatie gaan
- **THEN** verschillen voldoende en afhankelijk minstens ΔE 20, en elk ander
  paar ook

#### Scenario: Tekst op een gekleurde cel

- **GIVEN** een rastercel met het oordeel voldoende
- **WHEN** die rendert
- **THEN** is de cel gevuld met de voldoende-kleur en heeft de letter
  minstens 4,5:1 contrast daarop

#### Scenario: Niet van toepassing is geen oordeel

- **GIVEN** een antwoord met "niet van toepassing"
- **WHEN** het rendert
- **THEN** draagt het geen van de vier oordeelkleuren
