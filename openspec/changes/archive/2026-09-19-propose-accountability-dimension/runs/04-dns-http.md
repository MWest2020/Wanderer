# Habitat run 04 — SOA + security.txt (tasks 5.1–5.2)

Contract: `openspec/changes/2026-09-19-propose-accountability-dimension/`
(specs/scanner/spec.md requirements "SOA probe observes RNAME
contactability" and "HTTP probe fetches security.txt"; design.md
"External systems and failure modes" and the SMTP valkuil).

## Scope — ONLY these tasks
- [ ] 5.1 New soa probe (`internal/probe/soa`, wired into the scanner's
  probe list like the existing probes): query the zone's SOA and emit
  `dns.soa` with MNAME + RNAME, plus the resolution result of the RNAME
  mailbox domain (resolves yes/no, MX present yes/no). The RNAME is in
  RFC 1035 form — the first unescaped dot separates local part from
  domain (`hostmaster.voorbeeld.nl` → `hostmaster@voorbeeld.nl`), and an
  escaped dot (`\.`) belongs to the local part. NO SMTP connections,
  ever: existence + MX is the passive ceiling. On query failure emit
  `dns.soa.unavailable` and let other probes continue. Fixtures: a
  normal zone, an escaped-dot RNAME, and a lame delegation (timeout).
- [ ] 5.2 `internal/probe/http`: additionally fetch
  `/.well-known/security.txt` over HTTPS and emit `http.securitytxt`
  with presence, parseability, Contact and Expires (RFC 9116). A 404 is
  a VALID observation → present=false, not an error. Only a
  transport-level failure emits `http.securitytxt.unavailable`. An HTML
  page at the well-known path → parseable=false. Cap the body read so a
  huge response cannot blow up memory. Fixtures: valid file, 404,
  HTML-at-path, transport failure.

No assessor rules in this run — findings are observations only. Tests
are table-driven and use httptest / a stub resolver, never the live
network.

Done = these two checkboxes checked, and 5.1–5.2 ticked in tasks.md.

## Out of scope for this run
No rules, no YAML lists, no variants probe, no whois/ns_holder work (run
03), no UI. Follow-up 7.4 is not yours.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/, see openspec/config.yaml);
`openspec validate 2026-09-19-propose-accountability-dimension --strict`
green. Budget is $5 — leave room to write your run report.
