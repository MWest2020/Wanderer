---
status: draft
last_reviewed: 2026-09-20
---

# Accountability: the passive ceiling

The `accountability` dimension (see
[assessor.md](../reference/assessor.md#the-accountability-dimension))
asks who is answerable for a piece of infrastructure, and whether
that party is actually reachable. Two of its inputs — the RDAP
registrant and the RDAP expiration event — hit a hard ceiling before
Wanderer's own design choices even come into play: what the protocol
exposes, and what individual registries choose to publish through it.
This note records both limits explicitly, so "n.v.t." on a `.nl`
report reads as a documented boundary rather than a missing feature.

## RDAP fields we wish existed

RDAP (RFC 9083) is a real improvement over WHOIS-43 — structured
JSON, defined entity roles, a stable HTTP transport — but it still
does not carry two kinds of information that would sharpen the
accountability picture:

- **Actor/escalation relationships.** RDAP models *roles*
  (`registrant`, `registrar`, `reseller`, `administrative`,
  `technical`, `abuse`) but not an *escalation path* between them —
  which contact is authoritative when the registrant is unreachable,
  or which internal relationship binds a reseller to the registrar of
  record beyond both simply appearing in the same entity tree. Wanderer
  infers "a reseller sits between registrant and registrar" from
  nesting (`entities[].entities[]`), which is a convention several
  registries follow, not a field RDAP defines. A first-class
  escalation-role vocabulary would let `no_reseller` (and a future
  "who do I escalate to" rule) read a declared relationship instead of
  inferring one from response shape.
- **ISO2-prefixed subject identifiers.** Entities carry a `handle`
  (registry-assigned, opaque, and namespaced per registry) but no
  globally comparable identifier analogous to, say, an ISO
  3166-1-alpha-2-prefixed legal-entity ID. Cross-referencing "is this
  registrant the same legal entity as the one behind that other
  domain" currently has no protocol-level anchor; Wanderer does not
  attempt it (name-string comparison against operator-declared names
  is the whole of `registrant_identifiable`'s matching logic — see
  design.md "Expected registrant matching").

Both gaps are IETF/ICANN protocol work, not something a scanner can
paper over by inference without turning "observation" into
"speculation". Wanderer does not implement a work-around for either;
this note exists so the gap is visible rather than silently absorbed
into "the registrant looked unmatched."

## The `.nl` passive ceiling

SIDN (the `.nl` registry) redacts registrant data for **every** `.nl`
domain and publishes **no** expiration event in its RDAP responses —
a blanket registry policy, not a per-domain choice by the registrant.
Concretely, for any `.nl` target:

- `whois.registrant_identity` always reports a redacted or absent
  name (frequently accompanied by an RFC 9537 `redacted` array, see
  the `rijksoverheid.nl` fixture referenced from the proposal).
- `whois.expiry` always reports `present: false` — there is no
  expiration event to parse, ever.

Two of the five plain-language accountability questions —
"do we know who is behind this domain" and "will the registration
lapse unexpectedly" — are therefore **structurally unanswerable** for
`.nl` via RDAP, not gaps in Wanderer's implementation. The
`registry_redaction.yaml` TLD list (seeded `nl: SIDN`) makes this an
explicit, checked-first branch in `registrant_identifiable`, and the
`not_published_by_registry` reason code does the same for
`domain_expiry`: both render **n.v.t.** with a reason, excluded from
the dimension's worst-score and completeness computation, rather than
"onbekend" (which would read as "we tried and failed") or a guessed
answer (see [assessor.md](../reference/assessor.md#reason-codes)).

**Wanderer does not work around this.** Three tempting work-arounds
were considered and rejected, not for lack of feasibility but because
each trades the scanner's passive, boring, auditable posture for a
capability that belongs to a different kind of tool:

- **Registrar APIs.** Most registrars expose an authenticated API that
  *does* return the registrant on request. Using it would mean
  Wanderer holds registrar credentials for every scanned domain — an
  authenticated external dependency, a new secret to rotate and leak,
  and a relationship Wanderer would need per registrar. Out of
  proportion to what one accountability rule needs.
- **Scraping the registrar's own web WHOIS page.** Fragile (HTML
  changes break it silently), against most registrars' terms of
  service, and reintroduces exactly the un-auditable "we scraped a web
  page" evidence Wanderer's Finding/Evidence model exists to avoid.
- **Live KVK (Dutch trade register) lookups.** Would "prove" a Dutch
  legal entity's existence independent of RDAP, but adds an
  authenticated external dependency, rate limits, and a data-quality
  rabbit hole for a deployment model where organisations scan
  themselves — they already know their own name. Operator-declared
  `expected_registrant` names are boring, auditable, and correct for
  that model; a live KVK lookup would answer a question nobody
  disputed.

For `.nl` fleets, the dimension's story is carried by the other three
rules — `no_reseller`, `soa_rname`, and `securitytxt` — plus
`ns_holder_transparent`, none of which touch registry-redacted
fields. Phase-2 work on any registry-redaction work-around is
explicitly out of scope for this change (see the proposal's "Scope /
Not in scope"); this note is the record of why, so a future proposal
that wants to revisit it starts from the actual constraint instead of
rediscovering it.
