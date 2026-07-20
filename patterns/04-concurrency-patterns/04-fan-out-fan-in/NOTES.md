# Fan-out/Fan-in - Behavior Contract

Canonical Vault note: `04 - Eng - Fan-out & Fan-in：并行分发与结果合并.md`. Pattern identity: `concurrency.fan-out-fan-in`.

## Behavior Contract

A parent fans out identity-bearing child tasks and fans in injected completion receipts under an explicit all or first-success policy, preserving failures and cancellation evidence.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Scheduling, completion, and cancellation are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit queue/task state and standard-library synchronization tests |
| Go | small state model plus channels/WaitGroup race tests |
| Java | Java 21 collections plus CountDownLatch/Executor tests |
| JavaScript | deterministic state plus manual Promise completion |

## Proof Boundary

The lab proves deterministic merge and cancellation accounting. It is not a scheduler, remote cancellation guarantee, or parallel speed benchmark.

## Run

```bash
make -C patterns test-pattern PATTERN=concurrency.fan-out-fan-in
```
