#!/usr/bin/env python3
"""Black-box verification for the production Rate Limiter Lab server."""

from __future__ import annotations

import json
import sys
import time
import urllib.error
import urllib.request


BASE = sys.argv[1].rstrip("/")


def exchange(path: str, payload: object | None = None, headers: dict[str, str] | None = None):
    body = None if payload is None else json.dumps(payload).encode()
    request_headers = dict(headers or {})
    if body is not None:
        request_headers["Content-Type"] = "application/json"
    request = urllib.request.Request(BASE + path, data=body, headers=request_headers)
    try:
        response = urllib.request.urlopen(request, timeout=20)
    except urllib.error.HTTPError as error:
        response = error
    raw = response.read()
    parsed = json.loads(raw) if raw else None
    return response.status, response.headers, parsed


def wait_until_ready() -> None:
    deadline = time.monotonic() + 15
    last_error: Exception | None = None
    while time.monotonic() < deadline:
        try:
            status, _, body = exchange("/api/health")
            if status == 200 and body["status"] == "ok":
                return
        except Exception as error:  # pragma: no cover - only used during process startup
            last_error = error
        time.sleep(0.05)
    raise AssertionError(f"lab server did not become ready: {last_error}")


def point(at_ms: int, cost: float = 1, key: str = "alice") -> dict[str, object]:
    return {"atMs": at_ms, "cost": cost, "key": key}


FIXTURES = {
    "fixed-window": (
        {"limit": 2, "windowMs": 1000},
        [point(0), point(100), point(200), point(1000)],
        2,
    ),
    "sliding-window-log": (
        {"limit": 2, "windowMs": 1000},
        [point(0), point(100), point(200), point(1100)],
        2,
    ),
    "sliding-window-counter": (
        {"limit": 2, "windowMs": 1000},
        [point(0, 2), point(1000, 0.5), point(1250, 0.5)],
        3,
    ),
    "token-bucket": (
        {"capacity": 1, "ratePerSecond": 2},
        [point(0, 0.9999985), point(0, 0.0000015), point(0, 0.1)],
        2,
    ),
    "leaky-bucket": (
        {"capacity": 2, "ratePerSecond": 1},
        [point(0), point(0), point(0), point(500), point(1000)],
        2,
    ),
}


SYSTEM_SCENARIO_EXPECTED_CASES = {
    "policy-composition": [
        {"when": "Same client · requests #1–3", "result": "200 · all policies allowed", "kind": "allow"},
        {"when": "Same client · request #4", "result": "429 · rejected by per-client", "kind": "deny"},
        {"when": "Endpoint-wide request #7", "result": "429 · rejected by endpoint-wide", "kind": "deny"},
    ],
    "http-contract": [
        {"when": "Requests #1–3", "result": "200 · ALLOW", "kind": "allow"},
        {"when": "Request #4", "result": "429 · RateLimit-* + Retry-After", "kind": "deny"},
    ],
    "local-vs-shared": [
        {"when": "Memory · Replica A and B", "result": "Each replica has its own limit 3", "kind": "observe"},
        {
            "when": "Redis healthy · Replica A and B",
            "result": "Both replicas share one limit 3",
            "kind": "observe",
        },
    ],
    "redis-atomicity": [
        {
            "when": "Redis healthy",
            "result": "3×200 + 7×429 · atomic increment + TTL",
            "kind": "observe",
        },
        {"when": "Redis unavailable · fail-open", "result": "200 + degraded + bypass", "kind": "allow"},
        {"when": "Redis unavailable · fail-closed", "result": "503 + degraded + enforced", "kind": "deny"},
    ],
    "redis-outage": [
        {"when": "Redis healthy", "result": "Quota is enforced normally", "kind": "observe"},
        {"when": "Redis unavailable · fail-open", "result": "200 + degraded + bypass", "kind": "allow"},
        {"when": "Redis unavailable · fail-closed", "result": "503 + degraded + enforced", "kind": "deny"},
    ],
    "hot-key-sharding": [
        {
            "when": "Redis healthy · same key",
            "result": "Stable shard · first 3 requests return 200, then 429",
            "kind": "observe",
        },
        {
            "when": "Redis healthy · distinct keys",
            "result": "Keys can spread across four shards",
            "kind": "observe",
        },
        {"when": "Redis unavailable · fail-open", "result": "200 + degraded + bypass", "kind": "allow"},
        {"when": "Redis unavailable · fail-closed", "result": "503 + degraded + enforced", "kind": "deny"},
    ],
    "multi-region-quota": [
        {"when": "Region-local quota", "result": "Low latency, weaker global accuracy", "kind": "observe"},
        {
            "when": "Globally coordinated quota",
            "result": "Higher accuracy, added latency and dependency",
            "kind": "observe",
        },
        {
            "when": "Regional allocation",
            "result": "Bounded overshoot with explicit rebalancing",
            "kind": "observe",
        },
    ],
}


