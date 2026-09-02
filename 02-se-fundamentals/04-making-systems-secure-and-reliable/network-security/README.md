# Network Security Labs

Executable, defense-first labs for understanding network paths, browser and API trust boundaries, SSO/federation lifecycle, continuous access, and evidence closure.

## Safety boundary

- Every listener binds to `127.0.0.1` with an ephemeral port.
- Labs never scan or contact external targets.
- Labs do not use raw sockets or modify system DNS, routes, firewall, proxy, or certificate stores.
- Vulnerable or rejected behavior is explicit, local, and used only to prove defensive invariants.
- OIDC, SAML, TLS, proxy, and policy examples are educational models, not production implementations or conformance claims.

## Run

```bash
make verify
go run ./cmd/netsec-lab -lab LAB-NETSEC-07 -out evidence/lab07.json
go run ./cmd/netsec-browser
make refresh-evidence
```

`make verify` executes format checks, `go vet`, unit tests, race tests, every outcome of all ten labs, and the non-interactive browser fixture. Both reports use temporary files, so verification leaves the worktree unchanged. `make refresh-evidence` is the only target that intentionally refreshes `evidence/all.json` and `evidence/browser.json`.

`catalog.json` schema v2 declares the exact 12 Knowledge owners. LAB01–04 and LAB07–10 are `core`; LAB05/06 are adjacent exercises and do not claim owner coverage. Each mapping distinguishes `primary` from `integration` and `executable` from `model` coverage.

## Evidence model

Each lab must emit exactly these outcome classes:

- `normal`
- `reject`
- `dependency_failure`
- `degraded`
- `recovery`

Events keep desired and effective state separate and preserve correlation, policy revision, decision, action, and acknowledgement fields. Scenario evidence also records `scenario_id`, `stage`, `component`, `evidence_kind`, `observed_state`, `action_id`, and `precondition_revision`. Core outcomes are emitted only after the corresponding injected scenario assertion succeeds. `n/a` is used when a field is structurally required but semantically absent from a lower-layer lab.

## Real browser surface

`cmd/netsec-browser` starts a loopback-only browser application with:

- protected application and minimal OIDC-style Authorization Code + PKCE flow;
- `state`, `nonce`, issuer, audience, expiry, redirect URI and key-id negative checks;
- RP session, logout, deprovision and posture degradation;
- two-origin CORS and CSRF exercises;
- JSON evidence readback.

Browser evidence is explicitly attributed: OIDC to LAB07, logout/deprovision to LAB08, posture to LAB09, and CORS/CSRF to LAB05.
Use `go run ./cmd/netsec-browser -self-test -evidence /tmp/browser.json` for the same non-interactive loopback fixture without updating repository snapshots.
