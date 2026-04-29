# Fill MVP Gaps — Wanderer

## Context

Wanderer's MVP scanner suite is in. An audit identified two correctness bugs and four scope gaps that prevent the headline DICTU rules from producing evidence-backed scores on real scans. This change closes the correctness gaps, adds subdomain discovery, adds basic UI/dashboard, and adds the EU Cloud Sovereignty Framework (SEAL) rule pack alongside the existing DICTU one.

Boring-and-auditable applies throughout. No clever architecture; no plugin loaders; standard Go stdlib + chi + html/template. Every change must be explainable to an ISO-27001 auditor in one sentence.

## Goals (must)

1. **Fix the `dns.A` / `dns.a` casing bug.** `internal/assessor/dictu/rules.go:113` looks for ProbeID `"dns.A"` / `"dns.AAAA"`; the DNS probe emits lowercase. Lowercase the constants. Then add an integration test under `internal/assessor/dictu/integration_test.go` that runs the real DNS probe (with a fake `Resolver` returning fixed records) → real assessor → asserts `apex_ip_eea` produces `Soeverein` for an NL apex. This category of bug must be caught by tests in future.

2. **Wire DNS- and HTTP-discovered hosts into the IP probe.** Today the IP probe only resolves `target.Domain` and `target.Related[]`. Discovered MX hosts (from `dns.mx`) and third-party hosts (from `http.third_party`) are never IP-resolved, so `mx_vendor_jurisdiction`, `third_parties_eea`, and the third-party arm of `no_us_hyperscaler` always return `Onbekend` on real scans.

   Implement as a two-pass scan in `internal/scanner/scanner.go`: run DNS+TLS+HTTP first, collect all `Subject` values from findings whose ProbeID is in `{dns.mx, http.third_party}`, then run the IP probe with `target.Related = union(target.Related, discovered_hosts)`. Keep the probe interface unchanged. Document the two-pass behaviour in `docs/architecture.md`.

3. **SSRF guard on the HTTP and TLS probes.** The `POST /scans` endpoint accepts arbitrary domains. Add a `DialContext` to the default HTTP client and TLS dialer that rejects targets resolving to: loopback, link-local, RFC1918 (10/8, 172.16/12, 192.168/16), CGNAT (100.64/10), IPv6 ULA (fc00::/7), IPv6 link-local (fe80::/10), the AWS/GCP/Azure metadata IPs (169.254.169.254, fd00:ec2::254). Make this opt-out via `--allow-private-targets` for operators who genuinely scan internal hosts. Default is on. Add a unit test for each address family.

4. **Subdomain discovery (passive only, no Amass dependency).**
   - Mine SANs from the existing crt.sh response: every `name_value` becomes a candidate subdomain. Already-paid-for, no new I/O.
   - Add a tiny common-prefix probe: try `www`, `mail`, `m`, `api`, `auth`, `sso`, `mijn`, `loket`, `inloggen`, `nextcloud`, `webmail`, `vpn`, `portal`, `intranet`, `extranet`, `wachtwoord`, `wifi`, `gast`. Quick A-record lookup; record the resolving ones as `dns.subdomain` findings.
   - Discovered subdomains feed into the same two-pass logic as goal #2 (so they get IP-resolved).
   - Out of scope: brute-force wordlists, DNS-over-HTTPS, recursive walking. Those belong to Amass; see goal #5.

5. **Optional Amass importer.** Add `--amass <path-to-amass-json>` flag to `wanderer scan` and `wanderer serve` (the latter as a per-request field on `POST /scans`). When present, parse the Amass JSON output (`amass enum -json out.json` format), extract every FQDN, and merge into `target.Related`. Document in `docs/operator.md` with a recipe: `amass enum -passive -d gemeente-x.nl -json out.json && wanderer scan gemeente-x.nl --amass out.json`. No Amass dependency in `go.mod`; no shelling out; just file ingestion.

6. **EU Cloud Sovereignty Framework (SEAL) rule pack.** Add `internal/assessor/eucsf/` alongside the existing `dictu/` package. Implement at minimum:
   - `eucsf.sov2.cert_issuer_eu` — TLS issuer in EU jurisdiction → SEAL level
   - `eucsf.sov2.apex_jurisdiction` — apex IP ASN registered in EU → SEAL level
   - `eucsf.sov3.mx_jurisdiction` — MX host ASN in EU → SEAL level
   - `eucsf.sov4.operational_eu` — third-party hosts in EU → SEAL level
   - `eucsf.sov6.no_us_hyperscaler` — same as DICTU rule, mapped to SEAL

   Score on the SEAL 0–4 scale (`models.SEAL0` … `models.SEAL4`), separate from the existing DICTU `Score` type. Operator picks framework via `--framework dictu|eucsf|both` on `wanderer assess`. Both frameworks consume the same Findings — only the rule pack differs. Reference: <https://commission.europa.eu/document/download/09579818-64a6-4dd5-9577-446ab6219113_en>.

