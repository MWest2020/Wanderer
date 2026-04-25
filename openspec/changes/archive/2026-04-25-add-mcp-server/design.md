# Design: MCP Server

## Component placement

```
cmd/wanderer/mcp.go       # CLI entry: `wanderer mcp`
internal/mcp/
  server.go               # MCP protocol loop (read stdin, write stdout)
  tools.go                # tool definitions + dispatch
  resources.go            # resource URI handling
  schema.go               # JSON Schema fragments for tool params
```

The MCP server is a thin adapter. `tools.scanDomain` calls
`scanner.New(...).Scan(...)` — the same entry point the CLI uses. No
business logic lives inside `internal/mcp`.

## Transport

Stdio with JSON-RPC 2.0 framing, per the MCP spec
(<https://modelcontextprotocol.io>). Line-delimited JSON, one message
per line. Content-Length framing is not needed for stdio local
transport.

Tool calls and resource reads go through the same JSON-RPC dispatch
loop; `internal/mcp/server.go` dispatches by method name.

## MCP library choice

We evaluate two options at implementation time:

1. **Official Go SDK** if one exists and is maintained — preferred.
2. **Hand-rolled JSON-RPC loop** over `stdin`/`stdout` — small
   (~200 LOC), acceptable if no stable SDK.

An ADR will record which one we picked and why. Either choice is
reversible: the tool/resource handlers sit behind an interface.

## Tool: `scan_domain`

```json
{
  "name": "scan_domain",
  "description": "Run a Wanderer scan against a domain and return findings",
  "inputSchema": {
    "type": "object",
    "properties": {
      "domain":  {"type": "string"},
      "related": {"type": "array", "items": {"type": "string"}}
    },
    "required": ["domain"]
  }
}
```

Returns the `Scan` record (with Findings) serialised as JSON in the
`content` field of the MCP tool response.

**Synchronous** for the MVP — the tool call blocks until the scan
completes or the client disconnects. Wanderer's default 2-minute
global budget is acceptable for interactive use. Longer scans are a
scheduling concern, not an MCP concern.

## Tool: `assess_scan`

Wired only if `add-assessor` has merged. Compile-time guard via build
tag or separate package import. If not wired, the tool is simply
absent from the MCP `list_tools` response — no stub that returns
"unavailable", because that forces the client to handle a case we do
not need to create.

## Resources

```
wanderer://scans
wanderer://scans/{id}
wanderer://scans/{id}/findings
wanderer://assessments
wanderer://assessments/{id}
```

Resource URIs are read-only. Listing `wanderer://scans` returns a
summary list (ID, domain, status, started_at) — not the full
Findings set, so context windows stay manageable. Full data is
reachable by reading the specific URI.

## External systems

- **stdin / stdout** of the host process. The MCP client writes to
  our stdin; we write to its stdout. stderr is safe for logs and
  is what the host displays in diagnostics.
- **The local SQLite store**, via the same `*store.Store` handle the
  scanner uses.

Failure modes:

- **Client closes stdin.** The server exits cleanly after finishing
  any in-flight tool call.
- **Malformed JSON-RPC.** Respond with the JSON-RPC error envelope
  (`{"jsonrpc":"2.0","error":{"code":-32700,"message":"parse error"}}`)
  and stay alive for the next message.
- **Scan error mid-tool-call.** Return the partial Scan as the tool
  result with the Wanderer-level error embedded — same pattern as
  the HTTP API's `POST /scans` which returns `status: partial`.
- **Concurrent tool calls.** The MCP spec allows pipelining. Wanderer's
  store and scanner are concurrency-safe for reads; two concurrent
  `scan_domain` calls run independent scans.

## Clever valkuil

Tempting: expose every Wanderer internal as an MCP tool — "list
probes", "explain rule", "set probe timeout", "reload config". An
agent could theoretically do much more. But wide tool surfaces are
noise: agents choose poorly when faced with a dozen near-identical
tools. Ship the five tools an analyst actually asks for via natural
language, and stop there.

Also tempting: ship MCP as a Docker-hosted service with a web UI for
listing available MCP clients. That is a separate product. Stdio +
an install snippet for Claude Desktop / Claude Code is enough for the
MVP of this change.

## Install snippet (for `docs/mcp.md`)

```jsonc
// ~/.claude/mcp_servers.json (Claude Desktop) or
// ~/.claude/claude_desktop_config.json (depends on version)
{
  "mcpServers": {
    "wanderer": {
      "command": "/usr/local/bin/wanderer",
      "args": ["mcp", "--db", "/var/lib/wanderer/wanderer.db"]
    }
  }
}
```

## Tests

- Unit tests per tool handler, feeding a fake store.
- Integration test: spawn the MCP binary via `exec.Cmd`, pipe
  JSON-RPC messages, assert responses.
- Contract test: every tool's response round-trips through
  `json.Marshal`/`json.Unmarshal` against the MCP schema fragments.

## Stability + coverage

- `internal/mcp` is not a public API; third-party consumers should
  depend on the MCP protocol, not on our implementation package.
- Target coverage: 75%.
