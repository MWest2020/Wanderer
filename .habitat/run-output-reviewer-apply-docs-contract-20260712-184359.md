# Habitat reviewer — apply-docs-contract

_run_id reviewer-apply-docs-contract-20260712-184359 · verdict ok · 2026-07-12T18:45:02Z_

---

I've completed the review pass. Here's my verdict.

## Reviewer verdict: **NO DIFF TO REVIEW** (cannot PASS/FAIL)

**What I checked**
- Role: `.claude/agents/reviewer.md` — read-only, judge builder's diff against CLAUDE.md + the change `openspec/changes/apply-docs-contract/`.
- Project `CLAUDE.md`: **not present** in the repo (only `.claude/agents/*` role files exist).
- Change scope: read `proposal.md` + `tasks.md`.

**Finding**
There is no builder diff to review. The builder branch is empty relative to main:

- `git log main..habitat/builder/apply-docs-contract` → no commits
- `git diff main habitat/builder/apply-docs-contract` → **tree identical to main**
- My own branch (`habitat/reviewer/apply-docs-contract`) is also at `bfbdf02`, working tree clean, no stashes.

The `apply-docs-contract` change has **not been implemented yet**. The docs still show the pre-contract layout that the change is meant to migrate:
- `docs/` is flat (`docs/mcp.md`, `docs/architecture.md`, …) with no `how-to/` / `reference/` / `explanation/` split — contract §"Docs-contract" unmet.
- `docs/decisions/` (17 ADRs) not moved to `docs/explanation/adr/` — repo-specific task unmet.
- No front matter (`status` / `last_reviewed`) on any page — Check 2 unmet.
- `.mcp.json` already exists at root with the `TODO-change-3` placeholder (matches task 2.4), but this predates the change and isn't part of a reviewable diff.

**Why I'm not issuing PASS or FAIL**
A PASS/FAIL is a judgment on a diff. With zero changes, there's nothing to judge — and issuing FAIL would wrongly imply the builder produced defective work. My role is strictly read-only review; implementing the migration is the **builder's** job and is out of scope for me (touching it would also risk the reviewer overstepping the cage).

**What needs to happen next**
The builder agent (`habitat/builder/apply-docs-contract`) needs to run the change first. Once the builder pushes a diff, re-invoke me and I'll review it against:
1. **Scope** — implements the change's tasks, nothing more.
2. **Contract** — only `index.md` + `how-to/` + `reference/` + `explanation/` carry markdown; every page has `status` + `last_reviewed`; no `owner`; English only.
3. **Cage** — HARD FAIL if the diff touches `CLAUDE.md`, `.claude/agents/`, or CI config.
4. **No secrets.**

I've made no changes to the repository, consistent with the read-only reviewer role.
