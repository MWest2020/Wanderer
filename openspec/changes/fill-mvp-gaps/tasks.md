# Tasks: Fill MVP Gaps

## 1. Casing bug + integration discipline (goal #1)

- [ ] 1.1 Lowercase the `dns.A` / `dns.AAAA` constants in `internal/assessor/dictu/rules.go`
- [ ] 1.2 `internal/assessor/dictu/integration_test.go` runs the real DNS probe through the real DICTU rules and pins `apex_ip_eea`, `mx_vendor_jurisdiction`, and `third_parties_eea` against fixed-resolver fixtures
- [ ] 1.3 Document the test discipline in `docs/findings.md` ("Probe-ID / rule-ID drift is build-breaking")

## 2. Two-pass scanner (goal #2)

- [ ] 2.1 `internal/scanner/scanner.go` — split into pass 1 (DNS+TLS+HTTP) and pass 2 (IP)
- [ ] 2.2 `expandRelatedFromFindings` collects subjects from `{dns.mx, http.third_party, dns.subdomain}` and builds the enriched Target for pass 2 only
- [ ] 2.3 `cmd/wanderer/scan.go` `buildProbes` orders the IP probe last
- [ ] 2.4 `TestIPProbeReceivesDiscoveredHosts` pins the fan-out invariant
- [ ] 2.5 `docs/architecture.md` documents the two-pass behaviour

## 3. SSRF guard (goal #3)

- [ ] 3.1 `internal/scanner/ssrf.go` with `SafeDialer` and a static `*net.IPNet` table covering loopback, link-local, RFC1918, CGNAT, IPv6 ULA, IPv6 link-local, and the cloud metadata IPs
- [ ] 3.2 Wire the dialer into `internal/probe/http`'s `http.Client.Transport` and `internal/probe/tls`'s `tls.Dialer`
- [ ] 3.3 `--allow-private-targets` flag on `wanderer scan` and `wanderer serve`; default is on (private blocked)
- [ ] 3.4 `internal/probe/http`'s `POST /scans` handler refuses requests whose domain resolves only to private addresses unless the flag is set
- [ ] 3.5 Unit test per blocked family (IPv4 loopback, IPv4 RFC1918, IPv4 CGNAT, IPv4 metadata, IPv6 ULA, IPv6 link-local)

## 4. Passive subdomain discovery (goal #4)

- [ ] 4.1 `internal/probe/tls/tls.go` extracts SAN names from existing crt.sh response → `dns.subdomain` Findings (`source: ct_log`)
- [ ] 4.2 `internal/probe/dns/subdomains.go` probes 18 common prefixes via `LookupHost`; resolving names → `dns.subdomain` Findings (`source: prefix_probe`)
- [ ] 4.3 Wildcard detection: when every prefix resolves to the same IP set, emit one `dns.subdomain.wildcard` Finding instead of 18 spurious ones
- [ ] 4.4 Discovered subdomains feed pass 2 (already covered by goal #2's `expandRelatedFromFindings`)
- [ ] 4.5 Unit tests with a fake resolver covering hit / miss / wildcard

## 5. Amass importer (goal #5)

- [ ] 5.1 `internal/scanner/amass.go` parses Amass `enum -json` JSONL and returns `[]string` FQDNs
- [ ] 5.2 `--amass <file>` flag on `wanderer scan`; CLI errors fatal at startup
- [ ] 5.3 `POST /scans` body gains an `amass_json` field (string body or path); `wanderer serve` reads server-local paths only
- [ ] 5.4 `docs/operator.md` recipe: `amass enum -passive -d <domain> -json out.json && wanderer scan <domain> --amass out.json`
- [ ] 5.5 Unit tests with a sample Amass JSON fixture

## 6. SEAL rule pack (goal #6)

- [ ] 6.1 `pkg/models/seal.go` — `SealLevel` enum (SEAL0…SEAL4) and `Framework` enum
- [ ] 6.2 `internal/assessor/eucsf/rules.go` with the five rules listed in the proposal
- [ ] 6.3 `internal/assessor/eucsf/rules_test.go` — at least one EU and one non-EU fixture per rule
- [ ] 6.4 `wanderer assess --framework dictu|eucsf|both` plumbed through CLI + persistence
- [ ] 6.5 `docs/assessor.md` SEAL framework section with the score-mapping table

## 7. Read-only web UI (goal #7)

- [x] 7.1 `internal/ui/ui.go` with `html/template`, three handlers (index / scan / drift)
- [x] 7.2 `internal/ui/templates/{index,scan,drift}.tmpl` — vanilla HTML, vanilla CSS
- [x] 7.3 `internal/ui/static/main.css`
- [x] 7.4 HTTP Basic via `--ui-htpasswd <file>`; bcrypt-only (every other algorithm — `$apr1$` MD5, `{SHA}` SHA-1, `$5$`/`$6$` crypt — rejected at startup with an explicit "use bcrypt" error)
- [x] 7.5 Mount under `--ui` flag on the existing `wanderer serve` router
- [x] 7.6 `internal/ui/ui_test.go` hits each route with a fake store; asserts 200 + key strings
- [x] 7.7 Static-analysis test in the same file greps for any `r.Post|Patch|Delete|Put` in the package and fails the build

## 8. Concurrent probe execution (goal #8)

- [ ] 8.1 Pass 1 runs probes via `errgroup.WithContext`, per-probe panic recover converts to `nil` so the group survives
- [ ] 8.2 Global budget remains a single `context.WithTimeout` around the whole scan
- [ ] 8.3 Wall-clock test that asserts pass 1 finishes in roughly the slowest probe's duration, not the sum

## 9. WHOIS / RDAP probe (goal #9)

- [ ] 9.1 `internal/probe/whois/whois.go` calls `https://rdap.org/domain/<domain>` with a 5-second timeout
- [ ] 9.2 Emits `whois.registrant` (`country`), `whois.registrar` (`name`), `whois.unavailable` on failure
- [ ] 9.3 New rule `dictu.juridisch.registrar_jurisdiction` consults registrant country
- [ ] 9.4 Unit tests with httptest serving canned RDAP JSON
- [ ] 9.5 `docs/findings.md` whois section

## 10. Schema migration table (goal #10)

- [ ] 10.1 `internal/store/migrations.go` with a numbered slice and `schema_migrations` table
- [ ] 10.2 Convert the current schema into migration 001 and the `source_modus` ALTER into migration 002
- [ ] 10.3 Replace the string-matched `ALTER TABLE` tolerance with the migration runner
- [ ] 10.4 Tests: fresh DB applies all; DB at version N applies only N+1..M; failure rolls back

## 11. Cross-cutting

- [ ] 11.1 Update `README.md` to drop the "MVP landed" framing and link the SEAL framework
- [ ] 11.2 `docs/architecture.md` covers two-pass scanner, SSRF guard, framework selection, UI
- [ ] 11.3 `docs/findings.md` adds `dns.subdomain` and `whois.*` sections
- [ ] 11.4 `docs/assessor.md` adds the SEAL section
- [ ] 11.5 `CHANGELOG.md` — one entry per goal under `[Unreleased]`
- [ ] 11.6 ADR-0009 records the SEAL/DICTU dual-framework choice if the design decision proves load-bearing during implementation
