# Habitat builder — apply-docs-contract

_run_id builder-apply-docs-contract-20260712-190739 · verdict ok · 2026-07-12T19:14:58Z_

---

I've found a blocking situation and will stop here rather than proceed, per the builder role ("stop and report") and the "don't overwrite work you didn't create" rule.

## Report: change already implemented upstream — stopped before duplicating

**What I found.** The `apply-docs-contract` change is **already done** on the remote. `origin/docs/contract` contains commits:
- `d5d587c docs: apply handbook docs contract`
- `0c1d27e docs: check off apply-docs-contract task 4.1 (PR opened)`

Its `tasks.md` already checks off **4.1 (PR opened)**, so a PR for this change already exists. This branch was not created by my run.

**What I did.** Working from the seed (branch base `bfbdf02`), I independently implemented the full change on a local `docs/contract` branch (commit `744c38d`, unpushed): docs migrated into the contract structure, front matter on all 35 pages (`draft` + `last_reviewed: 2026-07-12`, stub `deprecated`), links rewritten, `index.md` authored, `decisions/README.md` stub, `.mcp.json` placeholder retained. Self-checks pass. My helper script was deleted; the diff is scoped to `docs/` + `tasks.md`.

**What I did NOT do (deliberately):**
- Did **not** force-push over `origin/docs/contract` — it holds work I didn't create, and overwriting it would clobber the existing PR.
- Did **not** open a second PR — the proposal mandates *"één branch, één PR."*

**Only meaningful divergence** between my version and the one already on the remote — the classification of two loose docs the proposal's repo-specific mapping never listed (`operator`, `scheduling`, `releasing`, `tutorial`):

| file | remote/existing PR | my local |
|------|-----|------|
| `operator.md` | `reference/` | `how-to/` |
| `scheduling.md` | `reference/` | `how-to/` |
| `releasing.md`, `tutorial.md` | `how-to/` | `how-to/` (same) |

The existing PR's choice (`reference/` for operator + scheduling) actually matches how the old `docs/README.md` itself grouped them (under a "Reference:" heading), so **I'd defer to the existing PR** — it's a defensible reading and it's already open.

**My recommendation:** review/merge the existing PR on `origin/docs/contract`; my local branch is redundant. Two items the proposal asked to carry into the PR body still need confirming from you regardless of which branch merges:
1. **License** is `NOASSERTION` — you must confirm **EUPL-1.2**; I did not choose one.
2. **Internal hostname** in test fixtures / archived openspec files is out of scope (no code changes) — flagged only.

Want me to delete my local branch, or diff it in detail against the existing PR so you can cherry-pick anything?
