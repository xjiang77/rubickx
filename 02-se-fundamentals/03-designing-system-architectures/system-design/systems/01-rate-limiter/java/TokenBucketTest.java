import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.LongSupplier;
import org.junit.jupiter.api.Test;

public class TokenBucketTest {

    /** 可手动推进的假时钟（纳秒）。 */
    static final class Clock {
        long t = 0;

        long now() {
            return t;
        }

        void advance(double seconds) {
            t += (long) (seconds * 1_000_000_000.0);
        }

        void set(double seconds) {
            t = (long) (seconds * 1_000_000_000.0);
        }
    }

    @Test
    void invalidConfigFailsFast() {
        assertThrows(IllegalArgumentException.class, () -> new TokenBucket(0, 1));
        assertThrows(IllegalArgumentException.class, () -> new TokenBucket(-1, 1));
        assertThrows(IllegalArgumentException.class, () -> new TokenBucket(Double.POSITIVE_INFINITY, 1));
        assertThrows(IllegalArgumentException.class, () -> new TokenBucket(Double.NaN, 1));
        assertThrows(IllegalArgumentException.class, () -> new TokenBucket(1, -1));
        assertThrows(IllegalArgumentException.class, () -> new TokenBucket(1, Double.POSITIVE_INFINITY));
        assertThrows(IllegalArgumentException.class, () -> new TokenBucket(1, Double.NaN));
        assertThrows(
                IllegalArgumentException.class,
                () -> new TokenBucket(1, 1, (LongSupplier) null));
    }

    @Test
    void burstThenEmpty() {
        Clock c = new Clock();
        TokenBucket b = new TokenBucket(5, 2, c::now);
        int allowed = 0;
        for (int i = 0; i < 5; i++) {
            if (b.allow()) {
                allowed++;
            }
        }
        assertEquals(5, allowed);
        assertFalse(b.allow());
    }

    @Test
    void refillOverTime() {
        Clock c = new Clock();
        TokenBucket b = new TokenBucket(5, 2, c::now);
        for (int i = 0; i < 5; i++) {
            b.allow();
        }
        c.advance(1.0); // 补 2 个令牌
        assertTrue(b.allow());
        assertTrue(b.allow());
        assertFalse(b.allow());
    }

    @Test
    void cap() {
        Clock c = new Clock();
        TokenBucket b = new TokenBucket(5, 10, c::now);
        c.advance(100); // 理论补 1000，但封顶 5
        int allowed = 0;
        for (int i = 0; i < 10; i++) {
            if (b.allow()) {
                allowed++;
            }
        }
        assertEquals(5, allowed);
    }

    @Test
    void fractionalRefillAndMultiTokenCost() {
        Clock c = new Clock();
        TokenBucket b = new TokenBucket(5, 1, c::now);
        assertTrue(b.allow(2.5));
        assertEquals(2.5, b.tokens(), 1e-9);
        assertFalse(b.allow(3));
        c.advance(0.5);
        assertTrue(b.allow(3));
        assertEquals(0, b.tokens(), 1e-9);
    }

    @Test
    void invalidCostIsRejectedWithoutChangingState() {
        Clock c = new Clock();
        TokenBucket b = new TokenBucket(5, 1, c::now);
        assertTrue(b.allow(2));
        c.advance(1);
        for (double n : new double[] {0, -1, Double.POSITIVE_INFINITY, Double.NEGATIVE_INFINITY, Double.NaN}) {
            assertFalse(b.allow(n));
            assertEquals(3, b.tokens(), 1e-9);
        }
        assertTrue(b.allow(4)); // 非法请求没有消耗令牌，也没有推进补充基准
    }

    @Test
    void clockRollbackDoesNotMoveRefillBaselineBackwards() {
        Clock c = new Clock();
        TokenBucket b = new TokenBucket(1, 1, c::now);
        assertTrue(b.allow());
        c.set(1);
        assertTrue(b.allow());
        c.set(0.5);
        assertFalse(b.allow());
        c.set(1.5);
        assertTrue(b.allow(0.5));
        assertFalse(b.allow(0.5));
    }

    @Test
    void concurrentSafe() throws InterruptedException {
        TokenBucket b = new TokenBucket(1000, 0); // 无补充，恰好 1000 令牌
        AtomicLong allowed = new AtomicLong();
        int workers = 100;
        CountDownLatch ready = new CountDownLatch(workers);
        CountDownLatch start = new CountDownLatch(1);
        CountDownLatch done = new CountDownLatch(workers);
        Thread[] threads = new Thread[workers];
        for (int i = 0; i < workers; i++) {
            threads[i] = new Thread(() -> {
                ready.countDown();
                try {
                    start.await();
                    for (int j = 0; j < 50; j++) {
                        if (b.allow()) {
                            allowed.incrementAndGet();
                        }
                    }
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                } finally {
                    done.countDown();
                }
            });
            threads[i].start();
        }
        assertTrue(ready.await(5, TimeUnit.SECONDS));
        start.countDown();
        assertTrue(done.await(10, TimeUnit.SECONDS));
        assertEquals(1000, allowed.get());
    }
}
