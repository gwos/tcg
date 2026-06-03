#!/usr/bin/env bash

_log() { printf '[%s] DOCKER_CMD %b\n' "$(date '+%Y-%m-%d %H:%M:%S%z')" "$*" >&2; }

# Set DOCKER_CMD_SET_E to any non-empty value to opt into strict fail-fast mode
[ -n "${DOCKER_CMD_SET_E:-}" ] && set -e && _log "DOCKER_CMD_SET_E : strict fail-fast mode"

. "${DOCKER_CMD_D:-/docker_cmd.d}/_inc"

_log "EXEC $@"
exec "$@"
