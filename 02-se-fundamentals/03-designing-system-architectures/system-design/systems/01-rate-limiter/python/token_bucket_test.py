import os
import sys
import threading

import pytest

sys.path.insert(0, os.path.dirname(__file__))
from token_bucket import TokenBucket


class FakeClock:
    def __init__(self) -> None:
        self.t = 0.0

    def __call__(self) -> float:
        return self.t

    def advance(self, dt: float) -> None:
        self.t += dt

    def set(self, t: float) -> None:
        self.t = t


@pytest.mark.parametrize(
    ("capacity", "refill_rate"),
    [
        (0, 1),
        (-1, 1),
        (float("inf"), 1),
        (float("nan"), 1),
        (1, -1),
        (1, float("inf")),
        (1, float("nan")),
        (None, 1),
    ],
)
def test_invalid_config_fails_fast(capacity, refill_rate):
    with pytest.raises(ValueError):
        TokenBucket(capacity=capacity, refill_rate=refill_rate)


def test_invalid_clock_fails_fast():
    with pytest.raises(ValueError):
        TokenBucket(capacity=1, refill_rate=1, now=None)


def test_burst_then_empty():
    clk = FakeClock()
    tb = TokenBucket(capacity=5, refill_rate=2, now=clk)
    assert sum(tb.allow() for _ in range(5)) == 5  # 满桶可瞬时放行 5 个
    assert tb.allow() is False                      # 桶空，拒绝


def test_refill_over_time():
    clk = FakeClock()
    tb = TokenBucket(capacity=5, refill_rate=2, now=clk)
    for _ in range(5):
        tb.allow()
    clk.advance(1.0)               # 1s 补 2 个令牌
    assert tb.allow() is True
    assert tb.allow() is True
    assert tb.allow() is False     # 补充的 2 个用完


def test_cap():
    clk = FakeClock()
    tb = TokenBucket(capacity=5, refill_rate=10, now=clk)
    clk.advance(100)               # 理论补 1000，但封顶 capacity
    assert sum(tb.allow() for _ in range(10)) == 5


def test_fractional_refill_and_multi_token_cost():
    clk = FakeClock()
    tb = TokenBucket(capacity=5, refill_rate=1, now=clk)
    assert tb.allow(2.5) is True
    assert tb.tokens() == pytest.approx(2.5)
    assert tb.allow(3) is False
    clk.advance(0.5)
    assert tb.allow(3) is True
    assert tb.tokens() == pytest.approx(0)


@pytest.mark.parametrize("n", [0, -1, float("inf"), float("-inf"), float("nan")])
def test_invalid_cost_is_rejected_without_changing_state(n):
    clk = FakeClock()
    tb = TokenBucket(capacity=5, refill_rate=1, now=clk)
    assert tb.allow(2) is True
    clk.advance(1)
    assert tb.allow(n) is False
    assert tb.tokens() == pytest.approx(3)
    assert tb.allow(4) is True  # 非法请求没有消耗令牌，也没有推进补充基准


def test_clock_rollback_does_not_move_refill_baseline_backwards():
    clk = FakeClock()
    tb = TokenBucket(capacity=1, refill_rate=1, now=clk)
    assert tb.allow() is True
    clk.set(1)
    assert tb.allow() is True
    clk.set(0.5)
    assert tb.allow() is False
    clk.set(1.5)
    assert tb.allow(0.5) is True
    assert tb.allow(0.5) is False


def test_concurrent_safe():
    # 无补充（rate=0），恰好 1000 个令牌；100 线程 ×50 次抢，线程安全则恰好放行 1000。
    tb = TokenBucket(capacity=1000, refill_rate=0)
    allowed = 0
    lock = threading.Lock()
    start = threading.Barrier(101)

    def worker():
        nonlocal allowed
        start.wait()
        for _ in range(50):
            if tb.allow():
                with lock:
                    allowed += 1

    threads = [threading.Thread(target=worker) for _ in range(100)]
    for t in threads:
        t.start()
    start.wait()
    for t in threads:
        t.join()
    assert allowed == 1000
