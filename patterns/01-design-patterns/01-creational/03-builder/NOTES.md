# Builder - Behavior Contract

Canonical Vault note: `03 - Eng - Builder：分步构建与验证.md`. Pattern identity: `gof.creational.builder`.

## Behavior Contract

A mutable builder collects fields, validates all invariants at build time, returns an independent immutable request, and resets before the next build.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; fixture IDs and expected values are never passed to production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocol/ABC, dataclass, or controlled module state |
| Go | small interfaces and explicit coded errors |
| Java | interfaces/records and Java 21 stable APIs |
| JavaScript | structural contracts and explicit runtime errors |

## Proof Boundary

The lab proves construction and reset semantics for an in-memory ChatRequest. It does not replace domain validation or provider contract tests.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.creational.builder
```
