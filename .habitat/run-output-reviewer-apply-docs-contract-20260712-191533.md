# Habitat reviewer — apply-docs-contract

_run_id reviewer-apply-docs-contract-20260712-191533 · verdict ok · 2026-07-12T19:17:39Z_

---

Review complete. All evidence gathered; issuing verdict as reviewer (read-only, no fixes).

---

# Review: `apply-docs-contract` — **PASS** ✅

Judged the builder's diff (`main...HEAD`, 41 files) against `.claude/agents/reviewer.md` and the change under `openspec/changes/apply-docs-contract/`. No root `CLAUDE.md` present, so that leg is moot.

## 1. Scope — PASS
- Docs restructure matches the repo-specific mapping in `proposal.md`: 17 ADRs + template + README → `docs/explanation/adr/` (numbering preserved); `architecture/agent/maintainability` → `explanation/`; `mcp/egress/exporters/findings/drift/observability/assessor` → `reference/`; how-to pages under `how-to/`.
- `.mcp.json` placed at root with the template and the `TODO-change-3` placeholder intact (task 2.4). ✓
- `tasks.md` edit is checkbox-only; 4.1 (open PR) correctly left unchecked — that's Mark's step. ✓
- **Note (non-blocking, for Mark):** the diff also carries habitat harness artifacts — `.habitat/audit.jsonl`, `.habitat/run-output-*.md`, `.habitat/run-report-*.html`, and root `run-report.json`. These are the runner's own audit trail, not builder content, and touch no cage/CI/secret surface. They arguably don't belong in a `docs:` PR — worth gitignoring or dropping before merge, but not a builder scope violation.

## 2. Contract — PASS
- **Allowed dirs only:** the sole markdown outside `index.md`/`how-to/`/`reference/`/`explanation/` is `docs/decisions/README.md` — a `status: deprecated` redirect stub that the proposal *explicitly mandates* ("stub in `docs/decisions/README.md` achterlaten"). Legitimate.
- **Front matter:** all 35 pages carry `status` + `last_reviewed`. Values are 34× `draft` + 1× `deprecated` (stub) — all in the allowed `current|draft|deprecated` set. Migrated-without-review pages correctly sit at `draft` / `last_reviewed: 2026-07-12`.
- **No `owner`:** confirmed — the only "owner" token is a GitHub URL template `{owner}/{repo}` in `how-to/releasing.md`.
- **One language:** English throughout. (The `"is dit goed of niet"` hit in `architecture.md` is a quoted example phrase in otherwise-English prose, pre-existing content.)
- `index.md` gives a one-paragraph description, links to `../README.md`, and links every section — all targets resolve.

## 3. Cage intact — PASS
No `CLAUDE.md`, `.claude/agents/`, or CI/workflow files in the diff. (The grep matches were false positives on substring — `de**ci**sions`, `open-spec-**workflow**` — both plain docs files.)

## 4. No secrets — PASS
No credentials, keys, tokens, or secret-bearing URLs. The two "secret" grep hits are documentation prose about the HMAC shared secret and egress redaction — not real secrets. `.mcp.json` URL is the placeholder.

---

### Deferred to PR body (per proposal, reviewer confirms unaddressed-by-design, not defects)
- License still `NOASSERTION` — Mark must confirm EUPL-1.2; builder correctly did **not** pick one.
- Internal hostname in test fixtures / archived openspec files — explicitly out of scope; no code changed. ✓

**Verdict: PASS.** The change is ready for Mark to open the PR (task 4.1) and merge; I'd only ask him to decide whether the habitat/run-report artifacts should ship in that PR.
