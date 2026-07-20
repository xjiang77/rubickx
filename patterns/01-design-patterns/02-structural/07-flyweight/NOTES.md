# Flyweight - Behavior Contract

Canonical Vault note: `07 - Eng - Flyweight：共享内在状态.md`. Pattern identity: `gof.structural.flyweight`.

## Behavior Contract

Model metadata is shared by intrinsic identity while tenant route state remains external and conflicting definitions fail explicitly.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocols/composition and coded exceptions |
| Go | small interfaces, explicit composition, coded errors |
| Java | interfaces/classes on Java 21 stable APIs |
| JavaScript | structural contracts and explicit wrappers |

## Proof Boundary

The lab proves identity reuse and state separation in one process. It does not establish a distributed cache or quantify memory savings.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.structural.flyweight
```
