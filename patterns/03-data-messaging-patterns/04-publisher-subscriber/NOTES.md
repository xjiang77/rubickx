# Publisher-Subscriber - Behavior Contract

Canonical Vault note: `04 - Eng - Publisher-Subscriber：一对多事件分发.md`. Pattern identity: `data-messaging.publisher-subscriber`.

## Behavior Contract

A publisher fans immutable topic events to every matching subscriber and records independent delivery outcomes without short-circuiting on one failure.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Failures and crash points are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit in-memory state and coded exceptions |
| Go | small state values and coded errors |
| Java | Java 21 collections/records and coded exceptions |
| JavaScript | deterministic objects and coded errors |

## Proof Boundary

The lab proves deterministic topic matching and failure isolation. It is not a durable broker, consumer group, ordering, or replay implementation.

## Run

```bash
make -C patterns test-pattern PATTERN=data-messaging.publisher-subscriber
```
