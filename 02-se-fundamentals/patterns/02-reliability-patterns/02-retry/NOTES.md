# Retry - Behavior Contract

Canonical Vault note: `02 - Eng - Retry：重试预算与退避.md`. Pattern identity: `reliability.retry`.

## Behavior Contract

Only transient outcomes consume a bounded retry schedule; permanent errors stop immediately and tests record delays without sleeping.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Time and failures are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit policy objects and fake scheduler state |
| Go | small state structs/interfaces and coded errors |
| Java | Java 21 classes/records with injected events |
| JavaScript | deterministic objects and injected timing data |

## Proof Boundary

The lab proves classification and deterministic attempt budgeting. It is not a production retry library.

## Run

```bash
make -C patterns test-pattern PATTERN=reliability.retry
```
