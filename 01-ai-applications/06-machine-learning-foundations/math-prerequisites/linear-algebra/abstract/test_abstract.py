#!/usr/bin/env python3
"""抽象向量空间验证器：3Blue1Brown 第 16 集的数值示例 + numpy 随机对拍 + doctest。

运行（仓库根目录）：make test-linalg-abstract
实现由本人手写（C5）。不得放宽容差、删除用例，或用库函数替换 abstract.py 中的实现。
骨架状态下本脚本预期失败；不得改为 skip。
"""
import doctest
import os
import random
import sys
import traceback
from collections.abc import Callable, Iterator
from typing import Any

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)

Case = tuple[str, bool]
"""一条用例：(名称, 是否通过)。"""

# x^2 + 3x + 5。课页坐标，低次在前。
F_POLY = [5, 3, 1]
# 4x^7 - 5x^2。
G_POLY = [0, 0, -5, 0, 0, 0, 0, 4]
# x^3 + 5x^2 + 4x + 5。导数矩阵乘的是这组坐标。
WALK = [5, 4, 5, 1]
# (1/9) x^3 - x。导数是 (1/3) x^2 - 1。
CUBIC = [0, -1, 0, 1 / 9]
# 课页用来说明「在同一个输入上相加」的那个输入。
X_DEMO = -4


def close(a: Any, b: Any, tol: float = 1e-7) -> bool:
    """数值或数值序列是否在相对容差内相等。

    Examples:
        >>> close([1.0, 0.0], [1, 0])
        True
        >>> close(1.0, 1.1)
        False
    """
    if isinstance(a, (list, tuple)):
        return len(a) == len(b) and all(close(x, y, tol) for x, y in zip(a, b))
    return abs(a - b) <= tol * max(1.0, abs(a), abs(b))


def raises(exc: type[BaseException], fn: Callable[..., Any], *args: Any) -> bool:
    """调用 fn(*args) 是否抛出 exc。

    Examples:
        >>> raises(ValueError, int, "x")
        True
        >>> raises(ValueError, int, "3")
        False
    """
    try:
        fn(*args)
    except exc:
        return True
    return False


def _std_add(u: list[float], v: list[float]) -> list[float]:
    return [a + b for a, b in zip(u, v)]


def _std_scale(c: float, v: list[float]) -> list[float]:
    return [c * x for x in v]


def _add_assoc_bump(u: list[float], v: list[float]) -> list[float]:
    """只把 [1, 0] + [1, 1] 从 [2, 1] 改成 [3, 1]。

    样本 {[1, 0], [0, 1]} 两两相加时碰不到这对输入，交换律仍成立。
    ([1, 0] + [1, 0]) + [0, 1] 是 [2, 1]，[1, 0] + ([1, 0] + [0, 1]) 走到这处改写，变成 [3, 1]。
    """
    if u == [1, 0] and v == [1, 1]:
        return [3, 1]
    return [u[0] + v[0], u[1] + v[1]]


def _add_zero_bump(u: list[float], v: list[float]) -> list[float]:
    """两侧加上 [0, 0] 时给第一个分量加 1。v + (-v) 不会以零向量为加数。"""
    zero = [0, 0]
    if u == zero and v != zero:
        return [v[0] + 1, v[1]]
    if v == zero and u != zero:
        return [u[0] + 1, u[1]]
    return [u[0] + v[0], u[1] + v[1]]


def _scale_bad_one(c: float, v: list[float]) -> list[float]:
    if c == 1:
        return [2 * x for x in v]
    return [c * x for x in v]


def _scale_bad_neg(c: float, v: list[float]) -> list[float]:
    if c == -1:
        return [0 * x for x in v]
    return [c * x for x in v]


def _scale_skip_sum(c: float, v: list[float]) -> list[float]:
    if v == [1, 1]:
        return [0, 0]
    return [c * x for x in v]


def _scale_kill(target: float) -> Callable[[float, list[float]], list[float]]:
    def scale(c: float, v: list[float]) -> list[float]:
        if c == target:
            return [0 * x for x in v]
        return [c * x for x in v]

    return scale


