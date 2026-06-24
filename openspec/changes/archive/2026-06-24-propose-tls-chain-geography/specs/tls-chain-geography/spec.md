# Delta for tls-chain-geography

> New capability — graduates from `research-high-signal-observability`
> (closes Wave 2, the sibling split off from CDN/front detection). Leads
> with the observed CA + chain fact; the existing
> `wand.juridisch.cert_issuer_eea` rule annotates it. DICTU dimension:
> Juridisch (CA jurisdiction). The seventh and last who/where signal.

## ADDED Requirements

### Requirement: Wanderer names the CA that issued a target's certificate and where it sits

Wanderer SHALL, after reading a target's leaf `tls.issuer` (issuer
organisation, common name, country) and `tls.chain` (intermediates), emit
one aggregate observed Finding that names the recognisable issuing
**Certificate Authority** (derived from a known-CA table keyed on the
issuer organisation and common name) and its **jurisdiction** (from the
issuer country when present), and states the certificate **chain** (leaf
issuer → intermediates). The Finding SHALL read as a plain statement —
"the TLS certificate for {domain} is issued by {CA} ({country})" — and
SHALL retain the raw issuer organisation, common name, and intermediate
names as evidence so the observed fact stands even when the friendly CA
name is uncertain. This Finding is an observation, not a verdict; the
EEA-issuer score is carried separately by `wand.juridisch.cert_issuer_eea`.

#### Scenario: A non-EEA CA reads as a named authority

- **GIVEN** a target whose leaf certificate issuer organisation is "Let's
  Encrypt" with issuer country "US"
- **WHEN** the scan reads `tls.issuer` and `tls.chain`
- **THEN** the scan contains an aggregate `tls.chain_geography` Finding
  naming the CA "Let's Encrypt" and country "US",
- **AND** the raw issuer organisation and chain are retained as evidence,
- **AND** `wand.juridisch.cert_issuer_eea` still scores the issuer as
  outside the EEA

#### Scenario: Issuer country absent still names the CA

- **GIVEN** a leaf certificate whose issuer organisation is "DigiCert" but
  whose issuer country field is empty
- **WHEN** the scan synthesises the chain geography
- **THEN** the Finding names the CA "DigiCert" with the jurisdiction
  reported as undetermined, and does not fail

#### Scenario: Unrecognised issuer falls back to the raw organisation

- **GIVEN** a leaf certificate whose issuer is absent from the CA table
- **WHEN** the scan synthesises the chain geography
- **THEN** the CA is named from the raw issuer organisation (or common
  name when the organisation is empty), so the observed fact still stands

#### Scenario: The certificate signal renders as a first-class flow row

- **GIVEN** a scan whose `cert_issuer_eea` rule produced a verdict
- **WHEN** the Sovereignty overview is rendered
- **THEN** a **Certificate** flow row shows the issuer verdict alongside
  the other who/where flows

#### Scenario: No certificate evidence does not fail

- **GIVEN** a target whose TLS probe produced no `tls.issuer` finding (the
  probe failed or the service is plain HTTP)
- **WHEN** the scan synthesises the chain geography
- **THEN** no `tls.chain_geography` Finding is emitted and the scan
  completes
