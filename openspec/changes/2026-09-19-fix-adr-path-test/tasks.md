# Tasks: fix-adr-path-test

## 1. Fix — run 01
- [ ] 1.1 `internal/ui/playwright_coverage_test.go` leest
  `docs/explanation/adr/` in plaats van `docs/decisions/`; eventuele
  foutmeldingen en commentaar in dat bestand noemen het nieuwe pad.
- [ ] 1.2 `go build ./...`, `go vet ./...` en `go test ./...` volledig groen
  (offline, uit vendor/).
- [ ] 1.3 `openspec validate 2026-09-19-fix-adr-path-test --strict` groen.

## 2. Afronding
- [ ] 2.1 Delta's toepassen op `openspec/specs/project-hygiene/spec.md` en de
  change archiveren (na merge).
