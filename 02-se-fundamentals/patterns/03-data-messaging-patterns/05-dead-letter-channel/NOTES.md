# Dead Letter Channel - Behavior Contract

Canonical Vault note: `05 - Eng - Dead Letter Channel：隔离、诊断与重放.md`. Pattern identity: `data-messaging.dead-letter-channel`.

## Behavior Contract

A bounded consumer isolates permanent or exhausted messages with reason and attempts, then replays only explicitly selected dead records while preserving failure evidence.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Failures and crash points are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit in-memory state and coded exceptions |
| Go | small state values and coded errors |
| Java | Java 21 collections/records and coded exceptions |
| JavaScript | deterministic objects and coded errors |

## Proof Boundary

The lab proves deterministic classification, isolation, and replay state. It is not a broker DLQ, durable store, or safe production replay system.

## Run

```bash
make -C patterns test-pattern PATTERN=data-messaging.dead-letter-channel
```
