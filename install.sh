#!/bin/sh
# Installs a versioned dashboard + private CLI bundle. Run as the intended owner.
set -eu
repository=${DASHBOARD_REPOSITORY:-cronitorio/crontab-dashboard}
version=${DASHBOARD_VERSION:-}
prefix=${INSTALL_DIR:-/usr/local/bin}
case "$(uname -s)" in Linux) platform=linux ;; Darwin) platform=darwin ;; *) echo 'Supported systems: Linux and macOS' >&2; exit 1 ;; esac
case "$(uname -m)" in x86_64|amd64) arch=amd64 ;; arm64|aarch64) arch=arm64 ;; armv7l|armv6l) arch=arm ;; *) echo 'Unsupported architecture' >&2; exit 1 ;; esac
if [ -z "$version" ]; then
 # Resolve the redirect without executing or parsing remote shell/JSON.
 url=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repository/releases/latest")
 version=${url##*/}
fi
case "$version" in ''|*[!a-zA-Z0-9._-]*) echo 'Invalid release version' >&2; exit 1 ;; esac
base=${DASHBOARD_RELEASE_BASE_URL:-https://github.com/$repository/releases/download/$version}
mkdir -p "$prefix/.crontab-dashboard"
prefix=$(CDPATH= cd -- "$prefix" && pwd)
staging=$(mktemp -d "$prefix/.crontab-dashboard/.install-XXXXXX")
trap 'rm -rf "$staging"' EXIT HUP INT TERM
archive="${platform}_${arch}.tar.gz"
curl -fsSL --retry 3 "$base/$archive" -o "$staging/bundle.tar.gz"
curl -fsSL --retry 3 "$base/$archive.sha256" -o "$staging/expected"
expected=$(tr -d '[:space:]' < "$staging/expected")
case "$expected" in ''|*[!a-fA-F0-9]*) echo 'Invalid dashboard checksum' >&2; exit 1 ;; esac
[ "${#expected}" -eq 64 ] || { echo 'Invalid dashboard checksum length' >&2; exit 1; }
if command -v sha256sum >/dev/null 2>&1; then actual=$(sha256sum "$staging/bundle.tar.gz" | cut -d ' ' -f 1); else actual=$(shasum -a 256 "$staging/bundle.tar.gz" | cut -d ' ' -f 1); fi
[ "$expected" = "$actual" ] || { echo 'Dashboard checksum mismatch' >&2; exit 1; }
for binary in crontab-dashboard cronitor; do
 tar -xzOf "$staging/bundle.tar.gz" "$binary" > "$staging/$binary"
 [ -s "$staging/$binary" ] || { echo "Missing $binary in dashboard bundle" >&2; exit 1; }
 chmod 755 "$staging/$binary"
 "$staging/$binary" --help >/dev/null
 done
# Unique bundle directories preserve the prior installation on failures, and
# avoid altering binaries used by an already-running dashboard.
bundle="$prefix/.crontab-dashboard/$version-$(basename "$staging")"
mv "$staging" "$bundle"
staging=$(mktemp -d "$prefix/.crontab-dashboard/.links-XXXXXX")
ln -s "$bundle/crontab-dashboard" "$staging/crontab-dashboard"
# An existing standalone CLI is preserved. Dashboard runs its private sibling.
managed_cli=false
if [ -L "$prefix/cronitor" ]; then
 case "$(readlink "$prefix/cronitor")" in "$prefix"/.crontab-dashboard/*/cronitor) managed_cli=true ;; esac
fi
if { [ ! -e "$prefix/cronitor" ] && [ ! -L "$prefix/cronitor" ]; } || [ "$managed_cli" = true ]; then
 ln -s "$bundle/cronitor" "$staging/cronitor"
 mv -f "$staging/cronitor" "$prefix/cronitor"
fi
mv -f "$staging/crontab-dashboard" "$prefix/crontab-dashboard"
printf 'Installed dashboard %s and its companion CLI in %s\n' "$version" "$bundle"
printf 'Start with: %s/crontab-dashboard\n' "$prefix"
