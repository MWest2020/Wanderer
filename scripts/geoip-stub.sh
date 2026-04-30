#!/usr/bin/env bash
# Build a deterministic, empty-but-valid GeoLite2-shaped mmdb so the
# test suite can exercise the populated-but-empty branch of the IP
# probe without a real MaxMind license.
#
# The output mmdb has the same schema (`GeoLite2-ASN`) the IP probe
# expects but contains zero networks, so every lookup returns
# "not found" cleanly.
#
# Usage:
#   ./scripts/geoip-stub.sh /tmp/geolite2-stub.mmdb
#
# Requires: go (any 1.20+). Does NOT require clang, llvm, or
# anything else from the eBPF builder image.

set -euo pipefail

OUT="${1:-}"
if [[ -z "$OUT" ]]; then
    echo "usage: $0 <output-path>" >&2
    exit 2
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# scripts/geoip-stub/main.go uses //go:build ignore so it stays out
# of `go build ./...` but `go run` invokes it directly. The
# mmdbwriter dep is fetched into the module cache on first run; it
# does not become a production dependency because no non-ignored
# package imports it.
exec go run ./scripts/geoip-stub/main.go "$OUT"
