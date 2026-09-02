"""Deterministic rate-limiter algorithms that emit semantic execution traces."""

from __future__ import annotations

import copy
import math
from dataclasses import dataclass
from typing import Any, Callable


class InvalidRequest(ValueError):
    """The JSON value is valid, but does not satisfy the runner contract."""


MAX_SAFE_INTEGER_MS = 9_007_199_254_740_991


def _number(value: Any, name: str, *, positive: bool = False) -> float:
    if isinstance(value, bool) or not isinstance(value, (int, float)):
        raise InvalidRequest(f"{name} must be a number")
    try:
        value = float(value)
    except OverflowError as failure:
        raise InvalidRequest(f"{name} must be finite") from failure
    if not math.isfinite(value) or (positive and value <= 0):
        qualifier = "a positive finite number" if positive else "finite"
        raise InvalidRequest(f"{name} must be {qualifier}")
    return value


def _quantity(value: Any, name: str) -> float:
    result = _number(value, name, positive=True)
    if result > MAX_SAFE_INTEGER_MS:
        raise InvalidRequest(
            f"{name} must be a positive finite number no greater than "
            f"{MAX_SAFE_INTEGER_MS}"
        )
    return result


def _rounded(value: float) -> int | float:
    # Quantize the actual IEEE-754 value; exact binary ties go away from zero.
    value = float(value)
    magnitude = math.floor(abs(value) * 1_000_000 + 0.5) / 1_000_000
    result = math.copysign(magnitude, value)
    if result == 0:
        return 0
    if result.is_integer():
        return int(result)
    return result


def _normalise(value: Any) -> Any:
    if isinstance(value, float):
        return _rounded(value)
    if isinstance(value, dict):
        return {key: _normalise(item) for key, item in value.items()}
    if isinstance(value, list):
        return [_normalise(item) for item in value]
    return value


@dataclass(frozen=True)
class TimelineItem:
    at_ms: float
    cost: float
    key: str


class Observer:
    def __init__(self) -> None:
        self.events: list[dict[str, Any]] = []

    def emit(
        self,
        step_id: str,
        item: TimelineItem,
        before: dict[str, Any],
        after: dict[str, Any],
        decision: dict[str, Any] | None,
        reason: str,
    ) -> None:
        self.events.append(
            {
                "seq": len(self.events) + 1,
                "stepId": step_id,
                "actor": item.key,
                "timestampMs": _rounded(item.at_ms),
                "before": _normalise(copy.deepcopy(before)),
                "after": _normalise(copy.deepcopy(after)),
                "decision": _normalise(copy.deepcopy(decision)),
                "reason": reason,
            }
        )


def _decision(
    allowed: bool,
    remaining: float,
    retry_after_ms: float,
    reset_at_ms: float,
    reason: str,
) -> dict[str, Any]:
    return {
        "allowed": allowed,
        "remaining": _rounded(max(0.0, remaining)),
        "retryAfterMs": _rounded(max(0.0, retry_after_ms)),
        "resetAtMs": _rounded(max(0.0, reset_at_ms)),
        "reason": reason,
    }


def validate_timeline(value: Any) -> list[TimelineItem]:
    if not isinstance(value, list):
        raise InvalidRequest("requestTimeline must be an array")
    if len(value) > 100:
        raise InvalidRequest("requestTimeline must contain at most 100 items")
    result: list[TimelineItem] = []
    previous = -1.0
    for index, raw in enumerate(value):
        if not isinstance(raw, dict):
            raise InvalidRequest(f"requestTimeline[{index}] must be an object")
        at_ms = _number(raw.get("atMs"), f"requestTimeline[{index}].atMs")
        cost = _quantity(raw.get("cost"), f"requestTimeline[{index}].cost")
        key = raw.get("key")
        if (
            not at_ms.is_integer()
            or at_ms < 0
            or at_ms > MAX_SAFE_INTEGER_MS
            or at_ms < previous
        ):
            raise InvalidRequest(
                "requestTimeline atMs must be non-negative, non-decreasing safe integer milliseconds"
            )
        try:
            key_bytes = key.encode("utf-8") if isinstance(key, str) else b""
        except UnicodeEncodeError:
            key_bytes = b""
        if not isinstance(key, str) or not key or not key_bytes or len(key_bytes) > 128:
            raise InvalidRequest(
                f"requestTimeline[{index}].key must be a non-empty UTF-8 string "
                "of at most 128 bytes"
            )
        result.append(TimelineItem(at_ms, cost, key))
        previous = at_ms
    return result


