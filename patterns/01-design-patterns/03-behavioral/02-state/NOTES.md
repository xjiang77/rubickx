# State - Behavior Contract

Canonical Vault note: `02 - Eng - State：状态驱动行为与转移.md`. Pattern identity: `gof.behavioral.state`.

## Behavior Contract

Each export state owns legal events and returns the next state; invalid transitions fail and retry recovery is explicit.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves deterministic in-memory transitions. It is not a durable workflow engine.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.state
```
