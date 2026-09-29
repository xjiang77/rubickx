#!/usr/bin/env python3
"""attention 玩具：三个 value 向量，手调 score，看 softmax 权重与输出点怎样在三角形内移动。

运行：python3 attention_toy.py
输出 = Σ softmax(score)_i · v_i。softmax 权重非负且和为 1，所以输出永远落在三个 value 向量
围成的三角形内（含边界）；score 拉大只会把输出推向某个顶点，推不出三角形。
"""
import math
from canvas import Canvas, show

VALUES = {"v1": (1, 0), "v2": (0, 1), "v3": (2, 2)}   # 正文推理示例 4 的三个 value 向量
SCALE = 6                                             # 画图放大倍数，方便看清


def softmax(scores):
    m = max(scores)
    e = [math.exp(s - m) for s in scores]
    z = sum(e)
    return [x / z for x in e]


def attention_output(scores):
    """给定三个 score，返回 (softmax 权重, 输出向量)。输出 = Σ 权重_i · value_i。"""
    weights = softmax(scores)
    return weights, output(weights)


def output(weights):
    vs = list(VALUES.values())
    return (sum(w * v[0] for w, v in zip(weights, vs)), sum(w * v[1] for w, v in zip(weights, vs)))


def draw(weights):
    c = Canvas()
    names = list(VALUES)
    pts = [(x * SCALE, y * SCALE) for x, y in VALUES.values()]
    for i in range(3):                              # 三角形的边
        c.segment(pts[i], pts[(i + 1) % 3], "·")
    for n, p in zip(names, pts):
        c.put(p[0], p[1], n[1])                     # 顶点标 1、2、3
    ox, oy = output(weights)
    c.put(ox * SCALE, oy * SCALE, "O")
    show(c)
    print("权重 = [" + ", ".join(f"{w:.3f}" for w in weights) + f"]，和 = {sum(weights):.3f}，输出 O = ({ox:.3f}, {oy:.3f})")



def main():
    print("顶点 1、2、3 是 value 向量 v1, v2, v3（放大 6 倍画出）。O 是 attention 输出。")
    for name, scores in [("均匀 score", [0, 0, 0]),
                         ("正文示例：权重约 0.7/0.2/0.1", [1.9459, 0.6931, 0.0]),
                         ("score 拉大：几乎只看 v3", [0, 0, 8])]:
        print(f"\n--- {name}：score = {scores}")
        draw(softmax(scores))

    print("\n自己试：输入三个 score（如 2 0 -1），q 退出。")
    while True:
        try:
            s = input("score > ").strip()
        except EOFError:
            break
        if s in ("q", "quit", ""):
            break
        try:
            scores = [float(x) for x in s.split()]
            assert len(scores) == 3
        except (ValueError, AssertionError):
            print("需要三个数")
            continue
        draw(softmax(scores))


if __name__ == "__main__":
    main()
