# 0005. MCP transport: hand-rolled stdio JSON-RPC

- Status: accepted
- Date: 2026-04-25

## Context

The `add-mcp-server` change exposes Wanderer over the Model Context
Protocol so Claude Code, Claude Desktop, and other MCP hosts can
drive scans and read findings as first-class context.

The protocol surface we actually need for the MVP of this capability
is small: stdio transport with line-delimited JSON-RPC 2.0,
`initialize`, `tools/list`, `tools/call`, `resources/list`,
`resources/read`, plus standard JSON-RPC error envelopes.

Two implementation paths were on the table:

1. Pull in an official or community Go SDK for MCP (e.g.
   `modelcontextprotocol/go-sdk`).
2. Hand-roll a stdio JSON-RPC loop in `internal/mcp/`.

## Decision

We hand-roll a minimal stdio JSON-RPC server in `internal/mcp/`. It
is ~300 lines of Go, uses only `encoding/json` and `bufio` from the
standard library, and lives entirely in `internal/` so it is not
part of Wanderer's public Go API.

This decision is reversible: tool and resource handlers sit behind
the small `Tool` and `Resource` interfaces defined in
`internal/mcp/server.go`. Swapping the dispatcher out for an SDK
later is a localised change.

## Consequences

- **Zero new top-level dependencies.** The dependency policy
  ([ADR-0003](0003-dependency-policy.md)) calls for stdlib first; this
  fits. `go build ./...` keeps working without internet access to a
  module cache populated only with the SDK.
- **Protocol surface limited to what we need.** We do not implement
  prompts, notifications, or sampling. If a future change needs them,
  either extend `internal/mcp` or migrate to an SDK at that point.
- **MCP version drift is on us.** When the protocol revises, we must
  update the dispatcher manually. For MVP purposes this is cheap —
  the surface area we use is the stable subset that has been in MCP
  since its first public spec.
- **No spec compliance tests against an upstream conformance suite.**
  We rely on the integration test (spawn `wanderer mcp`, exchange
  JSON-RPC, assert responses) and on Claude Desktop / Claude Code
  acting as the real-world conformance check.

When to revisit: if a credible Go SDK reaches v1 and our dispatcher
grows past ~600 lines, the trade-off flips and we should adopt the
SDK in a follow-up change.
