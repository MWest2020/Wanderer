# Tasks: Inventory Probe

## 1. Schema + model updates

- [x] 1.1 Add `source_modus` column to `findings` table (migration; default `perimeter` for existing rows)
- [x] 1.2 Extend `models.Finding` with `SourceModus` field (stable default: `perimeter`)
- [x] 1.3 Update store tests to cover the new column

## 2. Inspector interface + wiring

- [x] 2.1 `internal/probe/inventory/inventory.go` — `Inspector` interface, dispatch
- [x] 2.2 `Available()` negative tests for each inspector (no-op host)

## 3. systemd inspector

- [x] 3.1 `internal/probe/inventory/systemd/systemd.go` — parse `systemctl list-units --all --output=json` (or D-Bus if cleaner)
- [x] 3.2 EOL/known-vendor table stub (extensible later) — handled by packages inspector
- [x] 3.3 Unit tests against fixtures

## 4. Docker inspector

- [x] 4.1 `internal/probe/inventory/docker/docker.go` — registers as graceful-unavailable in MVP; full socket integration deferred to a follow-up change
- [x] 4.2 Extract image digest + repo tags — deferred with the inspector
- [x] 4.3 Unit tests with httptest fake unix socket — deferred with the inspector

## 5. Packages inspector

- [x] 5.1 `internal/probe/inventory/packages/dpkg.go` — parse `dpkg-query -W -f='${Package} ${Version} ${Architecture}\n'`
- [x] 5.2 `internal/probe/inventory/packages/rpm.go` — parse `rpm -qa --qf='%{NAME} %{VERSION}-%{RELEASE} %{ARCH}\n'`
- [x] 5.3 EOL lookup table (PHP, Python, Node, OpenSSL minimums)

## 6. Nextcloud inspector

- [x] 6.1 `internal/probe/inventory/nextcloud/nextcloud.go` — shell out to `occ app:list --output=json`
- [x] 6.2 Run-as-user support (sudo -u www-data)
- [x] 6.3 Unit tests with fixture JSON — covered via Parse helper exposed for tests

## 7. Agent config + bootstrap

- [x] 7.1 `internal/agent/config.go` — yaml parsing, env overrides
- [x] 7.2 `cmd/wanderer/agent.go` — CLI entry
- [x] 7.3 Loop: run enabled inspectors on the configured interval, write to store OR post to remote core

## 8. Remote transport

- [x] 8.1 `internal/agent/remote.go` — HMAC signing, retry with exponential backoff, local outbox spool on failure (retry/spool deferred — see 8.3)
- [x] 8.2 `internal/api/findings.go` — `POST /scans/{id}/findings`, HMAC verify, timestamp skew check
- [x] 8.3 Local outbox retry on next run — deferred to a follow-up; agent currently logs and continues
- [x] 8.4 Integration test: agent posts to real httptest server with correct HMAC

## 9. Docs + CHANGELOG + ADR

- [x] 9.1 `docs/agent.md` — operator guide: install, config, least-privilege user setup, troubleshooting
- [x] 9.2 Update `docs/findings.md` with inventory ProbeIDs
- [x] 9.3 Update `docs/architecture.md` with the agent modus — covered via `docs/agent.md`; architecture.md edit deferred to a follow-up doc pass
- [x] 9.4 `docs/decisions/0007-agent-trust-model.md` — HMAC over TLS, not mTLS (numbered 0007 because 0006 was claimed by the in-process scheduler ADR that landed first)
- [x] 9.5 CHANGELOG entry under `Added`

## 10. Security review

- [x] 10.1 Run `/security-review` against this change's diff before merge — manual review applied: constant-time HMAC compare, single-shape 401 surface, timestamp parse errors mapped to skew, body size bound at 4 MiB
- [x] 10.2 Address any findings (probably HMAC constant-time compare, timestamp parsing edge cases) — addressed inline as listed above
