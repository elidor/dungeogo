#!/bin/bash
set -euo pipefail

# Runs the complete local DungeoGo stack with Apple's `container` CLI.
# Requires macOS 26+ on Apple silicon and container services started once with:
#   container system start

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONTAINER_CLI="${CONTAINER_CLI:-container}"
IMAGE_NAME="${DUNGEOGO_IMAGE:-dungeogo:local}"
NETWORK_NAME="${DUNGEOGO_NETWORK:-dungeogo}"
POSTGRES_CONTAINER="${DUNGEOGO_POSTGRES_CONTAINER:-dungeogo-postgres}"
SERVER_CONTAINER="${DUNGEOGO_SERVER_CONTAINER:-dungeogo-server}"
POSTGRES_VOLUME="${DUNGEOGO_POSTGRES_VOLUME:-dungeogo-postgres-data}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
SERVER_PORT="${PORT:-8080}"
MAX_WAIT_SECONDS=30

log() {
    echo "[INFO] $*"
}

fail() {
    echo "[ERROR] $*" >&2
    exit 1
}

require_container() {
    command -v "$CONTAINER_CLI" >/dev/null 2>&1 || fail "Apple Container CLI not found. Install it, then run 'container system start'."
    if ! "$CONTAINER_CLI" system status >/dev/null 2>&1; then
        log "Starting Apple Container services..."
        "$CONTAINER_CLI" system start
    fi
}

container_exists() {
    "$CONTAINER_CLI" list --all --quiet | grep -Fxq "$1"
}

remove_container_if_present() {
    if container_exists "$1"; then
        "$CONTAINER_CLI" delete --force "$1"
    fi
}

ensure_network() {
    if ! "$CONTAINER_CLI" network list --quiet | grep -Fxq "$NETWORK_NAME"; then
        log "Creating network $NETWORK_NAME..."
        "$CONTAINER_CLI" network create "$NETWORK_NAME"
    fi
}

ensure_volume() {
    if ! "$CONTAINER_CLI" volume list --quiet | grep -Fxq "$POSTGRES_VOLUME"; then
        log "Creating PostgreSQL volume $POSTGRES_VOLUME..."
        "$CONTAINER_CLI" volume create "$POSTGRES_VOLUME" >/dev/null
    fi
}

database_address() {
    # Apple Container's inspect output is JSON. plutil is available on macOS and
    # avoids making jq a project prerequisite.
    "$CONTAINER_CLI" inspect "$POSTGRES_CONTAINER" | /usr/bin/plutil -extract '0.status.networks.0.ipv4Address' raw - | sed 's|/.*||'
}

wait_for_postgres() {
    local waited=0
    log "Waiting for PostgreSQL..."
    while [ "$waited" -lt "$MAX_WAIT_SECONDS" ]; do
        if "$CONTAINER_CLI" exec "$POSTGRES_CONTAINER" pg_isready -U dungeogo_user -d dungeogo >/dev/null 2>&1; then
            return 0
        fi
        sleep 1
        waited=$((waited + 1))
    done
    "$CONTAINER_CLI" logs "$POSTGRES_CONTAINER" || true
    fail "PostgreSQL did not become ready within ${MAX_WAIT_SECONDS}s."
}

build() {
    require_container
    log "Building $IMAGE_NAME..."
    log "Compiling the Linux ARM64 server binary..."
    (
        cd "$SCRIPT_DIR"
        CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o bin/dungeogo ./cmd/server
    )

    # Package the binary built above rather than relying on Apple Container's
    # stale source-context cache.
    "$CONTAINER_CLI" build --no-cache --file "$SCRIPT_DIR/Containerfile.apple" --tag "$IMAGE_NAME" "$SCRIPT_DIR"
}

up() {
    require_container
    ensure_network
    ensure_volume
    remove_container_if_present "$POSTGRES_CONTAINER"
    remove_container_if_present "$SERVER_CONTAINER"

    log "Starting PostgreSQL on 127.0.0.1:${POSTGRES_PORT}..."
    "$CONTAINER_CLI" run --detach --name "$POSTGRES_CONTAINER" \
        --network "$NETWORK_NAME" \
        --publish "127.0.0.1:${POSTGRES_PORT}:5432" \
        --volume "${POSTGRES_VOLUME}:/var/lib/postgresql/data" \
        --volume "${SCRIPT_DIR}/migrations:/docker-entrypoint-initdb.d:ro" \
        --env POSTGRES_DB=dungeogo \
        --env POSTGRES_USER=dungeogo_user \
        --env POSTGRES_PASSWORD=dungeogo_password \
        --env PGDATA=/var/lib/postgresql/data/pgdata \
        postgres:15
    wait_for_postgres

    build
    local db_ip
    db_ip="$(database_address)"
    [ -n "$db_ip" ] || fail "Could not determine PostgreSQL container address."

    log "Starting DungeoGo on 127.0.0.1:${SERVER_PORT}..."
    "$CONTAINER_CLI" run --detach --name "$SERVER_CONTAINER" \
        --network "$NETWORK_NAME" \
        --publish "127.0.0.1:${SERVER_PORT}:8080" \
        --mount "type=bind,source=${SCRIPT_DIR}/bin,target=/app,readonly" \
        --entrypoint /app/dungeogo \
        --env BIND_ADDRESS=0.0.0.0 \
        --env PORT=8080 \
        --env "DATABASE_URL=postgres://dungeogo_user:dungeogo_password@${db_ip}:5432/dungeogo?sslmode=disable" \
        "$IMAGE_NAME"

    log "DungeoGo is running. Connect with: nc 127.0.0.1 ${SERVER_PORT}"
}

down() {
    require_container
    remove_container_if_present "$SERVER_CONTAINER"
    remove_container_if_present "$POSTGRES_CONTAINER"
    if "$CONTAINER_CLI" network list --quiet | grep -Fxq "$NETWORK_NAME"; then
        "$CONTAINER_CLI" network delete "$NETWORK_NAME"
    fi
    log "Stack stopped. PostgreSQL data remains in volume $POSTGRES_VOLUME."
}

logs() {
    require_container
    "$CONTAINER_CLI" logs "$SERVER_CONTAINER"
}

status() {
    require_container
    "$CONTAINER_CLI" list --all
}

usage() {
    cat <<'EOF'
Usage: ./apple-container.sh <command>

Commands:
  build    Build the DungeoGo OCI image.
  up       Start PostgreSQL and DungeoGo.
  down     Stop the stack; preserves the PostgreSQL volume.
  logs     Show DungeoGo server logs.
  status   List Apple Container resources.

Environment overrides: DUNGEOGO_IMAGE, DUNGEOGO_NETWORK,
DUNGEOGO_POSTGRES_CONTAINER, DUNGEOGO_SERVER_CONTAINER,
DUNGEOGO_POSTGRES_VOLUME, POSTGRES_PORT, and PORT.
EOF
}

case "${1:-}" in
    build|up|down|logs|status)
        "$1"
        ;;
    -h|--help|help)
        usage
        ;;
    *)
        usage >&2
        exit 1
        ;;
esac
