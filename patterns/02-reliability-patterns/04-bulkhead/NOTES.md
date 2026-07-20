# Bulkhead - Behavior Contract

Canonical Vault note: `04 - Eng - Bulkhead：资源分区与故障 containment.md`. Pattern identity: `reliability.bulkhead`.

## Behavior Contract

Independent resource pools reject only within their own capacity and require explicit release before reuse.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Time and failures are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit policy objects and fake scheduler state |
| Go | small state structs/interfaces and coded errors |
| Java | Java 21 classes/records with injected events |
| JavaScript | deterministic objects and injected timing data |

## Proof Boundary

The lab proves deterministic token accounting. It is not a thread pool or connection pool.

## Run

```bash
make -C patterns test-pattern PATTERN=reliability.bulkhead
```
