"""数学前置（topic 1.6.0）：用数值求导核对导数、梯度与链式法则。

例子与 ELI5 页一致：单参数预测模型 ŷ = w·x + b，误差 e = ŷ − y，损失 L = e²。
只用标准库，便于在任何环境运行：python3 gradcheck.py 打印逐步推演。
"""
from __future__ import annotations

from typing import Callable, Dict, Iterable, List, Tuple

Sample = Tuple[float, float]  # (x, y)


def numeric_derivative(f: Callable[[float], float], x: float, h: float = 1e-5) -> float:
    """中心差分：[f(x + h) − f(x − h)] / 2h。"""
    return (f(x + h) - f(x - h)) / (2 * h)


def loss(w: float, b: float, samples: Iterable[Sample]) -> float:
    """平方误差之和：Σ (w·x + b − y)²。"""
    return sum((w * x + b - y) ** 2 for x, y in samples)


def analytic_gradient(w: float, b: float, samples: Iterable[Sample]) -> Dict[str, float]:
    """链式法则：∂L/∂w = Σ 2e · x，∂L/∂b = Σ 2e，其中 e = w·x + b − y。"""
    gw = gb = 0.0
    for x, y in samples:
        e = w * x + b - y
        gw += 2 * e * x   # ∂L/∂e · ∂e/∂ŷ · ∂ŷ/∂w = 2e · 1 · x
        gb += 2 * e       # ∂L/∂e · ∂e/∂ŷ · ∂ŷ/∂b = 2e · 1 · 1
    return {"w": gw, "b": gb}


def numeric_gradient(w: float, b: float, samples: List[Sample], h: float = 1e-5) -> Dict[str, float]:
    """对每个参数分别做数值求导（偏导数：只改它、其他不动）。"""
    return {
        "w": numeric_derivative(lambda v: loss(v, b, samples), w, h),
        "b": numeric_derivative(lambda v: loss(w, v, samples), b, h),
    }


def chain_factors(w: float, x: float, y: float) -> Dict[str, float]:
    """单个样本、b = 0 时，链式法则里的三个局部导数。"""
    e = w * x - y
    return {"dL/de": 2 * e, "de/dyhat": 1.0, "dyhat/dw": x}


def descend(w: float, b: float, samples: List[Sample], lr: float, steps: int,
            train_bias: bool = True) -> List[Tuple[float, float, float]]:
    """梯度下降：θ ← θ − η · ∇L。返回每一步的 (w, b, L)，第 0 项是初始值。"""
    history = [(w, b, loss(w, b, samples))]
    for _ in range(steps):
        g = analytic_gradient(w, b, samples)
        w -= lr * g["w"]
        if train_bias:
            b -= lr * g["b"]
        history.append((w, b, loss(w, b, samples)))
    return history


def main() -> None:
    one = [(2.0, 10.0)]
    print("前向：ŷ = w·x = 6，e = ŷ − y = −4，L = e² = 16")
    f = chain_factors(3.0, 2.0, 10.0)
    print("局部导数：", f, " 相乘 ∂L/∂w =", f["dL/de"] * f["de/dyhat"] * f["dyhat/dw"])
    print("数值核对：w = 3.01 时 L =", round(loss(3.01, 0.0, one), 4))
    two = [(2.0, 10.0), (1.0, 4.0)]
    print("两个样本：解析梯度", analytic_gradient(3.0, 0.0, two), "数值梯度",
          {k: round(v, 4) for k, v in numeric_gradient(3.0, 0.0, two).items()})
    print("梯度下降（只调 w，η = 0.05）：", [round(L, 4) for _, _, L in descend(3.0, 0.0, one, 0.05, 4, train_bias=False)])
    print("学习率过大（η = 0.25）：", [(round(w, 2), round(L, 2)) for w, _, L in descend(3.0, 0.0, one, 0.25, 3, train_bias=False)])


if __name__ == "__main__":
    main()
