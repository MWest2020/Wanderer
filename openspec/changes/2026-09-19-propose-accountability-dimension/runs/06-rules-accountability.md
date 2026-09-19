# Habitat run 06 — accountability rules (tasks 7.1–7.2, accountability only)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(specs/assessor/spec.md, design.md "Reason codes", "Privacy proxy vs
registry redaction", "Expected registrant matching").

## Scope — ONLY these tasks
- [ ] 7.1 Two YAML lists next to the rules in `internal/assessor/wand`,
  embedded and loaded like `package_vendors.yaml`:
  `registry_redaction.yaml` (per TLD the registry that redacts
  registrant data for everyone; seed `nl: SIDN`) and
  `privacy_proxies.yaml` (commercial proxy services: "Domains By
  Proxy", "Whois Privacy", …). Loader + table-driven tests.
- [ ] 7.2a The FIVE accountability rules, registered in the wand pack on
  dimension `accountability`:
  - `registrant_identifiable` — order matters: TLD on the redaction list
    → onbekend + reason `registry_redacted` WHATEVER the vcard says;
    else proxy-list match → afhankelijk; else name matches a name from
    the scan's `config.expected_registrant` finding (case-insensitive,
    legal-form suffixes B.V./N.V./Stichting normalised away, no fuzzy
    matching) → soeverein; present but unmatched → voldoende; RDAP
    unavailable → onbekend + reason `probe_unavailable`.
  - `no_reseller` — reseller present → afhankelijk (name it); direct →
    soeverein; roles unparseable → onbekend.
  - `soa_rname` — mailbox domain resolves + MX → soeverein; resolves
    without MX → voldoende; does not resolve → afhankelijk; SOA failed →
    onbekend. The verdict SHALL say delivery itself was not verified.
  - `securitytxt` — present + parseable + Contact + unexpired Expires →
    soeverein; present but expired or missing fields → voldoende; absent
    (404 is data) → afhankelijk; transport failure → onbekend.
  - `ns_holder_transparent` — all identifiable → soeverein; some →
    voldoende (name the opaque domains); none → afhankelijk; no lookup
    succeeded → onbekend.
  Table-driven tests covering EVERY onbekend and n.v.t. path, plus the
  rijksoverheid.nl fixture: `no_reseller` soeverein, registrant onbekend
  with reason `registry_redacted`.

Verdict strings stay English/neutral for now; the Dutch answer-sheet
copy is run 07's string table, and rules must not hard-code UI copy
beyond their one-sentence verdict.

Done = these two checkboxes checked; tick 7.1 in tasks.md and, for 7.2,
tick it only when run 07 has added the two operationeel rules — instead
note in your run report that 7.2's accountability half is done.

## Out of scope for this run
`domain_expiry`, `variant_convergence`, the NL string table, the 7.4
follow-up, and all UI work — those are run 07 and run 08.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/); `openspec validate 2026-09-19-propose-accountability-dimension
--strict` green. Budget is $5 — leave room for your run report.
