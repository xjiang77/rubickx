#!/usr/bin/env python3
"""终端画板演示：向量、首尾相接的加法、数乘，以及共线向量的张成。

运行：python3 draw_vectors.py
每一幕对应知识正文 02 的一个示例；看完再改 V、W 试试。notebook 里的同一演示用 SVG 画。
"""
from typing import TypeAlias

from canvas import Canvas, show

Point: TypeAlias = tuple[float, float]

V: Point = (3, 1)
W: Point = (1, 2)
TARGET: Point = (7, 4)
"""正文「逐步推演」用的两个向量与目标：2·V + 1·W = TARGET。"""


def scaled(v: Point, k: float) -> Point:
    """k·v。

    Examples:
        >>> scaled((3, 1), 2)
        (6, 2)
    """
    return (k * v[0], k * v[1])


def added(v: Point, w: Point) -> Point:
    """v + w。

    Examples:
        >>> added((6, 2), (1, 2))
        (7, 4)
    """
    return (v[0] + w[0], v[1] + w[1])


def scene_two_vectors() -> None:
    print(f"第 1 幕：v = {V} 与 w = {W}，从原点出发的两个箭头")
    canvas = Canvas()
    canvas.arrow(V, "v", label="v")
    canvas.arrow(W, "w", label="w")
    show(canvas)


def scene_tip_to_tail() -> None:
    print("\n第 2 幕：线性组合 2·v + 1·w。先走 2·v（记作 a），再从它的尖端接上 w（记作 b），终点是 [7, 4]（记作 T）")
    canvas = Canvas()
    tip = canvas.arrow(scaled(V, 2), "a", label="a")
    canvas.arrow(W, "b", tail=tip, label="b")
    canvas.arrow(TARGET, "=", label="T")
    show(canvas)
    end = added(scaled(V, 2), W)
    print("终点坐标：", end, "== TARGET:", end == TARGET)


def scene_scaling() -> None:
    print("\n第 3 幕：数乘。−1.5·v 与 v 在同一条直线上，方向相反、长 1.5 倍（记作 m）")
    canvas = Canvas()
    canvas.arrow(V, "v", label="v")
    canvas.arrow(scaled(V, -1.5), "m", label="m")
    show(canvas)


def scene_span(q: Point, q_name: str, title: str) -> None:
    """把 a·V + b·q 在 a、b 取 −2…2 整数时的落点用 '+' 标出。"""
    print(title)
    canvas = Canvas()
    for a in range(-2, 3):
        for b in range(-2, 3):
            point = added(scaled(V, a), scaled(q, b))
            canvas.put(point[0], point[1], "+")
    canvas.arrow(V, "v", label="v")
    canvas.arrow(q, q_name, label=q_name)
    show(canvas)


def main() -> None:
    scene_two_vectors()
    scene_tip_to_tail()
    scene_scaling()
    scene_span(scaled(V, 2), "u", "\n第 4 幕：共线的 v 与 u = 2v。任意 a·v + b·u 都落在同一条直线上：")
    scene_span(W, "w", "对照：把 u 换成 w = [1, 2]，同样的系数覆盖的是一片平行四边形网格：")


if __name__ == "__main__":
    main()
