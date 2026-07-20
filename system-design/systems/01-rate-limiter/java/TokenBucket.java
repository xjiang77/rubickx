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

    /**
     * 全参构造函数。
     *
     * @param capacity  桶容量（突发上限，令牌数）
     * @param refillRate 补充速率（令牌/秒），决定长期平均放行速率
     * @param nowNanos  时钟来源，返回纳秒级时间；便于注入假时钟做确定性测试
     */
    public TokenBucket(double capacity, double refillRate, LongSupplier nowNanos) {
        if (!Double.isFinite(capacity) || capacity <= 0) {
            throw new IllegalArgumentException("capacity must be finite and greater than zero");
        }
        if (!Double.isFinite(refillRate) || refillRate < 0) {
            throw new IllegalArgumentException("refillRate must be finite and non-negative");
        }
        if (nowNanos == null) {
            throw new IllegalArgumentException("nowNanos must not be null");
        }
        this.capacity = capacity;
        this.refillRate = refillRate;
        this.nowNanos = nowNanos;
        // 初始化时桶是满的：启动即可瞬间放行最多 capacity 个请求，
        // 用于扛住刚启动时的流量洪峰（突发），之后才按 refillRate 慢慢补充
        this.tokens = capacity;
        // 记录上次补充时间，作为后续计算时间差的基准
        this.last = nowNanos.getAsLong();
    }

    /** 便捷方法：每次尝试取走 1 个令牌。 */
    public synchronized boolean allow() {
        return allow(1);
    }

    /**
     * 尝试取走 n 个令牌。
     *
     * @param n 本次请求需要的令牌数（可为小数，便于按比例限流）
     * @return true 表示放行；false 表示令牌不足、请求被限流
     */
    public synchronized boolean allow(double n) {
        if (!Double.isFinite(n) || n <= 0) {
            return false;
        }
        // 1. 读取当前时间，与上次补充时间做差得到"流逝时间"
        long t = nowNanos.getAsLong();
        // 2. 换算成秒（纳秒 / 1e9），用于按速率补充令牌
        double elapsed = Math.max(0, ((double) t - last) / 1_000_000_000.0);
        // 3. 时钟回拨时不倒退基准，避免后续重复补充
        if (t > last) {
            last = t;
        }
        // 4. 按"速率 × 时间"补充令牌，且不超过桶容量（上限截断）
        tokens = Math.min(capacity, tokens + elapsed * refillRate);
        // 5. 令牌充足则扣减并放行，否则拒绝（不扣减，保持当前水位）
        if (tokens >= n) {
            tokens -= n;
            return true;
        }
        return false;
    }

    /** 读取最近一次 allow 物化后的令牌数；本方法不主动补充。 */
    public synchronized double tokens() {
        return tokens;
    }
}
