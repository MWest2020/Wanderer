## ADDED Requirements

### Requirement: Een regel draagt zijn drempels als waarde

Een `Rule` SHALL zijn beslisgrenzen als gegeven meedragen (naam,
waarde, eenheid), zodat de UI ze kan tonen en een test ze kan
controleren. Een regel die op een grens beslist SHALL die grens niet
alleen in zijn vergelijking hebben staan. Regels zonder grens (een
aanwezig-of-niet-controle) SHALL een lege lijst hebben.

#### Scenario: Grenzen zijn uitleesbaar

- **GIVEN** de regel `wand.operationeel.domain_expiry`
- **WHEN** de UI zijn grenzen opvraagt
- **THEN** krijgt die 90 dagen en 30 dagen terug, met hun betekenis

#### Scenario: Grens en vergelijking lopen niet uiteen

- **GIVEN** een regel waarvan de vergelijking een andere waarde
  gebruikt dan de meegedragen grens
- **WHEN** de tests draaien
- **THEN** falen ze
