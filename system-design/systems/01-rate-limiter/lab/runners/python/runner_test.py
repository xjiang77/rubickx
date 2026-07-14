import json
import subprocess
import sys
from pathlib import Path

import pytest


RUNNER = Path(__file__).with_name("runner.py")
FIXTURES = json.loads((Path(__file__).parents[2] / "fixtures" / "core-parity.json").read_text())


def run_lines(*requests: dict | str) -> list[dict]:
    input_text = "\n".join(
        request if isinstance(request, str) else json.dumps(request)
        for request in requests
    ) + "\n"
    completed = subprocess.run(
        [sys.executable, str(RUNNER)],
        input=input_text,
        text=True,
        capture_output=True,
        check=False,
    )
    assert completed.returncode == 0, completed.stderr
    return [json.loads(line) for line in completed.stdout.splitlines()]


def request(algorithm: str, config: dict, timeline: list[dict]) -> dict:
    return {
        "scenarioId": "contract-test",
        "algorithm": algorithm,
        "language": "python",
        "config": config,
        "requestTimeline": timeline,
        "storeMode": "memory",
    }


def test_jsonl_seam_runs_real_token_bucket_and_recovers_after_bad_input():
    responses = run_lines(
        "not json",
        request(
            "token-bucket",
            {"capacity": 2, "ratePerSecond": 1},
            [
                {"atMs": 0, "cost": 1, "key": "alice"},
                {"atMs": 0, "cost": 1, "key": "alice"},
                {"atMs": 0, "cost": 1, "key": "alice"},
                {"atMs": 1000, "cost": 1, "key": "alice"},
            ],
        ),
    )

    assert responses[0]["error"]["code"] == "invalid_json"
    assert [item["allowed"] for item in responses[1]["decisions"]] == [
        True,
        True,
        False,
        True,
    ]
    assert responses[1]["events"][-1]["stepId"] == "token.decision"
    assert set(responses[1]["events"][-1]) == {
        "seq",
        "stepId",
        "actor",
        "timestampMs",
        "before",
        "after",
        "decision",
        "reason",
    }


@pytest.mark.parametrize(
    ("algorithm", "config", "expected"),
    [
        ("fixed-window", {"limit": 2, "windowMs": 1000}, [True, True, False, True]),
        ("sliding-window-log", {"limit": 2, "windowMs": 1000}, [True, True, False, True]),
        ("sliding-window-counter", {"limit": 2, "windowMs": 1000}, [True, True, False, False]),
        ("leaky-bucket", {"capacity": 2, "ratePerSecond": 1}, [True, True, False, True]),
    ],
)
def test_jsonl_seam_supports_each_algorithm(algorithm, config, expected):
    timeline = [
        {"atMs": 0, "cost": 1, "key": "alice"},
        {"atMs": 0, "cost": 1, "key": "alice"},
        {"atMs": 0, "cost": 1, "key": "alice"},
        {"atMs": 1000, "cost": 1, "key": "alice"},
    ]
    response = run_lines(request(algorithm, config, timeline))[0]
    assert [item["allowed"] for item in response["decisions"]] == expected


def test_invalid_request_does_not_stop_following_lines():
    responses = run_lines(
        request("fixed-window", {"limit": 0, "windowMs": 1000}, []),
        request(
            "fixed-window",
            {"limit": 1, "windowMs": 1000},
            [{"atMs": 0, "cost": 1, "key": "alice"}],
        ),
    )
    assert responses[0]["error"]["code"] == "invalid_request"
    assert responses[1]["decisions"][0]["allowed"] is True


@pytest.mark.parametrize("case", FIXTURES, ids=lambda case: case["name"])
def test_shared_boundary_and_time_jump_fixtures(case):
    response = run_lines(
        request(case["algorithm"], case["config"], case["requestTimeline"])
    )[0]
    assert [item["allowed"] for item in response["decisions"]] == case["expectedAllowed"]
    assert [item["remaining"] for item in response["decisions"]] == case["expectedRemaining"]
    assert response["decisions"][-1]["reason"] == case["expectedLastReason"]
    assert response["decisions"][-1]["retryAfterMs"] == case["expectedLastRetryAfterMs"]
    if "expectedLastResetAtMs" in case:
        assert response["decisions"][-1]["resetAtMs"] == case["expectedLastResetAtMs"]


