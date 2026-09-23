# Proposal: standards-feed — the standards dimension, fed on the live instance

## Why

v0.9.0 shipped the `standards` dimension (DNSSEC, SPF/DKIM/DMARC,
STARTTLS/DANE, IPv6, RPKI, TLS configuration). On the live instance it has
never had a finding: nothing imports one. Every standards rule reads "niet
gemeten" on every target. Measured 2026-09-23: `wanderer version` v0.9.2 in the
pod, zero standards findings, and the demo page says nothing about DNSSEC or
mail security.

The pieces exist. netnl measures and exports `netnl-findings/v1`
(`internetnl results <id> --format findings`); Wanderer imports it
(`wanderer import internetnl <file>`). What is missing is the path between the
two on the cluster.

## The constraint that shapes it

The Wanderer Deployment says it in its own manifest:
`# RWO-volume + SQLite: nooit twee schrijvers`. A CronJob that mounts the
volume and runs `wanderer import` would be a second writer. So the import has
to go through the running server, as an HTTP route — the same place
`POST /scans` already writes from.

## What changes

1. **Wanderer: `POST /imports/internetnl`.** Body is a `netnl-findings/v1`
   document. It runs the same import as the CLI — one implementation, not two
   — and answers with the same counts: imported, skipped as unknown target,
   skipped as already imported from that file.
2. **The route needs a token, and refuses without one.** The full REST API is
   reachable on the tailnet without authentication (`tailscale.yaml`). An open
   import route would let anything on the tailnet write DNSSEC or mail verdicts
   into the reports. `WANDERER_IMPORT_TOKEN` set → the route checks
   `Authorization: Bearer`; unset → every request is refused, and the startup
   log says the import route is inactive. Same shape as the agent intake, which
   refuses everything while no agent is enrolled.
3. **homelab: a weekly CronJob** that runs the unchanged netnl CLI against the
   facade — web and mail — exports findings, and posts them to that route.
   Its own netnl tenant, so the facade's audit trail shows who measured.

## Which domains

Only hosts the operator controls. That is netnl's own rule for the facade
("meet alleen hosts die je zelf beheert"), stated in the existing measure
CronJob. The Wanderer fleet also watches `rijksoverheid.nl`, `ncsc.nl` and
`digid.nl`; those stay "niet gemeten" on the standards dimension, and that is
honest, not a gap.

## Not in scope

The v2 facade webhook (`2026-09-22-internetnl-facade-v2-stub.md`) — the facade
pushing the file itself. This change puts the receiving end in place; the
webhook can later post to the same route without a change here.
