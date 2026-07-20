# Saga - Behavior Contract

Canonical Vault note: `03 - Eng - Saga：跨边界补偿与恢复.md`. Pattern identity: `data-messaging.saga`.

## Behavior Contract

A deterministic orchestrator executes local steps, compensates completed steps in reverse after failure, and surfaces compensation failure as recovery_required.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Failures and crash points are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit in-memory state and coded exceptions |
| Go | small state values and coded errors |
| Java | Java 21 collections/records and coded exceptions |
| JavaScript | deterministic objects and coded errors |

## Proof Boundary

The lab proves orchestration state transitions and compensation order. It is not a durable workflow engine or distributed transaction implementation.

## Run

```bash
make -C patterns test-pattern PATTERN=data-messaging.saga
```
