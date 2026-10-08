#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
target=${1:?Usage: package-release.sh platform_arch dashboard_binary output_dir}
binary=${2:?Usage: package-release.sh platform_arch dashboard_binary output_dir}
output=${3:?Usage: package-release.sh platform_arch dashboard_binary output_dir}
mkdir -p "$output"
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT HUP INT TERM
cp "$binary" "$staging/crontab-dashboard"
"$ROOT/scripts/fetch-cli.sh" "$target" "$staging/cronitor"
cp "$ROOT/LICENSE" "$ROOT/CLI_VERSION" "$ROOT/CLI_REPOSITORY" "$staging/"
chmod 755 "$staging/crontab-dashboard" "$staging/cronitor"
archive="$output/$target.tar.gz"
tar -czf "$archive" -C "$staging" crontab-dashboard cronitor LICENSE CLI_VERSION CLI_REPOSITORY
if command -v sha256sum >/dev/null 2>&1; then sha256sum "$archive" | cut -d ' ' -f 1 > "$archive.sha256"; else shasum -a 256 "$archive" | cut -d ' ' -f 1 > "$archive.sha256"; fi
