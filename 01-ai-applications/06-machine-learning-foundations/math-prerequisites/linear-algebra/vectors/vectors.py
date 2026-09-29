"""向量与空间 —— 本人手写 (C5)。下面只有骨架，实现留空。

向量用 list[float] 表示。不得 import numpy 或其他数值库。
规格见 EXERCISE.md；数值示例见知识库正文 02。
"""


def add(v, w):
    """按分量相加。"""
    raise NotImplementedError


def scale(c, v):
    """每个分量乘以标量 c。"""
    raise NotImplementedError


def linear_combination(coeffs, vectors):
    """sum(c_i * v_i)。coeffs 与 vectors 等长。"""
    raise NotImplementedError


def is_collinear(v, w, tol=1e-9):
    """v 与 w 是否共线（存在 k 使 w = k*v 或 v = k*w）。零向量与任何向量共线。"""
    raise NotImplementedError


def solve_2x2(v, w, t):
    """求 (a, b) 使 a*v + b*w == t（均为二维）。v、w 共线时返回 None。"""
    raise NotImplementedError


def coords_in_basis(t, b1, b2):
    """t 在基 {b1, b2} 下的坐标 (a, b)。b1、b2 共线时抛出 ValueError。"""
    raise NotImplementedError


def in_span(t, v, w, tol=1e-9):
    """三维：t 是否在不共线的 v、w 的张成内。"""
    raise NotImplementedError


def is_independent_3(v1, v2, v3, tol=1e-9):
    """三个三维向量是否线性无关。"""
    raise NotImplementedError


def weighted_sum(weights, vectors):
    """权重非负且和为 1（容差 1e-9）时返回 linear_combination，否则抛出 ValueError。"""
    raise NotImplementedError
