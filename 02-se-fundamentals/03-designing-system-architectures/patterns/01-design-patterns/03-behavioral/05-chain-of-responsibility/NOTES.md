# Chain of Responsibility - Behavior Contract

Canonical Vault note: `05 - Eng - Chain of Responsibility：有序处理链.md`. Pattern identity: `gof.behavioral.chain-of-responsibility`.

## Behavior Contract

Auth, quota, and terminal handlers form one ordered chain; rejection short-circuits and per-request trace never leaks.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves deterministic in-memory handler ordering. It is not an authorization or quota system.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.chain-of-responsibility
```
