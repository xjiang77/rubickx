# Pro Git snapshot

Source URL: https://git-scm.com/book/en/v2

This snapshot is prepared for harness evaluation. It is a compressed description of the source, not a verbatim copy.

- The material starts by explaining why Git is different from centralized version control systems. The key shift is to think in snapshots rather than file-by-file deltas.
- A recurring mental model is the relationship between working tree, staging area (index), and repository history. Beginners often memorize commands, but the book keeps returning to these three states so commands feel predictable.
- Branching is presented as cheap pointer movement. That framing makes merge, fast-forward, and rebase easier to reason about.
- Collaboration chapters emphasize remote collaboration: fetch, pull, push, pull request style workflows, and the importance of understanding divergence before force-pushing.
- The material is reference-heavy. It is excellent for accuracy, but many readers need extra scaffolding: exercises, checkpoints, and a recommended order.
- Good downstream transformation would likely turn the reference into a Chinese lesson sequence with practical exercises such as resolving a merge conflict, reading the commit graph, and deciding when to use rebase versus merge.
