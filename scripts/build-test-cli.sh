#!/bin/sh
set -eu
ROOT=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$ROOT"
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT HUP INT TERM
cp go.mod "$staging/test.mod"
cp go.sum "$staging/test.sum"
go build -mod=mod -modfile "$staging/test.mod" -o dist/cronitor-test scripts/cli-test-main.go
