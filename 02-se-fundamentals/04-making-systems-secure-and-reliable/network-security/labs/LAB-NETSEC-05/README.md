# LAB-NETSEC-05 — Browser origin, CORS, cookie and CSRF boundaries

This is an adjacent exercise and does not claim canonical owner coverage for the current Network & Security batch.

## Question

Which controls restrict JavaScript reads, ambient credential sending and state-changing requests, and why are they not interchangeable?

## Topology

`origin A UI -> browser policy -> origin B API` plus same-origin `/csrf` mutation.

## Mechanisms

- Origin B emits an exact `Access-Control-Allow-Origin` only for the intended origin and route.
- Missing CORS headers make the real browser reject script access.
- The server independently requires a transaction-bound CSRF token.
- CSP restricts connection targets and script execution on the lab page.

## Invariants

- CORS is not authorization and does not prevent non-browser requests.
- SameSite reduces some ambient-cookie cases but is not a complete CSRF policy.
- CSP is damage containment, not input validation.
- Cross-origin dependency failure is not an authentication success.

## Browser run

```bash
go run ./cmd/netsec-browser
```

Open `/boundary`, run allowed/denied CORS and valid/missing CSRF actions, then inspect `/evidence`.
