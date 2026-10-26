#!/bin/bash
# Use an existing PostgreSQL instance. Each test creates its own disposable database.
set -euo pipefail
cd "$(dirname "$0")"
export DUNGEOGO_REQUIRE_DB=1
if [ "$#" -eq 0 ]; then
    set -- ./...
fi
exec go test -count=1 -timeout=120s "$@"
