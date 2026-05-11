# Proposal: Nextcloud integration — direction-finding

> **Status:** Design pass — DIRECTION still open. The four
> open questions below are the gate. No code lands until
> Mark answers Q1 (which direction).
>
> Mark mentioned "uiteindelijk integrate met nextcloud" on
> 2026-05-10. The phrasing is open enough that this proposal
> reads four plausible directions and asks for a pick before
> investing implementation effort.

## Intent

Wanderer already has a thin Nextcloud inventory inspector
(`internal/probe/inventory/nextcloud/`) that shells out to
`occ app:list --output=json` and emits one Finding per
installed Nextcloud app. The MVP scope was "the parser
works"; nothing scores those Findings yet and the inspector
is gated on `inspectors.nextcloud.enabled` in the agent
config — disabled by default.

A real Nextcloud integration could go four directions:

1. **Nextcloud as a target** — score the sovereignty posture
   *of* the Nextcloud instance the agent is inspecting.
   Trusted domains, external storage backends (S3 endpoint
   URLs → ASN/country), OIDC IdP jurisdictions, federation
   peers, marketplace app jurisdictions. This is the
   direction with the strongest fit to Wanderer's core
   mission.

2. **Nextcloud as the output** — publish Wanderer scans /
   Assessments *into* a Nextcloud instance as files
   (`/Wanderer/<org>/<scan>.json`), Deck cards, or Talk room
   notifications. Operators read results where they already
   work.

3. **Nextcloud as auth** — accept Nextcloud as the OIDC
   provider for `wanderer serve --ui`. Removes the htpasswd
   stop-gap, lets operators reuse their existing Nextcloud
   accounts.

4. **Nextcloud as distribution** — ship Wanderer (the
   server) as a Nextcloud app, installable via the
   marketplace. Heavy: the runtime would be PHP-bridged Go,
   or a sidecar pattern.

These are not all mutually exclusive, but they have very
different implementation costs and timelines. Option 4 is
the heaviest by an order of magnitude.

## Recommendation

Default to **(1) Nextcloud as a target**. It is the closest
fit to Wanderer's mission ("score the sovereignty posture
of digital infrastructure"), reuses the existing inspector
plumbing, lands as one or two new probes + a small
host-rule wave, and a Conduction customer running Nextcloud
gets immediate value. Options 2 and 3 are useful follow-ups
but address operator ergonomics rather than the core
sovereignty story. Option 4 is a separate product decision.

## Scope (if direction = 1)

**In scope:**

- Extend `nextcloud` inspector with three new probe queries:
  - `occ config:list system trusted_domains` → one
    `inventory.nextcloud.trusted_domain` Finding per entry
  - `occ config:list system objectstore` → one
    `inventory.nextcloud.objectstore` Finding per S3-style
    backend with `endpoint` + `bucket` (Wanderer's existing
    geoip annotation enriches with ASN/country)
  - `occ user_oidc:provider list` → one
    `inventory.nextcloud.oidc_provider` Finding per
    configured IdP with `issuer_url` (assessor reuses the
    existing `tls.issuer` jurisdiction logic)
- Three host-shaped rules:
  - `wand.nextcloud.objectstore_eu` — afhankelijk when any
    S3 backend resolves to a US-headquartered hyperscaler
  - `wand.nextcloud.oidc_provider_eu` — afhankelijk when an
    OIDC IdP's issuer URL resolves to a US ASN
  - `eucsf.sov6.nextcloud_supply_chain` — combined SEAL
    analogue
- Operator doc walkthrough (one section in
  `docs/operator.md`).

**Out of scope (deferred to follow-ups whether (1) ships or
not):**

- Nextcloud Talk room notifications (option 2)
- OIDC consumer in `wanderer serve` (option 3)
- Marketplace app packaging (option 4)
- Federation peer scoring (each peer's TLS issuer
  jurisdiction → another rule; lands after the core three
  work)

## Open questions

1. **Which direction?** (1, 2, 3, 4, or a combo).
   Recommendation: ship (1) first; queue (2) and (3) as
   separate proposals after (1) lands.

2. **Where does the new probe data flow?** The current
   `nextcloud` inspector emits `inventory.nextcloud.app`
   Findings. New probe queries would extend the same
   ProbeID family (`inventory.nextcloud.trusted_domain`
   etc.) or move under a new family
   (`config.nextcloud.*`)? Recommendation: extend the
   existing family — the inventory modus is the natural
   home for "facts about what the host is configured to
   trust".

3. **How do `wanderer agent` and `wanderer scan` differ
   here?** Today only the agent runs the nextcloud
   inspector. The objectstore / OIDC probes still make sense
   in agent context (they need `occ` access and the file is
   `/etc/nextcloud/config.php` on the host). For a remote
   `wanderer scan` against `https://cloud.example.nl` Mark
   may want a separate perimeter probe that hits well-known
   Nextcloud endpoints (`/status.php`, `/ocs/...`) — that's
   a different scope. Recommendation: this proposal stays
   agent-side; add a separate `propose-nextcloud-perimeter`
   change later if the remote angle is also desired.

4. **OIDC user_oidc app dependency.** The OIDC probe
   assumes the `user_oidc` app is installed (it's the
   official Nextcloud OIDC client). Many production
   installs use `oidc_login` (community) or `social_login`
   instead. The probe needs a fall-back path or it should
   declare an explicit app dependency in the proposal.
   Recommendation: probe `user_oidc` first; emit
   `inventory.nextcloud.oidc.unavailable` with the
   discovered alternative app name when it's missing, so
   the operator knows the gap is data, not a Wanderer bug.

## Passive / active boundary

All probe additions are read-only shell-outs to `occ`.
No writes, no network calls except the existing geoip
annotation, no schema change beyond the (possible) new
ProbeID family if Q2 picks "new family".

## Risks

- **Direction churn.** If Mark picks (2) after this lands,
  none of the work transfers — option 2's surface is a
  publisher, not a probe extension. Mitigation: do not
  start (1)'s implementation until Q1 is answered.
- **`occ` availability.** On hosts where Nextcloud runs in
  Docker, the agent's `occ` path needs a container exec
  shim. The existing inspector dodges this with the
  `OccPath` + `RunAs` config; the new queries inherit the
  same constraint. Mitigation: explicit operator-doc
  guidance + a clear unavailable-reason finding.
- **PHP CLI as a stable contract.** `occ` output format
  has shifted between Nextcloud majors (24, 26, 28). The
  existing inspector parses 28; we should add a version
  probe (`occ status --output=json` exposes
  `versionstring`) and gate parsing on it. Mitigation:
  ship the version probe in this wave and have each parser
  declare which majors it covers.

## Parallel-safe

Touches `internal/probe/inventory/nextcloud/`,
`internal/assessor/wand/`, `internal/assessor/eucsf/`,
docs. No UI change required (the rule catalogue picks up
new rules automatically). Migration-free.
