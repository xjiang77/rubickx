# LAB-NETSEC-10 — Enterprise access evidence-closure capstone

## Question

Can an architect explain one access decision from client intent through network and identity layers to enforcement acknowledgement and recovery?

## End-to-end chain

`client -> resolver -> transport/TLS -> gateway -> IdP/RP session -> PDP -> PEP -> resource -> telemetry -> signal -> action -> ack -> evidence`

The resolver, TLS, gateway, and IdP/RP segments in this chain are integration context, not executable claims of LAB10. Its executable slice is endpoint telemetry collection → typed posture-signal derivation → PDP decision → three-target action/ACK/NACK/unknown → rollback → effective-state probe/readback.

## State transition

1. Three PEP targets begin at effective `allow-v1`.
2. Policy v2 produces `ack` at PEP-A, `nack` at PEP-B, and `unknown` at offline PEP-C.
3. The controller rolls reachable targets back to v1 while preserving PEP-C as unknown.
4. PEP-C recovers and new policy v3 receives three acknowledgements with effective `deny-v3`.

## Invariants

- Desired, delivered, effective and observed states never collapse into one boolean.
- A negative acknowledgement is useful evidence, not a reason to overwrite history.
- Deny and recovery use the same correlation identifiers and revision lineage.
- No single log line proves the entire chain; evidence is joined across owners.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-10 -out evidence/lab10.json
```

Use `make verify` to execute this capstone with every preceding lab.

## Knowledge mapping

Primary executable coverage: Distributed Enforcement Lifecycle, Network Observability, and Detection to Recovery. Integration executable coverage: Continuous Trust. Integration model coverage: End-to-End Network Path and Enterprise Access Network.
