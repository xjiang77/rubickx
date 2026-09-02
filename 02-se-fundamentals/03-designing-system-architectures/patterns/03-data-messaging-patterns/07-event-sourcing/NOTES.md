# Event Sourcing - Behavior Contract

Canonical Vault note: `07 - Eng - Event Sourcing：事件事实与状态重建.md`. Pattern identity: `data-messaging.event-sourcing`.

## Behavior Contract

An aggregate rebuilds state by ordered event replay and appends a validated new event only when the command expected version matches the stream version.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Failures and crash points are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit in-memory state and coded exceptions |
| Go | small state values and coded errors |
| Java | Java 21 collections/records and coded exceptions |
| JavaScript | deterministic objects and coded errors |

## Proof Boundary

The lab proves fold and optimistic append semantics in memory. It is not an event store, snapshot system, schema migration framework, or multi-aggregate transaction.

## Run

```bash
make -C patterns test-pattern PATTERN=data-messaging.event-sourcing
```
