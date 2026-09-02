#!/usr/bin/env python3
"""Read one RunRequest per stdin line and write one RunResponse per stdout line."""

from __future__ import annotations

import json
import sys

from algorithms import InvalidRequest, run_request


def error(code: str, message: str) -> dict:
    return {"error": {"code": code, "message": message}}


def handle_line(line: str) -> dict:
    try:
        request = json.loads(line)
    except json.JSONDecodeError as exc:
        return error("invalid_json", f"invalid JSON: {exc.msg}")
    try:
        return run_request(request)
    except InvalidRequest as exc:
        return error("invalid_request", str(exc))
    except Exception as exc:  # Keep the long-lived JSONL runner usable after one failed line.
        return error("internal_error", f"runner failed: {type(exc).__name__}")


def main() -> None:
    for raw_line in sys.stdin:
        if not raw_line.strip():
            continue
        response = handle_line(raw_line)
        print(json.dumps(response, separators=(",", ":"), ensure_ascii=False), flush=True)


if __name__ == "__main__":
    main()
