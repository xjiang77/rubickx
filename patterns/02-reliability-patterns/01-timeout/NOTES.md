# Timeout - Behavior Contract

Canonical Vault note: `01 - Eng - Timeout：截止时间与未知结果.md`. Pattern identity: `reliability.timeout`.

## Behavior Contract

An absolute deadline is propagated across sequential operations; side-effect timeout yields unknown outcome without real time.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Time and failures are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit policy objects and fake scheduler state |
| Go | small state structs/interfaces and coded errors |
| Java | Java 21 classes/records with injected events |
| JavaScript | deterministic objects and injected timing data |

## Proof Boundary

The lab proves deterministic budget accounting. It does not cancel real work or infer a remote side effect did not happen.

## Run

```bash
make -C patterns test-pattern PATTERN=reliability.timeout
```
