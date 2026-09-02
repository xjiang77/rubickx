# LAB-NETSEC-06 — Server-side request and API abuse controls

This is an adjacent exercise and does not claim canonical owner coverage for the current Network & Security batch.

## Question

How does a service prevent user-controlled data from becoming an unbounded network destination, object reference or parser ambiguity?

## Mechanisms

- URL admission requires `http/https` and an explicit loopback allowlist.
- Object authorization compares authenticated subject with resource owner for every object.
- Conflicting message lengths are treated as HTTP ambiguity and rejected.
- Redirect and policy-refresh semantics remain part of the destination decision.

## Invariants

- Validate every redirect hop and the final resolved destination.
- Authentication does not imply authorization for an object identifier.
- Canonicalization occurs once at a defined boundary; downstream consumers receive the canonical form.
- Dependency failure never expands the egress allowlist.

## Outcomes

Normal requires every layer to allow. Any layer can reject. Degraded mode is bounded to a known-safe read set, and recovery re-evaluates the original request under a new revision.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-06 -out evidence/lab06.json
```
