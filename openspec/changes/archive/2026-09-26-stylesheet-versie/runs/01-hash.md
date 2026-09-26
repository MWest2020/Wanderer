# Habitat run 01 — stylesheet met inhoudsversie (tasks 1.1–1.2)

Contract: `openspec/changes/2026-09-26-stylesheet-versie/` (proposal: the
measured Cloudflare HIT with age 4046 s, specs/web-ui/spec.md).

## Where things are
- `internal/ui/ui.go`: `Templates()` builds the FuncMap; `assets` is the
  embedded FS (`fs.Sub(assets, "static")` is served at `/ui/static/`).
- Ten templates in `internal/ui/templates/` carry
  `<link rel="stylesheet" href="/ui/static/main.css">`.

## Decisions already made
1. Hash = first 12 hex characters of sha256 over the embedded `main.css`,
   computed once (package-level, e.g. `sync.OnceValue`), not per request.
2. Template function `stylesheet` (no arguments) returns
   `/ui/static/main.css?v=<hash>`; every template uses
   `href="{{stylesheet}}"`.
3. The file server ignores the query string, so serving does not change.
4. Nothing else: no other assets, no headers, no build step.

## Tests
- Go: render each template that links the stylesheet and assert the link
  equals `/ui/static/main.css?v=` + the hash of the current embedded file.
- Go: scan `internal/ui/templates/*.tmpl` and fail on any literal
  `href="/ui/static/main.css"` — so a new template cannot reintroduce it.
- Each once red with its fix removed; say so. CHANGELOG `### Fixed`.

## Done means
`go build ./...`, `go vet ./...`, `go test ./...` green,
`openspec validate 2026-09-26-stylesheet-versie --strict` green. Budget $3.
