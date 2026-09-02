#!/usr/bin/env bash
set -euo pipefail

BIN=${1:?"usage: verify-redis.sh /path/to/rate-limiter-lab"}
ROOT=$(cd "$(dirname "$0")/.." && pwd)
if ! command -v redis-server >/dev/null 2>&1; then
  echo "redis-server is required for verify-redis" >&2
  exit 1
fi

free_port() {
  python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'
}

REDIS_PORT=$(free_port)
PORT_ONE=$(free_port)
PORT_TWO=$(free_port)
TMP=$(mktemp -d -t rate-limiter-lab-redis.XXXXXX)
REDIS_PID=
PID_ONE=
PID_TWO=

cleanup() {
  for pid in "${PID_ONE}" "${PID_TWO}" "${REDIS_PID}"; do
    if [[ -n "${pid}" ]]; then
      kill "${pid}" 2>/dev/null || true
      wait "${pid}" 2>/dev/null || true
    fi
  done
  rm -rf "${TMP}"
}
trap cleanup EXIT

redis-server --bind 127.0.0.1 --port "${REDIS_PORT}" --save "" --appendonly no --dir "${TMP}" >"${TMP}/redis.log" 2>&1 &
REDIS_PID=$!

LAB_ADDR="127.0.0.1:${PORT_ONE}" LAB_ROOT="${ROOT}" REDIS_ADDR="127.0.0.1:${REDIS_PORT}" "${BIN}" >"${TMP}/server-one.log" 2>&1 &
PID_ONE=$!
LAB_ADDR="127.0.0.1:${PORT_TWO}" LAB_ROOT="${ROOT}" REDIS_ADDR="127.0.0.1:${REDIS_PORT}" "${BIN}" >"${TMP}/server-two.log" 2>&1 &
PID_TWO=$!

REDIS_ADDR="127.0.0.1:${REDIS_PORT}" python3 "${ROOT}/scripts/verify_redis.py" healthy "http://127.0.0.1:${PORT_ONE}" "http://127.0.0.1:${PORT_TWO}"

kill "${REDIS_PID}"
wait "${REDIS_PID}" 2>/dev/null || true
REDIS_PID=
python3 "${ROOT}/scripts/verify_redis.py" outage "http://127.0.0.1:${PORT_ONE}"
