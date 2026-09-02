import java.time.Duration;
import java.util.function.LongSupplier;

/** 用前一固定窗口的线性权重，近似统计最近一个窗口内的请求量。 */
public class SlidingWindowCounter {
    private final double limit;
    private final long windowNanos;
    private final LongSupplier nowNanos;
    private double previousCount;
    private double currentCount;
    private long currentWindowStart;

    public SlidingWindowCounter(double limit, Duration window) {
        this(limit, window, System::nanoTime);
    }

    public SlidingWindowCounter(double limit, Duration window, LongSupplier nowNanos) {
        if (!Double.isFinite(limit) || limit <= 0) {
            throw new IllegalArgumentException("limit must be finite and greater than zero");
        }
        if (window == null || window.isZero() || window.isNegative()) {
            throw new IllegalArgumentException("window must be greater than zero");
        }
        if (nowNanos == null) {
            throw new IllegalArgumentException("nowNanos must not be null");
        }

        long nanos;
        try {
            nanos = window.toNanos();
        } catch (ArithmeticException exc) {
            throw new IllegalArgumentException("window is too large", exc);
        }
        if (nanos <= 0) {
            throw new IllegalArgumentException("window must be at least one nanosecond");
        }

        this.limit = limit;
        this.windowNanos = nanos;
        this.nowNanos = nowNanos;
        this.currentWindowStart = nowNanos.getAsLong();
    }

    public synchronized boolean allow() {
        return allow(1);
    }

    public synchronized boolean allow(double n) {
        if (!Double.isFinite(n) || n <= 0) {
            return false;
        }

        long t = nowNanos.getAsLong();
        long elapsed = t > currentWindowStart ? t - currentWindowStart : 0;
        long windowsElapsed = elapsed / windowNanos;
        if (windowsElapsed == 1) {
            previousCount = currentCount;
            currentCount = 0;
        } else if (windowsElapsed >= 2) {
            previousCount = 0;
            currentCount = 0;
        }

        if (windowsElapsed >= 1) {
            currentWindowStart += windowsElapsed * windowNanos;
            elapsed = t - currentWindowStart;
        }

        double previousWeight = 1.0 - (double) elapsed / windowNanos;
        double estimate = currentCount + previousCount * previousWeight;
        if (estimate + n > limit) {
            return false;
        }
        currentCount += n;
        return true;
    }
}
