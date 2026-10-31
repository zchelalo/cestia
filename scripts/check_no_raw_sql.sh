#!/bin/sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root/apps/api"
exec go run ./cmd/checkrawsql -root .
