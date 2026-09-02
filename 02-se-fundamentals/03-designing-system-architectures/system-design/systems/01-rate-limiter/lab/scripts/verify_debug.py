#!/usr/bin/env python3
"""Exercise the public debug API against a real Delve DAP session."""

from __future__ import annotations

import json
import sys
import time
import urllib.error
import urllib.request


BASE = sys.argv[1].rstrip("/")


def exchange(path: str, method: str = "GET", payload: object | None = None):
    body = None if payload is None else json.dumps(payload).encode()
    request = urllib.request.Request(
        BASE + path,
        method=method,
        data=body,
        headers={"Content-Type": "application/json"} if body is not None else {},
    )
    try:
        response = urllib.request.urlopen(request, timeout=25)
    except urllib.error.HTTPError as error:
        response = error
    raw = response.read()
    return response.status, json.loads(raw) if raw else None


deadline = time.monotonic() + 15
while time.monotonic() < deadline:
    try:
        status, health = exchange("/api/health")
        if status == 200:
            assert health["debugAvailable"] is True
            break
    except Exception:
        time.sleep(0.05)
else:
    raise AssertionError("lab server did not become ready")

status, snapshot = exchange(
    "/api/debug/sessions",
    method="POST",
    payload={
        "algorithm": "token-bucket",
        "config": {"capacity": 2, "ratePerSecond": 1},
        "requestTimeline": [
            {"atMs": 0, "cost": 1, "key": "alice"},
            {"atMs": 0, "cost": 1, "key": "alice"},
            {"atMs": 1000, "cost": 1, "key": "alice"},
        ],
        "breakpointStepId": "token.refill",
    },
)
assert status == 201, snapshot
session_id = snapshot["sessionId"]
assert snapshot["status"] == "stopped" and snapshot["line"] > 0
assert snapshot["source"]["path"] == "server/algorithms.go"
assert "@step:token.refill" in snapshot["source"]["content"]
assert snapshot["stackFrames"] and snapshot["locals"]
assert {item["name"] for item in snapshot["locals"]} & {"point", "state", "t"}

for command in ("next", "continue", "restart"):
    status, next_snapshot = exchange(
        f"/api/debug/sessions/{session_id}/commands", method="POST", payload={"command": command}
    )
    assert status == 200, next_snapshot
    assert next_snapshot["status"] == "stopped" and next_snapshot["line"] > 0
    assert next_snapshot["stackFrames"] and next_snapshot["locals"]

status, _ = exchange(f"/api/debug/sessions/{session_id}", method="DELETE")
assert status == 204
status, body = exchange(
    f"/api/debug/sessions/{session_id}/commands", method="POST", payload={"command": "next"}
)
assert status == 400 and body["error"]["code"] == "debug_command_failed"
print("Debug verification passed: selected token-bucket in real algorithms.go, source/line, stack/locals, next, continue, restart, cleanup")
