#!/usr/bin/env bash

set -euo pipefail

ROOT="web"
PORT="${PAGES_CHECK_PORT:-8123}"

required_files=(
  "$ROOT/index.html"
  "$ROOT/styles.css"
  "$ROOT/favicon.svg"
)

for file in "${required_files[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Missing required file: $file" >&2
    exit 1
  fi
done

while IFS= read -r asset; do
  [[ -z "$asset" ]] && continue
  if [[ ! -f "$ROOT/$asset" ]]; then
    echo "Missing referenced local asset: $ROOT/$asset" >&2
    exit 1
  fi
done < <(
  grep -Eo '(href|src)="\./[^"#?]+(\?[^"#]*)?"' "$ROOT/index.html" \
    | sed -E 's/^(href|src)="\.\/([^"?#]+).*/\2/' \
    | sort -u || true
)

python3 -m http.server "$PORT" -d "$ROOT" >/tmp/rubickx-pages-check.log 2>&1 &
server_pid=$!
trap 'kill "$server_pid" >/dev/null 2>&1 || true' EXIT

sleep 1
curl -fsSI "http://127.0.0.1:${PORT}/index.html" >/dev/null

echo "Pages basic gate passed."
