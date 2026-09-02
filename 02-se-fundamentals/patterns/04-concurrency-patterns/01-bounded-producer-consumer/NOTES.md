# Bounded Producer-Consumer - Behavior Contract

Canonical Vault note: `01 - Eng - Bounded Producer-Consumer：容量与背压.md`. Pattern identity: `concurrency.bounded-producer-consumer`.

## Behavior Contract

A bounded FIFO accepts items only within explicit capacity, reports backpressure without overwriting, and closes admission while allowing queued work to drain.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Scheduling, completion, and cancellation are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit queue/task state and standard-library synchronization tests |
| Go | small state model plus channels/WaitGroup race tests |
| Java | Java 21 collections plus CountDownLatch/Executor tests |
| JavaScript | deterministic state plus manual Promise completion |

## Proof Boundary

The lab proves non-blocking bounded-buffer state transitions. It is not a lock-free queue, durable broker, or runtime throughput benchmark.

## Run

```bash
make -C patterns test-pattern PATTERN=concurrency.bounded-producer-consumer
```
