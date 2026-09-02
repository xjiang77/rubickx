# Observer - Behavior Contract

Canonical Vault note: `06 - Eng - Observer：状态通知与订阅生命周期.md`. Pattern identity: `gof.behavioral.observer`.

## Behavior Contract

A deployment subject emits per-observer receipts, isolates one observer failure, and applies unsubscription only to later events.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves synchronous in-memory subscription lifecycle. It does not guarantee durable or exactly-once delivery.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.observer
```
