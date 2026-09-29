#!/usr/bin/env python3
"""用 numpy 复现知识库正文 02（向量与空间）的示例。只供阅读与对照，不是练习答案。

运行：python3 demo.py
每一段先打印题目，再打印 numpy 的算法与结果；assert 保证与正文数值一致。
"""

import numpy as np

np.set_printoptions(precision=4, suppress=True)


def section(title):
    print("\n== " + title)


section("加法与数乘：[1,2] + [3,-1]，2*[1,2] - 3*[3,-1]")
v, w = np.array([1, 2]), np.array([3, -1])
print("v + w =", v + w)
print("2v - 3w =", 2 * v - 3 * w)
assert (v + w == [4, 1]).all() and (2 * v - 3 * w == [-7, 7]).all()

section("坐标即基向量的缩放：(3,-2) = 3*i_hat + (-2)*j_hat")
i_hat, j_hat = np.array([1, 0]), np.array([0, 1])
print("3*i_hat + (-2)*j_hat =", 3 * i_hat + (-2) * j_hat)

section("线性组合求解：[7,4] = a*[3,1] + b*[1,2]")
# 把 v、w 作为矩阵的两列，解 M @ [a, b] = t。矩阵的含义见正文 03。
M = np.array([[3, 1], [1, 2]], dtype=float).T
a, b = np.linalg.solve(M, [7, 4])
print("(a, b) =", (a, b))
assert np.allclose([a, b], [2, 1])

section("共线判定：[3,1] 与 [6,2]，[2,-4] 与 [-3,6]")
for p, q in [([3, 1], [6, 2]), ([2, -4], [-3, 6]), ([3, 1], [1, 2])]:
    rank = np.linalg.matrix_rank(np.array([p, q], dtype=float))
    print(f"{p} 与 {q}: rank = {rank} ->", "共线" if rank == 1 else "不共线")

section("换基坐标：[4,2] 在基 {[1,1],[1,-1]} 下")
B = np.array([[1, 1], [1, -1]], dtype=float).T
coords = np.linalg.solve(B, [4, 2])
print("坐标 =", coords, "；验证 B @ coords =", B @ coords)
assert np.allclose(coords, [3, 1])

section("张成判定：[1,2,3] 与 [1,2,4] 是否在 span{[1,0,1],[0,1,1]} 内")
v, w = np.array([1, 0, 1], dtype=float), np.array([0, 1, 1], dtype=float)
for t in ([1, 2, 3], [1, 2, 4]):
    A = np.array([v, w]).T
    coef, residual, *_ = np.linalg.lstsq(A, np.array(t, dtype=float), rcond=None)
    inside = np.allclose(A @ coef, t)
    print(f"{t}: 最小二乘系数 {coef}，重构 {A @ coef} ->", "在张成内" if inside else "不在")

section("线性相关判定：[1,0,1],[0,1,1],[1,1,2]")
V = np.array([[1, 0, 1], [0, 1, 1], [1, 1, 2]], dtype=float)
print("rank =", np.linalg.matrix_rank(V), "（3 个向量、秩 2 -> 线性相关，张成为平面）")
c = np.array([-1, -1, 1])
print("零组合 c = (-1,-1,1): c @ V =", c @ V)
assert np.linalg.matrix_rank(V) == 2 and np.allclose(c @ V, 0)

section("attention 输出为 value 向量的加权平均")
scores = np.array([1.9459, 0.6931, 0.0])  # 选得使 softmax 约为 [0.7, 0.2, 0.1]
weights = np.exp(scores) / np.exp(scores).sum()
Vals = np.array([[1, 0], [0, 1], [2, 2]], dtype=float)
print("softmax 权重 =", weights, "，和 =", weights.sum())
print("输出 = weights @ V =", weights @ Vals)
assert np.allclose(weights @ Vals, [0.9, 0.4], atol=1e-3)

print("\n全部示例与正文数值一致。")
