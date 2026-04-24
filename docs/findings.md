# Findings Reference

Every probe produces `models.Finding` records with a shared shape. This
document is the catalogue: which ProbeIDs exist, what they mean, which
attributes they carry, and which DICTU dimension they inform.

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
