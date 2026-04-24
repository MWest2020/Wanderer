# Tasks: Egress Probe

## 1. Redaction core

- [ ] 1.1 `internal/probe/egress/redact.go` — key-pattern + value-pattern + URL-credentials redaction
- [ ] 1.2 Golden-file tests with ≥30 snippets covering AWS keys, Slack tokens, DB URLs, PEM blocks, JWTs
- [ ] 1.3 Tested at ≥95% coverage before classifier or scanners are wired in

## 2. Classifier

- [ ] 2.1 `internal/probe/egress/classify.go` — pattern table, confidence scoring
- [ ] 2.2 `internal/probe/egress/vendors.yaml` — seed vendor/region list
- [ ] 2.3 `classify_test.go` — one case per pattern, plus ambiguous-host fallback cases

## 3. Scanners

- [ ] 3.1 `scanners/configfiles.go` — walk configured paths, handle yaml/toml/ini/env/json
- [ ] 3.2 `scanners/procenv.go` — iterate `/proc/*`, read `environ` where permitted
- [ ] 3.3 `scanners/systemd.go` — parse unit files `EnvironmentFile=` + `Environment=`
- [ ] 3.4 Per-scanner unit tests with fixtures

## 4. Probe implementation

- [ ] 4.1 `internal/probe/egress/egress.go` — runs scanners, classifies, redacts, resolves hosts
- [ ] 4.2 Integrate with IP probe for ASN/country annotation
- [ ] 4.3 Emit `egress.host_resolution.unavailable` exactly once per run when IP probe is missing

## 5. Agent wiring

- [ ] 5.1 Extend `wanderer-agent.yaml` schema with `egress:` block (scanners, paths, enabled flags)
- [ ] 5.2 Register egress probe in the agent run loop
- [ ] 5.3 Integration test: run agent in a fixture dir, assert expected Findings

## 6. Docs + CHANGELOG + ADR

- [ ] 6.1 `docs/egress.md` — what it catches, what it misses (runtime-only URLs), redaction guarantee, example config
- [ ] 6.2 Update `docs/findings.md` with egress ProbeIDs
- [ ] 6.3 Update `docs/architecture.md` (perimeter/inventory/egress triad)
- [ ] 6.4 `docs/decisions/0007-egress-redaction.md` — the redaction contract and its test discipline
- [ ] 6.5 CHANGELOG entry under `Added`

## 7. Security review

- [ ] 7.1 `/security-review` on the diff, with focus on redaction false-negatives
- [ ] 7.2 Add any new patterns surfaced by review to the redactor and its tests
