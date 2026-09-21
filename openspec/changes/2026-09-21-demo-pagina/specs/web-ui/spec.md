## ADDED Requirements

### Requirement: De demopagina toont één domein en verder niets

De UI SHALL een publieke route `/demo` kennen die zonder aanmelding de
laatste voltooide scan toont van het domein uit `demo.target`: het
antwoord in één zin, de stromen als vragen met antwoorden, en het
uitklapbare bewijs. De pagina SHALL geen andere domeinen noemen, geen
vloot, geen organisaties en geen regelpagina's met andere targets
tonen, en SHALL geen scan kunnen starten. Zonder `demo.target` SHALL
de route niet bestaan (404), en SHALL dat één keer in het opstartlog
staan.

#### Scenario: Bezoeker zonder account

- **GIVEN** een instantie met `demo.target: westerweel.work` en een
  voltooide scan van dat domein
- **WHEN** iemand zonder aanmelding `/demo` opent
- **THEN** ziet die het antwoord en de onderbouwing van westerweel.work,
  met de datum van de scan

#### Scenario: Geen andere domeinen

- **GIVEN** een instantie die ook andere domeinen volgt
- **WHEN** de demopagina rendert
- **THEN** komt geen enkel ander domein in de pagina voor, ook niet in
  een link of een regeloverzicht

#### Scenario: Demo staat uit

- **GIVEN** een instantie zonder `demo.target`
- **WHEN** iemand `/demo` opent
- **THEN** krijgt die 404, en heeft het opstartlog gezegd dat de demo
  uit staat

#### Scenario: Nog geen scan

- **GIVEN** `demo.target` is gezet maar er is nog geen voltooide scan
- **WHEN** iemand `/demo` opent
- **THEN** zegt de pagina dat er nog geen meting is, in plaats van een
  leeg oordeel te tonen

---

### Requirement: De demo start geen scans

De demoroute SHALL alleen tonen wat op schema is gemeten. Er SHALL geen
manier zijn om vanaf de demopagina een scan te starten, ook niet met een
handmatig verzoek.

#### Scenario: Scanpoging vanaf de demo

- **WHEN** iemand zonder aanmelding een scan probeert te starten via de
  demoroute
- **THEN** wordt dat geweigerd en start er geen scan
