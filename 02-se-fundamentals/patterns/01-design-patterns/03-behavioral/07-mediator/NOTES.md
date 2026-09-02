# Mediator - Behavior Contract

Canonical Vault note: `07 - Eng - Mediator：集中协作协议.md`. Pattern identity: `gof.behavioral.mediator`.

## Behavior Contract

Customer, agent, and bot communicate only through one support mediator with explicit sender-to-recipient rules.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves in-memory collaboration routing. It does not provide messaging durability or business workflow persistence.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.mediator
```
