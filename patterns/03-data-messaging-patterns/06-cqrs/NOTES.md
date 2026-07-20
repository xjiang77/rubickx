# CQRS - Behavior Contract

Canonical Vault note: `06 - Eng - CQRS：写模型与读投影.md`. Pattern identity: `data-messaging.cqrs`.

## Behavior Contract

A command model advances authoritative versioned state while a separate projection applies records to explicit targets and exposes lag without mutating the write model.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Failures and crash points are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit in-memory state and coded exceptions |
| Go | small state values and coded errors |
| Java | Java 21 collections/records and coded exceptions |
| JavaScript | deterministic objects and coded errors |

## Proof Boundary

The lab proves versioned write/projection semantics in memory. It is not a database, broker, durable projector, or CQRS platform.

## Run

```bash
make -C patterns test-pattern PATTERN=data-messaging.cqrs
```
