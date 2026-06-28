import java.util.function.LongSupplier;

/**
 * 令牌桶限流器：以固定速率补充令牌，每次请求取走若干，桶空则拒绝。
 *
 * <p>capacity = 突发上限；refillRate = 长期平均速率（令牌/秒）。
 * 时钟通过 {@link LongSupplier}（纳秒）注入，测试用假时钟推进，不靠 sleep。
 *
 * <p>并发：方法用 {@code synchronized} 保护"读-补-判-扣-写"复合操作。
 * 也可换 {@code ReentrantLock}（可超时/可中断）或对纯计数用 {@code AtomicLong}。
 */
public class TokenBucket {
    private final double capacity;
    private final double refillRate;
    private final LongSupplier nowNanos;
    private double tokens;
    private long last;

    public TokenBucket(double capacity, double refillRate) {
        this(capacity, refillRate, System::nanoTime);
    }

    public TokenBucket(double capacity, double refillRate, LongSupplier nowNanos) {
        this.capacity = capacity;
        this.refillRate = refillRate;
        this.nowNanos = nowNanos;
        this.tokens = capacity;
        this.last = nowNanos.getAsLong();
    }

    public synchronized boolean allow() {
        return allow(1);
    }

    public synchronized boolean allow(double n) {
        long t = nowNanos.getAsLong();
        double elapsed = (t - last) / 1_000_000_000.0;
        last = t;
        tokens = Math.min(capacity, tokens + elapsed * refillRate);
        if (tokens >= n) {
            tokens -= n;
            return true;
        }
        return false;
    }

    public synchronized double tokens() {
        return tokens;
    }
}
