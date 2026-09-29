#!/usr/bin/env python3
"""向量与空间验证器：知识正文 02 的数值示例 + numpy 随机对拍 + 各模块的 doctest。

运行（仓库根目录）：make test-linalg
实现由本人手写（C5）。不得放宽容差、删除用例，或用库函数替换 vectors.py 中的实现。
骨架状态下本脚本预期失败；不得改为 skip。
"""
import doctest
import os
import random
import sys
import traceback
from collections.abc import Iterator
from typing import Any

HERE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, HERE)
sys.path.insert(0, os.path.join(HERE, "playground"))

Case = tuple[str, bool]
"""一条用例：(名称, 是否通过)。"""


def close(a: Any, b: Any, tol: float = 1e-7) -> bool:
    """数值或数值序列是否在相对容差内相等。

    Examples:
        >>> close([1.0, 2.0], [1.0, 2.0 + 1e-9])
        True
        >>> close(1.0, 1.1)
        False
    """
    if isinstance(a, (list, tuple)):
        return len(a) == len(b) and all(close(x, y, tol) for x, y in zip(a, b))
    return abs(a - b) <= tol * max(1.0, abs(a), abs(b))


def raises(exc: type[BaseException], fn, *args) -> bool:
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


def cases() -> Iterator[Case]:
    """逐条产出正文 02 的数值用例。"""
    from vectors import (add, coords_in_basis, in_span, is_collinear, is_independent_3,
                         linear_combination, scale, solve_2x2, weighted_sum)

    # 步骤 1：加法与数乘
    yield "add [1,2]+[3,-1]", close(add([1, 2], [3, -1]), [4, 1])
    yield "scale -1.5*[2,-4]", close(scale(-1.5, [2, -4]), [-3, 6])
    yield "2*[1,2]-3*[3,-1] = [-7,7]", close(linear_combination([2, -3], [[1, 2], [3, -1]]), [-7, 7])
    yield "[7,4] = 2*[3,1]+1*[1,2]", close(linear_combination([2, 1], [[3, 1], [1, 2]]), [7, 4])

    # 步骤 3：共线
    yield "[3,1] vs [6,2] collinear", is_collinear([3, 1], [6, 2]) is True
    yield "[2,-4] vs [-3,6] collinear", is_collinear([2, -4], [-3, 6]) is True
    yield "[3,1] vs [1,2] not collinear", is_collinear([3, 1], [1, 2]) is False
    yield "zero vector collinear with anything", is_collinear([0, 0], [5, 7]) is True

    # 步骤 2：解二维方程组
    yield "solve: [7,4] in v=[3,1], w=[1,2] -> (2,1)", close(list(solve_2x2([3, 1], [1, 2], [7, 4])), [2, 1])
    yield "solve: [8,1] in v=[1,2], w=[2,-1] -> (2,3)", close(list(solve_2x2([1, 2], [2, -1], [8, 1])), [2, 3])
    yield "solve: collinear -> None", solve_2x2([3, 1], [6, 2], [7, 4]) is None

    # 步骤 4：换基、张成、线性无关
    yield "coords of [4,2] in {[1,1],[1,-1]} = (3,1)", close(list(coords_in_basis([4, 2], [1, 1], [1, -1])), [3, 1])
    yield "coords of [3,-2] in standard basis", close(list(coords_in_basis([3, -2], [1, 0], [0, 1])), [3, -2])
    yield "coords: collinear basis raises", raises(ValueError, coords_in_basis, [1, 1], [1, 2], [2, 4])
    yield "[1,2,3] in span{[1,0,1],[0,1,1]}", in_span([1, 2, 3], [1, 0, 1], [0, 1, 1]) is True
    yield "[1,2,4] not in span", in_span([1, 2, 4], [1, 0, 1], [0, 1, 1]) is False
    yield "[1,0,1],[0,1,1],[1,1,2] dependent", is_independent_3([1, 0, 1], [0, 1, 1], [1, 1, 2]) is False
    yield "[2,0,0],[0,3,0],[4,6,0] dependent", is_independent_3([2, 0, 0], [0, 3, 0], [4, 6, 0]) is False
    yield "standard basis independent", is_independent_3([1, 0, 0], [0, 1, 0], [0, 0, 1]) is True
    yield "[1,0,1],[0,1,1],[0,0,1] independent", is_independent_3([1, 0, 1], [0, 1, 1], [0, 0, 1]) is True

    # 工程应用：attention 加权平均
    yield "attention: 0.7,0.2,0.1 -> [0.9,0.4]", close(weighted_sum([0.7, 0.2, 0.1], [[1, 0], [0, 1], [2, 2]]), [0.9, 0.4])
    yield "attention: weights must sum to 1", raises(ValueError, weighted_sum, [0.5, 0.7], [[1, 0], [0, 1]])
    yield "attention: weights must be nonnegative", raises(ValueError, weighted_sum, [1.5, -0.5], [[1, 0], [0, 1]])


