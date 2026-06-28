import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.util.concurrent.atomic.AtomicLong;
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
    void concurrentSafe() throws InterruptedException {
        TokenBucket b = new TokenBucket(1000, 0); // 无补充，恰好 1000 令牌
        AtomicLong allowed = new AtomicLong();
        Thread[] threads = new Thread[100];
        for (int i = 0; i < 100; i++) {
            threads[i] = new Thread(() -> {
                for (int j = 0; j < 50; j++) {
                    if (b.allow()) {
                        allowed.incrementAndGet();
                    }
                }
            });
            threads[i].start();
        }
        for (Thread t : threads) {
            t.join();
        }
        assertEquals(1000, allowed.get());
    }
}
