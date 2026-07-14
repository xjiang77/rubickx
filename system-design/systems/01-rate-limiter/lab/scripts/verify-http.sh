#!/usr/bin/env bash
set -euo pipefail

BIN=${1:?"usage: verify-http.sh /path/to/rate-limiter-lab"}
ROOT=$(cd "$(dirname "$0")/.." && pwd)
PORT=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()')
LOG=$(mktemp -t rate-limiter-lab-http.XXXXXX)
PID=

cleanup() {
  if [[ -n "${PID}" ]]; then
    kill "${PID}" 2>/dev/null || true
    wait "${PID}" 2>/dev/null || true
  fi
  rm -f "${LOG}"
}
trap cleanup EXIT

LAB_ADDR="127.0.0.1:${PORT}" LAB_ROOT="${ROOT}" REDIS_ADDR="" "${BIN}" >"${LOG}" 2>&1 &
PID=$!

if ! python3 "${ROOT}/scripts/verify_http.py" "http://127.0.0.1:${PORT}"; then
  echo "--- server log ---" >&2
  tail -100 "${LOG}" >&2
  exit 1
fi
