# Habitat run 01 — verdict in mensentaal (tasks 1.1–1.3)

Contract: `openspec/changes/2026-09-20-answer-first-ui/` (proposal,
specs/web-ui/spec.md requirement "The answer is one sentence with its
reason").

## Scope — ONLY these tasks
- [ ] 1.1 One function in `internal/ui` that turns an Assessment into
  the headline verdict: `soeverein`/`voldoende` → ja, `afhankelijk` →
  nee, `onbekend` → onbekend. It also returns (a) how many questions
  could not be answered and (b) the single observation that decided a
  "nee" — the worst-scoring flow with its verdict text, so the headline
  can name it. An unknown SHALL NEVER become a ja; a mix of ja and
  onbekend is not a plain ja.
  Reuse `SovereigntyFlows` (`internal/ui/flows.go`) — it already groups
  rationales into hosting/mail/DNS/transit/CDN/third-parties and the
  certificate; do not build a second grouping.
- [ ] 1.2 The Dutch headline copy in ONE string table, in the pattern of
  `internal/assessor/wand/accountability_nl.yaml` (embedded, loaded,
  tested). Templates and Go code contain no answer copy.
- [ ] 1.3 Table-driven tests: all soeverein → ja; one afhankelijk →
  nee naming that flow; hosting soeverein + DNS onbekend → not a plain
  ja and the wording says one question is unanswered; everything
  onbekend → onbekend; an assessment predating a dimension does not
  count as nee.

No template or route changes in this run — that is run 02 and 03. This
run delivers the function, the copy and the tests.

## Over bouwen en testen in de kooi — lees dit eerst
De eerste `go build ./...` in de kooi duurt minuten: `modernc.org/sqlite`
is getranspileerde C en staat gevendord in de repo. Draai daarom tijdens
het werk alleen je eigen pakket:

    go test ./internal/ui/...

en bewaar één volledige `go build ./...` + `go vet ./...` + `go test ./...`
voor het eind. Ga NIET zoeken naar een snellere manier, en zet GOFLAGS of
GOPROXY niet om — die staan goed (offline uit vendor/).

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green (offline from
vendor/); `openspec validate 2026-09-20-answer-first-ui --strict` green.
Budget is $4.
