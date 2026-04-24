# Tasks: Inventory Probe

## 1. Schema + model updates

- [ ] 1.1 Add `source_modus` column to `findings` table (migration; default `perimeter` for existing rows)
- [ ] 1.2 Extend `models.Finding` with `SourceModus` field (stable default: `perimeter`)
- [ ] 1.3 Update store tests to cover the new column

## 2. Inspector interface + wiring

- [ ] 2.1 `internal/probe/inventory/inventory.go` — `Inspector` interface, dispatch
- [ ] 2.2 `Available()` negative tests for each inspector (no-op host)

## 3. systemd inspector

- [ ] 3.1 `internal/probe/inventory/systemd/systemd.go` — parse `systemctl list-units --all --output=json` (or D-Bus if cleaner)
- [ ] 3.2 EOL/known-vendor table stub (extensible later)
- [ ] 3.3 Unit tests against fixtures

## 4. Docker inspector

- [ ] 4.1 `internal/probe/inventory/docker/docker.go` — Docker Engine API via socket (stdlib net/http over unix socket)
- [ ] 4.2 Extract image digest + repo tags
- [ ] 4.3 Unit tests with httptest fake unix socket

## 5. Packages inspector

- [ ] 5.1 `internal/probe/inventory/packages/dpkg.go` — parse `dpkg-query -W -f='${Package} ${Version} ${Architecture}\n'`
- [ ] 5.2 `internal/probe/inventory/packages/rpm.go` — parse `rpm -qa --qf='%{NAME} %{VERSION}-%{RELEASE} %{ARCH}\n'`
- [ ] 5.3 EOL lookup table (PHP, Python, Node, OpenSSL minimums)

## 6. Nextcloud inspector

- [ ] 6.1 `internal/probe/inventory/nextcloud/nextcloud.go` — shell out to `occ app:list --output=json`
- [ ] 6.2 Run-as-user support (sudo -u www-data)
- [ ] 6.3 Unit tests with fixture JSON

## 7. Agent config + bootstrap

- [ ] 7.1 `internal/agent/config.go` — yaml parsing, env overrides
- [ ] 7.2 `cmd/wanderer/agent.go` — CLI entry
- [ ] 7.3 Loop: run enabled inspectors on the configured interval, write to store OR post to remote core

## 8. Remote transport

- [ ] 8.1 `internal/agent/remote.go` — HMAC signing, retry with exponential backoff, local outbox spool on failure
- [ ] 8.2 `internal/api/findings.go` — `POST /scans/{id}/findings`, HMAC verify, timestamp skew check
- [ ] 8.3 Local outbox retry on next run
- [ ] 8.4 Integration test: agent posts to real httptest server with correct HMAC

## 9. Docs + CHANGELOG + ADR

- [ ] 9.1 `docs/agent.md` — operator guide: install, config, least-privilege user setup, troubleshooting
- [ ] 9.2 Update `docs/findings.md` with inventory ProbeIDs
- [ ] 9.3 Update `docs/architecture.md` with the agent modus
- [ ] 9.4 `docs/decisions/0006-agent-trust-model.md` — HMAC over TLS, not mTLS
- [ ] 9.5 CHANGELOG entry under `Added`

## 10. Security review

- [ ] 10.1 Run `/security-review` against this change's diff before merge
- [ ] 10.2 Address any findings (probably HMAC constant-time compare, timestamp parsing edge cases)
