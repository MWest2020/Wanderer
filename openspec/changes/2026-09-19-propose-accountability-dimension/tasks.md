# Tasks: accountability dimension

Implementation runs through habitat, one run per cluster below
(`runs/<nn>-*.md` is the `HABITAT_TASK_REF` for that run). A builder
run checks off only the tasks of its own cluster.

## 1. Design gate
- [x] 1.1 Review design.md: variants-probe politeness + SSRF-guard
  reuse; privacy-proxy / registry-redaction YAML location;
  expected_registrant config shape. → outcome in design.md "Design
  gate outcome".
- [ ] 1.2 Decide attribution wording (README + docs/explanation) for
  the idea lineage (Stegink) — same pattern as the DICTU credit.
  Needed before run 08.

## 2. Foundation — run 01
- [x] 2.1 `models.DimensionHint`: add `accountability`; `Valid()`
  accepts it.
- [x] 2.2 Rename `DICTUDimensions` → `WandDimensions`, append
  `accountability`; no consumer assumes a count (fix
  `report_test.go`).
- [x] 2.3 Reason codes: `Reason` on `RuleResult`, `reason`
  (`omitempty`) on `models.Rationale`; one registry table mapping code
  → class (`structural`/`gap`) + subject (`target`/`scanner`) seeded with the four codes in
  design.md; unknown code fails a test.
- [x] 2.4 `scoreDimension`: structural rationales excluded from worst
  score and completeness denominator; all-structural dimension →
  n.v.t.; table-driven tests incl. old assessment JSON without
  `reason` loading unchanged.

## 3. Organisation — run 02
- [x] 3.1 Migration: `organisations.expected_registrant` (JSON array,
  default `[]`); model + store read/write.
- [x] 3.2 `wanderer org add --expected-registrant NAME` (repeatable);
  `org show` prints the list.
- [x] 3.3 Scanner records `config.expected_registrant` for the scan's
  organisation (empty list → finding with empty list, not omitted).

## 4. RDAP — run 03
- [x] 4.1 whois: recursive entity parsing; registrant identity /
  reseller / status / expiry findings; redacted-vcard fixture.
- [x] 4.2 Fixture from live `rijksoverheid.nl` RDAP (2026-09-19):
  registrant redacted, registrar "Rijksoverheid", no reseller, no
  expiration event.
- [x] 4.3 scanner: NS registrable-domain RDAP lookups with per-scan
  cache; no-RDAP-TLD fixture.

## 5. DNS + HTTP observations — run 04
- [x] 5.1 soa probe: SOA + RNAME resolution findings; lame-delegation
  fixture; no SMTP.
- [x] 5.2 http: security.txt fetch + RFC 9116 parse; 404-is-data and
  HTML-at-path fixtures.

## 6. Variants probe — run 05
- [x] 6.1 8 paths, ≤ 5 hops/path, 24-connection budget,
  `not_followed_budget`, stop on already-verified origin.
- [x] 6.2 SSRF guard on every hop; private-redirect fixture.
- [x] 6.3 `scanner_no_ipv6`: v6 paths `not_tested` when the scanner has
  no IPv6 route (structural, subject scanner).

- [x] 6.4 Live-smoke defect (agent-lxc, 2026-09-20): `detectIPv6` counts
  ANY non-link-local IPv6 address, so a Tailscale ULA (`fd7a::/128`)
  reads as "the scanner has IPv6" while no public v6 route exists. The
  four v6 paths then came back `unreachable` and would be charged to the
  target. Capability detection must mean reachability, not "an address
  exists": ULA/CGNAT addresses do not count, and the check is a real
  short-timeout dial, made once per scan and injectable for tests.
- [x] 6.5 Live-smoke defect (same scan): the "stop on an already-verified
  origin" optimisation marked `www v6 https` as `reachable` although no
  v6 connection was ever made — the origin had been verified over v4.
  The shortcut SHALL NOT cross address families: a path may only be
  reported reachable over the family it was actually dialed on.
  Regression test with a v4-only stub.

