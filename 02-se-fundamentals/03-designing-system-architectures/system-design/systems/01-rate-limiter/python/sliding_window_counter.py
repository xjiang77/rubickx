import math
import threading
import time
from typing import Callable


class SlidingWindowCounter:
    """用前一固定窗口的线性权重，近似统计最近一个窗口内的请求量。"""

    def __init__(
        self,
        limit: float,
        window_seconds: float,
        now: Callable[[], float] = time.monotonic,
    ) -> None:
        try:
            limit = float(limit)
            window_seconds = float(window_seconds)
        except (TypeError, ValueError) as exc:
            raise ValueError("limit and window_seconds must be numbers") from exc
        if not math.isfinite(limit) or limit <= 0:
            raise ValueError("limit must be finite and greater than zero")
        if not math.isfinite(window_seconds) or window_seconds <= 0:
            raise ValueError("window_seconds must be finite and greater than zero")
        if not callable(now):
            raise ValueError("now must be callable")

        self.limit = limit
        self.window_seconds = window_seconds
        self._now = now
        self._previous_count = 0.0
        self._current_count = 0.0
        self._current_window_start = now()
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
            elapsed = max(0.0, t - self._current_window_start)
            windows_elapsed = int(elapsed // self.window_seconds)
            if windows_elapsed == 1:
                self._previous_count = self._current_count
                self._current_count = 0.0
            elif windows_elapsed >= 2:
                self._previous_count = 0.0
                self._current_count = 0.0

            if windows_elapsed >= 1:
                self._current_window_start += windows_elapsed * self.window_seconds
                elapsed = max(0.0, t - self._current_window_start)

            previous_weight = 1.0 - elapsed / self.window_seconds
            estimate = self._current_count + self._previous_count * previous_weight
            if estimate + n > self.limit:
                return False
            self._current_count += n
            return True
