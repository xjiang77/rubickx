# LAB-NETSEC-08 — Federation, logout and provisioning lifecycle

## Question

What happens when one identity session fans out into multiple RP sessions and logout or deprovision succeeds only partially?

## State model

- IdP session, RP-A session, RP-B session, token grant and directory account are separate state owners.
- Front-channel logout depends on the browser; back-channel logout is direct but can still fail delivery.
- A failed RP-B acknowledgement preserves `rp-b active` as effective state and schedules bounded retry.
- SCIM-style deprovision creates a tombstone and denies new sessions while existing sessions converge to inactive.

## SAML fixture contract

The lab uses Go `crypto/ed25519` to sign and verify an educational federation fixture, then executes audience/recipient/request correlation, validity, and replay checks. It deliberately does not implement or claim XMLDSig/SAML interoperability or a Kerberos KDC.

## Invariants

- One logout click is not proof that every RP session ended.
- Session closure is acknowledged per RP and remains observable until convergence.
- `(issuer, subject)` is the account-linking key; mutable email is an attribute.
- Deprovision and group removal cannot depend only on the next interactive login.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-08 -out evidence/lab08.json
```

## Knowledge mapping

Integration executable coverage: SSO and Federation. The signature and lifecycle state machine execute locally; wire-level SAML/XMLDSig and Kerberos remain model boundaries.