@pytest.mark.parametrize(
    ("algorithm", "config"),
    [
        ("fixed-window", {"limit": 2, "windowMs": 1000}),
        ("sliding-window-log", {"limit": 2, "windowMs": 1000}),
        ("sliding-window-counter", {"limit": 2, "windowMs": 1000}),
        ("token-bucket", {"capacity": 2, "ratePerSecond": 1}),
        ("leaky-bucket", {"capacity": 2, "ratePerSecond": 1}),
    ],
)
def test_empty_timeline_is_a_successful_noop_for_every_algorithm(algorithm, config):
    response = run_lines(request(algorithm, config, []))[0]
    assert response == {"events": [], "decisions": []}


@pytest.mark.parametrize(
    "timeline",
    [
        [{"atMs": -1, "cost": 1, "key": "alice"}],
        [
            {"atMs": 2, "cost": 1, "key": "alice"},
            {"atMs": 1, "cost": 1, "key": "alice"},
        ],
    ],
    ids=["negative-time", "non-monotonic-time"],
)
def test_invalid_time_is_rejected_at_the_public_seam(timeline):
    response = run_lines(request("fixed-window", {"limit": 2, "windowMs": 1000}, timeline))[0]
    assert response["error"]["code"] == "invalid_request"


def test_fractional_time_and_more_than_100_items_are_rejected():
    fractional = [{"atMs": 0.5, "cost": 1, "key": "alice"}]
    oversized = [{"atMs": index, "cost": 1, "key": "alice"} for index in range(101)]
    responses = run_lines(
        request("fixed-window", {"limit": 2, "windowMs": 1000}, fractional),
        request("fixed-window", {"limit": 2, "windowMs": 1000}, oversized),
    )
    assert [response["error"]["code"] for response in responses] == [
        "invalid_request",
        "invalid_request",
    ]


def test_time_above_javascript_max_safe_integer_is_rejected():
    response = run_lines(
        request(
            "fixed-window",
            {"limit": 2, "windowMs": 1000},
            [{"atMs": 9_007_199_254_740_992, "cost": 1, "key": "alice"}],
        )
    )[0]
    assert response["error"]["code"] == "invalid_request"
    assert "safe integer milliseconds" in response["error"]["message"]


@pytest.mark.parametrize(
    ("algorithm", "config"),
    [
        ("fixed-window", {"limit": 2, "windowMs": 1000}),
        ("token-bucket", {"capacity": 2, "ratePerSecond": 1}),
        ("leaky-bucket", {"capacity": 2, "ratePerSecond": 1}),
    ],
)
def test_each_request_trace_is_state_continuous(algorithm, config):
    response = run_lines(
        request(
            algorithm,
            config,
            [
                {"atMs": 0, "cost": 1, "key": "alice"},
                {"atMs": 1000, "cost": 1, "key": "alice"},
            ],
        )
    )[0]
    events = response["events"]
    assert len(events) == 4
    for index in range(0, len(events), 2):
        assert events[index]["after"] == events[index + 1]["before"]


@pytest.mark.parametrize(
    ("algorithm", "config"),
    [
        ("fixed-window", {"limit": 0, "windowMs": 1000}),
        ("fixed-window", {"limit": 2, "windowMs": 0.5}),
        ("token-bucket", {"capacity": 2, "ratePerSecond": 0}),
    ],
)
def test_non_positive_config_is_rejected(algorithm, config):
    response = run_lines(request(algorithm, config, []))[0]
    assert response["error"]["code"] == "invalid_request"


