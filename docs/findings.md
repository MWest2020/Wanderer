# Findings Reference

Every probe produces `models.Finding` records with a shared shape. This
document is the catalogue: which ProbeIDs exist, what they mean, which
attributes they carry, and which DICTU dimension they inform.

> **Probe-ID / rule-ID drift is build-breaking.** Each ProbeID listed
> here is referenced by string from at least one assessor rule
> (`internal/assessor/dictu`, `internal/assessor/eucsf`). The DICTU
> integration tests in `internal/assessor/dictu/integration_test.go`
> drive real probes through real rules; renaming a ProbeID without
> updating the matching rule fails CI immediately rather than silently
> returning Onbekend on every real scan. Treat this catalogue as the
> contract — not a description.

## Finding shape

```go
type Finding struct {
    ID            string              // assigned on persist
    ScanID        string              // set by the scanner
    ProbeID       string              // "dns.mx", "tls.issuer", ...
    DimensionHint DimensionHint       // optional DICTU dimension
    CriteriumHint string              // optional DICTU criterium ref
    Subject       string              // the thing being described
    Severity      Severity
    Attributes    map[string]any      // probe-specific structured data
    Evidence      []byte              // raw source material (cert PEM, TXT record, ...)
    CreatedAt     time.Time
}
```

See `pkg/models/finding.go` for the canonical definition.

## Severity ladder

| Severity        | Meaning                                                  |
| --------------- | -------------------------------------------------------- |
| `info`          | Neutral fact. Not a problem, not praise.                 |
| `observation`   | Noteworthy fact that may matter in context.              |
| `concern`       | Fact that likely bears on sovereignty posture.           |
| `finding`       | Fact the assessor should almost certainly reflect in the score. |

The ladder is deliberately coarse. Fine-grained weighting belongs to
the assessor, not to the probe.

## DICTU dimension hints

| Hint              | DICTU dimension |
| ----------------- | --------------- |
| `juridisch`       | Juridisch       |
| `technologie`     | Technologie     |
| `data_ai`         | Data & AI       |
| `operationeel`    | Operationeel    |
| `mens`            | Mens            |
| (empty)           | no hint — raw observation |

## Scanner-level meta findings

These are produced by the orchestrator, not by any individual probe.

| ProbeID           | Severity  | When produced                                                 |
| ----------------- | --------- | ------------------------------------------------------------- |
| `<probe>.timeout` | info      | A probe exceeded its per-probe timeout.                       |
| `<probe>.error`   | concern   | A probe returned a non-timeout error.                         |
| `<probe>.panic`   | concern   | A probe panicked; recovered by the scanner.                   |

`<probe>` is the probe ID: `dns`, `tls`, `ip`, `http`.

## DNS probe — `internal/probe/dns`

| ProbeID          | Severity      | Dimension     | Attributes (non-exhaustive)                        |
| ---------------- | ------------- | ------------- | -------------------------------------------------- |
| `dns.a`          | info          | —             | `address` (IPv4); on failure: `error`, `kind`      |
| `dns.aaaa`       | info          | —             | `address` (IPv6)                                   |
| `dns.mx`         | observation   | `data_ai`     | `host`, `preference`; Evidence: `<pref> <host>`    |
| `dns.ns`         | observation   | `operationeel`| `host`                                             |
| `dns.cname`      | observation   | `data_ai`     | `target` (only recorded if non-apex)               |
| `dns.txt.spf`    | observation   | `data_ai`     | `record`, `kind: "spf"`                            |
| `dns.txt.dkim`   | observation   | `data_ai`     | `record`, `kind: "dkim"`                           |
| `dns.txt.dmarc`  | observation   | `data_ai`     | `record`, `kind: "dmarc"`                          |
| `dns.txt.other`  | info          | `data_ai`     | `record`, `kind: "other"`                          |
| `dns.caa`        | observation   | `operationeel`| `flag`, `tag`, `value`                             |
| `dns.subdomain`  | observation   | —             | `apex_domain`, `source: "ct_log" \| "prefix_probe"`, `addresses` (prefix_probe only) |
| `dns.subdomain.wildcard` | info  | —             | `apex_domain`, `source: "prefix_probe"`, `hit_count`, `ips` |

**Subdomain sources.** `dns.subdomain` Findings come from two passive
sources: SubjectAlternativeName entries on the apex certificate
(`source: "ct_log"`) and a small fixed prefix sweep
(`internal/probe/dns/subdomains.go`, `source: "prefix_probe"`). When
every prefix resolves to the same IP set the probe collapses the noise
into a single `dns.subdomain.wildcard` Finding rather than 18 spurious
hits.

**Failure variants.** For `dns.a`, `dns.mx`, `dns.ns`, `dns.txt`,
`dns.caa`, a lookup that fails produces a finding of severity `info`
with `error: "<string>"` and `kind: "nxdomain" | "timeout" |
"temporary" | "error"`.

