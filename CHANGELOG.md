# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
once a first release is cut. Until then every entry lives under
`[Unreleased]`.

## [Unreleased]

### Added

- **Sovereignty overview** (`propose-sovereignty-overview`, ADR-0015).
  A synthesis panel on the assessment page that re-groups the
  already-scored rule rationales into an ordered "what goes where" set
  of flows — Hosting, Mail, DNS, Transit path, CDN / hyperscaler, Third
  parties — each with its observed verdict and score. Pure presentation
  (`internal/ui/SovereigntyFlows`): no new collection, no jurisdiction
  logic in the view. The synthesis that answers the "weinigzeggend"
  feedback — the org/host as the spider in the web. (Textual list
  first; an interactive graph is a follow-up.)
- **DNS-hosting jurisdiction** (`propose-dns-hosting-jurisdiction`).
  The scanner now feeds `dns.ns` (nameserver) hosts to the IP probe,
  and a new `wand.juridisch.ns_vendor_jurisdiction` rule scores who
  runs the authoritative DNS and where — "your DNS is run by Cloudflare
  (US)" — mirroring the MX-jurisdiction rule (all-EEA → soeverein,
  split → voldoende, none → afhankelijk). Wave-1 lead from the
  high-signal observability direction. (`dns_redundancy` previously
  scored only NS *count*, never location.)
- **Transit-path probe** (`propose-transit-path-probe`). A new
  `transit` probe traces the network path to a target and attributes
  each hop — IP, reverse DNS, ASN, organisation, country, RTT — plus a
  `transit.path` aggregate, answering "where is this actually hosted
  and which jurisdictions does the traffic cross". Uses the
  unprivileged `tracepath`/`traceroute` tool (degrades to
  `transit.unavailable` when absent), reuses the GeoLite2 enrichment,
  and handles no-reply hops as gaps. New `wand.transit.eu_path` rule
  scores the destination jurisdiction (EEA → soeverein, non-EEA →
  afhankelijk) and names non-EEA transit hops informationally. First
  instance of the high-signal observability direction
  (`research-high-signal-observability`). New `transit-probe`
  capability spec.

First tagged release. Everything below was developed pre-1.0; the tag
exists so downstream consumers (e.g. the `wanderer-exapp` Nextcloud
ExApp) can pin a reproducible version instead of `@main`.

### Added

- **Nextcloud as OIDC login** for `wanderer serve --ui` — accept a
  Nextcloud (or any OIDC provider) as the identity hub. New
  `internal/auth/oidc` (go-oidc/v3 + x/oauth2, lazy discovery),
  server-side SQLite session table (migration 006), a gate that falls
  back through session → htpasswd break-glass → OIDC redirect, and a
  `serve.yaml` `oidc:` block. See ADR-0013.
- **Nextcloud as output** — publish each completed scan into a
  Nextcloud over WebDAV as a JSON-LD + Markdown bundle. New
  `internal/export/nextcloud` + a post-scan publisher hook (bounded
  retry, never fails the scan, ADR-0008 redaction + Evidence-drop
  before publish) and a `serve.yaml` `nextcloud:` block.
- **Marketplace distribution decided** — Wanderer ships as a Nextcloud
  AppAPI ExApp (architecture D) from the downstream `wanderer-exapp`
  repo; the core stays PHP/Composer-free. See ADR-0014.
- **Playwright smoke tests for the UI.** New `tests/playwright/`
  with `@playwright/test` 1.59.1 pinned, two spec files
  (`dar.spec.ts` covering DAR layering + organisation scope
  persistence + Analysis-page nav; `reporting-catalogue.spec.ts`
  covering the rule catalogue + status column), and a
  Makefile-driven workflow: `make playwright-install` (with
  signature verification, no lifecycle scripts per the global
  supply-chain rule), `make playwright` (boots `wanderer
  serve` against the configured DB, runs the spec set, tears
  down). Webserver block in `playwright.config.ts` handles the
  boot. New Go test
  `internal/ui/playwright_coverage_test.go` walks
  `docs/decisions/` for `## UI surface` sections and fails if
  a matching Playwright spec is missing — the ADR coverage
  contract from the OpenSpec proposal.

### Fixed

- The Reporting nav tab now threads the active org filter
  (`?org=<slug>`) through to `/ui/reporting`. The link was
  scope-aware in `fix-nav-org-context` but lost the filter in
  a later edit during the DAR layer restructure;
  `add-reporting-status-column` made the catalogue scope-aware
  again, so the nav link now matches. Caught by the new
  Playwright spec.

### Changed

- Reporting catalogue gained a compact "current state" column —
  worst score reached on the rule across the snapshots in
  scope, plus an "X of Y targets" triage hint. The full
  per-score matrix stays on Analysis; this is a hint so an
  operator with many rules can spot the problem rules at a
  glance. Catalogue is org-aware via `?org=<slug>` (mirrors
  Analysis). Rules with no recorded rationale render "no
  rationale yet". Spec amendment in
  `openspec/specs/web-ui/spec.md`.

