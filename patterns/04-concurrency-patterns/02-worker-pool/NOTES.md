# Worker Pool - Behavior Contract

Canonical Vault note: `02 - Eng - Worker Pool：受控并行与收口.md`. Pattern identity: `concurrency.worker-pool`.

## Behavior Contract

A fixed-size pool assigns every admitted job to the earliest available worker, records one terminal execution, and returns only after all jobs are joined.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Scheduling, completion, and cancellation are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit queue/task state and standard-library synchronization tests |
| Go | small state model plus channels/WaitGroup race tests |
| Java | Java 21 collections plus CountDownLatch/Executor tests |
| JavaScript | deterministic state plus manual Promise completion |

## Proof Boundary

The lab proves deterministic scheduling and join accounting. It is not an OS thread pool, fairness proof, or throughput benchmark.

## Run

```bash
make -C patterns test-pattern PATTERN=concurrency.worker-pool
```
