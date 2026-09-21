#!/usr/bin/env bash
# Runs the exact commands published in the README's quickstart section
# (the curl one-liner and the docker one-liner), so a quickstart that
# has drifted from reality fails the build instead of failing the
# first person who tries it. Run standalone: bash scripts/check-quickstart.sh
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readme="$repo_root/README.md"

if [[ ! -f "$readme" ]]; then
	echo "check-quickstart: README.md not found at $readme" >&2
	exit 1
fi

# extract_command <marker> prints the single line inside the fenced
# code block that immediately follows the given HTML-comment marker,
# e.g. "<!-- quickstart:curl -->" followed by a ```sh fence on the
# next line and the command on the line after that.
extract_command() {
	local marker="$1" marker_line offset
	marker_line="$(grep -n -F -- "$marker" "$readme" | cut -d: -f1 | head -n1)"
	if [[ -z "$marker_line" ]]; then
		echo "check-quickstart: marker '$marker' not found in README.md" >&2
		return 1
	fi
	offset=$((marker_line + 2))
	sed -n "${offset}p" "$readme"
}

curl_cmd="$(extract_command '<!-- quickstart:curl -->')"
docker_cmd="$(extract_command '<!-- quickstart:docker -->')"

if [[ -z "$curl_cmd" ]]; then
	echo "check-quickstart: empty curl command extracted from README.md" >&2
	exit 1
fi
if [[ -z "$docker_cmd" ]]; then
	echo "check-quickstart: empty docker command extracted from README.md" >&2
	exit 1
fi

work_dir="$(mktemp -d)"
trap 'rm -rf "$work_dir"' EXIT

echo "== quickstart: curl =="
echo "+ $curl_cmd"
(cd "$work_dir" && bash -c "$curl_cmd")

echo "== quickstart: docker =="
echo "+ $docker_cmd"
bash -c "$docker_cmd"

echo "check-quickstart: OK"
