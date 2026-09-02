# Proxy - Behavior Contract

Canonical Vault note: `04 - Eng - Proxy：访问控制与延迟加载.md`. Pattern identity: `gof.structural.proxy`.

## Behavior Contract

A protection proxy authorizes before lazily creating the real document store and reuses one subject for repeated authorized reads.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocols/composition and coded exceptions |
| Go | small interfaces, explicit composition, coded errors |
| Java | interfaces/classes on Java 21 stable APIs |
| JavaScript | structural contracts and explicit wrappers |

## Proof Boundary

The lab proves ordering and process-local lazy lifecycle. It does not provide distributed authorization, cache invalidation, or remote transport.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.structural.proxy
```
