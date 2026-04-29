# web-ui Specification

## Purpose
Read-only operator UI for Wanderer. Mounted at `/ui/` on the existing
chi router behind a `--ui` flag, served by `internal/ui` using Go
`html/template` and vanilla CSS. Provides browse access to targets,
scans, findings, drift, and assessments without exposing any mutating
endpoints — public-sector operators can review evidence in a browser
without enabling write paths.

## Requirements
### Requirement: Wanderer ships a read-only operator UI behind a flag

The `wanderer serve` command SHALL mount a read-only HTML interface
at `/ui/` on the existing chi router when `--ui` is set, rendering
its three pages (`/ui/`, `/ui/scans/{id}`,
`/ui/targets/{id}/drift`) with Go `html/template` and serving
vanilla CSS as static assets, so an operator can read scans,
findings, and drift in a browser without enabling any mutating
endpoint.

#### Scenario: UI flag enables the routes

- **Given** `wanderer serve --ui`
- **When** an operator opens `/ui/` in a browser
- **Then** the page lists every persisted Target with its last
  scan status and last Assessment score per framework

#### Scenario: UI flag absent keeps the routes off

- **Given** `wanderer serve` without `--ui`
- **When** any client requests `/ui/`
- **Then** the response status is 404
- **And** no template is rendered

#### Scenario: Scan page groups findings

- **Given** a stored Scan with findings across DNS, TLS, IP, HTTP
- **When** the operator opens `/ui/scans/<id>`
- **Then** the page renders one section per probe prefix
- **And** each finding's severity is colour-coded

---

### Requirement: UI authenticates via HTTP Basic when configured

When `--ui-htpasswd <file>` is set, every `/ui/*` request SHALL
require a matching credential from the htpasswd file, accepting
bcrypt and SHA-512 entries and rejecting MD5, with the htpasswd
file re-read on every request so an operator can rotate
credentials without restarting the binary.

#### Scenario: Bcrypt entry accepts the right password

- **Given** an htpasswd file containing one bcrypt entry for user
  `op` with password `correct horse battery staple`
- **When** a request to `/ui/` arrives with that Basic header
- **Then** the response is the index page with status 200

#### Scenario: Wrong password rejected

- **Given** the same htpasswd file
- **When** a request arrives with `op:wrong`
- **Then** the response status is 401
- **And** the `WWW-Authenticate: Basic` header is set

#### Scenario: MD5 entry rejected at config load

- **Given** an htpasswd file whose first entry uses MD5 (`$apr1$`)
- **When** the server starts
- **Then** the process exits non-zero
- **And** stderr names MD5 as the unsupported algorithm

---

### Requirement: UI surface stays read-only

The `internal/ui` package SHALL register only HTTP GET handlers
and SHALL NOT register any handler that mutates store state
(POST, PUT, PATCH, DELETE), enforced by a static-analysis test
that greps the package for those method names and fails the build
if any are present.

#### Scenario: Mutation handlers fail the build

- **Given** a contributor adds `r.Post("/ui/foo", ...)` to
  `internal/ui/ui.go`
- **When** `go test ./internal/ui/...` runs
- **Then** the package's static-analysis test fails with a clear
  message naming the offending file
