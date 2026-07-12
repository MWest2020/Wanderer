# Change: apply-docs-contract

> Seed-change, aangeleverd vanuit Westmarch (personal-handbook, change 2
> add-docs-contract). Uitvoering door een habitat-builder-Job; merge door Mark.

## Why

Wanderer is NLnet-traject-repo met de rijkste docs (17 ADR's), maar zonder contractstructuur of front matter.
Zonder uniform contract is handbook-aggregatie zinloos: het handbook (hub)
importeert `docs/` van dit repo at build time.

## What changes

- `docs/` volgens het contract hieronder (additief + migratie van bestaande
  docs; geen verwijderingen buiten wat hieronder expliciet staat).
- `.mcp.json` in de root volgens template (handbook-URL blijft placeholder
  `TODO-change-3`).
- Géén andere wijzigingen. Eén branch, één PR, titel:
  `docs: apply handbook docs contract`.

## Docs-contract (bindend, uit Westmarch add-docs-contract)

```
docs/
  index.md          # wat is dit, status, max 3 regels boilerplate
  how-to/           # taakgericht
  reference/        # feiten (config, API, schema's)
  explanation/      # waarom-besluiten; ADR's onder explanation/adr/NNNN-titel.md
```

- Lege mappen weglaten. Minimum viable: `index.md` + één reference-pagina.
- Front matter per pagina (YAML): `status: current|draft|deprecated` +
  `last_reviewed: <ISO-datum>`. GEEN `owner`-veld.
- Gemigreerde pagina's zonder inhoudelijke review: `status: draft`,
  `last_reviewed` = migratiedatum. Alleen een echte review zet `current`.
- Eén taal per repo (deze repo: English).
- README blijft; `docs/index.md` verwijst ernaar, vervangt niet.
- Bestaande losse docs migreren; stub met verwijzing achterlaten op de oude
  plek als er externe links naartoe kunnen bestaan.


## Repo-specifiek

- `docs/decisions/NNNN-*.md` (17 ADR's + template + README) →
  `docs/explanation/adr/`, nummering behouden; stub in `docs/decisions/README.md`
  achterlaten met verwijzing.
- Losse pagina's indelen: `architecture.md`, `agent.md`, `maintainability.md`
  → `explanation/`; `mcp.md`, `egress.md`, `exporters.md`, `findings.md`,
  `drift.md`, `observability.md`, `assessor.md` → `reference/`.
- `docs/README.md` inhoud opnemen in `docs/index.md`.
- Licentie staat op NOASSERTION: NIET zelf een licentie kiezen; noteer in de
  PR-body dat Mark EUPL-1.2 moet bevestigen.
- Interne hostnaam komt voor in test-fixtures en gearchiveerde openspec-files:
  BUITEN scope van deze change (geen codewijzigingen); alleen vermelden in de
  PR-body.
- Taal: Engels (publiek open-source-repo).

## Non-goals

- Geen merge (Mark merget), geen scope buiten deze change, geen wijzigingen
  aan CLAUDE.md / .claude/agents/ / CI.