def _window_config(config: Any) -> tuple[float, float]:
    if not isinstance(config, dict):
        raise InvalidRequest("config must be an object")
    limit = _quantity(config.get("limit"), "config.limit")
    window_ms = _number(config.get("windowMs"), "config.windowMs", positive=True)
    if not window_ms.is_integer() or window_ms > MAX_SAFE_INTEGER_MS:
        raise InvalidRequest(
            "config.windowMs must be a positive safe integer in milliseconds"
        )
    return limit, window_ms


def _bucket_config(config: Any) -> tuple[float, float]:
    if not isinstance(config, dict):
        raise InvalidRequest("config must be an object")
    capacity = _quantity(config.get("capacity"), "config.capacity")
    rate = _quantity(config.get("ratePerSecond"), "config.ratePerSecond")
    rate_per_ms = rate / 1000.0
    if rate_per_ms <= 0 or capacity > MAX_SAFE_INTEGER_MS * rate_per_ms:
        raise InvalidRequest(
            "config capacity and ratePerSecond must yield a finite maximum recovery "
            f"time no greater than {MAX_SAFE_INTEGER_MS} milliseconds"
        )
    maximum_recovery_ms = capacity / rate_per_ms
    if not math.isfinite(maximum_recovery_ms) or maximum_recovery_ms > MAX_SAFE_INTEGER_MS:
        raise InvalidRequest(
            "config capacity and ratePerSecond must yield a finite maximum recovery "
            f"time no greater than {MAX_SAFE_INTEGER_MS} milliseconds"
        )
    return capacity, rate


def run_fixed_window(config: Any, timeline: list[TimelineItem], observer: Observer) -> list[dict[str, Any]]:
    limit, window_ms = _window_config(config)
    states: dict[str, dict[str, float]] = {}
    decisions = []
    for item in timeline:
        window_start = math.floor(item.at_ms / window_ms) * window_ms
        state = states.setdefault(item.key, {"windowStartMs": window_start, "count": 0.0})
        before = copy.deepcopy(state)
        # @step:fixed.locate-window
        if state["windowStartMs"] != window_start:
            state["windowStartMs"] = window_start
            state["count"] = 0.0
        observer.emit("fixed.locate-window", item, before, state, None, "window_selected")

        decision_before = copy.deepcopy(state)
        allowed = state["count"] + item.cost <= limit
        if allowed:
            state["count"] += item.cost
        reason = "within_limit" if allowed else (
            "cost_exceeds_limit" if item.cost > limit else "limit_exceeded"
        )
        reset_at = window_start + window_ms
        # @step:fixed.decision
        decision = _decision(
            allowed,
            limit - state["count"],
            0 if allowed or item.cost > limit else reset_at - item.at_ms,
            reset_at,
            reason,
        )
        observer.emit("fixed.decision", item, decision_before, state, decision, reason)
        decisions.append(decision)
    return decisions


