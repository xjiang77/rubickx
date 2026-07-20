# Idempotent Consumer & Inbox - Behavior Contract

Canonical Vault note: `02 - Eng - Idempotent Consumer & Inbox：重复投递与效果去重.md`. Pattern identity: `data-messaging.idempotent-consumer-inbox`.

## Behavior Contract

A durable-style inbox applies one effect per stable message ID, recognizes same-payload duplicates, and rejects conflicting payload reuse.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Failures and crash points are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit in-memory state and coded exceptions |
| Go | small state values and coded errors |
| Java | Java 21 collections/records and coded exceptions |
| JavaScript | deterministic objects and coded errors |

## Proof Boundary

The lab proves deterministic receipt/effect semantics in memory. It does not provide a durable database transaction, retention policy, or broker integration.

## Run

```bash
make -C patterns test-pattern PATTERN=data-messaging.idempotent-consumer-inbox
```
