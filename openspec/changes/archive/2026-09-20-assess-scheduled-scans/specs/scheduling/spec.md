## ADDED Requirements

### Requirement: Een geplande scan levert ook een oordeel

Na elke geslaagde geplande scan SHALL de scheduler die scan beoordelen
met beide rule packs en het `Assessment` opslaan, zodat de UI een
oordeel toont zonder dat een mens een commando draait. Mislukt het
beoordelen, dan SHALL de scan bewaard blijven en de fout gelogd worden;
de volgende tick SHALL gewoon doorgaan. Een schedule MAY het oordelen
uitzetten met `assess: false`; ontbreekt het veld, dan staat het aan.

#### Scenario: Schema levert scan en oordeel

- **GIVEN** een schedule zonder `assess`-veld
- **WHEN** de tick een geslaagde scan oplevert
- **THEN** staat er naast de scan een assessment in de store, en toont
  de rapportpagina een oordeel in plaats van "no assessment has been
  produced for this scan yet"

#### Scenario: Beoordelen faalt

- **GIVEN** een scan die wel slaagt maar waarvan het beoordelen een
  fout geeft
- **WHEN** de tick afloopt
- **THEN** blijft de scan bewaard, wordt de fout gelogd en draait de
  volgende tick gewoon

#### Scenario: Bewust uitgezet

- **GIVEN** een schedule met `assess: false`
- **WHEN** de tick een geslaagde scan oplevert
- **THEN** wordt er geen assessment gemaakt