def run_sliding_log(config: Any, timeline: list[TimelineItem], observer: Observer) -> list[dict[str, Any]]:
    limit, window_ms = _window_config(config)
    states: dict[str, list[dict[str, float]]] = {}
    decisions = []
    for item in timeline:
        entries = states.setdefault(item.key, [])
        before = {"entries": copy.deepcopy(entries), "used": sum(entry["cost"] for entry in entries)}
        cutoff = item.at_ms - window_ms
        # @step:sliding-log.evict
        entries[:] = [entry for entry in entries if entry["atMs"] > cutoff]
        used = sum(entry["cost"] for entry in entries)
        after_evict = {"entries": copy.deepcopy(entries), "used": used}
        observer.emit("sliding-log.evict", item, before, after_evict, None, "expired_entries_removed")

        allowed = used + item.cost <= limit
        if allowed:
            entries.append({"atMs": item.at_ms, "cost": item.cost})
            used += item.cost
        reason = "within_limit" if allowed else (
            "cost_exceeds_limit" if item.cost > limit else "limit_exceeded"
        )
        retry_after = 0.0
        if not allowed and item.cost <= limit:
            required_release = used + item.cost - limit
            released = 0.0
            for entry in entries:
                released += entry["cost"]
                if released + 1e-9 >= required_release:
                    retry_after = max(0.0, entry["atMs"] + window_ms - item.at_ms)
                    break
        reset_at = item.at_ms + retry_after if not allowed else (
            entries[0]["atMs"] + window_ms if entries else item.at_ms
        )
        after = {"entries": copy.deepcopy(entries), "used": used}
        # @step:sliding-log.decision
        decision = _decision(allowed, limit - used, retry_after, reset_at, reason)
        observer.emit("sliding-log.decision", item, after_evict, after, decision, reason)
        decisions.append(decision)
    return decisions


def run_sliding_counter(config: Any, timeline: list[TimelineItem], observer: Observer) -> list[dict[str, Any]]:
    limit, window_ms = _window_config(config)
    states: dict[str, dict[str, float]] = {}
    decisions = []
    for item in timeline:
        current_start = math.floor(item.at_ms / window_ms) * window_ms
        state = states.setdefault(
            item.key,
            {"currentWindowStartMs": current_start, "currentCount": 0.0, "previousCount": 0.0},
        )
        before = copy.deepcopy(state)
        windows_elapsed = int((current_start - state["currentWindowStartMs"]) / window_ms)
        # @step:sliding-counter.rotate
        if windows_elapsed == 1:
            state["previousCount"] = state["currentCount"]
            state["currentCount"] = 0.0
            state["currentWindowStartMs"] = current_start
        elif windows_elapsed > 1:
            state["previousCount"] = 0.0
            state["currentCount"] = 0.0
            state["currentWindowStartMs"] = current_start
        observer.emit("sliding-counter.rotate", item, before, state, None, "windows_rotated")

        elapsed = item.at_ms - current_start
        previous_weight = max(0.0, 1.0 - elapsed / window_ms)
        # @step:sliding-counter.estimate
        estimated = state["currentCount"] + state["previousCount"] * previous_weight
        estimate_state = {**state, "previousWeight": previous_weight, "estimatedCount": estimated}
        observer.emit("sliding-counter.estimate", item, state, estimate_state, None, "weighted_count_estimated")

        allowed = estimated + item.cost <= limit
        if allowed:
            state["currentCount"] += item.cost
            estimated += item.cost
        reason = "within_limit" if allowed else (
            "cost_exceeds_limit" if item.cost > limit else "limit_exceeded"
        )
        reset_at = current_start + window_ms
        retry_after = 0.0
        if not allowed and item.cost <= limit:
            excess = max(0.0, estimated + item.cost - limit)
            until_boundary = max(0.0, current_start + window_ms - item.at_ms)
            previous_count = state["previousCount"]
            current_count = state["currentCount"]
            if current_count + item.cost <= limit and previous_count > 0:
                retry_after = min(until_boundary, excess * window_ms / previous_count)
            else:
                retry_after = until_boundary
                if current_count > 0:
                    retry_after += max(
                        0.0,
                        (current_count + item.cost - limit) * window_ms / current_count,
                    )
            reset_at = item.at_ms + retry_after
        after = {**state, "previousWeight": previous_weight, "estimatedCount": estimated}
        # @step:sliding-counter.decision
        decision = _decision(
            allowed,
            limit - estimated,
            retry_after,
            reset_at,
            reason,
        )
        observer.emit("sliding-counter.decision", item, estimate_state, after, decision, reason)
        decisions.append(decision)
    return decisions


