# Proposal: Nextcloud as output — publish Wanderer artefacts INTO Nextcloud

> **Status:** Design pass — awaiting Mark's scope call.
> Direction (2) of the four-direction Nextcloud integration
> proposal (direction 1, "Nextcloud as target", is shipped).
> No code lands until Mark picks a publication surface (the
> three options in Q1 below).

## Intent

The Wanderer team is a Conduction-style delivery shop that
runs Nextcloud as its day-to-day collaboration platform.
Today Wanderer's outputs — scans, Assessments, drift diffs —
live in the SQLite store and the `/ui/` browse layer. An
operator who wants to share a verdict with a customer has
to: open the UI, screenshot the page, paste it in Talk /
email / Deck.

This change publishes Wanderer artefacts directly into the
customer's Nextcloud, so a scheduled scan results in a
shareable artefact in their organisational Nextcloud space
without the screenshot step.

## Scope (depends on Q1)

Three publication surfaces, listed cheapest to richest:

1. **WebDAV file drop.** Wanderer writes a per-scan JSON
   bundle (or rendered HTML) to a configured Nextcloud
   directory via WebDAV. Operator sees
   `/Files/Wanderer/<org>/<scan_id>.json` in their Nextcloud.
   Pros: WebDAV is stdlib + basic-auth, no Nextcloud app
   needed. Cons: file-only, no in-app awareness.

2. **Talk room notification.** Wanderer posts a one-liner
   "scan complete: 3 afhankelijk on conduction.nl" to a
   configured Talk room. Pros: where operators already work.
   Cons: requires the operator to have a bot user in their
   Nextcloud + Talk app installed.

3. **Deck card per Assessment.** Wanderer creates a Deck card
   per scan in a configured board, including the verdict pill
   + a deep link to `/ui/scans/<id>/assessment`. Pros: visual
   triage surface, fits existing Deck workflows. Cons:
   requires Deck app + per-customer board configuration.

## Open questions

1. **Which publication surface?** (Q1: file drop / Talk room /
   Deck card / combination). Recommendation: **file drop
   first.** WebDAV is universally available on every Nextcloud
   install, doesn't need extra apps, and lets the operator
   choose their own downstream workflow (Talk auto-notify on
   `/Files/Wanderer/` changes, etc.). Talk + Deck land later
   as opt-in adapters.

2. **Push or pull?** Wanderer pushes to Nextcloud as a
   post-scan hook, OR Nextcloud polls Wanderer via the
   existing MCP / HTTP API. Recommendation: **push** — keeps
   the Nextcloud side passive, matches the existing
   `wanderer scan` cron model.

3. **What format does the file drop emit?** JSON-LD (machine
   readable), Markdown (human readable), HTML (rendered, with
   the same templates as `/ui/`), or all three? Recommendation:
   **JSON-LD + Markdown.** JSON-LD is durable + machine-
   diffable; Markdown opens nicely in Nextcloud's text app.
   HTML is the UI's job and shouldn't be duplicated.

4. **Authentication.** App password or full OAuth2? App
   passwords are simpler (basic auth + per-app token) and
   match Nextcloud's existing automation patterns. OAuth2 is
   richer but adds a token-refresh dance Wanderer doesn't
   need. Recommendation: **app password**, configured in
   `serve.yaml`'s new `nextcloud:` block.

## Risks

- **Customer-side configuration burden.** Operators need to
  generate an app password + decide on a target directory.
  Pre-launch doc effort scales with customer count.
- **WebDAV failure modes.** Customer Nextcloud goes down /
  rotates the app password / fills disk. Wanderer needs a
  bounded retry policy + a clear local fallback ("file
  publish failed, output still in `/ui/`").
- **Privacy.** A scan bundle contains TLS certificate
  Subject names, MX records, ASN data. The publication
  contract MUST honour the redaction rules in
  ADR-0008 (egress-redaction).

## Not in scope

- A "Wanderer Nextcloud app" — that's direction (4)
  (`propose-nextcloud-marketplace-app`).
- Nextcloud as an authentication provider for `wanderer
  serve --ui` — that's direction (3)
  (`propose-nextcloud-as-oidc`).
- Bi-directional sync (Nextcloud → Wanderer). Wanderer reads
  Nextcloud already via the agent inspector + the
  add-nextcloud-as-target rules; the publication direction
  is one-way.

## Parallel-safe

Touches `internal/export/` (new Nextcloud exporter alongside
csv / jsonl), `internal/serveconfig/` (new `nextcloud:`
block), docs. No schema change, no UI change, no agent
change. The exporter is opt-in via the new config block.
