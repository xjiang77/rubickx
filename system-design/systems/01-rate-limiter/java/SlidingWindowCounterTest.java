import static org.junit.jupiter.api.Assertions.assertEquals;
import static org.junit.jupiter.api.Assertions.assertFalse;
import static org.junit.jupiter.api.Assertions.assertThrows;
import static org.junit.jupiter.api.Assertions.assertTrue;

import java.time.Duration;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.TimeUnit;
import java.util.concurrent.atomic.AtomicLong;
import java.util.function.LongSupplier;
import org.junit.jupiter.api.Test;

public class SlidingWindowCounterTest {
    private static final Duration WINDOW = Duration.ofSeconds(10);

    static final class Clock {
        long t;

        Clock() {
            this(0);
        }

        Clock(double seconds) {
            set(seconds);
        }

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
        assertThrows(IllegalArgumentException.class, () -> new SlidingWindowCounter(0, WINDOW));
        assertThrows(IllegalArgumentException.class, () -> new SlidingWindowCounter(-1, WINDOW));
        assertThrows(
                IllegalArgumentException.class,
                () -> new SlidingWindowCounter(Double.POSITIVE_INFINITY, WINDOW));
        assertThrows(
                IllegalArgumentException.class,
                () -> new SlidingWindowCounter(Double.NaN, WINDOW));
        assertThrows(IllegalArgumentException.class, () -> new SlidingWindowCounter(1, Duration.ZERO));
        assertThrows(
                IllegalArgumentException.class,
                () -> new SlidingWindowCounter(1, Duration.ofSeconds(-1)));
        assertThrows(IllegalArgumentException.class, () -> new SlidingWindowCounter(1, null));
        assertThrows(
                IllegalArgumentException.class,
                () -> new SlidingWindowCounter(1, WINDOW, (LongSupplier) null));
    }

    @Test
    void currentWindowReachesLimitThenRejects() {
        SlidingWindowCounter counter = new SlidingWindowCounter(5, WINDOW, new Clock()::now);
        assertTrue(counter.allow(2.5));
        assertTrue(counter.allow(2.5));
        assertFalse(counter.allow());
    }

    @Test
    void exactRolloverKeepsFullPreviousWindowWeight() {
        Clock c = new Clock();
        SlidingWindowCounter counter = new SlidingWindowCounter(10, WINDOW, c::now);
        assertTrue(counter.allow(10));
        c.advance(10);
        assertFalse(counter.allow());
    }

    @Test
    void previousWindowHasHalfWeightAtHalfWindow() {
        Clock c = new Clock();
        SlidingWindowCounter counter = new SlidingWindowCounter(10, WINDOW, c::now);
        assertTrue(counter.allow(10));
        c.advance(15);
        assertTrue(counter.allow(5));
        assertFalse(counter.allow(0.001));
    }

    @Test
    void historyIsClearedAfterTwoWindows() {
        Clock c = new Clock();
        SlidingWindowCounter counter = new SlidingWindowCounter(10, WINDOW, c::now);
        assertTrue(counter.allow(10));
        c.advance(20);
        assertTrue(counter.allow(10));
    }

    @Test
    void rejectedRequestIsNotCounted() {
        Clock c = new Clock();
        SlidingWindowCounter counter = new SlidingWindowCounter(10, WINDOW, c::now);
        assertTrue(counter.allow(8));
        assertFalse(counter.allow(3));
        c.advance(15);
        assertTrue(counter.allow(6));
    }

    @Test
    void invalidCostIsRejectedWithoutChangingState() {
        SlidingWindowCounter counter = new SlidingWindowCounter(10, WINDOW, new Clock()::now);
        assertTrue(counter.allow(4));
        for (double n : new double[] {0, -1, Double.POSITIVE_INFINITY, Double.NEGATIVE_INFINITY, Double.NaN}) {
            assertFalse(counter.allow(n));
        }
        assertTrue(counter.allow(6));
    }

    @Test
    void clockRollbackDoesNotMoveWindowBaselineBackwards() {
        Clock c = new Clock(100);
        SlidingWindowCounter counter = new SlidingWindowCounter(10, WINDOW, c::now);
        assertTrue(counter.allow(10));
        c.set(110);
        assertFalse(counter.allow());
        c.set(105);
        assertFalse(counter.allow());
        c.set(115);
        assertTrue(counter.allow(5));
        assertFalse(counter.allow(0.001));
    }

    @Test
    void concurrentRequestsAllowExactlyTheLimit() throws InterruptedException {
        int workers = 50;
        int limit = 20;
        SlidingWindowCounter counter = new SlidingWindowCounter(limit, WINDOW, new Clock()::now);
        AtomicLong allowed = new AtomicLong();
        CountDownLatch ready = new CountDownLatch(workers);
        CountDownLatch start = new CountDownLatch(1);
        CountDownLatch done = new CountDownLatch(workers);

        for (int i = 0; i < workers; i++) {
            new Thread(() -> {
                ready.countDown();
                try {
                    start.await();
                    if (counter.allow()) {
                        allowed.incrementAndGet();
                    }
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                } finally {
                    done.countDown();
                }
            }).start();
        }

        assertTrue(ready.await(5, TimeUnit.SECONDS));
        start.countDown();
        assertTrue(done.await(10, TimeUnit.SECONDS));
        assertEquals(limit, allowed.get());
    }
}
