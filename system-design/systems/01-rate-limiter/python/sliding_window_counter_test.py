import os
import sys
import threading

import pytest

sys.path.insert(0, os.path.dirname(__file__))
from sliding_window_counter import SlidingWindowCounter


class FakeClock:
    def __init__(self, t: float = 0.0) -> None:
        self.t = t

    def __call__(self) -> float:
        return self.t

    def advance(self, dt: float) -> None:
        self.t += dt

    def set(self, t: float) -> None:
        self.t = t


@pytest.mark.parametrize(
    ("limit", "window_seconds"),
    [
        (0, 1),
        (-1, 1),
        (float("inf"), 1),
        (float("nan"), 1),
        (1, 0),
        (1, -1),
        (1, float("inf")),
        (1, float("nan")),
        (None, 1),
    ],
)
def test_invalid_config_fails_fast(limit, window_seconds):
    with pytest.raises(ValueError):
        SlidingWindowCounter(limit, window_seconds)


def test_invalid_clock_fails_fast():
    with pytest.raises(ValueError):
        SlidingWindowCounter(1, 1, now=None)


def test_current_window_reaches_limit_then_rejects():
    counter = SlidingWindowCounter(5, 10, now=FakeClock())
    assert counter.allow(2.5) is True
    assert counter.allow(2.5) is True
    assert counter.allow() is False


def test_exact_rollover_keeps_full_previous_window_weight():
    clk = FakeClock()
    counter = SlidingWindowCounter(10, 10, now=clk)
    assert counter.allow(10) is True
    clk.advance(10)
    assert counter.allow() is False


def test_previous_window_has_half_weight_at_half_window():
    clk = FakeClock()
    counter = SlidingWindowCounter(10, 10, now=clk)
    assert counter.allow(10) is True
    clk.advance(15)
    assert counter.allow(5) is True
    assert counter.allow(0.001) is False


def test_history_is_cleared_after_two_windows():
    clk = FakeClock()
    counter = SlidingWindowCounter(10, 10, now=clk)
    assert counter.allow(10) is True
    clk.advance(20)
    assert counter.allow(10) is True


def test_rejected_request_is_not_counted():
    clk = FakeClock()
    counter = SlidingWindowCounter(10, 10, now=clk)
    assert counter.allow(8) is True
    assert counter.allow(3) is False
    clk.advance(15)
    assert counter.allow(6) is True


@pytest.mark.parametrize("n", [0, -1, float("inf"), float("-inf"), float("nan")])
def test_invalid_cost_is_rejected_without_changing_state(n):
    counter = SlidingWindowCounter(10, 10, now=FakeClock())
    assert counter.allow(4) is True
    assert counter.allow(n) is False
    assert counter.allow(6) is True


def test_clock_rollback_does_not_move_window_baseline_backwards():
    clk = FakeClock(100)
    counter = SlidingWindowCounter(10, 10, now=clk)
    assert counter.allow(10) is True
    clk.set(110)
    assert counter.allow() is False
    clk.set(105)
    assert counter.allow() is False
    clk.set(115)
    assert counter.allow(5) is True
    assert counter.allow(0.001) is False


def test_concurrent_requests_allow_exactly_the_limit():
    workers = 50
    limit = 20
    counter = SlidingWindowCounter(limit, 10, now=FakeClock())
    start = threading.Barrier(workers + 1)
    results = []
    results_lock = threading.Lock()

    def worker():
        start.wait()
        allowed = counter.allow()
        with results_lock:
            results.append(allowed)

    threads = [threading.Thread(target=worker) for _ in range(workers)]
    for thread in threads:
        thread.start()
    start.wait()
    for thread in threads:
        thread.join()

    assert sum(results) == limit
