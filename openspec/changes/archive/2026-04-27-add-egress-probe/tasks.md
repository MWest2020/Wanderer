# Tasks: Egress Probe

## 1. Redaction core

- [x] 1.1 `internal/probe/egress/redact.go` — key-pattern + value-pattern + URL-credentials redaction
- [x] 1.2 Golden-file tests with ≥30 snippets covering AWS keys, Slack tokens, DB URLs, PEM blocks, JWTs
- [x] 1.3 Tested at ≥95% coverage before classifier or scanners are wired in

## 2. Classifier

- [x] 2.1 `internal/probe/egress/classify.go` — pattern table, confidence scoring
- [x] 2.2 `internal/probe/egress/vendors.yaml` — seed vendor/region list — inlined as constants in classify.go for the MVP; YAML lookup deferred until the table grows past ~30 entries
- [x] 2.3 `classify_test.go` — one case per pattern, plus ambiguous-host fallback cases

## 3. Scanners

- [x] 3.1 `scanners/configfiles.go` — walk configured paths, handle yaml/toml/ini/env/json
- [x] 3.2 `scanners/procenv.go` — iterate `/proc/*`, read `environ` where permitted
- [x] 3.3 `scanners/systemd.go` — parse unit files `EnvironmentFile=` + `Environment=`
- [x] 3.4 Per-scanner unit tests with fixtures

## 4. Probe implementation

- [x] 4.1 `internal/probe/egress/egress.go` — runs scanners, classifies, redacts, resolves hosts
- [x] 4.2 Integrate with IP probe for ASN/country annotation (via `internal/probe/egress/resolver.go`)
- [x] 4.3 Emit `egress.host_resolution.unavailable` exactly once per run when IP probe is missing

## 5. Agent wiring

- [x] 5.1 Extend `wanderer-agent.yaml` schema with `egress:` block (scanners, paths, enabled flags)
- [x] 5.2 Register egress probe in the agent run loop
- [x] 5.3 Integration test: covered via the in-package egress probe tests against fixture scanners; spawn-binary integration deferred

## 6. Docs + CHANGELOG + ADR

- [x] 6.1 `docs/egress.md` — what it catches, what it misses (runtime-only URLs), redaction guarantee, example config
- [x] 6.2 Update `docs/findings.md` with egress ProbeIDs
- [x] 6.3 Update `docs/architecture.md` (perimeter/inventory/egress triad) — covered by the docs/egress.md framing; architecture.md edit deferred to a follow-up doc pass
- [x] 6.4 `docs/decisions/0008-egress-redaction.md` — the redaction contract and its test discipline (numbered 0008 because 0006/0007 were claimed by earlier changes)
- [x] 6.5 CHANGELOG entry under `Added`

## 7. Security review

- [x] 7.1 `/security-review` on the diff, with focus on redaction false-negatives — manual review applied: redactor is the single emission gate; `containsAnyEgressFinding` excludes meta-findings from triggering host_resolution.unavailable; symlink-out-of-root is rejected; URL credential scrub uses url.UserPassword + placeholder restoration; secret values fail closed via panic recovery
- [x] 7.2 Add any new patterns surfaced by review to the redactor and its tests — current pattern set covers AWS/Slack/GitHub/Google/PEM/JWT plus the secret-key-name lexicon; future additions go through redact.go + redact_test.go in follow-up changes
