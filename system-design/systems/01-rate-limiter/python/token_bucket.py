import math
import threading
import time
from typing import Callable


class TokenBucket:
    """令牌桶限流器：以固定速率补充令牌，每次请求取走若干，桶空则拒绝。

    - capacity   桶容量（允许的瞬时突发量）
    - refill_rate 每秒补充的令牌数（长期平均速率）
    - now        可注入的单调时钟（默认 time.monotonic），测试用假时钟推进

    并发：用 threading.Lock 保护"读-补-判-扣-写"这段复合操作。
    注意 GIL 不能替代锁——`self._tokens -= n` 跨多条字节码，多线程会交叉丢更新。
    """

    def __init__(
        self,
        capacity: float,
        refill_rate: float,
        now: Callable[[], float] = time.monotonic,
    ) -> None:
        try:
            capacity = float(capacity)
            refill_rate = float(refill_rate)
        except (TypeError, ValueError) as exc:
            raise ValueError("capacity and refill_rate must be numbers") from exc
        if not math.isfinite(capacity) or capacity <= 0:
            raise ValueError("capacity must be finite and greater than zero")
        if not math.isfinite(refill_rate) or refill_rate < 0:
            raise ValueError("refill_rate must be finite and non-negative")
        if not callable(now):
            raise ValueError("now must be callable")

        self.capacity = capacity
        self.refill_rate = refill_rate
        self._now = now
        self._tokens = capacity
        self._last = now()
        self._lock = threading.Lock()

    def allow(self, n: float = 1.0) -> bool:
        try:
            n = float(n)
        except (TypeError, ValueError):
            return False
        if not math.isfinite(n) or n <= 0:
            return False

        with self._lock:
            t = self._now()
            elapsed = max(0.0, t - self._last)
            if t > self._last:
                self._last = t
            self._tokens = min(
                self.capacity,
                self._tokens + elapsed * self.refill_rate,
            )
            if self._tokens >= n:
                self._tokens -= n
                return True
            return False

    def tokens(self) -> float:
        """返回最近一次 allow 物化后的水位，不主动补充令牌。"""
        with self._lock:
            return self._tokens
