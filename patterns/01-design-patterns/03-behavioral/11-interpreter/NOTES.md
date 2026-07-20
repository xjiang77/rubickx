# Interpreter - Behavior Contract

Canonical Vault note: `11 - Eng - Interpreter：小型语言与语法树.md`. Pattern identity: `gof.behavioral.interpreter`.

## Behavior Contract

A tiny quota grammar parses whitelisted comparisons joined by `and`, evaluates against explicit context, and short-circuits deterministically.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves a deliberately small parser/interpreter. It is not a general expression language or sandbox.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.interpreter
```