def test_window_above_javascript_max_safe_integer_is_rejected():
    response = run_lines(
        request(
            "fixed-window",
            {"limit": 2, "windowMs": 9_007_199_254_740_992},
            [],
        )
    )[0]
    assert response["error"]["code"] == "invalid_request"
    assert "positive safe integer" in response["error"]["message"]


@pytest.mark.parametrize(
    ("algorithm", "config", "timeline"),
    [
        (
            "fixed-window",
            {"limit": 1e308, "windowMs": 1000},
            [{"atMs": 0, "cost": 1, "key": "alice"}],
        ),
        (
            "token-bucket",
            {"capacity": 1e308, "ratePerSecond": 1},
            [{"atMs": 0, "cost": 1, "key": "alice"}],
        ),
        (
            "token-bucket",
            {"capacity": 1, "ratePerSecond": 1e308},
            [{"atMs": 0, "cost": 1, "key": "alice"}],
        ),
        (
            "fixed-window",
            {"limit": 2, "windowMs": 1000},
            [{"atMs": 0, "cost": 1e308, "key": "alice"}],
        ),
        (
            "token-bucket",
            {"capacity": 1, "ratePerSecond": 5e-324},
            [{"atMs": 0, "cost": 1, "key": "alice"}],
        ),
    ],
    ids=[
        "limit-above-max-safe",
        "capacity-above-max-safe",
        "rate-above-max-safe",
        "cost-above-max-safe",
        "recovery-time-above-max-safe",
    ],
)
def test_unsafe_quantities_are_rejected_at_the_public_seam(algorithm, config, timeline):
    response = run_lines(request(algorithm, config, timeline))[0]
    assert response["error"]["code"] == "invalid_request"


def test_integer_quantity_too_large_for_float_is_invalid_not_internal():
    response = run_lines(
        request(
            "fixed-window",
            {"limit": 10**400, "windowMs": 1000},
            [{"atMs": 0, "cost": 1, "key": "alice"}],
        )
    )[0]
    assert response["error"]["code"] == "invalid_request"


def test_max_safe_quantities_are_accepted_at_the_public_seam():
    maximum = 9_007_199_254_740_991
    responses = run_lines(
        request(
            "fixed-window",
            {"limit": maximum, "windowMs": 1000},
            [{"atMs": 0, "cost": maximum, "key": "alice"}],
        ),
        request(
            "token-bucket",
            {"capacity": maximum, "ratePerSecond": maximum},
            [{"atMs": 0, "cost": maximum, "key": "alice"}],
        ),
    )
    assert [response["decisions"][0]["allowed"] for response in responses] == [True, True]


@pytest.mark.parametrize(
    "key",
    ["a" * 129, "界" * 43],
    ids=["ascii-129-bytes", "multibyte-129-bytes"],
)
def test_key_above_128_utf8_bytes_is_rejected_at_the_public_seam(key):
    response = run_lines(
        request(
            "fixed-window",
            {"limit": 1, "windowMs": 1000},
            [{"atMs": 0, "cost": 1, "key": key}],
        )
    )[0]
    assert response["error"]["code"] == "invalid_request"


def test_multibyte_key_at_128_utf8_bytes_is_accepted():
    key = "界" * 42 + "ab"
    response = run_lines(
        request(
            "fixed-window",
            {"limit": 1, "windowMs": 1000},
            [{"atMs": 0, "cost": 1, "key": key}],
        )
    )[0]
    assert response["decisions"][0]["allowed"] is True
    assert response["events"][0]["actor"] == key


@pytest.mark.parametrize(
    "timeline",
    [
        [{"atMs": 0, "key": "alice"}],
        [{"atMs": 0, "cost": 1}],
    ],
    ids=["missing-cost", "missing-key"],
)
def test_cost_and_key_are_required_at_the_public_seam(timeline):
    response = run_lines(
        request("fixed-window", {"limit": 1, "windowMs": 1000}, timeline)
    )[0]
    assert response["error"]["code"] == "invalid_request"
