# Facade - Behavior Contract

Canonical Vault note: `02 - Eng - Facade：子系统入口与编排边界.md`. Pattern identity: `gof.structural.facade`.

## Behavior Contract

A release facade coordinates validation, deployment, and health verification while preserving dry-run and failure semantics.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocols/composition and coded exceptions |
| Go | small interfaces, explicit composition, coded errors |
| Java | interfaces/classes on Java 21 stable APIs |
| JavaScript | structural contracts and explicit wrappers |

## Proof Boundary

The lab proves orchestration order and explicit failure propagation for in-memory subsystems. It does not implement deployment rollback or distributed transactions.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.structural.facade
```
