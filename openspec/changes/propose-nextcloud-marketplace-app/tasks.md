# Tasks: Wanderer as a Nextcloud marketplace app (design pending)

This proposal is paper-only. The only checkpoint is Mark's
go/no-go call on marketplace distribution.

## 1. Direction decisions

- [ ] 1.1 Q1 — is marketplace distribution a business
  priority for Wanderer? **NO code lands until this is
  yes.**
- [ ] 1.2 Q2 — if yes, who maintains the PHP shim's release
  cadence, PHP-version matrix, marketplace review?
- [ ] 1.3 Q3 — sidecar lifecycle (systemd / container /
  managed-by-Nextcloud).

## 2. If Q1=yes, follow-up

A separate `add-nextcloud-marketplace-app` proposal opens
with the picked architecture (A / B / C from this proposal)
and breaks the work down. Until then, this is the only
record.

## 3. Wrap-up

- [ ] 3.1 Mark's decision recorded in the proposal status block.
- [ ] 3.2 Either archive (no-go) or escalate to an
  implementation proposal (go).