def assert_trace_continuity(events: list[dict[str, object]], steps_per_request: int) -> None:
    assert len(events) % steps_per_request == 0, events
    for start in range(0, len(events), steps_per_request):
        request_events = events[start : start + steps_per_request]
        for previous, current in zip(request_events, request_events[1:]):
            assert current["before"] == previous["after"], (previous, current)


def verify_catalog_and_ui() -> None:
    status, _, catalog = exchange("/api/catalog")
    assert status == 200
    assert [item["id"] for item in catalog["languages"]] == ["python", "go", "java", "javascript"]
    assert len(catalog["algorithms"]) == 5
    assert len(catalog["scenarios"]) == 14
    assert {item["tier"] for item in catalog["scenarios"]} == {"core", "system"}
    system_scenario_ids = set()
    for scenario in catalog["scenarios"]:
        brief = scenario["brief"]
        assert brief["traffic"].strip()
        assert brief["expected"]["summary"].strip()
        assert bool(brief.get("conceptual")) == (scenario["id"] == "multi-region-quota")
        assert bool(brief.get("replicaScoped")) == (scenario["id"] == "local-vs-shared")
        if scenario["tier"] == "core":
            assert len(brief["expected"]["admissions"]) == len(scenario["requestTimeline"])
        else:
            actual_cases = brief["expected"]["cases"]
            assert actual_cases
            system_scenario_ids.add(scenario["id"])
            expected_cases = SYSTEM_SCENARIO_EXPECTED_CASES[scenario["id"]]
            assert len(actual_cases) == len(expected_cases), (scenario["id"], actual_cases, expected_cases)
            for index, (actual_case, expected_case) in enumerate(zip(actual_cases, expected_cases)):
                for field in ("when", "result", "kind"):
                    assert actual_case[field] == expected_case[field], (
                        scenario["id"],
                        index,
                        field,
                        actual_case[field],
                        expected_case[field],
                    )
    assert system_scenario_ids == set(SYSTEM_SCENARIO_EXPECTED_CASES), system_scenario_ids

    response = urllib.request.urlopen(BASE + "/", timeout=5)
    html = response.read().decode()
    assert response.status == 200 and "Rate Limiter Lab" in html


def verify_four_language_parity() -> None:
    for algorithm, (config, timeline, steps_per_request) in FIXTURES.items():
        canonical = None
        for language in ("go", "python", "java", "javascript"):
            status, _, run = exchange(
                "/api/runs",
                {
                    "scenarioId": "",
                    "algorithm": algorithm,
                    "language": language,
                    "config": config,
                    "requestTimeline": timeline,
                    "storeMode": "memory",
                },
            )
            assert status == 200, (language, algorithm, run)
            assert run["runId"] and run["language"] == language and run["algorithm"] == algorithm
            assert len(run["decisions"]) == len(timeline)
            assert run["source"]["content"] and run["source"]["path"]
            assert all(event["source"]["line"] > 0 for event in run["events"])
            assert_trace_continuity(run["events"], steps_per_request)
            if canonical is None:
                canonical = run["decisions"]
            else:
                assert run["decisions"] == canonical, (algorithm, language, canonical, run["decisions"])

        if algorithm == "sliding-window-counter":
            assert canonical[1]["allowed"] is False
            assert canonical[1]["retryAfterMs"] == 250
            assert canonical[1]["resetAtMs"] == 1250


def verify_input_boundaries() -> None:
    base = {
        "scenarioId": "",
        "algorithm": "token-bucket",
        "language": "go",
        "config": {"capacity": 2, "ratePerSecond": 1},
        "storeMode": "memory",
    }
    status, _, empty = exchange("/api/runs", {**base, "requestTimeline": []})
    assert status == 200 and empty["events"] == [] and empty["decisions"] == []

    invalid_inputs = [
        {**base, "requestTimeline": [{"atMs": 0.5, "cost": 1, "key": "alice"}]},
        {**base, "requestTimeline": [{"atMs": 9_007_199_254_740_992, "cost": 1, "key": "alice"}]},
        {**base, "requestTimeline": [point(index) for index in range(101)]},
        {**base, "config": {"capacity": 0, "ratePerSecond": 1}, "requestTimeline": [point(0)]},
        {**base, "config": {"capacity": 1, "ratePerSecond": 5e-324}, "requestTimeline": [point(0)]},
        {**base, "config": {"capacity": 1e308, "ratePerSecond": 1}, "requestTimeline": [point(0)]},
        {**base, "requestTimeline": [point(0, 1e308)]},
        {**base, "requestTimeline": [point(0, 1, "a" * 129)]},
        {**base, "requestTimeline": [point(0, 1, "界" * 43)]},
        {**base, "requestTimeline": [{"atMs": 0, "key": "alice"}]},
        {**base, "requestTimeline": [{"atMs": 0, "cost": 1}]},
        {**base, "requestTimeline": [{"atMs": 0, "cost": 1, "key": ""}]},
        {**base, "requestTimeline": [point(10), point(9)]},
        {**base, "requestTimeline": None},
        {key: value for key, value in base.items() if key != "requestTimeline"},
        {
            **base,
            "algorithm": "fixed-window",
            "config": {"limit": 1, "windowMs": 0.5},
            "requestTimeline": [point(0)],
        },
        {
            **base,
            "algorithm": "fixed-window",
            "config": {"limit": 1e308, "windowMs": 1000},
            "requestTimeline": [point(0)],
        },
    ]
    for payload in invalid_inputs:
        status, _, body = exchange("/api/runs", payload)
        assert status == 400 and body["error"]["code"] in {"invalid_request", "run_failed"}, body

    status, _, huge = exchange("/api/runs", {**base, "requestTimeline": [point(0, 3)]})
    assert status == 200
    assert huge["decisions"][0]["reason"] == "cost_exceeds_capacity"
    assert huge["decisions"][0]["retryAfterMs"] == 0