- **DAR layer roles refined.** Mark refined the model on
  2026-05-10: *"bij dashboarding enkel goed/niet goed, analysis
  zie je de matrix en reporting de regels."* The UI now
  reflects that:
  - **Dashboard** stays slim: per-framework verdict pills
    (worst-score reached) plus the headline-stats strip plus
    the organisations list. The External / Internal posture
    blocks, Top concerns table, and Recent activity table are
    removed from the Dashboard — they answered steering
    questions that belong on Analysis.
  - **Analysis** moves to `/ui/analysis` (with `?org=<slug>`
    filter). It's the rule × score-counts matrix that
    previously lived at `/ui/reporting`. The per-target slice
    (`/ui/targets`) is reachable as "per-target view →" from
    the matrix page.
  - **Reporting** at `/ui/reporting` becomes a rule catalogue:
    every registered rule with framework, dimension,
    description, and rationale — no scoring data. The per-rule
    detail page at `/ui/reporting/{framework}/{ruleID}` keeps
    its existing shape.
  Spec delta in `openspec/specs/web-ui/spec.md` adds the new
  layer requirements with scenarios; `docs/architecture.md`
  rewrites the "Read-only operator UI" section around the
  refined roles.

### Fixed

- The DAR cross-page navigation now persists the active
  organisation scope across every tab. Selecting an org and
  clicking Analysis or Reporting no longer dumps the operator
  out of their context — Dashboard threads to
  `/ui/orgs/<slug>`, Analysis to `/ui/targets?org=<slug>`,
  Reporting to `/ui/reporting?org=<slug>`. The Reporting
  index, rule-detail, and Targets pages render a "Scope:
  {orgName}" pill in the header so the filtered view is
  visibly distinct from the global one. The Reporting nav
  tab now appears on Analysis pages too (was previously
  omitted because those handlers passed `HasReporting=false`,
  left over from when Reporting did not yet exist).
- The `/ui/targets` page accepts `?org=<slug>` to filter the
  targets list to one organisation, mirroring the existing
  filter on `/ui/reporting`. Unknown slug → 404.

### Added

- **Organisations** as a first-class concept. Every Target
  (perimeter domain or agent host) now belongs to exactly one
  Organisation; a seeded `default` Organisation picks up any
  pre-existing or unspecified Target. Migration 005 lands the
  schema (new `organisations` table, `targets.organisation_id`
  column, `o_default` seed). The full surface ships in one
  change: CLI (`wanderer org add/list/show/rename` plus
  `--from-yaml` bulk seed), `--organisation <slug>` flag on
  `wanderer scan` with `flag > env > yaml > default` precedence,
  `core.organisation` field in the agent YAML, `scan.organisation`
  fallback in `serve.yaml`, optional per-schedule
  `organisation:` field, instance-wide `/ui/` listing every
  organisation with drill-in links, per-organisation dashboard
  at `/ui/orgs/{slug}`, optional `?org=<slug>` filter on
  `/ui/reporting`, and three MCP methods (`org_list`,
  `org_show`, `org_targets`). Mark approved every recommendation
  in the design pass on 2026-05-03; the resolved-decisions
  table is preserved in
  `openspec/changes/archive/2026-05-04-add-organisation-pivot/proposal.md`.
- Per-check Reporting page in the operator UI. New routes
  `/ui/reporting` (list every rule that has fired across the
  persisted Assessments, with distinct-target counts per
  `soeverein` / `voldoende` / `afhankelijk` / `onbekend`) and
  `/ui/reporting/{framework}/{ruleID}` (per-rule deep dive: the
  rule's description and rationale plus every target the rule
  fired on, with a back-link to the originating scan's
  assessment page). The Reporting tab in the cross-page nav now
  lights up — `redesign-dashboard-pontificaal` reserved the slot;
  this change wires it. New aggregator helpers `RuleSummary` and
  `RuleTargetRows` in `internal/ui/aggregate.go`. Distinct-target
  counts mirror the dashboard's Top concerns convention: a rule
  firing twice on one target counts once. Spec delta in
  `openspec/specs/web-ui/spec.md` adds the index and detail
  contracts with scenarios.

### Changed

- The read-only operator UI's `/ui/` page is now a Dashboard in
  the strict sense: a pontificaal headline (last scan, total
  scans, external + internal coverage counts, frameworks
  scored), an explicit **External posture** block for perimeter
  targets (`Kind=domain`), and an **Internal posture** block for
  agent-host targets (`Kind=host`) with empty-state copy
  pointing at `wanderer agent` when no host is reporting yet.
  A small `nav.tmpl` partial renders Dashboard / Analysis /
  Reporting tabs across every UI page so the layering is
  reinforced everywhere; the Reporting tab is omitted until the
  sibling `add-reporting-per-check` proposal lands. The page's
  Top concerns and Recent activity tables are unchanged. Spec
  delta in `openspec/specs/web-ui/spec.md` adds the headline
  and external/internal split as requirements.

### Added

