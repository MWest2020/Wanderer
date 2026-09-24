# Change: vloot-beheren-bereikbaar

## Why

Mark, 2026-09-23/24: "schiphol.nk eruit". He removed it in the Nextcloud
ExApp, then tried on `wanderer.westerweel.work` — and it never happened. The
prod database shows his login landing (2026-09-24 08:51 UTC) and nothing after.

Measured on a copy of the prod data (v0.11.0): **`/ui/` has no link to the
fleet manager.** The only route was `/ui/` → the Organisations table →
`/ui/orgs/default` → "vloot beheren". v0.11.0 hides the Organisations table
when there is one organisation (`vloot-in-beeld`, task 1.6), which removed the
last step of that route. Before that, the route existed but took two
unlabelled hops.

## What changes

`/ui/` links to fleet management when the fleet it shows belongs to one
organisation: a "vloot beheren →" link in the fleet section, pointing at that
organisation's `/ui/orgs/{slug}/fleet`. With more than one organisation, the
Organisations table stays the way in, and each organisation's dashboard has
its own link, as today.
