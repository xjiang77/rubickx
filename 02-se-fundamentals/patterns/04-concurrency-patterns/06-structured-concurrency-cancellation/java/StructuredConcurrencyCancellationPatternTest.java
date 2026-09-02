import static org.junit.jupiter.api.Assertions.assertEquals;

import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.CountDownLatch;
import org.junit.jupiter.api.Test;

class StructuredConcurrencyCancellationPatternTest {
    @Test void sharedContract() {
        ContractHarness.run("04-concurrency-patterns/06-structured-concurrency-cancellation/fixtures/contract.json", StructuredConcurrencyCancellationPattern::evaluate);
    }

    @Test void failureCancelsSiblingAndAllThreadsJoin() throws InterruptedException {
        CountDownLatch start = new CountDownLatch(1);
        CountDownLatch cancelled = new CountDownLatch(1);
        CountDownLatch done = new CountDownLatch(2);
        List<String> events = Collections.synchronizedList(new ArrayList<>());

        Thread failing = new Thread(() -> {
            await(start);
            events.add("failed");
            cancelled.countDown();
            done.countDown();
        });
        Thread sibling = new Thread(() -> {
            await(start);
            await(cancelled);
            events.add("cancelled");
            done.countDown();
        });

        failing.start();
        sibling.start();
        start.countDown();
        done.await();
        failing.join();
        sibling.join();

        assertEquals(List.of("failed", "cancelled"), events);
    }

    private static void await(CountDownLatch latch) {
        try {
            latch.await();
        } catch (InterruptedException error) {
            Thread.currentThread().interrupt();
            throw new AssertionError(error);
        }
    }
}
