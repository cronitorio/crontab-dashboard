# Crontab Guru Dashboard

A self-hosted web dashboard for managing local cron jobs: edit schedules, suspend
jobs, inspect running processes, run commands interactively, and connect a local
MCP client. Cronitor monitoring is optional.

This application was extracted from [Cronitor CLI](https://github.com/cronitorio/cronitor-cli).
Its frontend, server, authentication, MCP integration, and releases live here.

## Install

Download `install.sh` from a [published release](https://github.com/cronitorio/crontab-dashboard/releases),
inspect it, then run it as the intended installation owner:

```sh
DASHBOARD_VERSION=v0.1.0 sh install.sh
crontab-dashboard
```

For installation without root, set `INSTALL_DIR="$HOME/.local/bin"` and include
that directory in PATH, then start with
`CRONITOR_CONFIG="$HOME/.config/crontab-dashboard/cronitor.json" crontab-dashboard`.
First launch prompts for dashboard credentials. For
services, supply `CRONITOR_DASH_USER` and `CRONITOR_DASH_PASS` in the environment.

Every release bundle contains **both** `crontab-dashboard` and a pinned `cronitor`.
The installer puts them together in a private versioned directory and links the
dashboard into the install directory. It exposes `cronitor` on PATH only when no
CLI already exists there, and updates its own CLI link on subsequent installs.
An existing independently installed CLI is preserved; the dashboard
uses its own companion. No separate Go, Node, or CLI install is required.

You can also verify a release's `.sha256` checksum and extract its `.tar.gz`.
Keep both executables in the same directory. The dashboard prefers its sibling
`cronitor`, then PATH; `CRONTAB_DASHBOARD_CRONITOR` selects a specific executable
for development or custom installations. `CLI_VERSION` pins the bundled version.

## Existing installations

`/etc/cronitor/cronitor.json`, custom `--config` / `CRONITOR_CONFIG`, existing
`CRONITOR_*` settings, and scheduled `cronitor exec` commands remain compatible.
The CLI extraction branch keeps `cronitor dash` as a launcher for this application;
that behavior becomes available in the next CLI release.
Old CLIs retain their embedded dashboard; run `crontab-dashboard` directly to
select this application. Existing MCP clients can use:

```json
{"command":"crontab-dashboard","args":["--mcp-instance","default"]}
```

The standalone dashboard continues to accept `dash` as a command alias.
Update systemd's ExecStart to `/usr/local/bin/crontab-dashboard` and use
`Restart=always`. Existing configuration files are preserved.

## Monitoring and dependencies

The dashboard has its own executable and release cycle. It never imports the
CLI's `cmd` package. The cron parser, monitor/API types, and API client are reused
from a pinned CLI Go module; those are compile-time dependencies, not an extra
installation step. Dashboard-specific MCP and process-management code live here.

Scheduled monitored jobs keep their `cronitor exec` wrappers. Monitored “run now”
requests delegate to the installed CLI, which sends lifecycle events and logs.
Unmonitored local commands run directly and require no CLI or Cronitor account.

Dashboard updates verify and install both binaries together. Container updates
use image replacement instead of the application's binary updater. The bundled
CLI cannot be updated independently by the dashboard UI; `cronitor update` on
another CLI installation remains independent.

## Docker

The official image is intended to be published after the repository's final
organization/name is settled. `compose.yml` uses the proposed current image path:

```sh
export CRONITOR_DASH_USER=admin
read -r -s CRONITOR_DASH_PASS
export CRONITOR_DASH_PASS
docker compose -f compose.yml up -d
```

Or build locally with `docker build -t crontab-dashboard .` and set that image
in Compose. This image runs **cron and the dashboard**, and includes the pinned
CLI. It runs jobs inside the container. Install task executables/dependencies in
your own image derived from it, or mount the scripts and resources they need.
It does not automatically manage the host's cron daemon or host processes.

Configuration and user crontabs persist in named volumes. The port binds to
localhost; use an SSH tunnel or your existing access proxy for remote access.
Supply credentials at runtime. Both processes shut down together, and the
container exits if either cron or the dashboard fails. Version tags are preferred
over `latest` for installations that need predictable updates.

## Development

Go version is specified in `go.mod`; frontend builds use Node 22 or newer.

```sh
cd web
npm ci
npm run test:dependencies
npm run build
cd ..
go test ./...
go build -o dist/crontab-dashboard .
./scripts/fetch-cli.sh "$(go env GOOS)_$(go env GOARCH)" dist/cronitor
./dist/crontab-dashboard
```

To update the CLI dependency, change `CLI_VERSION` and the pinned CLI module
revision together, then verify the integration tests and all release targets.
If the CLI repository moves, update `CLI_REPOSITORY` and the Go import/module
pin in one change. The installed executable remains named `cronitor`.

Release tags use `vMAJOR.MINOR.PATCH`. Add the matching file under `docs/releases/`.
The release workflow builds frontend assets once, produces all five platform
bundles, verifies checksums, and publishes a complete release. The image workflow
is manually dispatched against an existing published tag. Set its GHCR package
visibility to public and verify an anonymous pull before advertising the image.

Source provenance: extracted from `cronitorio/cronitor-cli` at commit
`1c4eb731c492fa7e1a487e38077af184020e35ce`; shared library/companion pin initially
matches released CLI 33.8 (`71105ec2957e3f481db8aa5d354753834b9579fb`). The original
Functional Source License, Version 1.1, MIT Future License is preserved.
