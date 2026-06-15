# Proposal: DNS-hosting jurisdiction — score who runs your DNS, and where

> **Status:** Accepted + implemented (2026-06-15). Wave-1 lead from
> research-high-signal-observability; a direct mirror of the existing
> MX-jurisdiction rule, so it lands implemented rather than as a
> separate design pass.

## Why

The "who/where" of authoritative DNS was a blind spot. Wanderer already
scored MX jurisdiction ("your mail lands at Google/US") via the
dns.mx + ip.asn + `mx_vendor_jurisdiction` correlation, but the
equivalent for **nameservers** was missing: `dns_redundancy` only
counted NS hosts, never located them, and the scanner never even fed NS
hosts to the IP probe — so "your DNS is run by Cloudflare (US)" could
not be observed or scored. DNS is the control plane for every name an
organisation publishes; a non-EEA DNS operator resolves (and can
withhold/redirect) those names under foreign jurisdiction.

## What Changes

- The scanner adds `dns.ns` hosts to `Target.Related` (alongside the
  existing `dns.mx` / third-party / subdomain hosts), so the IP probe
  geo-locates them.
- A new `wand.juridisch.ns_vendor_jurisdiction` rule correlates the
  dns.ns hosts with their ip.asn lookups and scores the jurisdiction
  (all-EEA → soeverein, split → voldoende, none-EEA → afhankelijk),
  naming the country — the exact pattern of the MX rule.

## Scope / Not in scope

In: NS-host jurisdiction scoring. Out: DNSSEC, registrar lock, and
DNS-provider concentration (separate leads). Observed facts already
flow through the existing dns.ns + ip.asn findings; this change adds
the correlation + the missing NS→Related enrichment.

## Parallel-safe

One scanner line (NS → Related), one new rule file + registration, one
assessor spec requirement. No schema change.
