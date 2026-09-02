# Template Method - Behavior Contract

Canonical Vault note: `03 - Eng - Template Method：固定骨架与可变步骤.md`. Pattern identity: `gof.behavioral.template-method`.

## Behavior Contract

All ingestion jobs execute validate, format-specific transform, and persist in one fixed order.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves skeleton ordering with in-memory records. It does not implement files, schemas, or transactional persistence.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.template-method
```