- YAML config file for `wanderer serve`. New `--config <path>`
  flag (and `WANDERER_CONFIG` env var) loads a YAML covering every
  operator-tunable setting: `listen`, `db`, `geoip`, `ui`,
  `schedules`, and the `scan.*` block. Setting precedence is
  flag > env > YAML > default, so `serve.yaml` is the durable
  source of truth and a one-off `--addr :7070` still wins
  cleanly. The parser is strict — typos like `htpasswrd` fail
  at startup with an error naming the bad field. New package
  `internal/serveconfig/` with full unit-test coverage of every
  resolver layer. `docs/operator.md` carries the schema, an
  example config, and a sample systemd unit. When `--config` is
  unset, `wanderer serve` behaves byte-identically to today.

### Added

- arm64 BPF build target for the egress flow probe. `gen.go` now
  passes `-target amd64,arm64` to bpf2go; both
  `connect_x86_bpfel.{go,o}` and `connect_arm64_bpfel.{go,o}` ship
  in the repo. Selection between the two artefacts is a build-time
  concern resolved by `//go:build` constraints, so a `wanderer
  agent` binary built for either GOARCH carries exactly one BPF
  object — no runtime detection logic. Closes the arm64
  follow-up named in ADR-0010; the addendum to that ADR records
  the landing. Verified at change time with native `go build
  ./...` plus `GOARCH=arm64 go build
  ./internal/probe/egress/flow/...`. A CI matrix for the cross
  build is intentionally out of scope and tracked separately.

### Added

- Optional reverse DNS annotation on flow Findings. When the
  operator sets `egress.flow.reverse_dns.enabled: true`, each unique
  destination IP in the sampling window is resolved via the host's
  resolver and the resulting Finding gets a `reverse_dns:
  "<hostname>"` attribute. Off by default — a PTR query leaks the
  observation back through the host's DNS path, which a sovereignty
  monitor cannot default-on without consent. Per-IP cache inside
  the Aggregator window guarantees one PTR query per unique IP per
  tick. ADR-0010 carries the privacy-tradeoff addendum; spec delta
  in `openspec/specs/egress-probe/spec.md`. Configurable
  `egress.flow.reverse_dns.timeout` (default 500ms) caps each
  individual lookup.

### Changed (breaking)

- The first-party rule pack — formerly named **`dictu`** after the
  Dutch government's *Dienst ICT Uitvoering* — has been renamed to
  **`wand`** (Wanderer-NL) per
  [ADR-0011](docs/decisions/0011-rename-dictu-to-wand.md). DICTU's
  publicly-available *Toetsingsinstrument Soevereiniteit
  Clouddiensten* remains credited as the inspiration; the rule
  pack's identity is Conduction's. Affects: the Go package
  (`internal/assessor/dictu/` → `internal/assessor/wand/`), every
  rule ID (`dictu.<dim>.<short>` → `wand.<dim>.<short>`), the
  persisted `Assessment.Framework` value (`"dictu"` →
  `"wand"`), the CLI flag default (`--framework wand` is the new
  canonical), and every documentation surface. Schema migration
  v4 rewrites existing assessment rows on first `store.Open` —
  both the framework column and the JSON-encoded `criterium_id`
  strings inside the dimensions blob, in one transaction.
  External consumers reading the JSON API or the SQLite
  assessments table will see `framework: "wand"` after this
  change; integration code that pinned `"dictu"` needs updating.

### Added

- `--framework dictu` continues to work as a deprecated alias for
  `--framework wand` for one release. Operators using the alias
  see one stderr line at startup pointing at the canonical name
  and the docs that explain the rename. The alias goes away in
  the next release.
- `internal/ui/registry.go::lookupRule` accepts both `wand` and
  `dictu` as the framework key for one release so any DB row that
  bypassed the migration still renders against the live rule
  registry. Removed in the next release.
- ADR-0011 documents the legal motivation for the rename;
  ADR-0009 (dual-framework assessor) and ADR-0004 (assessor rule
  engine) gain rename addenda pointing at ADR-0011 so the
  package layout described in the older ADRs stays
  cross-referenceable.

### Added

- Operator UI Dashboard page (closes the DAR triad with the
  Analysis page that landed earlier today). `/ui/` now renders a
  posture summary (counts of targets per worst-dimension score
  per framework), top concerns (rules whose `afhankelijk`
  rationales span the most distinct targets, target-counted —
  one rule firing 50× on one target counts once), and recent
  activity (the five most recent scans across the estate, each
  linking to the per-scan Analysis page if it has an Assessment).
  An empty-state path renders a `wanderer scan <domain>` hint
  when no scans exist. New `internal/ui/aggregate.go` exposes
  pure-Go helpers (`WorstScore`, `PostureCounts`, `TopConcerns`,
  `RecentActivity`) covered by `aggregate_test.go`. End-to-end
  tests cover empty store, posture counts, target-counted
  concerns, and the activity-row cap.
  (`internal/ui/aggregate.go`, `internal/ui/aggregate_test.go`,
  `internal/ui/ui.go`,
  `internal/ui/templates/dashboard.tmpl`,
  `internal/ui/static/main.css`, `internal/ui/ui_test.go`)

### Changed

