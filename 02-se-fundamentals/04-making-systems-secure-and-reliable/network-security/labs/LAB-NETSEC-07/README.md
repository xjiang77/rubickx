# LAB-NETSEC-07 — OIDC-style Authorization Code and PKCE

## Question

How does a browser login bind the authorization response, code redemption, token and RP session to one transaction?

## Wire-shaped flow

1. RP creates `state`, `nonce` and PKCE verifier/challenge.
2. Authorization endpoint validates exact `client_id` + `redirect_uri` and issues a single-use code.
3. RP verifies `state` and response issuer before redeeming the code with the verifier.
4. RP validates RS256 signature, `kid`, `iss`, `aud`, `azp`, `nonce`, `exp`, `iat` and `nbf`.
5. Only then does it create an independent RP session.

## Negative paths

The runner executes a table-driven negative matrix for redirect, nonce, PKCE, replay, code injection, issuer, audience, expiry, and unknown key. Every case is asserted before RP session creation. The browser surface adds state/mix-up variants. Rotation retains the old key only for an explicit overlap window.

## Invariants

- OAuth authorization alone is not user authentication.
- Access token, ID token, authorization code and RP cookie are different credentials with different audiences and lifetimes.
- Email is never the stable subject key; use issuer + subject.
- Unknown `kid` does not trigger silent trust or arbitrary network fetch.

## Run

```bash
go run ./cmd/netsec-browser
```

This is an educational minimal profile, not an OpenID certification target.

## Knowledge mapping

Primary executable coverage: SSO and Federation, specifically the OIDC Authorization Code + PKCE path.
