## ADDED Requirements

### Requirement: The entry surface asks for a domain and answers it

`/ui/` SHALL lead with a single input that takes a domain (or selects a
host that reports through an agent) and, on submit, starts a scan and
navigates to that target's answer page. Below it SHALL sit the recently
answered targets as one-line verdicts. Scan IDs, rule IDs and the rule
matrix SHALL NOT appear on this surface. Submitting SHALL require a
signed-in user.

#### Scenario: A person types a domain

- **GIVEN** a signed-in operator on `/ui/`
- **WHEN** they enter `voorbeeld.nl` and submit
- **THEN** a scan starts and they land on that target's answer page
- **AND** they never see a scan ID to do it

#### Scenario: Not signed in

- **GIVEN** a request without a session
- **WHEN** it posts a domain to the scan route
- **THEN** it is refused and redirected to the login, and no scan starts

---

### Requirement: The answer is one sentence with its reason

The answer page SHALL open with one sentence in Dutch stating ja, nee
or onbekend for "does this stand under Dutch / European law?", naming
the observation that decided it. `soeverein` and `voldoende` SHALL read
as ja; `afhankelijk` SHALL read as nee; `onbekend` SHALL stay onbekend
and SHALL NEVER be promoted to ja. When questions could not be
answered, the headline SHALL say how many.

#### Scenario: Mail abroad decides the verdict

- **GIVEN** an assessment where every flow is soeverein except mail,
  which is afhankelijk on a US provider
- **WHEN** the answer page renders
- **THEN** the headline reads "Nee" and names the mail provider

#### Scenario: Unknown is not a yes

- **GIVEN** an assessment where hosting is soeverein and DNS is
  onbekend because GeoIP was unavailable
- **WHEN** the answer page renders
- **THEN** the headline does not read a plain "Ja", and states that one
  question could not be answered

---

### Requirement: The answer fills in while the scan runs

The answer page SHALL render from the findings that exist at that
moment and SHALL refresh itself until the scan is complete, without the
operator reloading. Each flow SHALL show its own state (nog bezig /
answered / niet gemeten). The page SHALL work without JavaScript, in
which case it falls back to a meta-refresh.

#### Scenario: Slow probe does not hold the page

- **GIVEN** a running scan where DNS and TLS have landed and the
  traceroute has not
- **WHEN** the operator opens the answer page
- **THEN** the DNS and TLS flows show their verdicts and the transit
  flow shows "nog bezig"

#### Scenario: Scan finishes while the page is open

- **GIVEN** the answer page open on a running scan
- **WHEN** the last probe lands
- **THEN** the page shows the final verdict without a manual reload

---

### Requirement: The reasoning is one click from the answer

From the answer page one link SHALL lead to the reasoning: the seven
sovereignty flows plus accountability, each as a plain-language
question with ja / nee / onbekend / n.v.t., the observed fact in the
verdict, and the evidence collapsed beneath. Rule IDs and protocol
jargon SHALL appear only inside the evidence.

#### Scenario: Explorer opens the reasoning

- **GIVEN** an answer page for a completed scan
- **WHEN** the operator follows the link to the reasoning
- **THEN** each flow is a question with an answer, and no rule ID
  appears outside an expanded evidence block

## MODIFIED Requirements

### Requirement: UI surface stays read-only

The UI SHALL remain read-only except for starting a scan, which a
signed-in user MAY do from the entry surface. Every other mutation
(targets, organisations, assessments, exports) SHALL stay with the API
and the CLI. When the instance has no authentication configured at all,
the scan route SHALL be refused rather than served open.

#### Scenario: Scanning is the only mutation

- **WHEN** the UI is served
- **THEN** the only route that changes state is the scan route, and it
  requires a signed-in user

#### Scenario: Mutation handlers fail the build

- **Given** a contributor adds `r.Post("/ui/foo", ...)` to
  `internal/ui/ui.go`, other than the sanctioned scan route
- **When** `go test ./internal/ui/...` runs
- **Then** the package's static-analysis test fails with a clear
  message naming the offending file

#### Scenario: No authentication configured

- **GIVEN** an instance started without OIDC and without htpasswd
- **WHEN** a request posts to the scan route
- **THEN** it is refused, and the startup log says the scan route is
  disabled for lack of authentication
