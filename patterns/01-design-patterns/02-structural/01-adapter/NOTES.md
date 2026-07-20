# Adapter - Boundary Translation Contract

Canonical Vault note: `01 - Eng - Adapter：边界转换与契约隔离.md`. Code identity is `gof.structural.adapter`; dynamic completion state lives only in [`PROGRESS.md`](../../../PROGRESS.md).

## Behavior Contract

The caller owns a stable `ChatClient` contract while the legacy provider uses a deployment ID, flattened prompt, vendor stop codes, and vendor errors. The Adapter must:

- map canonical request identity and payload to the legacy request;
- normalize response and error semantics;
- reject unsupported capabilities explicitly before invoking the provider;
- keep provider types out of the caller contract.

All four implementations read [`fixtures/contract.json`](fixtures/contract.json). Its cases cover nominal mapping, unknown-model boundary, provider failure, and provider-call lifecycle on unsupported capability.

## Language Mapping

| Language | Target expression | Adapter expression | Evidence boundary |
| --- | --- | --- | --- |
| Python | `Protocol` and frozen dataclasses | duck-typed client | incomplete providers fail at runtime |
| Go | consumer-owned `interface` | explicit value/error mapping | normalized errors remain inspectable with `errors.As` |
| Java | `interface` and records | compile-time contract | JUnit reads the shared JSON through Jackson |
| JavaScript | structural object contract | runtime validation | unsupported input fails before provider invocation |

The participant roles are equivalent; syntax and type-system guarantees are intentionally not forced to look identical.

## Neighbors

| Option | Primary intent | Why it is not Adapter |
| --- | --- | --- |
| Facade | simplify a multi-step subsystem entry | it may preserve the underlying semantics |
| Decorator | add composable behavior to the same contract | it should not translate the core protocol |
| Proxy | control access or lifecycle behind the same interface | it represents the original subject rather than a foreign dialect |
| DDD ACL | protect a bounded context's language | it can contain several adapters, policies, and translators |

## Proof Boundary

This lab proves deterministic mapping and fail-explicitly behavior against a fake provider. It does not prove network compatibility, streaming protocol correctness, production retry policy, vendor SDK behavior, or an entire anti-corruption layer.

## Run

```bash
make -C patterns test-pattern PATTERN=gof.structural.adapter
go test -race ./patterns/01-design-patterns/02-structural/01-adapter/go
```
