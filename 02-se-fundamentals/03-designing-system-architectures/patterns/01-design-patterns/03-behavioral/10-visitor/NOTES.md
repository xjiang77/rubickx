# Visitor - Behavior Contract

Canonical Vault note: `10 - Eng - Visitor：稳定结构上的新操作.md`. Pattern identity: `gof.behavioral.visitor`.

## Behavior Contract

Stable Equals/And nodes accept evaluation and field-collection visitors without embedding those operations in each node.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves double-dispatch-style operation separation over a small fixed AST. It does not parse or authorize production policies.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.visitor
```
