---
status: draft
last_reviewed: 2026-07-12
---

# 0007. Agent → core trust model: HMAC over TLS

- Status: accepted
- Date: 2026-04-26

## Context

The inventory probe introduces `wanderer agent`, a host-side mode
that runs on each observed host and ships `Finding` records back to
a central `wanderer serve` instance. That introduces a new trust
question: how does the core know which agent is allowed to write
which findings?

Three options were on the table:

1. **Bearer tokens.** A token per agent, sent in the `Authorization`
   header. Easy to deploy; trivially leaked or replayed by anyone who
   captures one.
2. **Mutual TLS.** The agent presents a client certificate from a
   private PKI; the core verifies it against the CA. Strong, but
   carries certificate rotation, CRL, and OCSP machinery as ongoing
   operational cost.
3. **HMAC-over-TLS.** Each agent has a per-host shared secret; every
   POST is signed with HMAC-SHA256 over `<timestamp>\n<body>` and
   includes a timestamp header. The core verifies the signature and
   rejects requests with timestamps outside ±5 minutes.

## Decision

The MVP ships option 3: HMAC-SHA256 over HTTPS, with a ±5-minute
timestamp window for replay protection.

The wire format:

```
POST /scans/{id}/findings
X-Wanderer-Agent: <hostname>
X-Wanderer-Timestamp: <RFC 3339>
X-Wanderer-Signature: base64(HMAC-SHA256(secret, timestamp + "\n" + body))
```

A constant-time HMAC compare (`crypto/hmac.Equal`) is used. Failures
return a single 401 status without leaking which check failed; that
prevents an attacker from mapping out registered hostnames.

Per-host secrets live behind an `AgentSecrets` interface; the MVP
ships a `StaticAgentSecrets` map suitable for ≤10 hosts. A
file-watching or vault-backed implementation can be added later
without touching the verifier.

## Consequences

- **Operational simplicity for small fleets.** A new host needs one
  secret in two places (`/etc/wanderer/agent.hmac` on the host,
  `AgentSecrets` on the core). No PKI, no rotation tooling, no CRL.
- **Replay protection is timestamp-based.** A request outside the
  ±5-minute window is rejected. We do not maintain a per-nonce
  cache; the timestamp window is the dedup mechanism.
- **TLS is still mandatory.** HMAC authenticates the request; it
  does not protect confidentiality. Operators MUST terminate TLS
  on the core (or at a reverse proxy in front of it). Unencrypted
  HTTP transport is explicitly out of scope.
- **Secret rotation requires a coordinated change.** Rotate by
  generating a new secret, updating the agent, then updating the
  core within the timestamp window. Bearer-token-style overlap
  windows are not supported in the MVP — when a fleet outgrows that
  ergonomic, revisit.
- **Information-leak discipline.** The 401 response is identical
  for unknown hostname, bad signature, and timestamp skew. The
  underlying error is logged on the core but not surfaced.

When to revisit: when the fleet exceeds ~10 hosts, when secret
rotation friction becomes noticeable, or when a regulator demands
mTLS specifically. At that point a follow-up change introduces
either workload identity (SPIFFE-style) or a full PKI.
