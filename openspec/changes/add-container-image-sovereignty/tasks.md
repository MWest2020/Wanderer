# Tasks: container image sovereignty

> Auto-mode straight-through. Decisions are recorded in the
> proposal's status block.

## 1. Match list + loader

- [ ] 1.1 Author
  `internal/assessor/container_registries.yaml` with the
  known US registries plus a vendor of record per entry.
- [ ] 1.2 Author `internal/assessor/container_registries.go`
  with `MatchRegistry(imageRef) (RegistryEntry, bool)` and
  `RegistryEntries() []RegistryEntry` (for tests + UI).
- [ ] 1.3 Loader test covers: bare name (`nginx` →
  docker.io match), library shorthand
  (`library/nginx` → docker.io match), explicit
  `gcr.io/foo/bar`, EU self-hosted miss
  (`harbor.example.de/...`).

## 2. Rules

- [ ] 2.1 `wand.docker.images_us_registry` — reads
  `inventory.docker.image`, classifies via `MatchRegistry`,
  emits afhankelijk on any hit with vendor names + image
  refs in the verdict; soeverein with sample evidence on a
  clean host; onbekend without findings.
- [ ] 2.2 `wand.docker.containers_us_registry` — same shape
  reading `inventory.docker.container`.
- [ ] 2.3 `eucsf.sov6.container_supply_chain` — combined
  rule reading both probe families.
- [ ] 2.4 Register all three in `DefaultRules()`.

## 3. Tests

- [ ] 3.1 Unit tests for each rule mirroring the
  host-rule pattern (clean host → soeverein with evidence,
  US hit → afhankelijk with verdict + evidence, no findings
  → onbekend, only meta → onbekend).
- [ ] 3.2 Update the rule-count test that pins the eucsf
  pack at N rules.

## 4. Fixture + Playwright

- [ ] 4.1 Extend `internal/fixtures/agent_host.go` with a
  curated docker surface: 3-4 images + 1-2 running
  containers, one of which references `gcr.io/foo/bar` so
  the rule fires.
- [ ] 4.2 Add
  `tests/playwright/specs/container-image-sovereignty.spec.ts`
  asserting the three rule rows surface with afhankelijk
  verdicts.

## 5. Docs

- [ ] 5.1 `docs/operator.md` — Docker inspector section gains
  a "sovereignty rules" paragraph naming the three rules and
  what they read.
- [ ] 5.2 `docs/architecture.md` — no change (the host-rule
  paragraph from add-host-side-scoring already covers the
  pattern; container rules are the same shape).

## 6. Verification

- [ ] 6.1 `go test ./...` clean.
- [ ] 6.2 `make playwright` clean.
- [ ] 6.3 `openspec validate add-container-image-sovereignty
  --strict` passes.

## 7. Wrap-up

- [ ] 7.1 Commit.
- [ ] 7.2 Merge spec delta into the canonical
  inventory-probe + assessor specs.
- [ ] 7.3 Archive under
  `openspec/changes/archive/2026-05-11-add-container-image-sovereignty/`.
