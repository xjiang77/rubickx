# LAB-NETSEC-09 — Continuous access and signal ordering

## Question

How can a PEP change an already-established session when device, identity or risk state changes?

## Mechanisms

- PDP input includes issuer/subject, device, resource, action, RP session and posture freshness.
- A typed SSF/CAEP-style envelope carries `jti`, event type, subject, device, issued-at, expiry, and posture state.
- The receiver deduplicates `jti`, rejects expired input, and never treats a local counter as a protocol sequence.
- Injected event loss produces stale posture and a bounded decision; an authoritative snapshot reconciles drift.
- Stale posture maps to explicit step-up or bounded low-risk degradation.

## Invariants

- Signal receipt is not enforcement; action and acknowledgement are distinct evidence.
- Duplicate delivery is idempotent; ordering is not inferred from a local revision.
- Missing signals do not silently mean compliant.
- Recovery never replays an older allow over a newer revoke.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-09 -out evidence/lab09.json
```

## Extend

Introduce receiver restart and signed delivery while preserving `jti`, expiry, freshness, and authoritative reconciliation semantics.

## Knowledge mapping

Primary executable coverage: Continuous Trust. Primary model coverage: Enterprise Access Network.
