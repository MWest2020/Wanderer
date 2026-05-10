.PHONY: build test lint run clean playwright-install playwright

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"

# Playwright defaults. The spec set runs against an existing demo
# DB — typically the one a developer built up by running a few
# `wanderer scan` calls (e.g. /tmp/wanderer-demo.db). A hermetic
# fixture loader is a planned follow-up; for now an operator
# runs `wanderer scan example.nl` once before `make playwright`.
PLAYWRIGHT_PORT ?= 8281
PLAYWRIGHT_DB   ?= /tmp/wanderer-demo.db

build:
	go build $(LDFLAGS) -o bin/wanderer ./cmd/wanderer

test:
	go test -race -cover ./...

lint:
	golangci-lint run ./...

run: build
	./bin/wanderer $(ARGS)

clean:
	rm -rf bin/ dist/ coverage.*

# playwright-install installs the Node deps with lifecycle scripts
# disabled (per the global supply-chain rule), verifies signatures,
# then downloads the Chromium binary. Run once on a fresh checkout
# and after every package-lock.json change.
playwright-install:
	cd tests/playwright && npm ci --ignore-scripts
	cd tests/playwright && npm audit signatures
	cd tests/playwright && npx playwright install chromium

# playwright runs the spec set against the configured DB.
# playwright.config.ts's webServer block boots `./bin/wanderer
# serve` on PLAYWRIGHT_PORT; the spec output lands in
# tests/playwright/playwright-report/.
playwright: build
	@test -e $(PLAYWRIGHT_DB) || (echo "ERROR: $(PLAYWRIGHT_DB) does not exist." && \
		echo "       Run \`./bin/wanderer scan <domain>\` first to populate it," && \
		echo "       or set PLAYWRIGHT_DB=<path> to point at a different DB." && exit 1)
	cd tests/playwright && \
		WANDERER_PLAYWRIGHT_PORT=$(PLAYWRIGHT_PORT) \
		WANDERER_PLAYWRIGHT_DB=$(PLAYWRIGHT_DB) \
		npx playwright test
