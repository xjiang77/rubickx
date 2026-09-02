#!/usr/bin/env python3
"""Black-box Redis checks shared by the healthy and outage phases."""

from __future__ import annotations

from concurrent.futures import ThreadPoolExecutor
import json
import os
import socket
import sys
import time
import urllib.error
import urllib.parse
import urllib.request


MODE = sys.argv[1]
BASES = [value.rstrip("/") for value in sys.argv[2:]]


def exchange(base: str, path: str, key: str):
    request = urllib.request.Request(base + path, headers={"X-RateLimit-Key": key})
    try:
        response = urllib.request.urlopen(request, timeout=5)
    except urllib.error.HTTPError as error:
        response = error
    raw = response.read()
    return response.status, response.headers, json.loads(raw)


def wait_until_ready(base: str) -> None:
    deadline = time.monotonic() + 15
    while time.monotonic() < deadline:
        try:
            if urllib.request.urlopen(base + "/api/health", timeout=1).status == 200:
                return
        except Exception:
            time.sleep(0.05)
    raise AssertionError(f"server did not become ready: {base}")


def wait_for_redis() -> None:
    host, raw_port = os.environ["REDIS_ADDR"].rsplit(":", 1)
    deadline = time.monotonic() + 10
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        try:
            with socket.create_connection((host, int(raw_port)), timeout=0.25) as connection:
                connection.sendall(b"*1\r\n$4\r\nPING\r\n")
                if connection.recv(64) == b"+PONG\r\n":
                    return
        except Exception as error:  # pragma: no cover - startup timing only
            last_error = error
        time.sleep(0.025)
    raise AssertionError(f"Redis did not become ready: {last_error}")


def wait_for_window_budget(window_ms: int, minimum_remaining_ms: int) -> None:
    remaining = window_ms - int(time.time() * 1000) % window_ms
    if remaining < minimum_remaining_ms:
        time.sleep((remaining + 50) / 1000)


for base in BASES:
    wait_until_ready(base)

if MODE == "healthy":
    assert len(BASES) == 2
    wait_for_redis()
    wait_for_window_budget(5000, 3000)

    local_key = f"local-{time.time_ns()}"
    local_path = "/demo/local-vs-shared?store=memory&replica=a&limit=1&window_ms=5000&failure=fail-closed"
    local_one = exchange(BASES[0], local_path, local_key)
    local_two = exchange(BASES[1], local_path, local_key)
    assert local_one[0] == 200 and local_two[0] == 200
    assert local_one[2]["scope"] == "replica-local" and local_two[2]["scope"] == "replica-local"

    shared_key = f"shared-{time.time_ns()}"
    shared_a = "/demo/local-vs-shared?store=redis&replica=a&limit=1&window_ms=5000&failure=fail-closed"
    shared_b = "/demo/local-vs-shared?store=redis&replica=b&limit=1&window_ms=5000&failure=fail-closed"
    shared_one = exchange(BASES[0], shared_a, shared_key)
    shared_two = exchange(BASES[1], shared_b, shared_key)
    assert shared_one[0] == 200 and shared_two[0] == 429
    assert shared_one[2]["scope"] == "shared" and shared_two[2]["scope"] == "shared"

    key = f"atomic-{time.time_ns()}"
    path = "/demo/shared?store=redis&limit=5&window_ms=5000&failure=fail-closed"
    with ThreadPoolExecutor(max_workers=20) as pool:
        futures = [pool.submit(exchange, BASES[index % 2], path, key) for index in range(20)]
    exchanges = [future.result() for future in futures]
    statuses = [item[0] for item in exchanges]
    assert statuses.count(200) == 5 and statuses.count(429) == 15, statuses
    assert all(item[1]["RateLimit-Remaining"] == "0" for item in exchanges if item[0] == 429)
    assert all(item[1]["Retry-After"] for item in exchanges if item[0] == 429)

    ttl_window_ms = 600
    wait_for_window_budget(ttl_window_ms, 450)
    ttl_key = f"ttl-{time.time_ns()}"
    ttl_path = f"/demo/ttl?store=redis&limit=1&window_ms={ttl_window_ms}&failure=fail-closed"
    assert exchange(BASES[0], ttl_path, ttl_key)[0] == 200
    assert exchange(BASES[1], ttl_path, ttl_key)[0] == 429
    remaining = ttl_window_ms - int(time.time() * 1000) % ttl_window_ms
    time.sleep((remaining + 100) / 1000)
    assert exchange(BASES[0], ttl_path, ttl_key)[0] == 200
    print("Redis healthy verification passed: local replica split, shared quota across two servers, atomic limit, and TTL reset")
elif MODE == "outage":
    assert len(BASES) == 1
    key = f"outage-{time.time_ns()}"
    open_status, open_headers, open_body = exchange(
        BASES[0], "/demo/outage?store=redis&limit=2&window_ms=1000&failure=fail-open", key
    )
    assert open_status == 200
    assert open_headers["X-RateLimit-Degraded"] == "true"
    assert open_body["decision"]["reason"] == "storage_unavailable_fail_open"

    closed_status, closed_headers, closed_body = exchange(
        BASES[0], "/demo/outage?store=redis&limit=2&window_ms=1000&failure=fail-closed", key
    )
    assert closed_status == 503
    assert closed_headers["X-RateLimit-Degraded"] == "true"
    assert closed_body["error"]["code"] == "rate_limit_store_unavailable"
    print("Redis outage verification passed: fail-open=200 degraded, fail-closed=503 degraded")
else:
    raise SystemExit(f"unknown mode: {MODE}")
