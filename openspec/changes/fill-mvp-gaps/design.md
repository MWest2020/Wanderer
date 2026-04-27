# Design: Fill MVP Gaps

## Component placement

```
internal/scanner/
  scanner.go           # extended: two-pass orchestration + concurrent pass 1
  ssrf.go              # new: dialer + ipNet table + opt-out flag plumbing
  amass.go             # new: Amass JSON file → []string FQDN merge
  subdomains.go        # new: SAN extraction + common-prefix probe
internal/probe/whois/
  whois.go             # new: RDAP lookup → whois.* findings
internal/assessor/eucsf/
  rules.go             # SEAL rule pack
  rules_test.go
internal/store/
  migrations.go        # new: numbered schema migrations + schema_migrations table
internal/ui/
  ui.go                # html/template renderer + route table
  templates/           # 3 .tmpl files
  static/              # vanilla CSS, no JS
  ui_test.go
pkg/models/
  seal.go              # SEAL level enum, separate from Score
  framework.go         # Framework enum (dictu | eucsf)
  assessment.go        # Framework field gains an enum, no breaking change
docs/
  architecture.md      # refreshed: two-pass + SSRF + frameworks
  findings.md          # adds dns.subdomain, whois.* sections
  assessor.md          # adds SEAL section
  operator.md          # adds Amass recipe
```

## Two-pass scanner

Today `scanner.New` returns a `*Scanner` whose `Scan(ctx, target)` runs every probe in `probes` concurrently. The new shape:

```go
// Pass 1 — probes that produce hosts to be IP-resolved later.
pass1 := []probe.Probe{dnsP, tlsP, httpP}
// Pass 2 — probes that consume pass-1 output.
pass2 := []probe.Probe{ipP}
```

Pass 1 runs with `errgroup`; results merge into a single `[]Finding`. The scanner then walks pass-1 findings, collects every `Subject` from `{dns.mx, http.third_party, dns.subdomain}`, builds an enriched `Target` whose `Related` is `union(target.Related, discovered)` *only for pass 2*, and runs pass 2 against it. Other passes still see the original target — important so pass-1 probes do not chase their own discovered hosts in the same scan.

The probe interface is unchanged. Pass membership is a static slice in `cmd/wanderer/scan.go`; no runtime selection.

## SSRF guard

```go
type SafeDialer struct {
    AllowPrivate bool
    Resolver     *net.Resolver
}

func (d SafeDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
    host, port, _ := net.SplitHostPort(addr)
    ips, err := d.Resolver.LookupIPAddr(ctx, host)
    if err != nil { return nil, err }
    var allowed []net.IPAddr
    for _, ip := range ips {
        if !d.AllowPrivate && isPrivateOrMetadata(ip.IP) {
            continue
        }
        allowed = append(allowed, ip)
    }
    if len(allowed) == 0 {
        return nil, ErrPrivateTargetBlocked
    }
    return (&net.Dialer{}).DialContext(ctx, network, net.JoinHostPort(allowed[0].IP.String(), port))
}
```

`isPrivateOrMetadata` checks against a static slice of `*net.IPNet` covering: 127.0.0.0/8, 169.254.0.0/16 (link-local + cloud metadata), 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16, 100.64.0.0/10 (CGNAT), `::1/128`, `fc00::/7` (ULA), `fe80::/10` (link-local), `fd00:ec2::254/128` (AWS IMDSv2 IPv6), 169.254.169.254/32 (cloud metadata).

The dialer is wired into `internal/probe/http`'s `http.Client.Transport.DialContext` and `internal/probe/tls`'s `tls.Dialer`. The opt-out is a `--allow-private-targets` flag on `wanderer scan` and `wanderer serve`, plumbed into `probe.Config`.

## Subdomain discovery

Two passive sources, both inside the existing TLS and DNS probes:

1. **crt.sh SAN mining** — `internal/probe/tls/tls.go` already calls crt.sh for CT logs. Each row's `name_value` is parsed (split on `\n`); each FQDN under the target apex becomes a `dns.subdomain` Finding with `source: ct_log` and an `apex_domain` attribute. No new HTTP call.

2. **Common-prefix probe** — a new helper in `internal/probe/dns/subdomains.go` runs `LookupHost` for the 18 names listed in the proposal (`www`, `mail`, `m`, `api`, `auth`, `sso`, `mijn`, `loket`, `inloggen`, `nextcloud`, `webmail`, `vpn`, `portal`, `intranet`, `extranet`, `wachtwoord`, `wifi`, `gast`). Resolving names emit `dns.subdomain` with `source: prefix_probe`.

Both feeds dedupe before emission and respect the per-probe timeout. The full prefix list lives in source as a `var commonPrefixes = []string{...}` so additions are a one-line PR.

## Amass importer

Amass `enum -json` produces newline-delimited JSON, one record per host:

```jsonl
{"name":"mail.example.nl","domain":"example.nl","addresses":[{"ip":"1.2.3.4"}]}
```

`internal/scanner/amass.go` reads the file, extracts `name` from each line (rejecting empty / non-FQDN), and returns `[]string`. The CLI/API merges that slice into `target.Related` before pass 1. Errors on the file path are fatal; malformed lines are skipped with a `slog.Warn`.

No `github.com/owasp-amass/amass` dependency; we never invoke Amass ourselves.

## SEAL rule pack

A new package `internal/assessor/eucsf/rules.go` with the same `Rule` shape the dictu pack uses. New types in `pkg/models`:

