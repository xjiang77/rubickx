# Circuit Breaker - Behavior Contract

Canonical Vault note: `03 - Eng - Circuit Breaker：失败隔离与恢复探测.md`. Pattern identity: `reliability.circuit-breaker`.

## Behavior Contract

A deterministic closed/open/half-open breaker rejects during cooldown and requires a successful probe before closing.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Time and failures are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit policy objects and fake scheduler state |
| Go | small state structs/interfaces and coded errors |
| Java | Java 21 classes/records with injected events |
| JavaScript | deterministic objects and injected timing data |

## Proof Boundary

The lab proves a simplified state machine. It is not a production rolling-window breaker.

## Run

```bash
make -C patterns test-pattern PATTERN=reliability.circuit-breaker
```