def cases() -> Iterator[Case]:
    """逐条产出第 16 集与公理清单的数值用例。"""
    from abstract import (
        add_values,
        addition_is_associative,
        addition_is_commutative,
        apply_on_basis,
        axioms_hold,
        derivative_matrix,
        has_additive_inverse,
        is_linear_on,
        matvec,
        poly_add,
        poly_close,
        poly_derivative,
        poly_eval,
        poly_scale,
        poly_values,
        preserves_addition,
        preserves_scaling,
        scale_by_one_is_identity,
        scale_distributes_over_addition,
        scale_distributes_over_scalar_addition,
        scale_is_compatible,
        scale_values,
        values_close,
        zero_is_identity,
        zero_poly,
        zero_values,
    )

    # 按点相加、按点放缩、零函数
    yield "add at x=-4", add_values([9], [-65616]) == [-65607]
    sample = [9]
    other = [-65616]
    total = add_values(sample, other)
    sample[0] = 0
    yield "add_values does not alias", total == [-65607]
    yield "add_values keeps the other list", other == [-65616]
    yield "coordinate-wise scale", scale_values(2, [1, 2, 3]) == [2, 4, 6]
    original = [1, 2]
    scaled = scale_values(3, original)
    original[0] = 0
    yield "scale by 0 is the zero sample", scale_values(0, [9, -65616]) == [0, 0]
    yield "scale_values does not alias", scaled == [3, 6]
    yield "zero on three points", zero_values(3) == [0, 0, 0]
    yield "zero on an empty grid", zero_values(0) == []
    yield "sample plus zero", add_values([9, -65616], zero_values(2)) == [9, -65616]
    grid = [1, 2]
    yield "values_close exact", values_close(grid, [1.0, 2.0]) is True
    yield "values_close within tol", values_close([1, 0], [1 + 1e-12, 0]) is True
    yield "values_close outside tol", values_close([1, 0], [1 + 1e-6, 0]) is False
    yield "values_close rejects a shorter list", values_close([1, 0], [1]) is False

    # 多项式这个「像向量的对象」
    yield "lesson polynomials add", poly_add(F_POLY, G_POLY) == [5, 3, -4, 0, 0, 0, 0, 4]
    copied_f = F_POLY[:]
    copied_g = G_POLY[:]
    poly_add(copied_f, copied_g)
    yield "poly_add does not mutate", copied_f == F_POLY and copied_g == G_POLY
    yield "cancellation is the zero polynomial", poly_add([1, -1], [-1, 1]) == [0]
    yield "middle zeros stay", poly_add([1, 0, 2], [0]) == [1, 0, 2]
    yield "tiny trailing coefficient drops", poly_scale(1, [1, 0, 5e-13]) == [1]
    yield "coefficient above the cutoff stays", poly_scale(1, [1, 0, 5e-11]) == [1, 0, 5e-11]
    yield "scale the quadratic", poly_scale(2, F_POLY) == [10, 6, 2]
    yield "scale a polynomial by 0", poly_scale(0, F_POLY) == [0]
    yield "zero polynomial", zero_poly() == [0]
    yield "polynomial plus zero", poly_add(F_POLY, zero_poly()) == F_POLY
    yield "eval at the lesson input", poly_eval(F_POLY, X_DEMO) == 9
    yield "eval the degree-7 polynomial", poly_eval(G_POLY, X_DEMO) == -65616
    yield "constant term is the value at 0", poly_eval(F_POLY, 0) == 5
    inputs = [-4, 0, 1, 2]
    yield "pointwise add matches coefficient add", close(
        add_values(poly_values(F_POLY, inputs), poly_values(G_POLY, inputs)),
        poly_values(poly_add(F_POLY, G_POLY), inputs),
    )
    yield "pointwise scale matches coefficient scale", close(
        scale_values(-2, poly_values(F_POLY, inputs)),
        poly_values(poly_scale(-2, F_POLY), inputs),
    )
    yield "zero polynomial samples as zeros", poly_values(zero_poly(), inputs) == zero_values(len(inputs))

    # 导数，以及它的矩阵
    walked = poly_derivative(WALK)
    yield "derivative of the walked polynomial", walked == [4, 10, 3]
    walked_input = WALK[:]
    poly_derivative(walked_input)
    yield "poly_derivative does not mutate", walked_input == WALK
    cubic_image = poly_derivative(CUBIC)
    yield "derivative of (1/9)x^3 - x", close(cubic_image, [-1, 0, 1 / 3])
    yield "derivative of 4x^7 - 5x^2", poly_derivative(G_POLY) == [0, -10, 0, 0, 0, 0, 28]
    yield "derivative of 1", poly_derivative([1]) == [0]
    yield "derivative of x", poly_derivative([0, 1]) == [1]
    yield "derivative of x^2", poly_derivative([0, 0, 1]) == [0, 2]
    yield "derivative of x^3", poly_derivative([0, 0, 0, 1]) == [0, 0, 3]
    yield "derivative of a constant", poly_derivative([7]) == [0]
    yield "derivative of zero", poly_derivative([0]) == [0]
    yield "trailing zeros in the input do not count", poly_derivative([1, 2, 0, 0]) == [2]
    yield "second derivative", poly_derivative([4, 10, 3]) == [10, 6]
    yield "derivative matrix for degree < 4", derivative_matrix(4) == [
        [0, 1, 0, 0],
        [0, 0, 2, 0],
        [0, 0, 0, 3],
        [0, 0, 0, 0],
    ]
    fresh = derivative_matrix(4)
    fresh[0][1] = 99
    yield "derivative matrix rows are new lists", derivative_matrix(4)[0] == [0, 1, 0, 0]
    yield "constants only", derivative_matrix(1) == [[0]]
    matrix_image = matvec(derivative_matrix(4), WALK)
    yield "lesson matrix-vector product", matrix_image == [4, 10, 3, 0]
    coords = WALK[:]
    image = matvec(derivative_matrix(4), coords)
    coords[0] = 0
    yield "matvec does not mutate the vector", coords[0] == 0 and image == [4, 10, 3, 0]
    yield "matrix agrees with the normalized derivative", poly_close([4, 10, 3], matrix_image) is True
    yield "degree 7 through an 8-by-8 block", matvec(derivative_matrix(8), G_POLY) == [
        0, -10, 0, 0, 0, 0, 28, 0,
    ]
    columns = [[row[j] for row in derivative_matrix(4)] for j in range(4)]
    basis_image = [[0], [1]]
    basis_coords = [5, 3]
    yield "linear map from basis images", apply_on_basis([[0], [1], [0, 2], [0, 0, 3]], WALK) == [4, 10, 3]
    yield "monomial basis keeps the polynomial", apply_on_basis([[1], [0, 1], [0, 0, 1]], F_POLY) == F_POLY
    yield "matrix columns are those images", apply_on_basis(columns, WALK) == [4, 10, 3]
    yield "empty linear combination is zero", apply_on_basis([], []) == [0]
    apply_on_basis(basis_image, basis_coords)
    yield "apply_on_basis does not mutate", basis_image == [[0], [1]] and basis_coords == [5, 3]

    # 两条线性公理，用导数和两个反例
    yield "derivative preserves this sum", close(
        poly_derivative(poly_add(WALK, CUBIC)),
        poly_add(poly_derivative(WALK), poly_derivative(CUBIC)),
    )
    yield "derivative preserves scaling by 9", close(
        poly_derivative(poly_scale(9, CUBIC)),
        poly_scale(9, poly_derivative(CUBIC)),
    )
    yield "preserves_addition on the derivative", preserves_addition(
        poly_derivative, WALK, CUBIC, poly_add, poly_close,
    ) is True
    yield "preserves_scaling on the derivative", preserves_scaling(
        poly_derivative, 9, CUBIC, poly_scale, poly_close,
    ) is True
    yield "translation breaks addition", preserves_addition(
        lambda v: [x + 1 for x in v],
        [9],
        [-65616],
        add_values,
        values_close,
    ) is False
    yield "absolute value breaks scaling", preserves_scaling(
        lambda v: [abs(x) for x in v],
        -1,
        [1, -2],
        scale_values,
        values_close,
    ) is False
    yield "derivative is linear on the lesson polynomials", is_linear_on(
        poly_derivative,
        [WALK, CUBIC, F_POLY],
        [9, 0, -1, 2],
        poly_add,
        poly_scale,
        poly_close,
    ) is True
    yield "doubling samples is linear", is_linear_on(
        lambda v: scale_values(2, v),
        [[1, -2], [3, 4]],
        [0, -3, 2],
        add_values,
        scale_values,
        values_close,
    ) is True
    yield "projection is linear", is_linear_on(
        lambda v: [v[0], 0],
        [[1, 2], [3, -4]],
        [2, 0, -1],
        add_values,
        scale_values,
        values_close,
    ) is True
    yield "a trailing zero still names the same polynomial", is_linear_on(
        lambda p: list(p) + [0],
        [F_POLY, [0, -1]],
        [2, -1],
        poly_add,
        poly_scale,
        poly_close,
    ) is True
    yield "translation is not linear", is_linear_on(
        lambda v: [x + 1 for x in v],
        [[9], [-65616]],
        [2, 0],
        add_values,
        scale_values,
        values_close,
    ) is False
    yield "squaring outputs is not linear", is_linear_on(
        lambda v: [x * x for x in v],
        [[1, -2], [3, 0]],
        [2],
        add_values,
        scale_values,
        values_close,
    ) is False
    yield "empty scalar list still checks addition", is_linear_on(
        lambda v: [2 * x for x in v],
        [[1, 0]],
        [],
        add_values,
        scale_values,
        values_close,
    ) is True
    yield "the empty grid is a vector space", axioms_hold(
        [[]], [2], [], add_values, scale_values, values_close,
    ) is True

    # 八条公理：普通样本为真，每条各有一个只打破它的运算
    samples = [[1, 0], [0, 1]]
    sample_scalars = [2, -3]
    yield "samples commute", addition_is_commutative(samples, add_values, values_close) is True
    yield "samples associate", addition_is_associative(samples, add_values, values_close) is True
    yield "sample zero", zero_is_identity(samples, [0, 0], add_values, values_close) is True
    yield "sample inverses", has_additive_inverse(
        samples, [0, 0], add_values, scale_values, values_close,
    ) is True
    yield "scale over vector addition", scale_distributes_over_addition(
        samples, sample_scalars, add_values, scale_values, values_close,
    ) is True
    yield "scale over scalar addition", scale_distributes_over_scalar_addition(
        samples, sample_scalars, add_values, scale_values, values_close,
    ) is True
    yield "scales compose", scale_is_compatible(samples, sample_scalars, scale_values, values_close) is True
    yield "scale by one", scale_by_one_is_identity(samples, scale_values, values_close) is True
    yield "left projection is not commutative", addition_is_commutative(
        [[9], [-65616]], lambda u, v: list(u), values_close,
    ) is False
    yield "subtraction is not associative", addition_is_associative(
        [[1, 0], [0, 1]], lambda u, v: [a - b for a, b in zip(u, v)], values_close,
    ) is False
    yield "tol accepts 1e-12", zero_is_identity(
        [[1, 0]],
        [0, 0],
        lambda u, v: [u[0] + v[0] + 1e-12, u[1] + v[1]],
        values_close,
        1e-9,
    ) is True
    yield "tol rejects 1e-6", zero_is_identity(
        [[1, 0]],
        [0, 0],
        lambda u, v: [u[0] + v[0] + 1e-6, u[1] + v[1]],
        values_close,
        1e-9,
    ) is False

    yield "function samples satisfy the eight axioms", axioms_hold(
        [[9, 1], [-65616, 0]],
        [2, -1, 0],
        [0, 0],
        add_values,
        scale_values,
        values_close,
    ) is True
    yield "lesson polynomials satisfy the eight axioms", axioms_hold(
        [F_POLY, G_POLY],
        [2, -1, 0],
        zero_poly(),
        poly_add,
        poly_scale,
        poly_close,
    ) is True

    scenarios = [
        ("commutative break is visible to axioms_hold", [[9], [-65616]], [2], [0], lambda u, v: list(u), _std_scale),
        ("only associativity fails", samples, [2], [0, 0], _add_assoc_bump, _std_scale),
        ("only the zero axiom fails", samples, [2], [0, 0], _add_zero_bump, _std_scale),
        ("only inverses fail", samples, [2, 3], [0, 0], _std_add, _scale_bad_neg),
        ("only scaling a sum fails", samples, [2], [0, 0], _std_add, _scale_skip_sum),
        ("only scaling a sum of scalars fails", [[1, 0]], [4], [0, 0], _std_add, _scale_kill(8)),
        ("only compatibility fails", [[1, 0]], [4], [0, 0], _std_add, _scale_kill(16)),
        ("only scaling by one fails", samples, [2, 3], [0, 0], _std_add, _scale_bad_one),
    ]
    for name, vectors, scalars, zero, add, scale in scenarios:
        flags = [
            addition_is_commutative(vectors, add, values_close),
            addition_is_associative(vectors, add, values_close),
            zero_is_identity(vectors, zero, add, values_close),
            has_additive_inverse(vectors, zero, add, scale, values_close),
            scale_distributes_over_addition(vectors, scalars, add, scale, values_close),
            scale_distributes_over_scalar_addition(vectors, scalars, add, scale, values_close),
            scale_is_compatible(vectors, scalars, scale, values_close),
            scale_by_one_is_identity(vectors, scale, values_close),
        ]
        got = axioms_hold(vectors, scalars, zero, add, scale, values_close)
        yield name, got is False and got is all(flags)

    only = {
        "only associativity fails": 1,
        "only the zero axiom fails": 2,
        "only inverses fail": 3,
        "only scaling a sum fails": 4,
        "only scaling a sum of scalars fails": 5,
        "only compatibility fails": 6,
        "only scaling by one fails": 7,
    }
    for name, vectors, scalars, zero, add, scale in scenarios:
        if name not in only:
            continue
        flags = [
            addition_is_commutative(vectors, add, values_close),
            addition_is_associative(vectors, add, values_close),
            zero_is_identity(vectors, zero, add, values_close),
            has_additive_inverse(vectors, zero, add, scale, values_close),
            scale_distributes_over_addition(vectors, scalars, add, scale, values_close),
            scale_distributes_over_scalar_addition(vectors, scalars, add, scale, values_close),
            scale_is_compatible(vectors, scalars, scale, values_close),
            scale_by_one_is_identity(vectors, scale, values_close),
        ]
        index = only[name]
        yield name + " isolates that axiom", flags[index] is False and all(
            flag for i, flag in enumerate(flags) if i != index
        )

    # 异常必须抛出
    yield "add_values length", raises(ValueError, add_values, [1, 2], [3])
    yield "zero_values negative", raises(ValueError, zero_values, -1)
    yield "zero_values bool", raises(ValueError, zero_values, True)
    yield "zero_values float", raises(ValueError, zero_values, 1.5)
    yield "poly_add empty", raises(ValueError, poly_add, [], [0])
    yield "poly_scale empty", raises(ValueError, poly_scale, 2, [])
    yield "poly_eval empty", raises(ValueError, poly_eval, [], 0)
    yield "poly_values empty", raises(ValueError, poly_values, [], [0])
    yield "poly_derivative empty", raises(ValueError, poly_derivative, [])
    yield "derivative_matrix zero", raises(ValueError, derivative_matrix, 0)
    yield "derivative_matrix negative", raises(ValueError, derivative_matrix, -3)
    yield "derivative_matrix bool", raises(ValueError, derivative_matrix, True)
    yield "matvec empty", raises(ValueError, matvec, [], [1])
    yield "matvec ragged", raises(ValueError, matvec, [[1, 2], [3]], [1, 2])
    yield "matvec bad width", raises(ValueError, matvec, [[1, 2], [3, 4]], [1])
    yield "apply_on_basis length", raises(ValueError, apply_on_basis, [[1]], [])
    yield "apply_on_basis empty image", raises(ValueError, apply_on_basis, [[]], [1])
    yield "is_linear_on empty", raises(
        ValueError, is_linear_on, poly_derivative, [], [1], poly_add, poly_scale, poly_close,
    )
    yield "mixed sample lengths", raises(
        ValueError,
        is_linear_on,
        lambda v: v,
        [[1, 2], [3]],
        [1],
        add_values,
        scale_values,
        values_close,
    )
    yield "commutative empty", raises(ValueError, addition_is_commutative, [], add_values, values_close)
    yield "associative empty", raises(ValueError, addition_is_associative, [], add_values, values_close)
    yield "zero axiom empty", raises(ValueError, zero_is_identity, [], [0], add_values, values_close)
    yield "inverse empty", raises(
        ValueError, has_additive_inverse, [], [0], add_values, scale_values, values_close,
    )
    yield "distribute over vectors empty scalar", raises(
        ValueError,
        scale_distributes_over_addition,
        [[1]],
        [],
        add_values,
        scale_values,
        values_close,
    )
    yield "compatibility empty", raises(ValueError, scale_is_compatible, [], [2], scale_values, values_close)
    yield "axioms empty vectors", raises(
        ValueError, axioms_hold, [], [1], [0], add_values, scale_values, values_close,
    )
    yield "axioms empty scalars", raises(
        ValueError, axioms_hold, [[1]], [], [0], add_values, scale_values, values_close,
    )


