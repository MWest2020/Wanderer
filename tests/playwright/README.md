# tests/playwright

Browser smoke tests for the Wanderer read-only UI.

## When to add a spec

Every OpenSpec change or ADR that lands UI-touching scenarios
gets a matching spec file in `specs/`. The Go doc-lint test at
`internal/ui/playwright_coverage_test.go` walks
`docs/decisions/` for `## UI surface` headings and confirms a
matching `tests/playwright/specs/<adr-slug>.spec.ts` exists.

## Install (one-shot)

```sh
make playwright-install
```

This runs:

1. `npm ci --ignore-scripts` (per the global supply-chain
   rule — postinstall scripts disabled).
2. `npm audit signatures` to verify the registry-signed
   packages.
3. `npx playwright install chromium` to download the headless
   Chromium binary into `~/.cache/ms-playwright/`. No
   `--with-deps` flag — system libraries are the host's
   responsibility, and Wanderer's dev hosts span RHEL + Debian
   so a one-size-fits-all `apt-get` does not apply.

## Run the specs

```sh
make playwright
```

Builds the `wanderer` binary, seeds a temp SQLite DB with the
demo dataset, boots the server on `127.0.0.1:8281`, runs every
spec under `specs/`, and tears the server down. Screenshots
from failures land in `playwright-report/`.

## Adding screenshots to docs

Scratch screenshots in `playwright-report/` are gitignored.
Curated screenshots for the operator docs live in
`docs/screenshots/` and are committed by hand — only the ones
the operator manual references.
