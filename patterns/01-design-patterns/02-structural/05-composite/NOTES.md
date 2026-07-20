# Composite - Behavior Contract

Canonical Vault note: `05 - Eng - Composite：树形结构与一致操作.md`. Pattern identity: `gof.structural.composite`.

## Behavior Contract

Leaf predicates and All/Any composites share one evaluation contract with deterministic short-circuit trace.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocols/composition and coded exceptions |
| Go | small interfaces, explicit composition, coded errors |
| Java | interfaces/classes on Java 21 stable APIs |
| JavaScript | structural contracts and explicit wrappers |

## Proof Boundary

The lab proves recursive composition for an acyclic in-memory tree. It does not parse a language or secure an authorization system.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.structural.composite
```
