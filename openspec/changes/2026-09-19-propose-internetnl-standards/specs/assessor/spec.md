# Delta for assessor

> Draft. New wand dimension `standards`, fed exclusively by imported
> `internetnl.*` findings. No first-party probes.

## ADDED Requirements

### Requirement: Standards rules score imported Internet.nl verdicts only

The wand rule pack SHALL provide six rules under the `standards`
dimension — `wand.standards.dnssec`, `wand.standards.mail_auth`,
`wand.standards.starttls_dane`, `wand.standards.ipv6`,
`wand.standards.rpki`, `wand.standards.tls_config` — each scoring
solely from imported `internetnl.web.*` / `internetnl.mail.*`
findings by verdict mapping (all passed → soeverein, warnings/mixed →
voldoende, failed → afhankelijk, not_tested/absent/stale → onbekend).
The rules SHALL NOT recompute or aggregate Internet.nl's percentage
score, and no first-party Wanderer finding SHALL feed these rules.

#### Scenario: Signed and valid DNSSEC

- **GIVEN** imported findings where every dnssec-category test is
  passed
- **WHEN** the assessor runs the wand rule pack
- **THEN** `wand.standards.dnssec` scores soeverein and links the
  Internet.nl report URL in the evidence

#### Scenario: Failed mail authentication

- **GIVEN** imported findings where DMARC tests are failed
- **WHEN** the assessor runs
- **THEN** `wand.standards.mail_auth` scores afhankelijk and names
  the failing standard(s)

#### Scenario: No import present

- **GIVEN** a target with perimeter findings but no `internetnl.*`
  findings
- **WHEN** the assessor runs
- **THEN** every standards rule scores onbekend with reason "not
  measured" and no other dimension is affected

---

### Requirement: Stale measurements are not presented as current

When a target's newest `internetnl.*` findings are older than the
configured `standards.max_age` (default 30 days), the standards rules
SHALL score onbekend and the verdict SHALL name the measurement date,
so an old pass can never mask a regression.

#### Scenario: Measurement past max_age

- **GIVEN** imported findings with measured_at 45 days ago and the
  default max_age
- **WHEN** the assessor runs
- **THEN** `wand.standards.dnssec` scores onbekend with "measurement
  stale" and the date

#### Scenario: Fresh re-import restores scoring

- **GIVEN** a subsequent import with measured_at yesterday
- **WHEN** the assessor runs
- **THEN** the rules score from the fresh findings and the stale ones
  are ignored

---

### Requirement: Standards rules never double-score first-party ground

Categories where Wanderer keeps first-party probes for evidence
granularity (security.txt, HTTPS variant convergence, security
headers, certificate validity) SHALL remain scored exclusively by
their existing first-party rules; the standards dimension SHALL NOT
add rules over the corresponding Internet.nl subtests.

#### Scenario: security.txt stays first-party

- **GIVEN** an import whose web results include Internet.nl's
  security.txt subtest
- **WHEN** the assessor runs
- **THEN** no standards rule scores security.txt and
  `wand.accountability.securitytxt` scores solely from the
  first-party `http.securitytxt` finding
