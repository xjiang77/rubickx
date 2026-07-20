# LAB-NETSEC-04 — Proxy routing and effective configuration

## Question

How can a control plane prove that a route or policy is effective at the data plane rather than merely declared?

## Topology

`client -> loopback proxy -> health-aware selection -> backend A/B -> response revision`

## Mechanisms

- The proxy performs real HTTP forwarding to two loopback backends and returns backend/config evidence.
- Route admission happens before forwarding.
- Reload is a state transition: desired -> delivered -> validated -> effective -> acknowledged.
- Invalid delivered configuration keeps effective v1 and emits `reload_rejected`.
- Recovery atomically activates v2 and changes selection from backend A to backend B.

## Invariants

- Unknown routes never become arbitrary upstream URLs.
- Health failure does not silently select an unverified backend.
- A successful distribution event is not proof of effective configuration.
- Recovery is a new revision, not mutation of old evidence.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-04 -out evidence/lab04.json
```

## Extend

Add PAC/system proxy, L4 tunnel and service-mesh variants while preserving the same state/evidence contract.

## Knowledge mapping

Primary executable coverage: Traffic Intermediation. Integration model coverage: Enterprise Access Network. Integration executable coverage: Distributed Enforcement Lifecycle.