- Operator UI: the previous flat targets table that lived at
  `/ui/` moved to `/ui/targets`. The dashboard at `/ui/` is the
  new entry point; the targets table is one click away ("All
  targets →" in the dashboard header). All in-page back-links
  ("← all targets") on scan / drift / assessment pages now
  point at the dashboard with a secondary "all targets" link
  pointing at `/ui/targets`. Operators with bookmarks to the
  flat table need to update them.
  (`internal/ui/ui.go`, `internal/ui/templates/index.tmpl`,
  `internal/ui/templates/scan.tmpl`,
  `internal/ui/templates/drift.tmpl`,
  `internal/ui/templates/assessment.tmpl`)

- Operator UI Analysis page: `/ui/scans/{id}/assessment` renders
  every persisted Assessment for the scan as one card per
  framework, dimension cards with the score badge and
  completeness flag, and one row per Rationale showing the rule's
  CriteriumID + verdict + a "why this matters" expandable detail
  with cited evidence Findings linking back to the scan-detail
  page. Scans without an Assessment render an empty-state hint
  pointing at `wanderer assess`. The scan-detail page gains an
  "Open assessment →" link when at least one Assessment exists.
  `assessor.Rule` grows a required `Rationale` field — every
  DICTU and EUCSF rule now ships with a paragraph explaining what
  it observes and why it matters; new
  `TestEveryRuleHasRationale` per pack fails the build if any
  rule's Rationale is empty or duplicates the Description.
  Read-only contract preserved: every new handler is GET-only,
  the existing static-analysis test still pins it.
  (`internal/assessor/rule.go`,
  `internal/assessor/dictu/rules.go`,
  `internal/assessor/eucsf/rules.go`,
  `internal/ui/registry.go`, `internal/ui/registry_test.go`,
  `internal/ui/ui.go`,
  `internal/ui/templates/assessment.tmpl`,
  `internal/ui/templates/scan.tmpl`,
  `internal/ui/static/main.css`, `internal/ui/ui_test.go`)

### Changed (breaking)

- `assessor.Rule.Rationale` is now a required field on every Rule
  registered with `dictu.DefaultRules()` /
  `eucsf.DefaultRules()`. External consumers that build their own
  rule packs must populate the field; an empty value is caught by
  the per-pack `TestEveryRuleHasRationale` rather than going
  silently into production.

### Added

- GeoLite2 onboarding: `wanderer scan` and `wanderer serve` emit one
  warning to stderr at startup when no `--geoip` / `WANDERER_GEOIP_ASN`
  is configured, so a fresh-installed operator notices the missing
  ASN data instead of silently scoring half the DICTU rules
  `onbekend`. New `--no-geoip` flag (and `WANDERER_GEOIP_OPTIONAL=1`
  env) silences the warning for offline labs / CI without changing
  runtime behaviour. New `scripts/geoip-stub.sh` produces a
  deterministic empty-but-valid GeoLite2-shaped mmdb so the test
  suite can exercise the populated-but-empty branch without a real
  MaxMind license. `docs/operator.md` gains an explicit "GeoLite2
  setup" section (license-key acquisition, recommended
  `geoipupdate` via systemd timer, opt-out, test stub);
  `docs/architecture.md` and `docs/tutorial.md` link into it.
  Adds dev-time dependency `github.com/maxmind/mmdbwriter` (used
  only by the stub script via `//go:build ignore`).
  (`cmd/wanderer/geoip.go`, `cmd/wanderer/geoip_test.go`,
  `cmd/wanderer/scan.go`, `cmd/wanderer/serve.go`,
  `scripts/geoip-stub.sh`, `scripts/geoip-stub/main.go`,
  `docs/operator.md`, `docs/architecture.md`, `docs/tutorial.md`)

- Egress flow probe **kernel attach** lands. Earlier today the
  userspace half (Aggregator, Inspector surface, classifier reuse,
  agent wiring, ADR-0010) shipped with the BPF object compile
  deferred. The deferral is now closed: a pinned Fedora-42 builder
  container (`build/bpf-builder/Dockerfile`) and a
  `./scripts/bpf-build.sh` driver run `go generate` against the
  bpf2go directive in `internal/probe/egress/flow/gen.go`,
  producing committed `connect_x86_bpfel.{go,o}` artefacts. New
  `kernel_linux.go` (build-tagged `linux && amd64`) loads the
  embedded BPF object via `cilium/ebpf`, attaches to
  `tracepoint/syscalls/sys_enter_connect`, and feeds a perf-ring
  reader into the existing Aggregator. `cmd/wanderer/agent.go`
  constructs the kernel source eagerly when `egress.flow.enabled:
  true`; load failures are captured on Flow.SourceErr so
  Available() reports the specific reason without crashing the
  agent. `go build ./...` still works on hosts without
  clang/llvm — only the developer regenerating the BPF object
  needs the builder container.
  (`build/bpf-builder/Dockerfile`, `scripts/bpf-build.sh`,
  `internal/probe/egress/flow/gen.go`,
  `internal/probe/egress/flow/kernel_linux.go`,
  `internal/probe/egress/flow/kernel_stub.go`,
  `internal/probe/egress/flow/connect_x86_bpfel.go` (generated),
  `internal/probe/egress/flow/connect_x86_bpfel.o` (generated),
  `cmd/wanderer/agent.go`,
  `docs/decisions/0010-ebpf-flow-probe.md`,
  `docs/egress.md`, `.gitignore`)

