#!/usr/bin/env python3
"""attention 玩具：三个 value 向量，手调 score，看 softmax 权重与输出点怎样在三角形内移动。

运行：python3 attention_toy.py
输出 = Σ softmax(score)_i · value_i。softmax 权重非负且和为 1，所以输出永远落在三个 value
向量围成的三角形内（含边界）；score 拉大只会把输出推向某个顶点，推不出三角形。
"""
import math
from typing import TypeAlias

from canvas import Canvas, show

Point: TypeAlias = tuple[float, float]

VALUES: dict[str, Point] = {"v1": (1, 0), "v2": (0, 1), "v3": (2, 2)}
"""正文推理示例 4 的三个 value 向量。"""
SCALE = 6
"""终端画图的放大倍数。"""


def softmax(scores: list[float]) -> list[float]:
    """把 score 变成非负、和为 1 的权重：softmax(s)_i = exp(s_i) / Σ exp(s_j)。

    先减去最大值再取指数，避免溢出；结果不变，因为分子分母同乘一个常数。

    Examples:
        >>> [round(w, 3) for w in softmax([0, 0, 0])]
        [0.333, 0.333, 0.333]
        >>> [round(w, 2) for w in softmax([1.9459, 0.6931, 0.0])]
        [0.7, 0.2, 0.1]
        >>> round(sum(softmax([5, -3, 100])), 9)
        1.0
    """
    shift = max(scores)
    exps = [math.exp(s - shift) for s in scores]
    total = sum(exps)
    return [e / total for e in exps]


def output(weights: list[float]) -> Point:
    """按权重对 VALUES 里的三个向量做加权和。

    Args:
        weights: 与 VALUES 顺序对应的三个权重。

    Returns:
        Σ weights[i] · value_i。

    Examples:
        >>> output([1, 0, 0])
        (1, 0)
        >>> tuple(round(c, 6) for c in output([0.7, 0.2, 0.1]))   # 浮点比较先取整
        (0.9, 0.4)
    """
    values = list(VALUES.values())
    x = sum(w * v[0] for w, v in zip(weights, values))
    y = sum(w * v[1] for w, v in zip(weights, values))
    return (x, y)


def attention_output(scores: list[float]) -> tuple[list[float], Point]:
    """给定三个 score，返回 (softmax 权重, 输出向量)。

    Examples:
        >>> weights, out = attention_output([0, 0, 8])
        >>> round(weights[2], 3), tuple(round(c, 3) for c in out)
        (0.999, (1.999, 1.999))
    """
    weights = softmax(scores)
    return weights, output(weights)


def draw(weights: list[float]) -> None:
    """在字符画板上画出三角形、三个顶点与输出点 O，并打印权重。"""
    canvas = Canvas()
    names = list(VALUES)
    corners = [(x * SCALE, y * SCALE) for x, y in VALUES.values()]
    for i in range(3):
        canvas.segment(corners[i], corners[(i + 1) % 3], "·")
    for name, corner in zip(names, corners):
        canvas.put(corner[0], corner[1], name[1])   # 顶点标 1、2、3
    ox, oy = output(weights)
    canvas.put(ox * SCALE, oy * SCALE, "O")
    show(canvas)
    weight_text = ", ".join(f"{w:.3f}" for w in weights)
    print(f"权重 = [{weight_text}]，和 = {sum(weights):.3f}，输出 O = ({ox:.3f}, {oy:.3f})")


def main() -> None:
    """先演示三组 score，再进入交互：输入三个 score，q 退出。"""
    print("顶点 1、2、3 是 value 向量 v1, v2, v3（放大 6 倍画出）。O 是 attention 输出。")
    demos = [
        ("均匀 score", [0, 0, 0]),
        ("正文示例：权重约 0.7/0.2/0.1", [1.9459, 0.6931, 0.0]),
        ("score 拉大：几乎只看 v3", [0, 0, 8]),
    ]
    for name, scores in demos:
        print(f"\n--- {name}：score = {scores}")
        draw(softmax(scores))

    print("\n自己试：输入三个 score（如 2 0 -1），q 退出。")
    while True:
        try:
            line = input("score > ").strip()
        except EOFError:
            break
        if line in ("q", "quit", ""):
            break
        parts = line.split()
        if len(parts) != 3:
            print("需要三个数")
            continue
        try:
            scores = [float(x) for x in parts]
        except ValueError:
            print("需要三个数")
            continue
        draw(softmax(scores))


if __name__ == "__main__":
    main()
