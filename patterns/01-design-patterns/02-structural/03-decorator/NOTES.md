# Decorator - Behavior Contract

Canonical Vault note: `03 - Eng - Decorator：可组合横切行为.md`. Pattern identity: `gof.structural.decorator`.

## Behavior Contract

Auth and trace decorators preserve one send contract, compose in declared order, and never leak request state.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocols/composition and coded exceptions |
| Go | small interfaces, explicit composition, coded errors |
| Java | interfaces/classes on Java 21 stable APIs |
| JavaScript | structural contracts and explicit wrappers |

## Proof Boundary

The lab proves deterministic wrapper composition around an in-memory HTTP client. It does not measure transport overhead or implement retries.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.structural.decorator
```