- Egress runtime flow probe (eBPF) — userspace half landed
  2026-04-29; kernel attach deferred to a follow-up that needs a
  clang/llvm toolchain. The new `internal/probe/egress/flow`
  package ships an Inspector with proper `Available()` detection
  (kernel BTF + CAP_BPF + loader presence), an Aggregator that
  dedups by `(destination_ip, destination_port)` and emits one
  Finding per unique destination through the existing classifier,
  and a `Run` method the agent invokes only when
  `egress.flow.enabled: true` is set in the agent config. Default
  config does not construct the inspector, so the agent emits no
  `egress.flow.*` findings (not even `unavailable`) until an
  operator opts in. The CO-RE C source for the
  `sys_enter_connect` tracepoint program ships at
  `internal/probe/egress/flow/bpf/connect.bpf.c` for review;
  ADR-0010 documents the kernel-version contract and the
  deferred bpf2go integration.
  (`internal/probe/egress/flow/flow.go`,
  `internal/probe/egress/flow/flow_test.go`,
  `internal/probe/egress/flow/bpf/connect.bpf.c`,
  `internal/agent/config.go`, `cmd/wanderer/agent.go`,
  `docs/egress.md`, `docs/decisions/0010-ebpf-flow-probe.md`)

### Changed

- `docs/architecture.md` is rewritten around the three-modi triad
  (perimeter / inventory / egress) with a Mermaid diagram, per-modus
  prose, and four how-to-extend sections (perimeter probe, inventory
  inspector, egress scanner, DICTU rule). Cross-references the
  per-capability docs (`assessor.md`, `agent.md`, `egress.md`,
  `mcp.md`, `scheduling.md`, `drift.md`, `exporters.md`,
  `operator.md`, `observability.md`, `maintainability.md`,
  `tutorial.md`) so a new contributor lands in the right reference
  for whatever they are touching. The probe-level meta-finding
  convention is documented inline. (`docs/architecture.md`)

### Added

- Docker inventory inspector reports containers and images. The
  placeholder that emitted only `inventory.docker.unavailable` has
  been replaced with real Engine API reads via a unix-socket
  http.Client (stdlib only, no Docker SDK dependency). New
  ProbeIDs: `inventory.docker.container` (with `image`,
  `image_digest`, `state`, `status`, `created_at`, `labels`) and
  `inventory.docker.image` (with `digest`, `repo_tags`,
  `size_bytes`, `created_at`). Non-2xx Engine responses surface as
  `inventory.docker.error` with the status code attached;
  permission-denied / socket-missing branches keep emitting
  `inventory.docker.unavailable`. Read-only contract enforced: the
  inspector issues only GET calls to `/v1.41/containers/json` and
  `/v1.41/images/json`.
  (`internal/probe/inventory/docker/docker.go`,
  `internal/probe/inventory/docker/client.go`,
  `internal/probe/inventory/docker/docker_test.go`,
  `docs/agent.md`, `docs/findings.md`)

- Remote-mode `wanderer agent` no longer drops findings on a
  transient network outage. New `internal/agent/outbox.go`
  persists any batch that fails to POST after three retries (0s /
  250ms / 1s with jitter) to a local directory (default
  `/var/lib/wanderer/agent/outbox`), and drains the directory on
  the next tick before collecting fresh findings. The outbox is
  bounded by `core.outbox_max_bytes` (default 100 MiB); a corrupt
  spool file is renamed `<name>.corrupt` and skipped so it does
  not block the drain. Configuration knobs `core.outbox_dir` and
  `core.outbox_max_bytes` are optional. Refactored
  `agent.Remote.Send` exposes `MarshalBatch` and `SendBytes` so
  the outbox can re-POST the exact same body without
  re-marshalling.
  (`internal/agent/outbox.go`, `internal/agent/outbox_test.go`,
  `internal/agent/remote.go`, `internal/agent/config.go`,
  `cmd/wanderer/agent.go`, `docs/agent.md`)

- Egress classifier vendor / region table is now loaded from
  `internal/probe/egress/vendors.yaml` via `//go:embed`, so a
  contributor adding a new log shipper or webhook host edits a YAML
  file rather than Go source. Operators can override the embedded
  list with `wanderer agent --vendors <path>` or
  `WANDERER_VENDORS=<path>`; a missing file, malformed YAML, invalid
  regex, or missing required key is fatal at agent start. The
  classifier publishes the per-vendor `rule_id` from the YAML in
  the Finding's `classifier_rule` attribute, so an auditor can
  trace each verdict back to the line that produced it.
  (`internal/probe/egress/vendors.go`,
  `internal/probe/egress/vendors.yaml`,
  `internal/probe/egress/vendors_test.go`,
  `internal/probe/egress/classify.go`,
  `cmd/wanderer/agent.go`, `docs/egress.md`)

