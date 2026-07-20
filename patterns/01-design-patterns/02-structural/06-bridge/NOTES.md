# Bridge - Behavior Contract

Canonical Vault note: `06 - Eng - Bridge：抽象与实现独立演化.md`. Pattern identity: `gof.structural.bridge`.

## Behavior Contract

Alert semantics and delivery channel formatting evolve independently and compose without combination-specific subclasses.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocols/composition and coded exceptions |
| Go | small interfaces, explicit composition, coded errors |
| Java | interfaces/classes on Java 21 stable APIs |
| JavaScript | structural contracts and explicit wrappers |

## Proof Boundary

The lab proves two independent in-memory variation axes. It does not model channel retries, credentials, or delivery receipts.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.structural.bridge
```
