# Memento - Behavior Contract

Canonical Vault note: `09 - Eng - Memento：快照与封装恢复.md`. Pattern identity: `gof.behavioral.memento`.

## Behavior Contract

Route-config mementos deep-copy originator state, remain opaque to the caretaker, and restore by explicit index.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves in-memory snapshot isolation. It is not an event store, backup, or external-side-effect rollback mechanism.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.memento
```
