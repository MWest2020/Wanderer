# mcp Specification

## Purpose
TBD - created by archiving change add-mcp-server. Update Purpose after archive.
## Requirements
### Requirement: MCP server speaks stdio JSON-RPC 2.0

The `wanderer mcp` command SHALL act as a Model Context Protocol
server over stdin/stdout, using line-delimited JSON-RPC 2.0 framing.

#### Scenario: Initialise + list tools

- **Given** a running `wanderer mcp` process with its stdin/stdout
  connected to a client
- **When** the client sends a valid `initialize` request
- **Then** the server responds with an `initialize` result declaring
  the protocol version and server capabilities
- **When** the client sends `tools/list`
- **Then** the response lists at minimum `scan_domain`, `get_scan`,
  `list_scans`

#### Scenario: Malformed request survives

- **Given** a running server
- **When** the client sends a line that is not valid JSON
- **Then** the server writes a JSON-RPC error response with code
  `-32700` ("Parse error")
- **And** the server remains running and ready for the next message

#### Scenario: Client disconnect

- **Given** a running server with no in-flight calls
- **When** stdin is closed
- **Then** the server exits with status 0 within 5 seconds

---

### Requirement: `scan_domain` tool produces a complete Scan

The `scan_domain` tool SHALL accept a domain (and optional related
domains) and return the full `Scan` record including all Findings, in
the same shape as the HTTP API's `POST /scans` response.

#### Scenario: Happy path

- **Given** a valid domain argument
- **When** the client calls `scan_domain`
- **Then** the tool response contains a `content` field with the Scan
  record JSON-encoded
- **And** `status` is one of `complete | partial | failed`
- **And** `findings` is a non-nil array

#### Scenario: Invalid domain

- **Given** an input `domain: ""`
- **When** the client calls `scan_domain`
- **Then** the response contains an MCP tool error describing the
  validation failure
- **And** no scan is persisted in the store

---

### Requirement: Resources reflect store state

Resource URIs under `wanderer://` SHALL read from the same store that
the CLI and HTTP API use. Writes via other surfaces are visible
through MCP resources without restarting the MCP server.

#### Scenario: Scan written via HTTP visible via MCP

- **Given** a scan persisted via `POST /scans`
- **When** an MCP client reads `wanderer://scans/{id}`
- **Then** the response contains the same Scan record as the HTTP
  `GET /scans/{id}` response

#### Scenario: Missing resource

- **Given** an unknown scan ID `s_missing`
- **When** the client reads `wanderer://scans/s_missing`
- **Then** the MCP response is a resource read error with a
  not-found message
- **And** the server stays running

---

### Requirement: Tool surface stays small

The MCP server SHALL expose no more than six tools in the MVP of this
change, and SHALL NOT expose tools that mutate configuration, probe
behaviour, or the database schema.

#### Scenario: No probe configuration tool

- **Given** the server is running
- **When** the client calls `tools/list`
- **Then** no tool name contains `config`, `reload`, or `reset`
- **And** the set of tools is limited to: `scan_domain`, `get_scan`,
  `list_scans`, and (when assessor is present) `assess_scan`,
  `get_assessment`

