---
status: draft
last_reviewed: 2026-07-12
---

# MCP server

Wanderer can speak the [Model Context Protocol][mcp] over stdio so
agents (Claude Code, Claude Desktop, other MCP clients) drive scans
and read findings as first-class context.

[mcp]: https://modelcontextprotocol.io

## Why

The wider the gap between "I want to ask about a domain's posture"
and "I run `wanderer scan` and read the output", the less Wanderer
gets used. MCP closes that gap: an analyst in a Claude Code session
can say *"scan example.nl and tell me about the jurisdictional
posture"* and the model calls `scan_domain` + `assess_scan` against
a local Wanderer process, all without leaving the conversation.

## Install

The MCP server runs as `wanderer mcp`. Configure your MCP host by
pointing at the Wanderer binary.

### Claude Desktop / Claude Code

Add to `~/.claude/mcp_servers.json` (or your host's equivalent):

```jsonc
{
  "mcpServers": {
    "wanderer": {
      "command": "/usr/local/bin/wanderer",
      "args": ["mcp", "--db", "/var/lib/wanderer/wanderer.db"]
    }
  }
}
```

The server reads JSON-RPC 2.0 messages on stdin, writes responses on
stdout, and emits structured logs on stderr (your host displays these
in diagnostics output).

### Flags

`wanderer mcp` accepts the same scanner flags as `wanderer serve`:

```
--db <path>              SQLite database path
--geoip <path>           GeoLite2-ASN mmdb (for IP probe)
--geoip-country <path>   Optional country mmdb
--per-probe-timeout 30s  Per-probe timeout
--budget 2m              Global scan timeout budget
--user-agent "Wanderer/0.x"
```

## Tools

The MVP exposes five tools. Wanderer deliberately does not expose
configuration, reload, or reset surfaces — those are operational
concerns, not analyst concerns.

| Tool             | Purpose                                                      |
| ---------------- | ------------------------------------------------------------ |
| `scan_domain`    | Run a fresh scan against a domain; returns the Scan + Findings. |
| `get_scan`       | Look up a stored Scan by ID.                                 |
| `list_scans`     | Enumerate recent scans (id, domain, status, started_at, finding_count). |
| `assess_scan`    | Score a stored scan against the DICTU rule set; returns and persists an Assessment. |
| `get_assessment` | Look up a stored Assessment by ID.                           |

## Resources

| URI                                  | Body                                  |
| ------------------------------------ | ------------------------------------- |
| `wanderer://scans`                   | Summary list of every stored scan.    |
| `wanderer://scans/{id}`              | Full Scan record with findings.       |
| `wanderer://scans/{id}/findings`     | Findings only (no scan envelope).     |
| `wanderer://assessments`             | List of every persisted Assessment.   |
| `wanderer://assessments/{id}`        | Full Assessment record.               |

Resources are read-only. Writes happen only through `scan_domain` and
`assess_scan` — the same code paths the CLI uses.

## Example conversation

User: *"Can you scan ribbel.nl and summarise the jurisdictional
posture?"*

Claude: *(invokes `scan_domain` with `{"domain": "ribbel.nl"}`,
receives the Scan record, then invokes `assess_scan` with the new
scan ID, reads the resulting Assessment, summarises in plain
language with citations to specific findings.)*

## Protocol surface

Wanderer implements a minimal subset of MCP:

- `initialize`, `ping`
- `tools/list`, `tools/call`
- `resources/list`, `resources/read`

JSON-RPC 2.0 framing. Line-delimited; one message per line. ADR-0005
records why this is hand-rolled rather than built on a third-party
SDK.

## Failure modes

- **Malformed JSON line.** The server replies with the JSON-RPC
  parse-error envelope (code `-32700`) and stays alive.
- **Unknown tool or method.** Replies with method-not-found (`-32601`).
- **Scan or store error during a tool call.** Reply is a successful
  JSON-RPC response with `result.isError = true` and a text content
  block describing the failure — the agent can read and explain it.
- **stdin closed.** The server exits cleanly within seconds.
