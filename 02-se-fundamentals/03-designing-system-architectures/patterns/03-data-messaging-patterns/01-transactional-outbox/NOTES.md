# Transactional Outbox - Behavior Contract

Canonical Vault note: `01 - Eng - Transactional Outbox：提交与发布.md`. Pattern identity: `data-messaging.transactional-outbox`.

## Behavior Contract

A local commit creates aggregate state and one pending outbox record atomically; relay crash windows preserve recoverability while allowing duplicate delivery after publish-before-mark.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Failures and crash points are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit in-memory state and coded exceptions |
| Go | small state values and coded errors |
| Java | Java 21 collections/records and coded exceptions |
| JavaScript | deterministic objects and coded errors |

## Proof Boundary

The lab proves local commit and deterministic relay state transitions. It is not a database transaction, broker, or exactly-once delivery implementation.

## Run

```bash
make -C patterns test-pattern PATTERN=data-messaging.transactional-outbox
```
