# Factory Method - Behavior Contract

Canonical Vault note: `01 - Eng - Factory Method：延迟具体创建.md`. Pattern identity: `gof.creational.factory-method`.

## Behavior Contract

A stable export workflow obtains a formatter through an overridable creation point. CSV and JSON creators return different products; unsupported formats fail before formatting.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; fixture IDs and expected values are never passed to production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | protocol/ABC, dataclass, or controlled module state |
| Go | small interfaces and explicit coded errors |
| Java | interfaces/records and Java 21 stable APIs |
| JavaScript | structural contracts and explicit runtime errors |

## Proof Boundary

The lab proves product selection and stable workflow behavior for in-memory records. It does not benchmark serialization, write files, or prescribe inheritance over first-class factory functions.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.creational.factory-method
```