## 7. Assessor rules + copy — run 06
- [x] 7.1 `privacy_proxies.yaml` + `registry_redaction.yaml` (seed
  `nl: SIDN`) + loaders (pattern: `package_vendors.yaml`).
- [x] 7.2 Five accountability rules + `domain_expiry` +
  `variant_convergence`; table-driven tests incl. every onbekend and
  n.v.t. path; the rijksoverheid fixture yields no_reseller soeverein,
  registrant n.v.t. (`registry_redacted`), expiry n.v.t.
  (`not_published_by_registry`).
- [x] 7.4 Follow-up from run 01 review: the engine forces a rationale's
  score to onbekend whenever it carries a reason (spec: "a rationale with
  a reason SHALL score onbekend" — run 01 passes `res.Score` through), and
  a dimension whose every rationale is structural is marked not
  applicable explicitly instead of only being derivable from its
  rationale list. Test both.
- [x] 7.3 `accountability_nl.yaml` string table (question, verdict per
  outcome, remediation) + load-time completeness test.
- [x] 7.5 Live-smoke defect (`go test ./...` outside the cage, in
  Europe/Amsterdam, 2026-09-20): `TestDomainExpiry` passed in UTC but
  failed elsewhere because its own assertion computed the expected
  date from local `time.Now()` instead of UTC — the cage cannot catch
  a test that only fails outside it. One zone for dates in verdicts:
  UTC, because RDAP `events` and RFC 9116 `Expires` both publish in
  UTC. `wand.operationeel.domain_expiry` and
  `wand.accountability.securitytxt` now format the named date via
  `.UTC()` explicitly; the test computes its expected date in UTC too.
  Added a regression test with a fixed, non-UTC zone
  (`time.FixedZone`) proving the named date does not shift.

## 8. UI — run 07
- [x] 8.1 Answer-sheet section on the assessment report (Dutch copy
  from the table, verdicts, remediation lines, evidence expanders).
- [x] 8.2 Accountability pill on the Overview rows; "not assessed" and
  "n.v.t." states; overall score names the dimensions it covers.
- [x] 8.3 Review pass against Wordsworth's tone/interaction patterns;
  scanner-subject reasons render as operator warnings, not answers;
  a `.nl` fleet must still read as a story (four non-n.v.t.
  questions carry it); verdict copy readable by a non-specialist or
  it goes back.
- [x] 8.5 De spec uit 8.4 draait niet: `tests/playwright/playwright.config.ts`
  kent hem niet (elke project noemt zijn specs in `testMatch`), en de
  `baseline`-fixture (`internal/fixtures`) bevat geen
  accountability-findings. Zet de spec in het baseline-project en vul de
  fixture aan met een scan die alle vier de antwoordtoestanden toont
  (ja / nee / onbekend / n.v.t.), zodat de spec echt iets bewijst.
- [ ] 8.4 Playwright smoke: expand evidence, onbekend vs n.v.t.
  rendering.

- [x] 8.6 UI-defect, gevonden toen de spec eindelijk draaide: de
  rapportpagina rendert per framework een dimensiekaart met
  `id="<dimensie>"`, dus met twee frameworks staan er twee elementen met
  `id="accountability"` op één pagina. Dubbele id's zijn ongeldige HTML
  en maken de ankerlink van de Overview-pil dubbelzinnig (de test faalt
  erop met "resolved to 2 elements"). Maak het anker uniek per framework
  (bijv. `wand-accountability`), laat de pil daarheen wijzen en pas de
  spec aan. Alleen de ankers; de kaartinhoud blijft.

## 9. Wrap-up — run 08
- [ ] 9.1 docs/reference/assessor.md + findings.md updates (reason
  codes, new findings, new dimension); CHANGELOG.
- [ ] 9.2 docs/explanation note: RDAP fields we wish existed
  (actor/escalation), the `.nl` passive ceiling, why both are out of
  scope.
- [ ] 9.3 Attribution text from 1.2.
- [ ] 9.4 Archive the change.
