# Exercise - Micrograd from scratch (D2 / R3)

> C5 hard constraint: the implementation is handwritten by the learner. An agent may explain concepts, review code, and maintain the verifier, but must not provide the finished implementation.

## Outcome

Build a scalar autograd engine with `Value`, reverse-mode topological backpropagation, and a small MLP that converges on a deterministic toy classification task.

## Contract

- `micrograd.py` provides `Value` with `.data`, `.grad`, `+`, `*`, `**`, `.tanh()`, and `.backward()`.
- Reused graph nodes must accumulate gradients correctly.
- `micrograd.py` provides `MLP`; `train_mlp.py` provides `train(seed=42) -> list[float]`.
- No third-party autograd implementation may replace the handwritten engine.

## Primary verifier

```bash
python3 nanochat/from-scratch/micrograd/test_micrograd.py
```

The verifier checks five finite-difference histories plus deterministic MLP convergence. Relative gradient error must remain below `1e-4`; the final loss must be at most 10% of the first loss. The current skeleton is expected to fail until the learner implements it.

Pytest discovery must also see a real failing/passing test; the verifier must never be marked `skip`.

## Learning connection

Compare the finished exercise with `nanochat_mlx/train.py` only after the closed-book attempt. Record experiment evidence in the Vault `nanochat - Experiment Log`; promote only reusable mechanism insight to AI Knowledge.
