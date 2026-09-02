# Abstract Factory - Behavior Contract

Canonical Vault note: `02 - Eng - Abstract Factory：一致对象族.md`. Pattern identity: `gof.creational.abstract-factory`.

## Behavior Contract

One selected factory creates a compatible queue and object-store family. Products from different providers are never assembled implicitly.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; fixture IDs and expected values are never passed to production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocol/ABC, dataclass, or controlled module state |
| Go | small interfaces and explicit coded errors |
| Java | interfaces/records and Java 21 stable APIs |
| JavaScript | structural contracts and explicit runtime errors |

## Proof Boundary

The lab proves family selection and compatibility descriptors only. It does not authenticate with cloud providers, provision resources, or compare vendors.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.creational.abstract-factory
```
