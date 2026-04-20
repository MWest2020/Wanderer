.PHONY: build test lint run clean

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags="-s -w -X main.Version=$(VERSION)"

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
