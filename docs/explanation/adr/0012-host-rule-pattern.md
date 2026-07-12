---
status: draft
last_reviewed: 2026-07-12
---

# 0012 — Vendor-jurisdiction rule pattern

**Status:** Accepted, codified 2026-05-11 after the pattern
landed across four rule families.

**Context.** Wanderer's assessor scores Findings against rule
sets per framework. The original perimeter rules (TLS issuer
jurisdiction, IP/ASN country, third-party hostnames) were
hand-rolled — each rule had its own match logic and its own
verdict shape. As the agent modus matured and started landing
inventory + Nextcloud + Docker findings, three new rule families
joined in quick succession:

- `add-host-side-scoring` (2026-05-11) — wand.host.no_us_telemetry_*
- `add-nextcloud-as-target` (2026-05-11) — wand.nextcloud.* + sov6.nextcloud_supply_chain
- `add-container-image-sovereignty` (2026-05-11) — wand.docker.* + sov6.container_supply_chain

Each of those landed reading a different probe family but
converged on the same shape: "classify the subject of an
inventory Finding against a curated vendor list, score the host
based on whether any subject hits the list". The fourth landing
made the convention real; this ADR records it so a fifth or
sixth rule family lands by copy-paste instead of re-derivation.

**Decision.** Vendor-jurisdiction rules SHALL follow these
conventions:

1. **The match list is YAML, embedded via `go:embed`.** Lists
   live under `internal/assessor/<list>.yaml`. The egress
   probe's `internal/probe/egress/vendors.yaml` is the source
   pattern. Operators can eyeball one file; contributors edit
   one file; reviewers diff one file. Embedded so the binary
   builds without runtime file dependencies; no
   operator-override mechanism until a real need surfaces.

2. **The rule cites inspected Finding IDs as negative evidence
   on the soeverein branch.** The assessor engine
   (`engine.go::scoreDimension`) forces verdicts without an
   `Evidence` slice back to `onbekend` — a soeverein verdict
   that carries zero IDs is silently degraded. Each rule pack
   ships a `sampleEvidence` helper that returns up to 10
   inspected Finding IDs from the soeverein branch. The
   verdict text MUST include the inspected count ("inspected
   1790 packages") so an operator reading the verdict sees the
   scale of negative evidence at a glance.

3. **Verdict text on the afhankelijk branch names the matched
   subject AND the vendor of record.** Operators acting on a
   verdict need both — the subject ("which thing on my host")
   and the vendor ("who is downstream"). Single-hit verdicts
   read `"<probe-shape> <subject> matches <vendor>"`;
   multi-hit verdicts roll them up but keep the subject list
   inline rather than hiding it behind Evidence.

4. **One wand rule per probe family; one EUCSF roll-up.** The
   wand split keeps each probe family's verdict independently
   readable — an operator who runs Docker but no Nextcloud
   sees the Docker rule fire and the Nextcloud rule stay
   onbekend, rather than a single verdict that conflates both.
   The EUCSF pack rolls them up under one SEAL supply-chain
   observation per the SEAL convention.

5. **Helpers stay local to each rule pack, not extracted.**
   `sampleEvidence` is duplicated across `wand/host_rules.go`,
   `wand/nextcloud_rules.go`, and `wand/docker_rules.go`. The
   bodies are five-line slices; a shared helper would save 15
   lines at the cost of one cross-package import and one
   indirection-by-name. Two reviewers can eyeball each pack's
   policy in one place. Reconsider only if a fifth pack joins.

**Consequences.**

- **Adding a fifth rule family is copy-paste.** The host rules
  are the cleanest reference: one new YAML, one new rules file,
  two unit tests (rule + loader), one fixture extension, one
  Playwright spec. The existing four families landed in roughly
  this order and each took one session.
- **Vendor list drift is a documented hazard.** Acquisitions
  shift jurisdiction (Quay → Red Hat → IBM; GitHub →
  Microsoft); the lists are small enough to keep current by
  hand. A "last reviewed" header in each YAML would help — not
  implemented yet, file as a follow-up if drift becomes a
  problem in production.
- **The "onbekend on empty Evidence" engine rule is now a
  hard contract.** Any new rule that conditionally returns
  soeverein-without-citations is silently broken. The pattern
  documented here prevents that, but a reviewer noticing
  "this rule returns soeverein but its Evidence is empty"
  should treat it as a bug.

**Alternatives considered.**

- *Shared sampler helper in `internal/assessor`*: rejected —
  five lines per pack is cheap; one import + one indirection
  per pack costs more readability than it saves.
- *Operator-override path for the YAML lists*: rejected for
  now — no operator has asked, embedded YAML is reviewable in
  the binary's source. Add the env var hook the day someone
  needs it.
- *One mega-rule reading every probe family*: rejected —
  operators want to see "which family tripped". A single rule
  output is harder to act on than three independent ones.
- *Cross-pack DRY (shared helpers in `internal/assessor`)*:
  the host telemetry YAML loader is already shared (lives in
  `internal/assessor/host_telemetry.go`). Beyond that, each
  pack stays self-contained. The cost of cross-pack coupling
  outweighs the saved lines.

**Not covered by this ADR.**

- Rules that don't read inventory subjects (e.g. drift-modus
  rules, perimeter aggregate rules). They follow the same
  Evidence convention but have their own shape conversations
  in their own ADRs (or future ones).
- The Nextcloud `country` annotation, which comes from a
  `HostResolver` interface the inspector accepts directly
  rather than a YAML list. Same Evidence + verdict-text
  contract; different match source. ADR-0013 if and when
  that shape grows beyond Nextcloud.

**See also.**

- ADR-0004 (Assessor rule engine) — the underlying
  rule/finding contract this pattern extends
- `openspec/specs/assessor/spec.md` — the
  "Host-rule soeverein verdicts cite negative evidence"
  requirement that formalises the negative-evidence contract
