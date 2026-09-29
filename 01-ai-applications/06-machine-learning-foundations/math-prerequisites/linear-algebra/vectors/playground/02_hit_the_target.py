#!/usr/bin/env python3
"""命中目标：给定基向量 b1、b2 与目标 t，输入系数 (a, b)，看 a·b1 + b·b2 落在哪里。

运行：python3 02_hit_the_target.py            # 默认关卡
      python3 02_hit_the_target.py --level 2  # 换基
每一关的答案就是 t 在基 {b1, b2} 下的坐标。命中即通关；输入 q 退出。
"""
import sys
from canvas import Canvas, show

LEVELS = {
    1: {"b1": (3, 1), "b2": (1, 2), "t": (7, 4), "hint": "正文「逐步推演」的例子"},
    2: {"b1": (1, 1), "b2": (1, -1), "t": (4, 2), "hint": "正文运算示例 2：换基坐标"},
    3: {"b1": (1, 2), "b2": (2, -1), "t": (8, 1), "hint": "测验题 3"},
    4: {"b1": (3, 1), "b2": (6, 2), "t": (7, 4), "hint": "基向量共线——这一关能通吗？"},
}


def play(level):
    L = LEVELS[level]
    b1, b2, t = L["b1"], L["b2"], L["t"]
    print(f"第 {level} 关（{L['hint']}）：b1 = {b1}，b2 = {b2}，目标 t = {t}")
    print("输入两个数 a b，使 a·b1 + b·b2 = t。")
    tries = 0
    while True:
        try:
            s = input("a b > ").strip()
        except EOFError:
            return False
        if s in ("q", "quit"):
            return False
        try:
            a, b = (float(x) for x in s.split())
        except ValueError:
            print("格式：两个数，用空格隔开")
            continue
        tries += 1
        p = (a * b1[0] + b * b2[0], a * b1[1] + b * b2[1])
        gap = (t[0] - p[0], t[1] - p[1])
        c = Canvas()
        tip = c.arrow((a * b1[0], a * b1[1]), "1", label="1")
        c.arrow((b * b2[0], b * b2[1]), "2", tail=tip, label="P")
        c.put(t[0], t[1], "T")
        show(c)
        print(f"落点 P = ({p[0]:g}, {p[1]:g})，距目标还差 {gap}")
        if abs(gap[0]) < 1e-9 and abs(gap[1]) < 1e-9:
            print(f"命中！用了 {tries} 次。t 在基 {{b1, b2}} 下的坐标是 ({a:g}, {b:g})。\n")
            return True
        det = b1[0] * b2[1] - b1[1] * b2[0]
        if abs(det) < 1e-9:
            print("提示：b1 与 b2 共线，所有落点都在一条直线上；目标不在这条线上时永远打不中。")


if __name__ == "__main__":
    level = int(sys.argv[sys.argv.index("--level") + 1]) if "--level" in sys.argv else 1
    while level in LEVELS and play(level):
        level += 1
