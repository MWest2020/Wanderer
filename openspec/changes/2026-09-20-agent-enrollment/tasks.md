# Tasks: agent-enrollment

## 1. Aanmelden en intrekken — run 01
- [ ] 1.1 Migratie + model: `agents` (hostnaam, geheim-hash, aangemeld
  op, ingetrokken op), en aanmeldtokens (hash, geldig tot, gebruikt op).
- [ ] 1.2 `POST /agents/enrol`: token inwisselen voor een eigen geheim;
  één keer bruikbaar, verloopt, en het geheim is niet terug te lezen.
- [ ] 1.3 CLI: een token aanmaken, agents tonen, een agent intrekken.
- [ ] 1.4 Tests incl. token twee keer, verlopen token, ingetrokken agent.

## 2. De kern luistert — run 02
- [ ] 2.1 `serve` bouwt de router met de agent-geheimen uit de store
  (`RouterWithSecrets`), met een startregel als er geen agents zijn.
- [ ] 2.2 De agent schrijft zijn geheim weg en gebruikt het daarna; bij
  een bestaand geheim meldt hij zich niet opnieuw aan.
- [ ] 2.3 Tests: aangemelde agent levert af, onbekende wordt geweigerd.

## 3. Eén keer is één keer — run 03
- [ ] 3.1 Partij-identificatie op de ontvangstroute; herhaling geeft
  "al ontvangen" zonder opnieuw op te slaan.
- [ ] 3.2 De outbox bewaart die identificatie over een herstart heen.
- [ ] 3.3 Tests: dezelfde partij twee keer, en een outbox die na een
  herstart leegloopt.

## 4. Documentatie
- [ ] 4.1 `docs/explanation/agent.md` en de operator-documentatie
  bijwerken; CHANGELOG onder `[Unreleased]`.
