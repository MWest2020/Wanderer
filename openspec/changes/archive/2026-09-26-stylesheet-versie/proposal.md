# Change: stylesheet-versie

## Why

v0.12.0 (new colour palette) went live on 2026-09-26, and the public address
kept serving the old stylesheet. Measured the same minute:

- `https://wanderer.westerweel.work/ui/static/main.css` → `cache-control:
  max-age=14400`, `cf-cache-status: HIT`, `age: 4046` — an old copy from
  Cloudflare's edge, while the pod served the new file.
- Two requests in a row got two different answers: one edge had the new file,
  the next still the old one.

So after every release, a person can see new pages with old colours and
layout for up to four hours, in their browser and at the edge. All ten
templates link the stylesheet by a fixed name
(`<link rel="stylesheet" href="/ui/static/main.css">`), so no cache can know it
changed.

## What changes

The stylesheet is linked with a version derived from its content:
`/ui/static/main.css?v=<first 12 hex of sha256>`. The hash is computed once at
startup from the embedded file. A new stylesheet has a new address; an
unchanged one keeps its address and stays cached, which is what the cache is
for.
