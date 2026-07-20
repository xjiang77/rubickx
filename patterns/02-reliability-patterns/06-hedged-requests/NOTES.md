# Hedged Requests - Behavior Contract

Canonical Vault note: `06 - Eng - Hedged Requests：尾延迟与冗余请求.md`. Pattern identity: `reliability.hedged-requests`.

## Behavior Contract

An idempotent read starts one hedge only after delay or primary failure, chooses earliest success, and records loser cancellation.

The shared fixture covers nominal, boundary, failure, and lifecycle or non-interference behavior. Time and failures are injected; implementations receive no case ID or expected value.

## Language Mapping

| Language | Expression |
| --- | --- |
| Python | explicit policy objects and fake scheduler state |
| Go | small state structs/interfaces and coded errors |
| Java | Java 21 classes/records with injected events |
| JavaScript | deterministic objects and injected timing data |

## Proof Boundary

The lab proves deterministic scheduling math. It is not a production RPC hedging library.

## Run

```bash
make -C patterns test-pattern PATTERN=reliability.hedged-requests
```
