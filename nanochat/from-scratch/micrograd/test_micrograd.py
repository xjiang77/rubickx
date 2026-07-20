#!/usr/bin/env python3
"""Micrograd R3 verifier: finite differences plus a deterministic MLP check.

The exercise implementation remains user-owned. Do not weaken the tolerance,
remove cases, or replace the handwritten autograd engine with a library.
"""

import os
import sys
import traceback

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

H = 1e-5
TOL = 1e-4


def rel_err(a, b):
    return abs(a - b) / max(1e-8, abs(a) + abs(b))


def evaluate():
    from micrograd import Value

    results = []

    def check_grads(name, build, inputs):
        def scalar_f(xs):
            vals = [Value(x) for x in xs]
            return build(vals).data

        vals = [Value(x) for x in inputs]
        out = build(vals)
        out.backward()
        analytic = [v.grad for v in vals]

        ok = True
        details = []
        for index in range(len(inputs)):
            xs_hi = list(inputs)
            xs_lo = list(inputs)
            xs_hi[index] += H
            xs_lo[index] -= H
            numeric = (scalar_f(xs_hi) - scalar_f(xs_lo)) / (2 * H)
            error = rel_err(analytic[index], numeric)
            details.append(
                f"  d/dx{index}: handwritten={analytic[index]:.6g} "
                f"numeric={numeric:.6g} relative_error={error:.2e}"
            )
            if error >= TOL:
                ok = False
        results.append((name, ok, "\n".join(details)))

    check_grads(
        "add_mul",
        lambda values: values[0] * values[1] + values[1] + values[0] * values[0],
        [3.0, -2.0],
    )
    check_grads(
        "pow",
        lambda values: values[0] ** 3 + values[1] ** 2 * values[0],
        [1.5, -0.7],
    )
    check_grads(
        "tanh_chain",
        lambda values: (values[0] * values[1] + values[0]).tanh() * values[1],
        [0.8, -1.2],
    )

    def diamond(values):
        edge = values[0] * values[1]
        return edge + values[0] + edge * edge

    check_grads("reuse_accumulate", diamond, [0.6, 1.1])

    def deep(values):
        current = values[0]
        for _ in range(20):
            current = (current * values[1] + values[0]).tanh()
        return current

    check_grads("deep_chain", deep, [0.5, 0.3])

    try:
        from train_mlp import train

        losses = train(seed=42)
        assert isinstance(losses, list) and len(losses) >= 20, "loss history must contain at least 20 points"
        ok = losses[-1] <= 0.10 * losses[0]
        results.append(
            (
                "mlp_train",
                ok,
                f"  loss: {losses[0]:.4f} -> {losses[-1]:.4f} "
                f"(required <= {0.10 * losses[0]:.4f})",
            )
        )
    except Exception:
        results.append(("mlp_train", False, "  " + traceback.format_exc(limit=2).strip()))

    return results


def test_micrograd_contract():
    results = evaluate()
    failures = [f"{name}\n{detail}" for name, ok, detail in results if not ok]
    assert not failures, "\n\n".join(failures)


def main():
    try:
        results = evaluate()
    except Exception:
        print("Verifier could not run because the implementation is missing or violates the contract:\n" + traceback.format_exc(limit=3))
        return 2

    passed = sum(1 for _, ok, _ in results if ok)
    for name, ok, detail in results:
        print(f"{'PASS' if ok else 'FAIL'}  {name}\n{detail}")
    print(f"\n{passed}/{len(results)} cases passed")
    return 0 if passed == len(results) else 1


if __name__ == "__main__":
    raise SystemExit(main())