def run_token_bucket(config: Any, timeline: list[TimelineItem], observer: Observer) -> list[dict[str, Any]]:
    capacity, rate = _bucket_config(config)
    states: dict[str, dict[str, float]] = {}
    decisions = []
    for item in timeline:
        state = states.setdefault(item.key, {"tokens": capacity, "lastRefillMs": item.at_ms})
        before = copy.deepcopy(state)
        elapsed = max(0.0, item.at_ms - state["lastRefillMs"])
        # @step:token.refill
        state["tokens"] = min(capacity, state["tokens"] + elapsed * rate / 1000.0)
        state["lastRefillMs"] = item.at_ms
        observer.emit("token.refill", item, before, state, None, "tokens_refilled")

        decision_before = copy.deepcopy(state)
        allowed = state["tokens"] + 1e-9 >= item.cost
        if allowed:
            state["tokens"] -= item.cost
        reason = "token_available" if allowed else (
            "cost_exceeds_capacity" if item.cost > capacity else "insufficient_tokens"
        )
        retry_after = (
            0.0
            if allowed or item.cost > capacity
            else (item.cost - state["tokens"]) / rate * 1000.0
        )
        reset_at = item.at_ms + (capacity - state["tokens"]) / rate * 1000.0
        # @step:token.decision
        decision = _decision(allowed, state["tokens"], retry_after, reset_at, reason)
        observer.emit("token.decision", item, decision_before, state, decision, reason)
        decisions.append(decision)
    return decisions


def run_leaky_bucket(config: Any, timeline: list[TimelineItem], observer: Observer) -> list[dict[str, Any]]:
    capacity, rate = _bucket_config(config)
    states: dict[str, dict[str, float]] = {}
    decisions = []
    for item in timeline:
        state = states.setdefault(item.key, {"water": 0.0, "lastLeakMs": item.at_ms})
        before = copy.deepcopy(state)
        elapsed = max(0.0, item.at_ms - state["lastLeakMs"])
        # @step:leaky.drain
        state["water"] = max(0.0, state["water"] - elapsed * rate / 1000.0)
        state["lastLeakMs"] = item.at_ms
        observer.emit("leaky.drain", item, before, state, None, "queued_work_drained")

        decision_before = copy.deepcopy(state)
        allowed = state["water"] + item.cost <= capacity + 1e-9
        if allowed:
            state["water"] += item.cost
        reason = "queue_has_capacity" if allowed else (
            "cost_exceeds_capacity" if item.cost > capacity else "queue_full"
        )
        retry_after = (
            0.0
            if allowed or item.cost > capacity
            else (state["water"] + item.cost - capacity) / rate * 1000.0
        )
        reset_at = item.at_ms + state["water"] / rate * 1000.0
        # @step:leaky.decision
        decision = _decision(allowed, capacity - state["water"], retry_after, reset_at, reason)
        observer.emit("leaky.decision", item, decision_before, state, decision, reason)
        decisions.append(decision)
    return decisions


RUNNERS: dict[str, Callable[[Any, list[TimelineItem], Observer], list[dict[str, Any]]]] = {
    "fixed-window": run_fixed_window,
    "sliding-window-log": run_sliding_log,
    "sliding-window-counter": run_sliding_counter,
    "token-bucket": run_token_bucket,
    "leaky-bucket": run_leaky_bucket,
}


def run_request(request: Any) -> dict[str, Any]:
    if not isinstance(request, dict):
        raise InvalidRequest("request must be an object")
    algorithm = request.get("algorithm")
    if algorithm not in RUNNERS:
        raise InvalidRequest(f"unsupported algorithm: {algorithm!r}")
    timeline = validate_timeline(request.get("requestTimeline"))
    observer = Observer()
    decisions = RUNNERS[algorithm](request.get("config"), timeline, observer)
    return {"events": observer.events, "decisions": decisions}