- `pkg/models.TargetKind` distinguishes a perimeter `domain` Target
  from an agent `host` Target. Public-domain validation (one dot,
  printable ASCII) still applies to `domain` Targets; `host` Targets
  use the new `NormaliseHost` helper which trims and lowercases but
  drops the TLD requirement so a bare Linux hostname like
  `webapp-01` round-trips. Migration 003 adds a `kind` column to the
  `targets` table with a `'domain'` default. New `Store.GetTarget`
  exposes the row, including Kind, by ID.
  (`pkg/models/target.go`, `pkg/models/target_test.go`,
  `internal/store/migrations.go`, `internal/store/sqlite.go`,
  `internal/store/store_test.go`, `cmd/wanderer/agent.go`,
  `docs/agent.md`)

### Fixed

- `wanderer agent` no longer fails with `domain: "<hostname>" has no
  TLD` on hosts whose `hostname` is a bare label. The agent now sets
  `Kind: TargetKindHost` on the bootstrap Target, so the validator
  uses host normalisation rather than the public-domain rules.

- Assessor rules that count or aggregate Findings by `ProbeID` now skip
  meta-rows (an `error` attribute, `no_answer: true`, or
  `unavailable: true`) before treating them as evidence. The
  `dictu.data_ai.mx_present` smoke test on a `.invalid` domain
  previously returned `voldoende` because the lookup-error `dns.mx`
  rows counted as configured mail exchangers; it now returns
  `onbekend`. New helper `assessor.IsEvidenceLike` plus a regression
  test pin the invariant. The probe-level meta-finding convention is
  now documented in `docs/findings.md`.
  (`internal/assessor/rule.go`, `internal/assessor/rule_test.go`,
  `internal/assessor/dictu/rules.go`,
  `internal/assessor/dictu/rules_test.go`, `docs/findings.md`)
- DICTU rule `dictu.juridisch.apex_ip_eea` looked for `ProbeID == "dns.A"`
  / `"dns.AAAA"` while the DNS probe emits the lowercase variants
  documented in `docs/findings.md`. The unit test agreed with the rule
  (also using uppercase) so the bug was invisible in CI but caused
  every real scan to return Onbekend for the apex jurisdiction. Rule
  and unit test are now both lowercase, with a comment pinning the
  invariant. (`internal/assessor/dictu/rules.go`,
  `internal/assessor/dictu/rules_test.go`)
- Scanner now feeds DNS- and HTTP-discovered hosts into the IP probe
  before it runs. Previously the IP probe only resolved `target.Domain`
  and the operator-provided `target.Related`; MX hosts (`dns.mx`) and
  third-party hosts (`http.third_party`) found by other probes were
  never looked up, so `dictu.juridisch.mx_vendor_jurisdiction`,
  `dictu.technologie.third_parties_eea`, and the third-party half of
  `dictu.technologie.no_us_hyperscaler` silently returned Onbekend on
  every real scan. New `expandRelatedFromFindings` helper builds an
  enriched target only for the IP probe (other probes still see the
  original target). `buildProbes` now orders the IP probe last so the
  HTTP probe has had a chance to discover third parties.
  (`internal/scanner/scanner.go`, `cmd/wanderer/scan.go`)

### Added