7. **Read-only web UI.** Add `internal/ui/` with Go `html/template` rendering. No SPA, no JS framework, no build step. Tailwind-via-CDN is fine if you must, but vanilla CSS is preferred. Mounted at `GET /ui/` on the existing chi router behind a `--ui` flag (default off). Three pages:
   - `/ui/` — list of targets, last scan status, last assessment score per framework
   - `/ui/scans/{id}` — findings grouped by probe, with severity colouring
   - `/ui/targets/{id}/drift` — drift findings since a date picker

   Auth: HTTP Basic via `--ui-htpasswd <file>`. Boring. No sessions, no JWT, no OAuth. Operators put it behind a reverse proxy if they want anything fancier.

## Goals (should)

8. Concurrent probe execution. Probes are independent except for the two-pass IP step; run DNS+TLS+HTTP concurrently in pass 1, then IP in pass 2. Use `errgroup` with a per-probe panic recover; respect global budget. Cuts wall-clock per scan from ~30s to ~10s on a typical target.

9. WHOIS / RDAP probe. New `internal/probe/whois/` using RDAP (`https://rdap.org/domain/<domain>`), no WHOIS-43 sockets. Records registrant country and registrar name. Feeds a new `dictu.juridisch.registrar_jurisdiction` rule. Fail-soft: RDAP unavailable → single `whois.unavailable` finding.

10. Schema migration table. Replace the string-matched ALTER TABLE in `internal/store/sqlite.go` with a `schema_migrations` table and numbered up-only migrations. The current schema becomes migration 001. The `source_modus` ALTER becomes 002. Future migrations get a number. Auditable.

## Non-goals

- JavaScript-rendered third-party detection. Headless browsers are a future change.
- Vulnerability scanning. Out of scope by README-level promise.
- Subdomain brute-forcing with wordlists > 50 entries. Use Amass.
- Multi-tenancy in the UI. Single-tenant by design; deploy multiple instances if needed.
- Rewriting the assessor. Rules stay Go functions per ADR-0004.

## Order of work (suggested)

1. Goal #1 — casing bug + integration test. ~30 min. Unblocks honest demos.
2. Goal #3 — SSRF guard. ~2h. Needed before goal #7 ships UI publicly.
3. Goal #2 — two-pass scanner. ~half day. Makes the headline rules actually score.
4. Goal #10 — migration table. ~2h. Cheap; do before adding more tables for goals #6 and #7.
5. Goal #6 — SEAL rule pack. ~1 day. Mostly mechanical; copy-modify the DICTU rules.
6. Goal #4 — passive subdomain discovery. ~half day.
7. Goal #5 — Amass importer. ~2h.
8. Goal #7 — UI. ~1–2 days.
9. Goals #8 and #9 — should-haves.

## Test discipline

Every goal needs at least one test that would have caught the bug or gap it closes. Specifically:

- Goal #1: integration test from goal #1.
- Goal #2: end-to-end test that scans a fake `gemeente-x.nl` (with stub probes returning realistic `dns.mx` and `http.third_party` findings) and asserts `mx_vendor_jurisdiction` produces `Soeverein` evidence.
- Goal #3: unit test per blocked address family asserting the dialer returns an error.
- Goal #6: at minimum one test per SEAL rule with EU and non-EU fixtures.
- Goal #7: a `go test` that hits each UI route and asserts 200 + a known string in the body.

## What to update along the way

- `README.md` — update the "MVP landed" framing; the gaps in the audit are no longer accurate after this change.
- `docs/architecture.md` — document the two-pass scanner, the SSRF guard, and the framework selection.
- `docs/findings.md` — add `dns.subdomain`, `whois.*` probe IDs.
- `docs/assessor.md` — add the SEAL framework section.
- `CHANGELOG.md` — one entry per goal under `[Unreleased]`.
- `openspec/changes/fill-mvp-gaps/` — proposal.md (this file), design.md, tasks.md, specs/.

## Out-of-band

When this change archives, propose a follow-up `add-headless-browser-probe` for JS-rendered third-party detection. Note in passing.