def _as_list(value: Any) -> list[float]:
    if hasattr(value, "tolist"):
        value = value.tolist()
    return [float(x) for x in value]


def same_poly(got: Any, expected: Any, tol: float = 1e-8) -> bool:
    """忽略高次上的 0，比较两个多项式。"""
    left = _as_list(got)
    right = _as_list(expected)
    n = max(len(left), len(right), 1)
    left.extend([0.0] * (n - len(left)))
    right.extend([0.0] * (n - len(right)))
    return close(left, right, tol)


def _normalized(coeffs: list[float]) -> bool:
    if coeffs == [0] or coeffs == [0.0]:
        return True
    return len(coeffs) >= 1 and abs(coeffs[-1]) > 1e-12


def parity(n: int = 200, seed: int = 1616) -> list[str] | None:
    """与 numpy.polynomial 随机对拍 n 组；未安装 numpy 时返回 None。"""
    try:
        import numpy as np
        from numpy.polynomial.polynomial import polyadd, polyder, polyval
    except ImportError:
        return None
    from abstract import (
        add_values,
        derivative_matrix,
        is_linear_on,
        matvec,
        poly_add,
        poly_close,
        poly_derivative,
        poly_eval,
        poly_scale,
        scale_values,
    )

    rng = random.Random(seed)
    fails: list[str] = []

    def rand_poly() -> list[int]:
        return [rng.randint(-4, 4) for _ in range(rng.randint(1, 6))]

    for _ in range(n):
        p = rand_poly()
        q = rand_poly()
        scalar = rng.choice([-2, -1, 0, 1, 2, 3])
        added = poly_add(p, q)
        scaled = poly_scale(scalar, p)
        derived = poly_derivative(p)
        if not _normalized(added) or not _normalized(scaled) or not _normalized(derived):
            fails.append(f"not normalized {p} {q}")
            continue
        if not same_poly(added, polyadd(np.array(p, dtype=float), np.array(q, dtype=float))):
            fails.append(f"poly_add {p}+{q}")
        if not same_poly(scaled, scalar * np.array(p, dtype=float)):
            fails.append(f"poly_scale {scalar}*{p}")
        if not same_poly(derived, polyder(np.array(p, dtype=float))):
            fails.append(f"poly_derivative {p}")
        for x in (-2, -1, 0, 1, 2):
            if not close(poly_eval(p, x), float(polyval(x, np.array(p, dtype=float)))):
                fails.append(f"poly_eval {p} at {x}")
                break
        width = len(p)
        padded = matvec(derivative_matrix(width), p)
        expected = _as_list(polyder(np.array(p, dtype=float)))
        expected.extend([0.0] * (width - len(expected)))
        if not close(padded, expected[:width]):
            fails.append(f"derivative_matrix {p}")
        samples_f = [rng.randint(-5, 5) for _ in range(4)]
        samples_g = [rng.randint(-5, 5) for _ in range(4)]
        factor = rng.uniform(-3, 3)
        if not close(add_values(samples_f, samples_g), (np.array(samples_f) + np.array(samples_g)).tolist()):
            fails.append("add_values random")
        if not close(scale_values(factor, samples_f), (factor * np.array(samples_f, dtype=float)).tolist()):
            fails.append("scale_values random")
        triple = [p, q, rand_poly()]
        if not is_linear_on(
            poly_derivative, triple, [0, 1, -2, 0.5], poly_add, poly_scale, poly_close,
        ):
            fails.append(f"derivative lost linearity on {triple}")
    return fails


def run_doctests() -> bool:
    """运行 abstract.py 的 doctest；返回是否全部通过。"""
    import abstract

    result = doctest.testmod(abstract, verbose=False, optionflags=doctest.REPORT_ONLY_FIRST_FAILURE)
    status = "PASS" if result.failed == 0 else "FAIL"
    print(f"{status} doctest {abstract.__name__}: {result.attempted - result.failed}/{result.attempted}")
    return result.failed == 0


def main() -> None:
    ok = True
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL abstract.py 尚未实现（NotImplementedError）")
        ok = False
    except Exception:
        traceback.print_exc()
        ok = False
    if ok:
        fails = parity()
        if fails is None:
            print("SKIP parity: numpy 未安装（主用例已通过）")
        elif fails:
            ok = False
            for item in fails[:10]:
                print("FAIL parity: " + item)
            print(f"parity failures: {len(fails)}")
        else:
            print("PASS parity: 200 组随机对拍与 numpy 一致")
    ok = run_doctests() and ok
    print("RESULT:", "PASS" if ok else "FAIL")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
