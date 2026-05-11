.PHONY: build test lint run clean playwright-install playwright playwright-fixture

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"

# Playwright fixture directory. `make playwright-fixture` writes one
# DB per scenario under this directory; `make playwright` boots
# `wanderer serve` against each in turn via the three-project
# playwright.config.ts. Files are .gitignored.
PLAYWRIGHT_FIXTURE_DIR ?= tests/playwright/fixtures

build:
	go build $(LDFLAGS) -o bin/wanderer ./cmd/wanderer

test:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

run: build
	./bin/wanderer $(ARGS)

clean:
	rm -rf bin/ dist/ coverage.* tests/playwright/fixtures/*.db

# playwright-install installs the Node deps with lifecycle scripts
# disabled (per the global supply-chain rule), verifies signatures,
# then downloads the Chromium binary. Run once on a fresh checkout
# and after every package-lock.json change.
playwright-install:
	cd tests/playwright && npm ci --ignore-scripts
	cd tests/playwright && npm audit signatures
	cd tests/playwright && npx playwright install chromium

# playwright-fixture writes one SQLite per scenario via the
# internal/fixtures seeder. Re-runs are idempotent: each scenario
# removes the existing file and rebuilds from scratch so schema
# migrations run on every invocation.
playwright-fixture:
	@mkdir -p $(PLAYWRIGHT_FIXTURE_DIR)
	go run ./internal/fixtures/main --scenario baseline   --out $(PLAYWRIGHT_FIXTURE_DIR)/baseline.db
	go run ./internal/fixtures/main --scenario agent-host --out $(PLAYWRIGHT_FIXTURE_DIR)/agent-host.db
	go run ./internal/fixtures/main --scenario empty-org  --out $(PLAYWRIGHT_FIXTURE_DIR)/empty-org.db

# playwright runs the spec set against the three hermetic
# fixture DBs. The build + fixture targets are prerequisites so a
# fresh checkout runs end-to-end with one command. Output lands in
# tests/playwright/playwright-report/.
playwright: build playwright-fixture
	cd tests/playwright && npx playwright test
