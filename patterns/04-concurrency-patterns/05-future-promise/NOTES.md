# Future & Promise - Behavior Contract

Canonical Vault note: `05 - Eng - Future & Promise：结果句柄与完成语义.md`. Pattern identity: `concurrency.future-promise`.

## Behavior Contract

A future exposes one pending-to-terminal transition, preserves value, failure, or cancellation, and allows repeated non-consuming observation.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Scheduling, completion, and cancellation are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit queue/task state and standard-library synchronization tests |
| Go | small state model plus channels/WaitGroup race tests |
| Java | Java 21 collections plus CountDownLatch/Executor tests |
| JavaScript | deterministic state plus manual Promise completion |

## Proof Boundary

The lab proves single-completion state semantics. It is not an event-loop implementation, remote cancellation protocol, or async performance benchmark.

## Run

```bash
make -C patterns test-pattern PATTERN=concurrency.future-promise
```
