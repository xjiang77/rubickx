# Strategy - Behavior Contract

Canonical Vault note: `01 - Eng - Strategy：可替换决策算法.md`. Pattern identity: `gof.behavioral.strategy`.

## Behavior Contract

Cost and latency strategies select from the same healthy-candidate contract with deterministic tie-breaking.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves interchangeable in-memory selection algorithms. It does not implement health collection, weighted routing, or load balancing.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.strategy
```
