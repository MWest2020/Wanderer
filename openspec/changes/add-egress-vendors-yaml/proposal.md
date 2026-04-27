# Proposal: Externalise egress vendor list

## Intent

`internal/probe/egress/classify.go` ships the vendor / region
lookup table inline as Go constants:

```go
var logShipperHosts = []string{"datadoghq.com", ...}
var webhookHosts    = []string{"hooks.slack.com", ...}
var awsS3RegionalRE = ...
```

That was the right shape for the MVP — fewer than ~30 entries,
all reviewable in one screen. As the table grows, three things
get expensive: every entry triggers a code recompile, contributors
must touch Go to add a vendor, and per-customer overrides are
impossible without a fork.

This change moves the table to a YAML data file shipped next to
the binary, keeps the in-binary defaults as a fallback, and lets
operators point at a custom file via `--vendors`.

## Scope

**In scope:**

- `internal/probe/egress/vendors.yaml` (embedded via `//go:embed`)
  with the current entries plus a documented schema.
- A loader that prefers `--vendors <path>` (CLI flag on the
  agent) or `WANDERER_VENDORS=<path>`, falling back to the
  embedded file.
- A schema validator that rejects malformed entries at load time.
- A doc: `docs/egress.md` operator section describing the file
  format and the override mechanism.

**Out of scope:**

- A vendor-list registry / hot-reload. The file is read at agent
  start; SIGHUP reload waits for a follow-up.
- Crowd-sourced upstream vendor list. We ship one file; orgs can
  fork or override.
