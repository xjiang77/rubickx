#!/usr/bin/env python3
"""ASCII 画板：向量、首尾相接的加法、数乘，以及共线向量的张成。

运行：python3 draw_vectors.py
每一幕对应知识正文 02 的一个示例；看完再改下面的向量试试。
"""
from canvas import Canvas, show

V, W = (3, 1), (1, 2)          # 正文「逐步推演」用的两个向量
TARGET = (7, 4)                # 2·V + 1·W

def main():

    print("第 1 幕：v = %s 与 w = %s，从原点出发的两个箭头" % (V, W))
    c = Canvas()
    c.arrow(V, "v", label="v")
    c.arrow(W, "w", label="w")
    show(c)

    print("\n第 2 幕：线性组合 2·v + 1·w。先走 2·v（记作 a），再从它的尖端接上 w（记作 b），终点就是 [7, 4]（记作 T）")
    c = Canvas()
    two_v = (2 * V[0], 2 * V[1])
    tip = c.arrow(two_v, "a", label="a")
    c.arrow(W, "b", tail=tip, label="b")
    c.arrow(TARGET, "=", label="T")
    show(c)
    print("终点坐标：", (two_v[0] + W[0], two_v[1] + W[1]), "== TARGET:", (two_v[0] + W[0], two_v[1] + W[1]) == TARGET)

    print("\n第 3 幕：数乘。−1.5·v 与 v 在同一条直线上，方向相反、长 1.5 倍（记作 m）")
    c = Canvas()
    c.arrow(V, "v", label="v")
    c.arrow((-1.5 * V[0], -1.5 * V[1]), "m", label="m")
    show(c)

    print("\n第 4 幕：共线的 v = [3, 1] 与 u = [6, 2]（记作 u）。任意 a·v + b·u 都落在同一条直线上，")
    print("下面把 a、b 取遍 −2…2 的整数，用 '+' 标出全部组合的落点：")
    c = Canvas()
    U = (2 * V[0], 2 * V[1])
    for a in range(-2, 3):
        for b in range(-2, 3):
            c.put(a * V[0] + b * U[0], a * V[1] + b * U[1], "+")
    c.arrow(V, "v", label="v")
    c.arrow(U, "u", label="u")
    show(c)
    print("对照：把 u 换成 w = [1, 2]，同样的系数覆盖的是一片平行四边形网格，而不是一条线：")
    c = Canvas()
    for a in range(-2, 3):
        for b in range(-2, 3):
            c.put(a * V[0] + b * W[0], a * V[1] + b * W[1], "+")
    show(c)


if __name__ == "__main__":
    main()
