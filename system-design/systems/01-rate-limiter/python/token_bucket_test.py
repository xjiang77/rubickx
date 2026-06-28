import os
import sys
import threading

sys.path.insert(0, os.path.dirname(__file__))
from token_bucket import TokenBucket


class FakeClock:
    def __init__(self) -> None:
        self.t = 0.0

    def __call__(self) -> float:
        return self.t

    def advance(self, dt: float) -> None:
        self.t += dt


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


def test_concurrent_safe():
    # 无补充（rate=0），恰好 1000 个令牌；100 线程 ×50 次抢，线程安全则恰好放行 1000。
    tb = TokenBucket(capacity=1000, refill_rate=0)
    allowed = 0
    lock = threading.Lock()

    def worker():
        nonlocal allowed
        for _ in range(50):
            if tb.allow():
                with lock:
                    allowed += 1

    threads = [threading.Thread(target=worker) for _ in range(100)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    assert allowed == 1000
