# Change: drift-land-per-adres

## Why

The first scheduled scans on v0.12.1 (2026-09-28) produced ten
`drift.ip.country_changed` findings: seven for Cloudflare's
`jamie`/`lennox.ns.cloudflare.com` (westerweel.work), three for
`ns0.rijksoverheidnl.com` (rijksoverheid.nl, ncsc.nl, digid.nl). Mark asked to
measure how often it happens before deciding on an "anycast" treatment.

Measured on a copy of the prod data (47 scans, 21 consecutive perimeter-scan
pairs in 10 domains): **the country of a host never changed.** Per host, the
set of (address, country) was identical in every pair (118 comparisons). All
11 `drift.ip.country_changed` findings in the history are false.

The cause is `ipCountryChanged` (`internal/drift/rules.go`): it keys the
previous scan's `ip.asn` findings by **host name only**. A host with several
addresses (IPv4 + IPv6, several anycast addresses) keeps only the country of
whichever address came last, and every address of the new scan is compared
with that one country. `lennox.ns.cloudflare.com` has six addresses in CA, US,
CR and GB — it "changes country" on almost every scan. The existing test uses a
host with one address, so it could not see this.

## What changes

The rule compares **per address**: an address that is in both scans and whose
country differs is a country change. An address that appears or disappears is
not a country change (that is a different question, and the NS/MX set rules
already cover the hosts).
