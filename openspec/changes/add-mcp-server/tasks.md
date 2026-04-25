# Tasks: MCP Server

## 1. Protocol evaluation + ADR

- [x] 1.1 Evaluate official Go MCP SDK availability + maturity
- [x] 1.2 Decide SDK vs hand-roll; record in `docs/decisions/0005-mcp-transport.md`

## 2. Server scaffolding

- [x] 2.1 `internal/mcp/server.go` — JSON-RPC dispatcher over stdio
- [x] 2.2 `internal/mcp/tools.go` — tool registration + schema validation
- [x] 2.3 `internal/mcp/resources.go` — URI parsing + dispatch
- [x] 2.4 `internal/mcp/schema.go` — JSON Schema fragments

## 3. Tool handlers

- [x] 3.1 `scan_domain` — calls `scanner.Scan`, returns Scan JSON
- [x] 3.2 `get_scan` — reads from store
- [x] 3.3 `list_scans` — paginated list
- [x] 3.4 `assess_scan` / `get_assessment` — gated on assessor presence (build tag or import gate)

## 4. Resource handlers

- [x] 4.1 `wanderer://scans` — summary list
- [x] 4.2 `wanderer://scans/{id}` — full Scan
- [x] 4.3 `wanderer://scans/{id}/findings` — findings-only view
- [x] 4.4 `wanderer://assessments` + `wanderer://assessments/{id}` — gated on assessor

## 5. CLI

- [x] 5.1 `cmd/wanderer/mcp.go` — flag parsing (`--db`, `--geoip`, timeouts), server bootstrap, signal handling
- [x] 5.2 Register `mcp` in `cmd/wanderer/main.go`

## 6. Tests

- [x] 6.1 Unit tests per tool handler with a fake store
- [x] 6.2 Integration test: spawn `wanderer mcp`, pipe JSON-RPC, assert responses
- [x] 6.3 Contract test: each tool response round-trips through JSON Schema validation

## 7. Docs + CHANGELOG

- [x] 7.1 `docs/mcp.md` — install snippet for Claude Desktop + Claude Code, list of tools, example conversation
- [x] 7.2 Update `docs/README.md` index
- [x] 7.3 CHANGELOG entry under `Added`
