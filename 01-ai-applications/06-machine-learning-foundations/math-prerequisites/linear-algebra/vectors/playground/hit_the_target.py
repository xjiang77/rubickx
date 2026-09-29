#!/usr/bin/env python3
"""命中目标：给定基向量 b1、b2 与目标 t，输入系数 (a, b)，看 a·b1 + b·b2 落在哪里。

运行：python3 hit_the_target.py            # 默认关卡
      python3 hit_the_target.py --level 2  # 换基
每一关的答案就是 t 在基 {b1, b2} 下的坐标。命中即通关；输入 q 退出。
"""
import sys
from typing import TypeAlias

from canvas import Canvas, show

Point: TypeAlias = tuple[float, float]

LEVELS: dict[int, dict] = {
    1: {"b1": (3, 1), "b2": (1, 2), "t": (7, 4), "hint": "正文「逐步推演」的例子"},
    2: {"b1": (1, 1), "b2": (1, -1), "t": (4, 2), "hint": "正文运算示例 2：换基坐标"},
    3: {"b1": (1, 2), "b2": (2, -1), "t": (8, 1), "hint": "测验题 3"},
    4: {"b1": (3, 1), "b2": (6, 2), "t": (7, 4), "hint": "基向量共线——这一关能通吗？"},
}


def combine(a: float, b: float, b1: Point, b2: Point) -> Point:
    """计算 a·b1 + b·b2。

    Examples:
        >>> combine(2, 1, (3, 1), (1, 2))
        (7, 4)
    """
    return (a * b1[0] + b * b2[0], a * b1[1] + b * b2[1])


def is_collinear(b1: Point, b2: Point, tol: float = 1e-9) -> bool:
    """两个二维向量是否共线：b1[0]·b2[1] − b1[1]·b2[0] 是否为 0。

    Examples:
        >>> is_collinear((3, 1), (6, 2))
        True
        >>> is_collinear((3, 1), (1, 2))
        False
    """
    return abs(b1[0] * b2[1] - b1[1] * b2[0]) <= tol


def draw_attempt(a: float, b: float, b1: Point, b2: Point, t: Point) -> None:
    """画出 a·b1（标 1）、接上的 b·b2（标 P）以及目标 T。"""
    canvas = Canvas()
    tip = canvas.arrow((a * b1[0], a * b1[1]), "1", label="1")
    canvas.arrow((b * b2[0], b * b2[1]), "2", tail=tip, label="P")
    canvas.put(t[0], t[1], "T")
    show(canvas)


def play(level: int) -> bool:
    """玩一关。返回 True 表示命中通关，False 表示玩家退出。"""
    spec = LEVELS[level]
    b1, b2, t = spec["b1"], spec["b2"], spec["t"]
    print(f"第 {level} 关（{spec['hint']}）：b1 = {b1}，b2 = {b2}，目标 t = {t}")
    print("输入两个数 a b，使 a·b1 + b·b2 = t。")
    tries = 0
    while True:
        try:
            line = input("a b > ").strip()
        except EOFError:
            return False
        if line in ("q", "quit"):
            return False
        try:
            a, b = (float(x) for x in line.split())
        except ValueError:
            print("格式：两个数，用空格隔开")
            continue
        tries += 1
        point = combine(a, b, b1, b2)
        gap = (t[0] - point[0], t[1] - point[1])
        draw_attempt(a, b, b1, b2, t)
        print(f"落点 P = ({point[0]:g}, {point[1]:g})，距目标还差 {gap}")
        if abs(gap[0]) < 1e-9 and abs(gap[1]) < 1e-9:
            print(f"命中！用了 {tries} 次。t 在基 {{b1, b2}} 下的坐标是 ({a:g}, {b:g})。\n")
            return True
        if is_collinear(b1, b2):
            print("提示：b1 与 b2 共线，所有落点都在一条直线上；目标不在这条线上时永远打不中。")


def main() -> None:
    """从命令行指定的关卡开始，依次玩到最后一关。"""
    level = int(sys.argv[sys.argv.index("--level") + 1]) if "--level" in sys.argv else 1
    while level in LEVELS and play(level):
        level += 1


if __name__ == "__main__":
    main()