def verify_memory_http_contract() -> None:
    key = f"http-{time.time_ns()}"
    statuses = []
    final_headers = None
    for _ in range(3):
        status, headers, body = exchange(
            "/demo/search?store=memory&limit=2&window_ms=1000&failure=fail-open",
            headers={"X-RateLimit-Key": key},
        )
        statuses.append(status)
        final_headers = headers
        assert body["decision"]["allowed"] is (status == 200)
    assert statuses == [200, 200, 429]
    assert final_headers["RateLimit-Limit"] == "2"
    assert final_headers["RateLimit-Remaining"] == "0"
    assert final_headers["Retry-After"] == "1"
    assert final_headers["RateLimit-Policy"] == "2;w=1"

    status, _, _ = exchange(
        "/demo/search?store=memory&limit=2&window_ms=1000&failure=fail-open",
        headers={"X-RateLimit-Key": key + "-independent"},
    )
    assert status == 200

    for path in (
        "/demo/search?store=unknown",
        "/demo/search?failure=unknown",
        "/demo/search?limit=0",
        "/demo/search?window_ms=0",
    ):
        status, _, _ = exchange(path)
        assert status == 400, path


def verify_policy_composition() -> None:
    path = "/demo/policy-composition?store=memory&limit=2&window_ms=3600000&failure=fail-closed"
    for _ in range(2):
        assert exchange(path, headers={"X-RateLimit-Key": "alice"})[0] == 200
    status, headers, body = exchange(path, headers={"X-RateLimit-Key": "alice"})
    assert status == 429 and body["rejectedBy"] == ["per-client"]
    assert [policy["id"] for policy in body["policies"]] == ["endpoint-wide", "per-client"]
    assert headers["RateLimit-Policy"] == "2;w=3600, 4;w=3600"

    assert exchange(path, headers={"X-RateLimit-Key": "bob"})[0] == 200
    status, _, body = exchange(path, headers={"X-RateLimit-Key": "charlie"})
    assert status == 429 and body["rejectedBy"] == ["endpoint-wide"]


def verify_replica_and_hot_key_scenarios() -> None:
    key = f"replica-{time.time_ns()}"
    local = "/demo/local-vs-shared?store=memory&limit=1&window_ms=3600000&failure=fail-closed"
    first_a = exchange(local + "&replica=a", headers={"X-RateLimit-Key": key})
    first_b = exchange(local + "&replica=b", headers={"X-RateLimit-Key": key})
    second_a = exchange(local + "&replica=a", headers={"X-RateLimit-Key": key})
    assert first_a[0] == 200 and first_b[0] == 200 and second_a[0] == 429
    assert first_a[2]["scope"] == "replica-local" and first_b[2]["replica"] == "b"
    assert exchange(local + "&replica=c", headers={"X-RateLimit-Key": key})[0] == 400

    hot = "/demo/hot-key-sharding?store=memory&limit=100&window_ms=3600000&failure=fail-closed"
    alice_one = exchange(hot, headers={"X-RateLimit-Key": "alice"})
    alice_two = exchange(hot, headers={"X-RateLimit-Key": "alice"})
    bob = exchange(hot, headers={"X-RateLimit-Key": "bob"})
    assert alice_one[0] == alice_two[0] == bob[0] == 200
    assert alice_one[2]["routing"]["shard"] == alice_two[2]["routing"]["shard"]
    assert alice_one[2]["routing"]["shard"] != bob[2]["routing"]["shard"]
    assert alice_one[2]["routing"]["strategy"] == "hash(key)%4"


wait_until_ready()
verify_catalog_and_ui()
verify_four_language_parity()
verify_input_boundaries()
verify_memory_http_contract()
verify_policy_composition()
verify_replica_and_hot_key_scenarios()
print("HTTP verification passed: catalog, embedded UI, 5x4 parity, trace/source, input safety, 429, policy composition, replica scope, and hot-key routing")
