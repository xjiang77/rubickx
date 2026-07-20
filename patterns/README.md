# Rubickx Patterns

`patterns/` is the executable evidence layer for the Dragon Vault Engineering Pattern Library. The fixed catalog contains 42 patterns and mirrors the Vault families:

```text
patterns/
├── 01-design-patterns/
│   ├── 01-creational/
│   ├── 02-structural/
│   └── 03-behavioral/
├── 02-reliability-patterns/
├── 03-data-messaging-patterns/
└── 04-concurrency-patterns/
```

## Contract

Every catalog entry must contain:

1. `NOTES.md` with the behavior contract, language mapping, proof boundary, and run command.
2. `fixtures/contract.json` with nominal, boundary, failure, and lifecycle or non-interference cases.
3. Python, Go, Java, and JavaScript implementations and tests.
4. A canonical Vault note with three scenarios, comparison, trade-offs, and evidence protocol.

Implementations calculate behavior from the input. They must not branch on fixture case IDs, return expected fixture values, use real network or sleep, or claim production-library guarantees.

## Sources of Truth

- [`catalog.json`](catalog.json): immutable identity, family, order, code path, Vault path, and contract path.
- [`PROGRESS.md`](PROGRESS.md): dynamic completion state after a vertical slice passes all tests.
- Vault `Engineering Patterns Source Map`: source provenance and catalog differences.

## Catalog And Golden Path

The catalog contains 23 classic design, 6 reliability, 7 data and messaging, and 6 concurrency patterns. [Adapter](01-design-patterns/02-structural/01-adapter/NOTES.md) remains the first golden slice: it translates a stable `ChatClient` contract to a legacy provider contract and verifies request mapping, response normalization, error normalization, and fail-explicitly capability behavior.

## Run

```bash
make -C patterns setup
make -C patterns test-pattern PATTERN=gof.structural.adapter
make -C patterns verify
make -C patterns verify-vault VAULT_ROOT=/Users/kevinxjiang/Obsidian/dragon-vault
```

`verify` fails while any catalog item is pending and runs all four language suites plus Go race detection. `verify-vault` is the final cross-repository gate and additionally validates frontmatter, three scenarios, navigation coverage, rubickx paths, and scoped wikilinks.
