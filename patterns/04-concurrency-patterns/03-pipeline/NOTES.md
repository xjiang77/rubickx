# Pipeline - Behavior Contract

Canonical Vault note: `03 - Eng - Pipeline：分阶段处理与取消.md`. Pattern identity: `concurrency.pipeline`.

## Behavior Contract

Items traverse ordered stages; a stage failure or downstream stop cancels remaining input and returns explicit terminal and cleanup evidence.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Scheduling, completion, and cancellation are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit queue/task state and standard-library synchronization tests |
| Go | small state model plus channels/WaitGroup race tests |
| Java | Java 21 collections plus CountDownLatch/Executor tests |
| JavaScript | deterministic state plus manual Promise completion |

## Proof Boundary

The lab proves stage order and cancellation accounting. It is not a streaming runtime, channel implementation, or throughput benchmark.

## Run

```bash
make -C patterns test-pattern PATTERN=concurrency.pipeline
```