```go
type SealLevel string
const (
    SEAL0 SealLevel = "seal_0"  // unknown
    SEAL1 SealLevel = "seal_1"
    SEAL2 SealLevel = "seal_2"
    SEAL3 SealLevel = "seal_3"
    SEAL4 SealLevel = "seal_4"  // sovereign
)

type Framework string
const (
    FrameworkDICTU Framework = "dictu"
    FrameworkEUCSF Framework = "eucsf"
)
```

`models.Assessment.Framework` is already a string; we tighten the doc to use the enum but keep the column flexible. The existing assessor engine accepts any rule pack as `[]assessor.Rule`; the SEAL pack returns `RuleResult` with the `Score` field carrying a `models.Score` mapped from `SealLevel` (`SEAL0 → ScoreOnbekend`, `SEAL4 → ScoreSoeverein`, etc.) so the existing aggregation logic works unchanged. The framework label distinguishes them in the persisted Assessment.

CLI: `wanderer assess <scan-id> --framework dictu|eucsf|both`. With `both`, two `Assessment` rows are persisted, one per framework, both citing the same Findings.

## Read-only web UI

Routes mounted on the existing chi router under `--ui` flag:

```go
r.Route("/ui", func(r chi.Router) {
    if htpasswd != "" {
        r.Use(basicAuth(htpasswdFile))
    }
    r.Get("/",                       indexHandler)
    r.Get("/scans/{id}",             scanHandler)
    r.Get("/targets/{id}/drift",     driftHandler)
})
```

Templates are `internal/ui/templates/{index,scan,drift}.tmpl`, parsed once at server start (no hot-reload). Static assets (`internal/ui/static/main.css`) served via `http.FileServer` under `/ui/static/`. The package contains zero HTTP method handlers other than GET; a static-analysis check in `ui_test.go` greps the package for `r.Post|Patch|Delete|Put` and fails the build if any are present.

The `basicAuth` middleware reads htpasswd entries lazily (re-read every request) so an operator can rotate without restarting. Bcrypt and SHA-512 entries supported; plain MD5 is rejected.

## RDAP / WHOIS probe

`internal/probe/whois/whois.go` issues `GET https://rdap.org/domain/<domain>` with a 5-second timeout and parses the response per RFC 7483:

- `entities[].roles` containing `registrant` → `whois.registrant` Finding with `country` from `vcardArray`.
- `entities[].roles` containing `registrar` → `whois.registrar` Finding with `name`.
- HTTP error or parse failure → single `whois.unavailable` Finding with `reason`.

Feeds a new DICTU rule `dictu.juridisch.registrar_jurisdiction` that maps registrant `country` against `eeaCountries`.

## Schema migration table

Replace the current ad-hoc `ALTER TABLE ADD COLUMN` (with string-matched "duplicate column name" tolerance) with a numbered migration list:

```go
var migrations = []migration{
    {Version: 1, Name: "initial_schema", Up: initialSchemaSQL},
    {Version: 2, Name: "add_source_modus", Up: addSourceModusSQL},
}
```

`schema_migrations` table tracks `(version int primary key, applied_at datetime)`. On `Open`, every migration whose version is not in the table runs in order in a single transaction. Failures roll the transaction back. The auditor reads one table to know which schema version is in production.

Future migrations add an entry to the slice; nothing else.

## Concurrent probe execution

Pass 1 already needs an orchestration shape that fans out and waits. `errgroup.WithContext` plus `g.Go(func() error { ... })` per probe, each wrapped in `defer recover()` that logs and returns `nil` so one panic does not poison the group. The global budget remains a single `context.WithTimeout` around the whole scan.

Per-probe timeouts are unchanged.

## Failure modes

| Cause                                   | Outcome                                                                     |
| --------------------------------------- | --------------------------------------------------------------------------- |
| crt.sh response missing `name_value`    | No subdomain mined; existing CT log finding still emitted                  |
| Common-prefix probe hits a wildcard A   | Each prefix resolves; many spurious `dns.subdomain` findings                 |
|                                         | Mitigation: dedupe by IP set; if all prefixes resolve to the same IPs,       |
|                                         | emit a single `dns.subdomain.wildcard` Finding instead                       |
| Amass file referenced but missing       | `wanderer scan` exits non-zero with a clear stderr message                   |
| RDAP returns 429                        | Exponential backoff up to 3 retries, then `whois.unavailable`                |
| SSRF guard blocks a legitimate target   | Operator runs with `--allow-private-targets`; documented in operator.md      |
| UI htpasswd file missing                | `wanderer serve --ui --ui-htpasswd /missing.htpasswd` exits at startup       |
| Migration N fails                       | Transaction rolls back; agent fails to start; operator inspects the error   |

## Out-of-band

When this change archives, file a follow-up `add-headless-browser-probe` for JS-rendered third-party detection. The MVP HTTP probe sees only the static HTML and misses anything injected by client-side bundles. Headless Chrome is heavy; defer until concrete demand.

## Stability + coverage targets

- `internal/scanner` ≥ 80% (worth tightening with the new fan-out logic in scope).
- `internal/probe/whois` ≥ 75%.
- `internal/assessor/eucsf` ≥ 85% (mirrors the DICTU pack's coverage target).
- `internal/ui` ≥ 70% (template rendering + auth middleware).
- `internal/store/migrations.go` ≥ 90% (mechanical, fully testable).
