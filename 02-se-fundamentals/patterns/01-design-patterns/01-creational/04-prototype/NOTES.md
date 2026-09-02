# Prototype - Behavior Contract

Canonical Vault note: `04 - Eng - Prototype：配置克隆与隔离.md`. Pattern identity: `gof.creational.prototype`.

## Behavior Contract

Cloning a validated routing policy creates a new identity and an independent fallback list before applying overrides.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; fixture IDs and expected values are never passed to production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocol/ABC, dataclass, or controlled module state |
| Go | small interfaces and explicit coded errors |
| Java | interfaces/records and Java 21 stable APIs |
| JavaScript | structural contracts and explicit runtime errors |

## Proof Boundary

The lab proves copy semantics for value state. It does not clone live resources, database identities, locks, or network clients.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.creational.prototype
```
