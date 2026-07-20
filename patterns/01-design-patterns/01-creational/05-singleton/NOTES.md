# Singleton - Behavior Contract

Canonical Vault note: `05 - Eng - Singleton：进程内唯一实例.md`. Pattern identity: `gof.creational.singleton`.

## Behavior Contract

A process-local provider catalog returns one identity, shares registrations, accepts idempotent duplicates, and rejects conflicting duplicates.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; fixture IDs and expected values are never passed to production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocol/ABC, dataclass, or controlled module state |
| Go | small interfaces and explicit coded errors |
| Java | interfaces/records and Java 21 stable APIs |
| JavaScript | structural contracts and explicit runtime errors |

## Proof Boundary

The lab proves process-local identity and synchronized registry semantics. It does not provide cross-process leadership, durable state, or service discovery.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.creational.singleton
```