**CAA caveat.** The Go standard library resolver does not expose CAA.
The default adapter returns an empty result (recorded as "no CAA
records"). Swap in a `miekg/dns`-based resolver if CAA visibility is
load-bearing for your assessment.

## TLS probe — `internal/probe/tls`

| ProbeID          | Severity                                    | Dimension     | Attributes                                                                 |
| ---------------- | ------------------------------------------- | ------------- | -------------------------------------------------------------------------- |
| `tls.handshake`  | concern                                     | `operationeel`| `error`, `kind: "timeout" \| "other"` — when the handshake could not be made |
| `tls.verify`     | concern                                     | `operationeel`| `verified: false`, `subject_cn` — only when the chain did not verify     |
| `tls.issuer`     | finding                                     | `juridisch`   | `issuer_cn`, `issuer_o`, `issuer_country`, `subject_cn`, `serial`, `not_before`, `not_after`, `signature_algo`, `public_key_algo`. Evidence: cert PEM |
| `tls.san`        | observation                                 | `data_ai`     | `dns_names`, `ip_addresses`                                                |
| `tls.validity`   | info \| observation (expiring) \| concern (expired) | `operationeel`| `not_before`, `not_after`, `days_left`, `expired?`, `expiring_soon?`       |
| `tls.chain`      | info                                        | `operationeel`| `length`, `intermediates` (list of intermediate CNs)                       |
| `tls.ct`         | observation \| info (unavailable)           | `juridisch` \| — | `total_entries`, `issuer_counts`; on failure: `unavailable: true`, `error` |

The TLS probe will first try a verified handshake; on failure it
retries with verification off so the certificate can still be
inspected. In that path it emits both `tls.verify` (concern) and the
usual cert-inspection findings.

## IP probe — `internal/probe/ip`

| ProbeID         | Severity    | Dimension   | Attributes                                      |
| --------------- | ----------- | ----------- | ----------------------------------------------- |
| `ip.asn`        | finding     | `juridisch` | `address`, `asn`, `organisation`, `country`     |
| `ip.resolve`    | info        | —           | `error` — hostname did not resolve              |
| `ip.lookup`     | info        | —           | `address`, `error` — IP present, MaxMind lookup failed |
| `ip.unavailable`| info        | —           | `reason` — probe started with no GeoLite2 DB    |

The IP probe fails fast at startup if the GeoLite2 DB is missing or
corrupt. It does not degrade silently.

## HTTP probe — `internal/probe/http`

| ProbeID                 | Severity    | Dimension      | Attributes                                                                 |
| ----------------------- | ----------- | -------------- | -------------------------------------------------------------------------- |
| `http.response`         | info        | —              | `status`, `final_url`, `scheme`, `server`, `powered_by`                    |
| `http.security_headers` | observation | `operationeel` | `present` (map of header→value), `missing` (list of header names)          |
| `http.scheme_downgrade` | concern     | `operationeel` | `reason` — HTTPS failed, HTTP fallback succeeded                           |
| `http.third_party`      | observation | `technologie`  | `source_domain`, `kinds` (one or more of `script`, `link`, `img`, `iframe`, `source`) |
| `http.robots_blocked`   | info        | —              | `robots_txt_fetched: true`; Evidence: robots.txt body                      |
| `http.fetch_failed`     | concern     | —              | `error` — neither HTTPS nor HTTP worked                                    |
| `http.parse_failed`     | info        | —              | `error` — response fetched but HTML parsing failed                         |

`http.third_party` has one Finding **per external host**. The `Subject`
is the external host, not the scanned domain. Use `source_domain` in
Attributes to link it back.

## WHOIS / RDAP probe — `internal/probe/whois`

Issues `GET https://rdap.org/domain/<domain>` with a 5-second
timeout and walks the returned vCard array per RFC 7483. Stdlib
`net/http` only — no WHOIS-43 socket, no third-party SDK.

| ProbeID            | Severity | Dimension   | Attributes                                  |
| ------------------ | -------- | ----------- | ------------------------------------------- |
| `whois.registrant` | finding  | `juridisch` | `country` (ISO 3166-1 alpha-2, uppercase)   |
| `whois.registrar`  | info     | —           | `name` (registrar organisation)             |
| `whois.unavailable`| info     | —           | `reason` — emitted on any network error, non-2xx, parse error, or response with no registrant/registrar entities, so the rest of the scan continues |

Consumed by `dictu.juridisch.registrar_jurisdiction`, which maps
EEA registrant countries to soeverein, anything outside the EEA to
afhankelijk, and absence (`whois.unavailable`) to onbekend.

## Drift Findings

Drift Findings are produced by the scheduler when a new scan differs
from the previous scan of the same target. Their `ProbeID` starts
with `drift.`; their `Attributes` always contain
`source_modus: "drift"`, `prev_scan_id`, and `curr_scan_id`.

| ProbeID                              | severity     | dimension       | attributes                                                  |
| ------------------------------------ | ------------ | --------------- | ----------------------------------------------------------- |
| `drift.baseline_established`         | info         | —               | (first scan for a target — no comparison possible)          |
| `drift.no_changes`                   | info         | —               | (two scans identical modulo IDs/timestamps)                 |
| `drift.tls.issuer_changed`           | finding      | `juridisch`     | `prev_issuer_cn`, `curr_issuer_cn`                          |
| `drift.tls.days_left_dropped`        | concern      | `operationeel`  | `prev_days_left`, `curr_days_left` (fired on crossing 30d)  |
| `drift.dns.mx_set_changed`           | observation  | `data_ai`       | `added: []`, `removed: []`                                  |
| `drift.dns.ns_set_changed`           | observation  | `operationeel`  | `added: []`, `removed: []`                                  |
| `drift.ip.country_changed`           | finding      | `juridisch`     | `prev_country`, `curr_country` (per host)                   |
| `drift.http.third_party_added`       | observation  | `technologie`   | `hosts: []` — newly seen subjects                           |
| `drift.http.third_party_removed`     | info         | `technologie`   | `hosts: []` — subjects that disappeared                     |

See [`drift.md`](drift.md) for the rule semantics and how the
assessor consumes drift Findings.

## Inventory Findings

Inventory Findings are produced by `wanderer agent` running on a
host. They carry `SourceModus = "inventory"` (top-level field on
`Finding` since the inventory probe landed) so the assessor's
completeness calculation can distinguish them from perimeter data.

| ProbeID                          | severity      | dimension       | attributes                                                       |
| -------------------------------- | ------------- | --------------- | ---------------------------------------------------------------- |
| `inventory.systemd.service`      | info          | `operationeel`  | `load_state`, `active_state`, `sub_state`, `description`        |
| `inventory.packages.dpkg`        | info / observation | `operationeel`  | `version`, `arch`, `status` (observation when EOL)         |
| `inventory.packages.rpm`         | info / observation | `operationeel`  | `version`, `arch`                                          |
| `inventory.nextcloud.app`        | info          | `technologie`   | `version`, `enabled`                                            |
| `inventory.<id>.unavailable`     | info          | —               | `reason` — inspector could not run on this host                 |
| `inventory.<id>.error`           | info          | —               | `error` — inspector ran but failed mid-run                      |

See [`agent.md`](agent.md) for how to run `wanderer agent` and how
inventory data raises a dimension's Completeness from `partial` to
`complete`.

## Egress Findings

Egress Findings are produced by the `wanderer agent`'s egress probe
when it identifies an external endpoint in a config file, a process
environment, or a systemd unit. They carry `SourceModus = "egress"`.
Every non-`unknown` Finding includes a `classifier_rule` attribute
naming the rule that fired. Secrets are redacted before any value
reaches Attributes or Evidence — see [`egress.md`](egress.md) and
ADR-0008 for the redaction contract.

| ProbeID                              | severity      | dimension       | attributes (in addition to `config_source`, `config_key`, `value`, `classifier_rule`, `confidence`) |
| ------------------------------------ | ------------- | --------------- | ----- |
| `egress.object_storage`              | observation   | `juridisch`     | `provider` (aws/gcs/azure/generic), `region` (best-effort), `asn`/`organisation`/`country` when GeoLite2 is wired |
| `egress.database`                    | observation   | `juridisch`     | `provider` (postgres/mysql/…), `port`                                                              |
| `egress.smtp`                        | observation   | `data_ai`       | `port`                                                                                              |
| `egress.oidc`                        | observation   | `data_ai`       | (host only)                                                                                         |
| `egress.log_shipper`                 | observation   | `operationeel`  | (host only)                                                                                         |
| `egress.webhook`                     | observation   | `technologie`   | (host only)                                                                                         |
| `egress.unknown`                     | info          | —               | URL-shaped value the classifier could not bucket                                                    |
| `egress.<scanner>.unconfigured`      | info          | —               | scanner enabled but no paths/sources to read                                                        |
| `egress.<scanner>.error`             | info          | —               | scanner ran but failed mid-run                                                                      |
| `egress.host_resolution.unavailable` | info          | —               | emitted at most once per run when no GeoLite2 DB is wired                                           |

## Reading tips

- Group by `ProbeID`'s first segment for a per-probe view (the CLI
  output already does this).
- `ip.asn` findings joined with `http.third_party` findings give you
  the non-EU-dependency picture — which hosts does the homepage load
  from, which country/ASN are those hosted in.
- `tls.issuer.issuer_country` is a strong jurisdictional signal on its
  own; combine with `dns.ns` to see who can unilaterally affect cert
  issuance and DNS continuity.
- `Evidence` always lets you audit without re-scanning. A finding is
  never "lost" — the raw source is retained.

## Stability promise

The `Finding` struct and the ProbeID namespaces in this document are
intended to be stable across MVP-era releases. Adding new ProbeIDs is
non-breaking; renaming or removing an existing one is breaking and
should come with a documented migration path.