- SSRF guard for outbound probe traffic. `internal/probe/ssrf.go` ships
  a `SafeDialer` plus a static `*net.IPNet` table covering loopback,
  link-local, RFC1918, CGNAT, IPv6 ULA, IPv6 link-local and the cloud
  metadata IPs (169.254.169.254, fd00:ec2::254). The dialer is wired
  into the HTTP probe's `http.Client.Transport` and the TLS probe's
  `tls.Dialer`. `wanderer scan` and `wanderer serve` gain
  `--allow-private-targets` (default off — private blocked); the
  `POST /scans` handler refuses requests whose domain resolves only to
  private addresses unless the flag is set, so an authenticated API
  client cannot turn the scanner into an internal-network probe. Unit
  tests cover one blocked family per address class.
  (`openspec/changes/fill-mvp-gaps` goal #3)
- Passive subdomain discovery. `internal/probe/tls/tls.go` extracts SAN
  names from the apex certificate and emits each as a `dns.subdomain`
  Finding tagged `source: "ct_log"`. `internal/probe/dns/subdomains.go`
  probes 18 common prefixes via the system resolver; resolving names
  emit `dns.subdomain` Findings tagged `source: "prefix_probe"`. When
  every prefix resolves to the same IP set, the probe collapses the
  noise into a single `dns.subdomain.wildcard` Finding rather than 18
  spurious hits. Discovered subdomains feed pass 2 of the scanner via
  `expandRelatedFromFindings` so the IP probe resolves them. Unit tests
  cover hit / miss / wildcard against a fake resolver.
  (`openspec/changes/fill-mvp-gaps` goal #4)
- Amass importer. `internal/scanner/amass.go` parses the JSONL produced
  by `amass enum -json` and returns the FQDN list to `target.Related`.
  `wanderer scan --amass <file>` and `POST /scans` `amass_json` field
  (server-local path only; no inline body) wire it through. Malformed
  JSON aborts at startup so an unenriched scan does not silently look
  fine. `docs/operator.md` documents the
  `amass enum -passive -d … -json out.json && wanderer scan … --amass
  out.json` recipe.
  (`openspec/changes/fill-mvp-gaps` goal #5)
- EU CSF / SEAL rule pack. New `pkg/models.SealLevel` (SEAL0–SEAL4)
  and `models.Framework` enum. `internal/assessor/eucsf/rules.go`
  ships five rules — `eucsf.sov2.cert_issuer_eu`,
  `eucsf.sov2.apex_jurisdiction`, `eucsf.sov3.mx_jurisdiction`,
  `eucsf.sov4.operational_eu`, `eucsf.sov6.no_us_hyperscaler` —
  with at least one EU and one non-EU fixture per rule.
  `wanderer assess --framework dictu|eucsf|both` selects which pack
  runs, plumbed through CLI flags, the HTTP API, and persistence.
  `docs/assessor.md` carries the SEAL section with the score-mapping
  table.
  (`openspec/changes/fill-mvp-gaps` goal #6)
- Read-only operator UI on `wanderer serve` (`--ui`). Mounts at
  `/ui/` with three GET-only routes: `/` (target overview with
  latest scan and worst-dimension framework score per target),
  `/scans/{id}` (findings grouped by probe prefix), and
  `/targets/{id}/drift` (drift findings since `?since=<RFC3339>`).
  Templates and CSS are embedded via `go:embed`; no JavaScript, no
  external assets. Authentication is HTTP Basic against an htpasswd
  file (`--ui-htpasswd <path>` or `WANDERER_UI_HTPASSWD`). Only
  bcrypt entries (`$2a$`/`$2b$`/`$2y$`) are accepted; `$apr1$` MD5,
  `{SHA}` SHA-1, `$5$` and `$6$` crypt entries are rejected at
  startup with an explicit "use bcrypt (`htpasswd -B`)" error — one
  algorithm = one battle-tested verification path. The htpasswd
  file is re-read on every request so credentials can rotate without
  restarting. `ui_test.go` includes a static-analysis check that
  greps the package source for `r.Post|Patch|Delete|Put` and fails
  the build if any mutating handler appears, locking in the
  read-only invariant. Default is off; the JSON API and existing
  flags are unchanged. (`internal/ui/`, `cmd/wanderer/serve.go`,
  `openspec/changes/fill-mvp-gaps` goal #7)
- Concurrent pass-1 execution. The scanner now fans out DNS, TLS,
  HTTP and WHOIS probes via `errgroup.WithContext`, each wrapped in a
  `defer recover()` that converts panics to a nil group error so a
  single misbehaving probe cannot poison the whole scan. Pass 2
  (the IP probe) still runs serially after the pass-1 join so it can
  consume hosts the others discovered. The global budget remains a
  single `context.WithTimeout`. A wall-clock test asserts pass 1
  finishes in roughly the slowest probe's duration, not the sum.
  (`openspec/changes/fill-mvp-gaps` goal #8)
- RDAP / WHOIS probe. `internal/probe/whois/whois.go` calls
  `https://rdap.org/domain/<domain>` (injectable for tests) with a
  5-second timeout and walks the vCard array per RFC 7483. Emits
  `whois.registrant` (country, juridisch dimension, finding
  severity), `whois.registrar` (name, info), and a single
  `whois.unavailable` Finding on any network/parse failure so the
  rest of the scan continues. New rule
  `dictu.juridisch.registrar_jurisdiction` consults the registrant
  country (EEA → soeverein, outside-EEA → afhankelijk, absent →
  onbekend). Stdlib `net/http` only — no WHOIS-43 socket, no
  third-party SDK. Tests cover happy path, HTTP error, malformed
  JSON, empty domain, and no-entities cases.
  (`openspec/changes/fill-mvp-gaps` goal #9)
- Numbered schema migrations. `internal/store/migrations.go`
  introduces a `schema_migrations` table and a versioned migration
  slice. The current schema becomes migration 001 (idempotent
  `CREATE TABLE IF NOT EXISTS` so pre-migration databases adopt it
  cleanly) and the `findings.source_modus` ALTER becomes migration
  002. The previous string-matched ALTER tolerance is removed in
  favour of the migration runner. Tests cover: fresh DB applies all
  migrations, a DB at version N applies only N+1..M, and a failing
  migration rolls back without recording its version.
  (`openspec/changes/fill-mvp-gaps` goal #10)
- End-to-end integration tests in `internal/assessor/dictu/integration_test.go`
  that drive the real DNS probe through the real DICTU rules. Pins the
  probe-ID/assessor-ID contract for `apex_ip_eea`,
  `mx_vendor_jurisdiction`, and `third_parties_eea` so future drift on
  either side breaks the build instead of silently returning Onbekend.
- Scanner unit test `TestIPProbeReceivesDiscoveredHosts` pins the
  fan-out invariant: the IP probe sees discovered hosts in
  `target.Related`, while other probes continue to see the original
  target.
- Egress probe: agent-side observation of where data goes when it
  leaves the host. New `internal/probe/egress/` walks configured
  config files (`.env`, `.yaml`, `.yml`, `.toml`, `.ini`, `.conf`,
  `.json`), `/proc/<pid>/environ`, and systemd unit files, then
  classifies discovered URLs and hosts into nine categories
  (`object_storage`, `database`, `smtp`, `oidc`, `log_shipper`,
  `webhook`, plus `unknown` / `unconfigured` / `error`). Findings
  are tagged with `SourceModus = "egress"` and carry a
  `classifier_rule` attribute. Optional GeoLite2 annotation adds
  ASN/organisation/country per host. The redactor runs in front of
  every value emission path; ADR-0008 documents the contract and
  test discipline. Symlinks pointing outside the configured root
  are not followed. `docs/egress.md` is the operator guide.
  (`openspec/changes/add-egress-probe`)
- Inventory probe + agent: new `wanderer agent` subcommand running
  host-side inspectors (systemd, dpkg, rpm; nextcloud opt-in;
  docker as graceful-unavailable placeholder pending a follow-up).
  New `pkg/models.SourceModus` field tags every Finding with its
  origin (perimeter / inventory / egress / drift) so the assessor's
  completeness calculation can distinguish them; the `findings`
  table gains a `source_modus` column with idempotent migration.
  New `POST /scans/{id}/findings` endpoint authenticates agents via
  HMAC-SHA256 over `<timestamp>\n<body>` with a ±5-minute replay
  window (constant-time compare; single 401 surface). Agent config
  in YAML covers local-mode (writes to a shared SQLite file) and
  remote-mode (HMAC-signed HTTPS to a central core). ADR-0007
  records the trust-model rationale; `docs/agent.md` is the
  operator guide.
  (`openspec/changes/add-inventory-probe`)
- Scheduling + drift: an in-process cron scheduler runs alongside
  `wanderer serve` (via `--schedules <file>`), invoking
  `scanner.Scan` per tick and feeding each new scan to the drift
  engine. New `internal/drift/` engine compares consecutive scans of
  the same target and emits `drift.*` Findings (TLS issuer, days
  left, MX/NS sets, IP country, HTTP third parties). New
  `wanderer diff <scan-a> <scan-b>` CLI prints would-be drift as
  markdown without touching the store. New `GET
  /targets/{id}/drift?since=<RFC3339>` HTTP endpoint. ADR-0006
  records why the scheduler is in-process rather than a Kubernetes
  CronJob.
  (`openspec/changes/add-scheduling`)
- MCP server: `wanderer mcp` subcommand speaks the Model Context
  Protocol over stdio (line-delimited JSON-RPC 2.0). Exposes five
  tools (`scan_domain`, `get_scan`, `list_scans`, `assess_scan`,
  `get_assessment`) and the `wanderer://` resource family for
  reading scans and assessments. Hand-rolled dispatcher in
  `internal/mcp/`; no new dependencies. ADR-0005 records the
  transport choice. `docs/mcp.md` carries the install snippet for
  Claude Desktop / Claude Code.
  (`openspec/changes/add-mcp-server`)
- Exporters: `wanderer export <findings|scans|assessments>` subcommand
  with `--format csv|jsonl`, file or stdout output, and composable
  `--scan`, `--probe`, `--dimension`, `--since`, `--until` selectors
  pushed down to the SQL query. Adds `internal/export/` writers,
  `store.ListFindings` / `ListScans` / `ListAssessments` query
  helpers (with a `Selectors` type), and `docs/exporters.md` for
  recipes (Excel, jq, Grafana, diff).
  (`openspec/changes/add-exporters`)
- Assessor: `pkg/models.Assessment` and its supporting types
  (`Score`, `Completeness`, `DimensionScore`, `Rationale`), the
  `internal/assessor` rule engine, the DICTU MVP rule set under
  `internal/assessor/dictu/` (10 rules across four dimensions),
  markdown/JSON/text report renderers, the `wanderer assess` CLI
  subcommand, and `POST /scans/{id}/assessments` +
  `GET /assessments/{id}` on the HTTP API. Assessments are persisted
  in a new `assessments` table via `store.CreateAssessment`,
  `store.GetAssessment`, and `store.ListAssessmentsForScan`. ADR-0004
  records why rules are Go functions rather than a DSL.
  (`openspec/changes/add-assessor`)
- Maintainability baseline: `CHANGELOG.md`, `CODEOWNERS`, the
  `docs/decisions/` ADR folder with seed records for the OpenSpec
  workflow, API stability classes, and dependency policy, plus
  `docs/maintainability.md` as the single contributor entry point.
  (`openspec/changes/add-maintainability-baseline`)
- Initial MVP scanner suite: DNS (A/AAAA/MX/NS/CNAME/TXT/CAA), TLS
  chain + crt.sh certificate-transparency lookup, IP→ASN→country via
  a local MaxMind GeoLite2 database, and HTTP apex fetch with
  third-party resource extraction. Findings persist to SQLite through
  `modernc.org/sqlite`, the `wanderer` CLI exposes `scan` and `serve`
  subcommands, and a chi-based HTTP API serves `POST /scans` and
  `GET /scans/{id}`. slog (JSON) and Prometheus counters are wired
  into scanner and probes; OpenTelemetry traces were intentionally
  deferred (see `docs/observability.md`).
  (`openspec/changes/archive/2026-04-24-init-mvp-scanners`)

[Unreleased]: https://github.com/MWest2020/wanderer/commits/main
