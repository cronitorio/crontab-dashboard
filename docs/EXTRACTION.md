# Extraction and release ordering

Dashboard source comes from Cronitor CLI commit
`1c4eb731c492fa7e1a487e38077af184020e35ce`. Git history remains in the original
repository; this initial import preserves the license and records its origin.
Shared Go library code and the companion executable initially pin released CLI
33.8. Production code imports its `lib` package; only the integration-test harness
imports `cmd` to route the real CLI's API traffic to local test servers.

## Rollout

1. Review this import and validate the repository's Linux/macOS and container CI.
2. Publish dashboard `v0.1.0`, verify all five bundles/checksums and the installer.
3. Merge the CLI extraction PR only once that installation path works. Until
   then it stays draft; CLI 33.8 users keep their existing dashboard.
4. Update the public crontab.guru dashboard installation documentation to point
   to this repository. Existing services can switch to `crontab-dashboard` while
   keeping configuration, cron entries, and runtime credentials.
5. Dispatch the image workflow against the published tag when the dashboard's
   final repository identity is confirmed. Verify package visibility and an
   anonymous pull before advertising Compose or closing CLI issue #40.

Moving the CLI to `monitor-io/cli` does not block this extraction. Keep the
released dependency pin until the move, then update `CLI_REPOSITORY`, the Go
module/imports, and checksums together in a tested dependency update. Keep the
native executable named `cronitor` so existing cron entries continue working.
Moving this dashboard repository itself affects the installer, updater, docs,
and GHCR package identity; settle that before publishing its official image.

## Verification of the initial import

Local verification includes dashboard Go race tests, CLI Go tests, frontend
build/dependency regression checks, real pinned-CLI event/log-upload integration
against local mock services, installer failure/upgrade fixtures, and container
supervisor fixtures. The native installer was exercised with a real macOS ARM
bundle and the CLI compatibility launcher. All five release targets compile;
Windows CLI compilation also passes.

The container smoke test checks authentication, API mutations, interactive job
output, an actual scheduled cron invocation, graceful shutdown, and persistence
of configuration/crontabs across container recreation. Workflows pass actionlint.

The inherited frontend uses Create React App. Its lockfile is preserved, including
its existing npm audit findings (102 during the initial local install). The focused
dependency checks cover earlier targeted fixes; they are not a clean dependency
audit. Modernizing that build toolchain is separate follow-up work.

These checks do not publish a release or image. Live monitoring/account integration
and a first release's anonymous download/pull still need release-time verification.
