#!/usr/bin/env bash
# Downloads a Wanderer release archive, verifies it against the
# release's checksums.txt, and extracts the binary. Used as the first
# step of action.yml. Runs entirely against the target release
# (github.com/<repo>/releases) — it talks to nowhere else.
#
# WANDERER_RELEASE_BASE_URL is an internal override for testing this
# script standalone against a local fixture server instead of a real
# GitHub release; it is not a documented action input.
set -euo pipefail

repo="${WANDERER_REPO:-MWest2020/wanderer}"
version="${WANDERER_VERSION:-}"
base_url="${WANDERER_RELEASE_BASE_URL:-https://github.com/${repo}/releases}"
dest_dir="${WANDERER_INSTALL_DIR:-$(mktemp -d)}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m | sed -e 's/x86_64/amd64/' -e 's/aarch64/arm64/')"

if [[ "$os" != "linux" && "$os" != "darwin" ]]; then
	echo "wanderer-action: unsupported runner OS '$os' — this action installs the linux/darwin tar.gz release archives only" >&2
	exit 1
fi

if [[ -z "$version" || "$version" == "latest" ]]; then
	download_root="$base_url/latest/download"
else
	download_root="$base_url/download/$version"
fi

archive="wanderer_${os}_${arch}.tar.gz"

echo "wanderer-action: downloading $archive from $download_root"
curl -sSL --fail -o "$dest_dir/$archive" "$download_root/$archive"
curl -sSL --fail -o "$dest_dir/checksums.txt" "$download_root/checksums.txt"

expected="$(awk -v f="$archive" '$2 == f { print $1 }' "$dest_dir/checksums.txt")"
if [[ -z "$expected" ]]; then
	echo "wanderer-action: $archive is not listed in checksums.txt — refusing to trust an unverifiable download" >&2
	exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
	actual="$(sha256sum "$dest_dir/$archive" | awk '{print $1}')"
else
	actual="$(shasum -a 256 "$dest_dir/$archive" | awk '{print $1}')"
fi

if [[ "$expected" != "$actual" ]]; then
	echo "wanderer-action: checksum mismatch for $archive (expected $expected, got $actual)" >&2
	exit 1
fi

tar -xzf "$dest_dir/$archive" -C "$dest_dir" wanderer
chmod +x "$dest_dir/wanderer"

echo "wanderer-action: checksum verified, binary at $dest_dir/wanderer"
if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
	echo "wanderer-bin=$dest_dir/wanderer" >>"$GITHUB_OUTPUT"
fi
