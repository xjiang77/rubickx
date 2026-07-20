# Structured Concurrency & Cancellation - Behavior Contract

Canonical Vault note: `06 - Eng - Structured Concurrency & Cancellation：任务所有权.md`. Pattern identity: `concurrency.structured-concurrency-cancellation`.

## Behavior Contract

A parent scope owns every child to a terminal state, propagates failure or cancellation to pending siblings, and returns only after all children are joined with zero leaks.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Scheduling, completion, and cancellation are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit queue/task state and standard-library synchronization tests |
| Go | small state model plus channels/WaitGroup race tests |
| Java | Java 21 collections plus CountDownLatch/Executor tests |
| JavaScript | deterministic state plus manual Promise completion |

## Proof Boundary

The fixture proves lifecycle accounting and the runtime tests prove deterministic synchronization primitives. This is not a production task-group runtime or remote cancellation guarantee.

## Run

```bash
make -C patterns test-pattern PATTERN=concurrency.structured-concurrency-cancellation
```
