# Delta for scanner

> Draft. Probe extensions feeding the accountability dimension. All
> probes stay isolated: any failure emits `*.unavailable` and never
> cascades.

## ADDED Requirements

### Requirement: whois probe emits identity, reseller, status, and expiry findings

The whois probe SHALL, from the same RDAP response it already
fetches, additionally emit `whois.registrant_identity` (name, kind,
privacy-proxy flag), `whois.reseller` (when a reseller entity is
present), `whois.status` (RDAP status codes), and `whois.expiry`
(the expiration event date, or `absent` when the registry publishes
none). Entities SHALL be parsed recursively, so a reseller nested
under the registrar entity is found. Redacted or missing fields SHALL
be emitted as explicit redacted/absent values, never omitted silently.

#### Scenario: Full RDAP response

- **GIVEN** an RDAP response with registrant vcard, reseller entity,
  status codes, and an expiration event
- **WHEN** the whois probe runs
- **THEN** all four new findings are emitted alongside the existing
  registrant/registrar findings

#### Scenario: Reseller nested under the registrar

- **GIVEN** an RDAP response whose reseller entity appears only inside
  the registrar entity's `entities` array
- **WHEN** the whois probe runs
- **THEN** `whois.reseller` names that reseller

#### Scenario: Organisation is its own registrar (live fixture)

- **GIVEN** the RDAP response for `rijksoverheid.nl` captured
  2026-09-19 (registrant "REDACTED FOR PRIVACY", registrar
  "Rijksoverheid", no reseller, events registration / last changed
  only)
- **WHEN** the whois probe runs
- **THEN** `whois.reseller` records no reseller and `whois.expiry`
  records `absent`

#### Scenario: Redacted vcard

- **GIVEN** an RDAP response whose registrant vcard is "REDACTED FOR
  PRIVACY"
- **WHEN** the whois probe runs
- **THEN** `whois.registrant_identity` is emitted with the proxy flag
  set, so the assessor can score afhankelijk instead of onbekend

---

### Requirement: SOA probe observes RNAME contactability

A new soa probe SHALL query the zone's SOA record and emit `dns.soa`
carrying MNAME and RNAME, plus the resolution result of the RNAME
mailbox domain (resolves yes/no, MX present yes/no). The probe SHALL
NOT open SMTP connections. On query failure it SHALL emit
`dns.soa.unavailable`.

#### Scenario: Normal zone

- **GIVEN** a zone whose SOA RNAME is `hostmaster.voorbeeld.nl`
- **WHEN** the soa probe runs
- **THEN** `dns.soa` reports the RNAME with resolves=true and the MX
  observation

#### Scenario: Lame delegation

- **GIVEN** a zone whose authoritative servers time out
- **WHEN** the soa probe runs
- **THEN** the probe emits `dns.soa.unavailable` within its timeout
  and other probes are unaffected

---

### Requirement: HTTP probe fetches security.txt

The HTTP probe SHALL fetch `/.well-known/security.txt` over HTTPS and
emit `http.securitytxt` with presence, parseability, Contact, and
Expires. A 404 SHALL be emitted as present=false (a valid
observation); only transport-level failure SHALL emit
`http.securitytxt.unavailable`.

#### Scenario: Present and valid

- **GIVEN** a target serving a parseable security.txt
- **WHEN** the HTTP probe runs
- **THEN** `http.securitytxt` carries Contact and Expires

#### Scenario: HTML error page at the path

- **GIVEN** a target answering 200 with an HTML page at the
  well-known path
- **WHEN** the HTTP probe runs
- **THEN** `http.securitytxt` is emitted with parseable=false

---

### Requirement: Variants probe observes path convergence

A new variants probe SHALL attempt the 8 paths apex/www × IPv4/IPv6 ×
http/https with redirect depth at most 5 per path, a hard budget of 24
connections per target across all paths, no retries, within the
scan's global timeout, and emit `http.variants` recording per path:
status (`reachable`, `unreachable`, `refused`, `not_followed_budget`,
`not_tested`), redirect chain, and final origin. A chain MAY stop as
soon as it reaches an origin already verified in the same run. Every
redirect hop SHALL pass the existing SSRF guard. When the scanner host
has no IPv6 route, v6 paths SHALL be recorded as `not_tested` with
reason `scanner_no_ipv6`. On total failure the probe SHALL emit
`http.variants.unavailable`.

#### Scenario: Converging paths

- **GIVEN** a target where all live paths redirect to
  `https://www.voorbeeld.nl`
- **WHEN** the variants probe runs
- **THEN** `http.variants` records convergence on that single origin

#### Scenario: Budget exhausted

- **GIVEN** a target whose paths each redirect 5 times before
  converging
- **WHEN** the variants probe runs
- **THEN** no more than 24 connections are opened
- **AND** the paths not reached are recorded as `not_followed_budget`

#### Scenario: Redirect into private address space

- **GIVEN** a path whose redirect chain points at 10.0.0.5
- **WHEN** the variants probe runs
- **THEN** the SSRF guard refuses the hop and the path is recorded as
  `refused`, not followed

---

### Requirement: Scanner looks up NS holder transparency

The scanner SHALL, for each unique registrable domain among the
target's `dns.ns` hosts, perform one RDAP lookup and emit
`whois.ns_holder` (registrant present/proxied/absent). Lookups SHALL
be cached per scan so N nameservers under one provider cost one
lookup; failures emit `whois.ns_holder.unavailable` per domain.

#### Scenario: Two providers, three nameservers

- **GIVEN** ns1/ns2.provider-a.nl and ns1.provider-b.eu
- **WHEN** the scan runs
- **THEN** exactly two RDAP lookups occur and two `whois.ns_holder`
  findings are emitted

#### Scenario: Registry without RDAP

- **GIVEN** a nameserver under a TLD with no RDAP service (404 from
  the bootstrap)
- **WHEN** the scan runs
- **THEN** `whois.ns_holder.unavailable` is emitted for that domain
  and the scan continues

---

### Requirement: Organisations declare expected registrant names

An organisation SHALL carry a list `expected_registrant` of names
(stored on the `organisations` table, default empty), set with
`wanderer org add <slug> --expected-registrant NAME` (repeatable;
`org add` upserts) and shown by `wanderer org show`. There SHALL be no
per-target override. At scan time the scanner SHALL record the
list of the scan's organisation as a `config.expected_registrant`
finding, also when the list is empty, so the assessment stays a
function of findings and shows which names were expected when the
scan ran.

#### Scenario: Declared names reach the assessment

- **GIVEN** organisation `voorbeeld` with `expected_registrant:
  ["Gemeente Voorbeeld"]`
- **WHEN** a scan of one of its targets runs
- **THEN** the scan contains a `config.expected_registrant` finding
  listing "Gemeente Voorbeeld"

#### Scenario: Changing the list does not rewrite history

- **GIVEN** a stored scan whose `config.expected_registrant` lists
  "Gemeente Voorbeeld"
- **WHEN** the organisation's list is later changed and the old scan
  is re-assessed
- **THEN** the re-assessment uses the names recorded in the old scan