def parity(n: int = 200, seed: int = 1605) -> list[str] | None:
    """与 numpy 随机对拍 n 组；返回失败描述列表，未安装 numpy 时返回 None。"""
    try:
        import numpy as np
    except ImportError:
        return None
    from vectors import coords_in_basis, in_span, is_independent_3, linear_combination, solve_2x2

    rng = random.Random(seed)

    def random_vector(k: int) -> list[int]:
        return [rng.randint(-5, 5) for _ in range(k)]

    fails: list[str] = []
    for _ in range(n):
        v, w, t = random_vector(2), random_vector(2), random_vector(2)
        matrix = np.array([v, w], dtype=float).T
        got = solve_2x2(v, w, t)
        if abs(np.linalg.det(matrix)) < 1e-9:
            if got is not None:
                fails.append(f"solve_2x2 collinear case {v},{w} returned {got}")
        else:
            expected = np.linalg.solve(matrix, np.array(t, dtype=float))
            if got is None or not close(list(got), list(expected), 1e-6):
                fails.append(f"solve_2x2 {v},{w},{t}: got {got}, expected {expected.tolist()}")
            if not close(list(coords_in_basis(t, v, w)), list(expected), 1e-6):
                fails.append(f"coords_in_basis {t},{v},{w}")

        a, b, c = random_vector(3), random_vector(3), random_vector(3)
        rank_abc = np.linalg.matrix_rank(np.array([a, b, c], dtype=float))
        if is_independent_3(a, b, c) != (rank_abc == 3):
            fails.append(f"is_independent_3 {a},{b},{c}: rank {rank_abc}")
        if np.linalg.matrix_rank(np.array([a, b], dtype=float)) == 2 and in_span(c, a, b) != (rank_abc == 2):
            fails.append(f"in_span {c} in {a},{b}: rank {rank_abc}")

        coeffs = [rng.uniform(-2, 2) for _ in range(3)]
        vecs = [random_vector(4) for _ in range(3)]
        expected = sum(k * np.array(x, dtype=float) for k, x in zip(coeffs, vecs))
        if not close(linear_combination(coeffs, vecs), expected.tolist(), 1e-9):
            fails.append("linear_combination random case")
    return fails


def run_doctests() -> bool:
    """运行 vectors.py 与 playground 各模块的 doctest；返回是否全部通过。"""
    import vectors
    import attention_toy, canvas, display_helpers, draw_vectors, hit_the_target, pixel_blend, svgplot

    ok = True
    for module in (vectors, svgplot, canvas, display_helpers, attention_toy, pixel_blend, draw_vectors, hit_the_target):
        result = doctest.testmod(module, verbose=False)
        status = "PASS" if result.failed == 0 else "FAIL"
        print(f"{status} doctest {module.__name__}: {result.attempted - result.failed}/{result.attempted}")
        ok = ok and result.failed == 0
    return ok


def main() -> None:
    ok = True
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL vectors.py 尚未实现（NotImplementedError）")
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
            for f in fails[:10]:
                print("FAIL parity: " + f)
            print(f"parity failures: {len(fails)}")
        else:
            print("PASS parity: 200 组随机对拍与 numpy 一致")
    ok = run_doctests() and ok
    print("RESULT:", "PASS" if ok else "FAIL")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
