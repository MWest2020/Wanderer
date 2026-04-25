# Architecture Decision Records

This folder captures decisions that shape Wanderer's design. The aim
is a short, append-only trail that a future maintainer can read in
one sitting to understand *why* the project looks the way it does —
not a comprehensive design doc.

## When to write an ADR

Write one when a decision constrains future work and is not obvious
from the code. Examples: choice of a cross-cutting dependency, a new
data-model contract, a passive-vs-active boundary, the stability
class of a public package.

Do not write one for tactical bug fixes, typo corrections, or
refactors that do not change external behaviour.

## Format

Lightweight [MADR](https://adr.github.io/madr/). Copy
[`0000-template.md`](0000-template.md) to `NNNN-short-title.md`,
using the next free number. Keep the status field current: `proposed`,
`accepted`, or `superseded by ADR-XXXX`. Decisions are additive —
do not delete old ADRs; supersede them.

## Index

| #    | Title                                                     | Status   |
|------|-----------------------------------------------------------|----------|
| 0001 | [OpenSpec workflow](0001-openspec-workflow.md)            | accepted |
| 0002 | [API stability classes](0002-api-stability-classes.md)    | accepted |
| 0003 | [Dependency policy](0003-dependency-policy.md)            | accepted |
| 0004 | [Assessor rule engine](0004-assessor-rule-engine.md)      | accepted |
| 0005 | [MCP transport](0005-mcp-transport.md)                    | accepted |
