# Load Shedding - Behavior Contract

Canonical Vault note: `05 - Eng - Load Shedding：过载拒绝与 Goodput.md`. Pattern identity: `reliability.load-shedding`.

## Behavior Contract

A deterministic admission controller accepts highest-priority requests within an explicit capacity and returns the rest as shed receipts.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Time and failures are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit policy objects and fake scheduler state |
| Go | small state structs/interfaces and coded errors |
| Java | Java 21 classes/records with injected events |
| JavaScript | deterministic objects and injected timing data |

## Proof Boundary

The lab proves priority admission semantics. It is not an autoscaler or production overload detector.

## Run

```bash
make -C patterns test-pattern PATTERN=reliability.load-shedding
```
