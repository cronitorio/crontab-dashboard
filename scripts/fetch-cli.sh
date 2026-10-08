#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
target=${1:?Usage: fetch-cli.sh platform_arch destination}
destination=${2:?Usage: fetch-cli.sh platform_arch destination}
case "$target" in linux_amd64|linux_arm64|linux_arm|darwin_amd64|darwin_arm64) ;; *) echo "Unsupported target: $target" >&2; exit 1 ;; esac
version=$(cat "$ROOT/CLI_VERSION")
repository=$(cat "$ROOT/CLI_REPOSITORY")
base=${CLI_RELEASE_BASE_URL:-https://github.com/$repository/releases/download/$version}
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT HUP INT TERM
curl -fsSL --retry 3 "$base/$target.tar.gz" -o "$staging/cli.tar.gz"
curl -fsSL --retry 3 "$base/$target.tar.gz.sha256" -o "$staging/expected"
expected=$(tr -d '[:space:]' < "$staging/expected")
case "$expected" in ''|*[!a-fA-F0-9]*) echo 'Invalid CLI checksum' >&2; exit 1 ;; esac
[ "${#expected}" -eq 64 ] || { echo 'Invalid CLI checksum length' >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$staging/cli.tar.gz" | cut -d ' ' -f 1); else actual=$(shasum -a 256 "$staging/cli.tar.gz" | cut -d ' ' -f 1); fi
[ "$expected" = "$actual" ] || { echo 'CLI checksum mismatch' >&2; exit 1; }
tar -xzOf "$staging/cli.tar.gz" cronitor > "$staging/cronitor"
[ -s "$staging/cronitor" ] || { echo 'CLI archive contains no binary' >&2; exit 1; }
chmod 755 "$staging/cronitor"
mkdir -p "$(dirname -- "$destination")"
mv "$staging/cronitor" "$destination"
