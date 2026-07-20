# LAB-NETSEC-02 — Transport framing, deadlines and backpressure

## Question

How do byte-stream semantics, timeouts and retries turn into application correctness or duplicate effects?

## Mechanisms

- `net.Pipe` fragments one frame and coalesces a second frame into the same write, proving that TCP-style reads are not messages.
- A bounded newline framer reconstructs both messages and rejects an oversized frame.
- A loopback server commits an idempotency-ledger side effect, delays its response past the client deadline, and leaves the client with an unknown remote-effect state.
- A capacity-one queue executes load shedding while preserving admitted work.
- The same idempotency key reconciles the committed result after timeout and asserts `applications=1`.

## Invariants

- Parsers cap frame size and reject incomplete frames.
- Timeout is not proof that the peer did nothing.
- Only the layer with operation semantics decides whether retry is safe.
- Reconnect starts a new framing state and retry budget.

## Outcomes

The five outcomes cover fragment/coalesce framing, oversized rejection, deadline ambiguity, bounded shedding and idempotent reconciliation.

## Knowledge mapping

Primary executable coverage: Transport Semantics. QUIC and production protocol stacks remain outside this stdlib lab.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-02 -out evidence/lab02.json
```

## Extend

Compare the same invariant in WebSocket frames, HTTP/2 streams and gRPC deadlines without assuming their recovery semantics are identical.
