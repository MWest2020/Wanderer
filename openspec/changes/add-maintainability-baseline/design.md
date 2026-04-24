# Design: Maintainability Baseline

## What lands, concretely

```
CHANGELOG.md                          # new
CODEOWNERS                            # new
docs/decisions/0000-template.md       # new
docs/decisions/0001-openspec-workflow.md       # new
docs/decisions/0002-api-stability-classes.md   # new
docs/decisions/0003-dependency-policy.md       # new
docs/decisions/README.md              # new, index
docs/maintainability.md               # new, single contributor entry point
```

No code changes, no test changes, no dependency changes.

## CHANGELOG format

Keep-a-Changelog (<https://keepachangelog.com>). Sections:
`Added`, `Changed`, `Deprecated`, `Removed`, `Fixed`, `Security`. One
`## [Unreleased]` heading, with concrete entries rolled under it
continuously. Entries are written in the imperative past tense,
reference the OpenSpec change that introduced them, and point at the
notable files.

Example entry:

```markdown
## [Unreleased]

### Added
- Maintainability baseline: CHANGELOG, ADRs, dependency/API stability
  policy. (`openspec/changes/add-maintainability-baseline`)
```

## ADR format

Lightweight MADR-style (<https://adr.github.io>). One file per decision
in `docs/decisions/NNNN-short-title.md` with front matter:

```markdown
# NNNN. Short title

- Status: proposed | accepted | superseded by ADR-XXXX
- Date: 2026-MM-DD

## Context
## Decision
## Consequences
```

Three seed ADRs ship with this change:

1. **0001-openspec-workflow**: all non-trivial changes go through
   `openspec/changes/<name>/` before landing. Points at the `/opsx:*`
   skill family.
2. **0002-api-stability-classes**: `pkg/models` is the stable public
   surface; `internal/*` is a private implementation detail and may
   change without notice. Third-party consumers who import `internal/*`
   do so at their own risk.
3. **0003-dependency-policy**: prefer the standard library; prefer
   pure-Go deps (no CGo) so cross-compilation stays trivial; pin
   versions in `go.mod`; each new dep gets a one-line rationale in
   the ADR or the proposal that introduced it.

## External systems

None — this change is repo-local only.

## Failure modes

- **Contributors do not update the CHANGELOG.** Mitigation: add a brief
  note to `docs/maintainability.md` and `CODEOWNERS`. A CI hook checking
  for CHANGELOG edits is explicitly **not** in scope — too ceremonial
  for a small project, trivial to add later if drift becomes visible.
- **ADRs rot.** Mitigation: status field on every ADR; superseded
  decisions point forward. Decisions should be additive; do not delete
  old ADRs.

## Clever valkuil

Tempting: ship a Husky-style commit-msg hook, GitHub PR template,
Dependabot config, labeller bot, required-status-checks matrix,
auto-changelog generation from conventional commits. All of that is
good at scale; on a two-person project at MVP stage it is pure
ceremony that slows every commit without materially improving quality.

The maintainability baseline is **documentation + one index file** on
purpose. Tools come later when the pain becomes real, not preemptively.

## How the six feature changes reference this

Each of the six parallel changes MUST, in its own design.md:

- Add its entry to `CHANGELOG.md` under `[Unreleased]`.
- Create an ADR if it introduces a non-obvious architectural choice
  (new transport protocol, new data type, new trust boundary).
- State its API stability class for any new exported type
  (public via `pkg/` or private via `internal/`).
- State its minimum test coverage target and the rationale if lower
  than the baseline (70% internal, 90% models).
