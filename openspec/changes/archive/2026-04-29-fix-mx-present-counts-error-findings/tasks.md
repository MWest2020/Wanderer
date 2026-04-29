# Tasks: Rules ignore meta Findings

## 1. Helper

- [x] 1.1 Add `IsEvidenceLike(f models.Finding) bool` to `internal/assessor/rule.go`
- [x] 1.2 Unit test the helper with one positive and three negative cases (error / no_answer / unavailable)

## 2. Rule fixes

- [x] 2.1 `mxPresent` consults `IsEvidenceLike` before counting
- [x] 2.2 Audit every other rule in `internal/assessor/dictu/rules.go` that does bare-ProbeID counting and apply the helper where it matters
- [x] 2.3 Regression test: NXDOMAIN-style finding set produces Onbekend on `mx_present`

## 3. Docs

- [x] 3.1 Document the meta-finding convention in `docs/findings.md`
- [x] 3.2 CHANGELOG entry under `### Fixed`

## Audit notes (2.2)

The other DICTU rules already filter meta-rows incidentally because
they require an evidence attribute the meta-row does not carry:

- `mxVendorJurisdiction`, `dnsRedundancy`: skip rows where the `host`
  attribute is empty (lookup-error rows have no `host`).
- `caaRestricts`: explicitly handles `no_answer` already; otherwise
  requires `tag` + `value`.
- `apexIpEEA`, `thirdPartiesEEA`, `noUSHyperscaler`, `certIssuerEEA`,
  `certValidity`, `registrarJurisdiction`: filter on `ProbeID`s
  (`ip.asn`, `tls.issuer`, `tls.validity`, `whois.registrant`,
  `http.third_party`) that are emitted only on the evidence path; the
  meta variants live under different ProbeIDs (`ip.unavailable`,
  `tls.ct.unavailable`, `whois.unavailable`, `http.parse_failed`) and
  are excluded by the existing string match.

`mxPresent` was the only rule using a bare `ProbeID == "dns.mx"`
counter, so it was the only rule needing the helper. The helper is
exported so future rules adopting the same pattern can lean on it.
