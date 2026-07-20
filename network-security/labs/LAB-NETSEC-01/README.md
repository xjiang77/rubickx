# LAB-NETSEC-01 — Resolver to TCP to HTTP path

## Question

When an application says “the network is down”, which stage actually failed and what evidence distinguishes it?

## Topology

`in-memory resolver/cache -> candidate address -> route/MTU decision -> 127.0.0.1 TCP listener -> HTTP framing -> response evidence`

## Mechanisms

- Resolve `service.lab` from a deterministic in-memory resolver/cache.
- Select a route with `net/netip` longest-prefix matching; independently reject no-route and allowed-route-over-PMTU cases before dialing.
- Dial loopback candidates on ephemeral ports, preserving a refused first candidate before fallback.
- Send and parse a real HTTP/1.1 exchange over the raw connection.
- Keep resolver, connect, protocol, application and recovery decisions separate.

## Invariants

- No dial occurs after a resolver failure.
- A fallback attempt records the failed candidate instead of rewriting history.
- A second lookup must be an asserted cache hit; recovery invalidates that entry and performs a fresh lookup.
- Application status never proves DNS, TCP or TLS independently; each stage has its own evidence.

## Outcomes

`normal` completes the path; `reject` denies an unowned route/oversized payload; `dependency_failure` injects an uncached resolver outage and stops before dial; `degraded` executes candidate fallback; `recovery` performs a fresh lookup and trace.

## Knowledge mapping

Primary executable coverage: End-to-End Network Path and DNS & Service Discovery. Primary model coverage: IP Connectivity. Integration executable coverage: Network Observability.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-01 -out evidence/lab01.json
```

## Non-guarantees

This lab does not reproduce DHCP, ARP/NDP, NAT, Internet routing or real packet loss. Those stages remain explicit model boundaries rather than hidden assumptions.
