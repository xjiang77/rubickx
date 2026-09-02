# LAB-NETSEC-03 — TLS identity and certificate lifecycle

## Question

What exactly does a successful TLS handshake prove, and which identity/configuration failures must remain fail-closed?

## Mechanisms

- Go `crypto/x509` generates two local CAs plus server/client certificates entirely in memory.
- A real loopback TLS 1.3 server requires and verifies a client certificate.
- SAN mismatch, expiry, unknown server CA, and missing client certificate are executed as separate failures.
- Trust rotation executes bounded CA overlap, then verifies that v2 succeeds and retired v1 no longer does.

## Invariants

- Encryption without peer-name verification is not authenticated transport.
- SNI selection and certificate hostname validation are separate operations.
- Expired, untrusted or wrong-name certificates do not fall back to plaintext.
- Failed rotation leaves the last-known-good revision explicit; it never claims the desired revision is effective.

## Outcomes

The lab records allow, reject, trust dependency failure, bounded drain and validated recovery.

## Run

```bash
go run ./cmd/netsec-lab -lab LAB-NETSEC-03 -out evidence/lab03.json
```

## Non-guarantees

The generated certificates are educational evidence, not a production PKI, revocation service, HSM, ACME flow, or deployment recipe.

## Knowledge mapping

Primary executable coverage: TLS & PKI.
