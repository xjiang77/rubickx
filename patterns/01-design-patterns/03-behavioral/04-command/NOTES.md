# Command - Behavior Contract

Canonical Vault note: `04 - Eng - Command：请求对象化与可逆执行.md`. Pattern identity: `gof.behavioral.command`.

## Behavior Contract

Set-route commands capture prior receiver state, execute through an invoker, and undo in exact LIFO order.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves in-memory reversible mutation. It does not make external side effects reversible or durable.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.command
```
