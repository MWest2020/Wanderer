#!/usr/bin/env bash
set -euo pipefail
readme="$(dirname "${BASH_SOURCE[0]}")/README.md"
extract() {
  local marker="$1"
  awk -v marker="$marker" '
    $0 == marker { found=1; next }
    found && /^```/ { if (infence) { exit } infence=1; next }
    found && infence { print; exit }
  ' "$readme"
}
echo "CURL: [$(extract '<!-- quickstart:curl -->')]"
echo "DOCKER: [$(extract '<!-- quickstart:docker -->')]"
