# Iterator - Behavior Contract

Canonical Vault note: `08 - Eng - Iterator：遍历协议与游标边界.md`. Pattern identity: `gof.behavioral.iterator`.

## Behavior Contract

A paginated iterator fetches lazily, distinguishes failure from exhaustion, and never prefetches after the client take limit.

The fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Implementations receive only `input`; case IDs and expected values never enter production code.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit objects/protocols and coded errors |
| Go | small interfaces and explicit state |
| Java | interfaces/records/classes on Java 21 |
| JavaScript | structural objects and explicit lifecycle |

## Proof Boundary

The lab proves deterministic cursor behavior over in-memory pages. It does not implement remote pagination or resource cleanup.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.behavioral.iterator
```
