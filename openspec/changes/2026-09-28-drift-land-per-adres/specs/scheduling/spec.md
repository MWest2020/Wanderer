## ADDED Requirements

### Requirement: Een landwissel is een wissel van hetzelfde adres

De drift-engine SHALL `drift.ip.country_changed` alleen melden voor een
IP-adres dat in beide scans voorkomt en waarvan het land verschilt. Een host
met meerdere adressen in verschillende landen SHALL geen landwissel opleveren
zolang elk adres zijn land houdt. Een adres dat erbij komt of wegvalt, SHALL
geen landwissel opleveren.

#### Scenario: Host met adressen in vier landen, niets veranderd

- **GIVEN** twee scans waarin `lennox.ns.cloudflare.com` dezelfde zes adressen
  heeft, in CA, US, CR en GB, elk met hetzelfde land
- **WHEN** de drift-engine ze vergelijkt
- **THEN** is er geen `drift.ip.country_changed`, en precies één
  `drift.no_changes` als verder niets verschilt

#### Scenario: Hetzelfde adres in een ander land

- **GIVEN** adres `192.0.2.10` van `example.nl` in NL in de vorige scan en in
  US in de nieuwe
- **WHEN** de drift-engine ze vergelijkt
- **THEN** is er één `drift.ip.country_changed` met dat adres, NL en US

#### Scenario: Een nieuw adres

- **GIVEN** een host die er in de nieuwe scan een adres in een ander land bij
  heeft
- **WHEN** de drift-engine ze vergelijkt
- **THEN** is er geen `drift.ip.country_changed`
