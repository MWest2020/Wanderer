# Proposal: MCP Server — Wanderer as a Claude-Native Tool

## Intent

Expose Wanderer via the Model Context Protocol so agents (Claude Code,
Claude Desktop, other MCP clients) can drive scans and read findings
as first-class context. This is how Wanderer stops being "a CLI you
remember to run" and becomes "a thing your assistant reaches for when
you ask it a sovereignty question".

Concretely: an analyst in a Claude Code session can say *"scan
gemeente-X.nl and summarise the jurisdictional posture"* and the model
calls `scan_domain` + `get_assessment` MCP tools against the local
Wanderer server, without the operator leaving the conversation.

## Scope

**In scope:**

- `wanderer mcp` subcommand that starts an MCP server over stdio
  (the canonical transport for local MCP hosts).
- **Tools:**
  - `scan_domain(domain, related?)` — kicks off a scan, returns the
    Scan record with all Findings inline.
  - `get_scan(id)` — returns a stored Scan + Findings.
  - `list_scans(limit?, since?)` — enumerate recent scans.
  - `assess_scan(scan_id, framework?)` — only wired when assessor is
    present; returns an Assessment.
  - `get_assessment(id)` — returns a stored Assessment.
- **Resources:**
  - `wanderer://scans` — directory of scan IDs with summaries.
  - `wanderer://scans/{id}` — full Scan record.
  - `wanderer://assessments/{id}` — full Assessment (if present).
- A short install snippet for Claude Desktop / Claude Code MCP config.

**Out of scope:**

- HTTP/SSE MCP transport — stdio is what local hosts want; network
  MCP requires authentication and is a separate change.
- Prompts served by the MCP server (templates). Plausible future, not
  needed for the MVP of this change.
- A GUI for configuring the MCP connection.

## DICTU dimensions informed

Indirect. MCP is a transport; it does not change what Wanderer
observes. It changes how humans consume the observations.

## Passive/active boundary

Same as the tool it wraps: scans invoked via MCP go through the same
scanner + probe pipeline as CLI scans. No new probes, no new network
activity patterns.

## Parallel-safe

Touches `cmd/wanderer/mcp.go` (new), `internal/mcp/` (new), and one
line in `cmd/wanderer/main.go` dispatch. No changes to scanner, store,
or probes.
