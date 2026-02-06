#!/bin/bash

set -euo pipefail

COMPOSE_FILE="docker-compose.yml"
SERVICE_NAME="postgres"
CONTAINER_NAME="dungeogo-postgres"
DB_HOST="127.0.0.1"
DB_PORT="5432"
MAX_WAIT_SECONDS=30
COMPOSE_CMD=()
RUNTIME_CMD=""

log() {
	echo "[INFO] $1"
}

warn() {
	echo "[WARN] $1"
}

err() {
	echo "[ERROR] $1"
}

detect_runtime() {
	if command -v docker >/dev/null 2>&1; then
		if docker compose version >/dev/null 2>&1; then
			COMPOSE_CMD=(docker compose)
			RUNTIME_CMD="docker"
			return 0
		fi
		if command -v docker-compose >/dev/null 2>&1; then
			COMPOSE_CMD=(docker-compose)
			RUNTIME_CMD="docker"
			return 0
		fi
	fi

	if command -v nerdctl >/dev/null 2>&1 && nerdctl compose version >/dev/null 2>&1; then
		COMPOSE_CMD=(nerdctl compose)
		RUNTIME_CMD="nerdctl"
		return 0
	fi

	return 1
}

is_db_reachable() {
	if command -v pg_isready >/dev/null 2>&1; then
		pg_isready -h "$DB_HOST" -p "$DB_PORT" >/dev/null 2>&1
		return $?
	fi

	# Fallback TCP check.
	bash -c "exec 3<>/dev/tcp/$DB_HOST/$DB_PORT" >/dev/null 2>&1
}

wait_for_db() {
	local waited=0
	log "Waiting for PostgreSQL on ${DB_HOST}:${DB_PORT}..."
	while [ "$waited" -lt "$MAX_WAIT_SECONDS" ]; do
		if is_db_reachable; then
			log "PostgreSQL is reachable."
			return 0
		fi
		sleep 1
		waited=$((waited + 1))
	done
	return 1
}

start_db_container() {
	log "Starting PostgreSQL container via $COMPOSE_FILE..."
	"${COMPOSE_CMD[@]}" -f "$COMPOSE_FILE" up -d "$SERVICE_NAME"
}

main() {
	if is_db_reachable; then
		log "PostgreSQL already running on ${DB_HOST}:${DB_PORT}."
		return 0
	fi

	warn "PostgreSQL is not running on ${DB_HOST}:${DB_PORT}."

	if ! detect_runtime; then
		err "No compatible container runtime found (docker/docker-compose/nerdctl compose)."
		exit 1
	fi

	start_db_container

	# Prefer container-level readiness when possible.
	if [ -n "$RUNTIME_CMD" ]; then
		"$RUNTIME_CMD" exec "$CONTAINER_NAME" pg_isready -U dungeogo_user -d dungeogo >/dev/null 2>&1 || true
	fi

	if ! wait_for_db; then
		err "PostgreSQL did not become ready within ${MAX_WAIT_SECONDS}s."
		exit 1
	fi
}

main "$@"
