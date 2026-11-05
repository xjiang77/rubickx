#!/usr/bin/env python3
"""维度之间验证器：3Blue1Brown 第 8–11 集的数值示例 + numpy 随机对拍 + doctest。

运行（仓库根目录）：make test-linalg-dimensions
实现由本人手写（C5）。不得放宽容差、删除用例，或用库函数替换 dimensions.py 中的实现。
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

Case = tuple[str, bool]
"""一条用例：(名称, 是否通过)。"""

LESSON_3X2 = [[2, 0], [-1, 1], [-2, 1]]
"""第 8 集：î → (2, −1, −2)，ĵ → (0, 1, 1)，二维到三维。"""

WIDE = [[3, 1, 4], [1, 5, 9]]
"""第 8 集画面上的 2×3：三维到二维。"""

SPAN = [[1, 0], [0, 1], [1, 1]]
"""A 组张成：列是 [1, 0, 1] 与 [0, 1, 1]。"""


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
    """逐条产出第 8–11 集与 A 组衔接的数值用例。"""
    from dimensions import apply_between, cross_product, dot_product, dual_vector, project

    # 第 8 集：维数变化
    yield "2d to 3d: i-hat", apply_between(LESSON_3X2, [1, 0]) == [2, -1, -2]
    yield "2d to 3d: j-hat", apply_between(LESSON_3X2, [0, 1]) == [0, 1, 1]
    yield "2d to 3d: (1, 1)", apply_between(LESSON_3X2, [1, 1]) == [2, 0, -1]
    yield "2d to 3d: zero", apply_between(LESSON_3X2, [0, 0]) == [0, 0, 0]
    yield "3d to 2d: i-hat", apply_between(WIDE, [1, 0, 0]) == [3, 1]
    yield "3d to 2d: j-hat", apply_between(WIDE, [0, 1, 0]) == [1, 5]
    yield "3d to 2d: k-hat", apply_between(WIDE, [0, 0, 1]) == [4, 9]
    yield "3d to 2d: (1, 1, 1)", apply_between(WIDE, [1, 1, 1]) == [8, 15]
    yield "2d to 1d: lesson 9", apply_between([[1, -2]], [4, 3]) == [-2]
    yield "2d to 1d: i-hat lands on 1", apply_between([[1, 2]], [1, 0]) == [1]
    yield "2d to 1d: j-hat lands on 2", apply_between([[1, 2]], [0, 1]) == [2]
    yield "A span as 2d to 3d", apply_between(SPAN, [1, 2]) == [1, 2, 3]
    yield "A span other combination", apply_between(SPAN, [3, -1]) == [3, -1, 2]
    yield "1d to 3d", apply_between([[2], [-1], [-2]], [3]) == [6, -3, -6]
    yield "square is the same formula", apply_between([[1, 0], [0, 1]], [3, -2]) == [3, -2]
    copied = [row[:] for row in LESSON_3X2]
    vector = [1, 1]
    apply_between(copied, vector)
    yield "apply_between does not mutate", copied == LESSON_3X2 and vector == [1, 1]

    # 第 9 集：点积
    yield "dot (1, 2) with (3, 4)", dot_product([1, 2], [3, 4]) == 11
    yield "dot is symmetric", dot_product([3, 4], [1, 2]) == 11
    yield "dot scales", dot_product([2, 4], [3, 4]) == 22
    yield "dot 4d lesson", dot_product([6, 2, 8, 3], [1, 8, 5, 3]) == 71
    yield "dot duality pair", dot_product([1, -2], [4, 3]) == -2
    yield "dot perpendicular", dot_product([1, 2], [2, -1]) == 0
    yield "dot zero vector", dot_product([0, 0], [5, 7]) == 0
    yield "dot with self is length squared", dot_product([3, 4], [3, 4]) == 25
    yield "dot of A columns", dot_product([1, 0, 1], [0, 1, 1]) == 1
    yield "dot of A target", dot_product([1, 2, 3], [1, 0, 1]) == 4

    # 投影到一维
    yield "project onto x-axis", project([4, 3], [1, 0]) == [4, 0]
    yield "project onto a multiple", project([3, 4], [6, 8]) == [3, 4]
    yield "project onto the opposite direction", project([3, 4], [-6, -8]) == [3, 4]
    yield "project perpendicular is zero", project([1, 2], [2, -1]) == [0, 0]
    yield "project zero vector", project([0, 0], [3, 4]) == [0, 0]
    yield "project A target onto first column", project([1, 2, 3], [1, 0, 1]) == [2, 0, 2]
    onto = project([1, 2], [3, 4])
    yield "project lesson pair", close(onto, [33 / 25, 44 / 25])
    yield "projected vector dots back to 11", close(dot_product(onto, [3, 4]), 11)
    onto_line = project([4, 3], [1, -2])
    yield "project duality direction", close(onto_line, [-2 / 5, 4 / 5])
    yield "that projection dots back to -2", close(dot_product(onto_line, [1, -2]), -2)

    # 对偶
    yield "dual of lesson 9", dual_vector([1, -2]) == [1, -2]
    yield "dual of number line", dual_vector([1, 2]) == [1, 2]
    landings = [1, -2]
    got = dual_vector(landings)
    landings.append(99)
    yield "dual returns a new list", got == [1, -2]
    yield "dual dot is the functional", dot_product(dual_vector([1, -2]), [4, 3]) == -2
    yield "dual row matrix matches", apply_between([dual_vector([1, -2])], [4, 3]) == [-2]

    # 第 10–11 集：叉积
    yield "2d unit square", cross_product([1, 0], [0, 1]) == 1
    yield "2d order flips sign", cross_product([0, 1], [1, 0]) == -1
    yield "2d 2 by 3 rectangle", cross_product([3, 0], [0, 2]) == 6
    yield "2d parallel", cross_product([2, -1], [4, -2]) == 0
    yield "i cross j is k", cross_product([1, 0, 0], [0, 1, 0]) == [0, 0, 1]
    yield "j cross i is -k", cross_product([0, 1, 0], [1, 0, 0]) == [0, 0, -1]
    yield "j cross k is i", cross_product([0, 1, 0], [0, 0, 1]) == [1, 0, 0]
    yield "i cross k is -j", cross_product([1, 0, 0], [0, 0, 1]) == [0, -1, 0]
    crossed = cross_product([2, -1, -2], [0, 1, 1])
    yield "cross of lesson columns", crossed == [1, -2, 2]
    yield "cross is perpendicular", (
        dot_product(crossed, [2, -1, -2]) == 0 and dot_product(crossed, [0, 1, 1]) == 0
    )
    yield "cross length squared is 9", dot_product(crossed, crossed) == 9
    yield "3d parallel", cross_product([1, 2, 3], [2, 4, 6]) == [0, 0, 0]
    swapped = cross_product([0, 1, 1], [2, -1, -2])
    yield "3d order flips sign", swapped == [-1, 2, -2]

    # 异常：必须抛出，不能 skip
    yield "apply dim mismatch", raises(ValueError, apply_between, LESSON_3X2, [1, 2, 3])
    yield "apply ragged", raises(ValueError, apply_between, [[1, 2], [3]], [1, 2])
    yield "apply empty", raises(ValueError, apply_between, [], [1])
    yield "apply empty row", raises(ValueError, apply_between, [[]], [])
    yield "dot length mismatch", raises(ValueError, dot_product, [1, 2], [3])
    yield "dot empty", raises(ValueError, dot_product, [], [])
    yield "project zero direction", raises(ValueError, project, [1, 2], [0, 0])
    yield "project length mismatch", raises(ValueError, project, [1, 2, 3], [1, 0])
    yield "project empty", raises(ValueError, project, [], [])
    yield "dual empty", raises(ValueError, dual_vector, [])
    yield "cross 4d", raises(ValueError, cross_product, [1, 2, 3, 4], [5, 6, 7, 8])
    yield "cross length mismatch", raises(ValueError, cross_product, [1, 2], [3, 4, 5])
    yield "cross 1d", raises(ValueError, cross_product, [1], [2])


def parity(n: int = 200, seed: int = 1658) -> list[str] | None:
    """与 numpy 随机对拍 n 组；返回失败描述列表，未安装 numpy 时返回 None。"""
    try:
        import numpy as np
    except ImportError:
        return None
    from dimensions import apply_between, cross_product, dot_product, dual_vector, project

    rng = random.Random(seed)
    fails: list[str] = []
    shapes = [(3, 2), (2, 3), (1, 2), (2, 1), (1, 3), (3, 1), (2, 2), (3, 3)]

    def random_matrix(rows: int, cols: int) -> list[list[int]]:
        return [[rng.randint(-5, 5) for _ in range(cols)] for _ in range(rows)]

    def random_vector(k: int) -> list[int]:
        return [rng.randint(-5, 5) for _ in range(k)]

    def nonzero(k: int) -> list[int]:
        vector = [0] * k
        while all(x == 0 for x in vector):
            vector = random_vector(k)
        return vector

    for _ in range(n):
        rows, cols = rng.choice(shapes)
        matrix = random_matrix(rows, cols)
        vector = random_vector(cols)
        expected = (np.array(matrix, dtype=float) @ np.array(vector, dtype=float)).tolist()
        got = apply_between([row[:] for row in matrix], vector[:])
        if not close(got, expected, 1e-6) or len(got) != rows:
            fails.append(f"apply_between {matrix} @ {vector}")

        dim = rng.choice([1, 2, 3, 4])
        left, right = random_vector(dim), random_vector(dim)
        if not close(dot_product(left, right), float(np.dot(left, right)), 1e-9):
            fails.append(f"dot_product {left} · {right}")
        if not close(dot_product(left, right), dot_product(right, left), 1e-9):
            fails.append(f"dot symmetry {left} {right}")

        direction = nonzero(dim)
        projected = project(left, direction)
        scale = float(np.dot(left, direction)) / float(np.dot(direction, direction))
        expect_proj = (scale * np.array(direction, dtype=float)).tolist()
        if not close(projected, expect_proj, 1e-6):
            fails.append(f"project {left} onto {direction}")
        elif not close(dot_product(projected, direction), dot_product(left, direction), 1e-6):
            fails.append(f"project dots back {left} onto {direction}")

        landings = random_vector(dim)
        dual = dual_vector(landings[:])
        if not close(dual, landings):
            fails.append(f"dual_vector {landings}")
        elif not close(dot_product(dual, left), float(np.dot(landings, left)), 1e-9):
            fails.append(f"dual dot {landings}")
        elif not close(apply_between([landings], left), [float(np.dot(landings, left))], 1e-9):
            fails.append(f"row times vector {landings}")

        for width in (2, 3):
            a, b = random_vector(width), random_vector(width)
            got_cross = cross_product(a, b)
            expect_cross = np.cross(np.array(a, dtype=float), np.array(b, dtype=float))
            if width == 2:
                if not close(got_cross, float(expect_cross), 1e-9):
                    fails.append(f"cross 2d {a} × {b}")
                elif not close(cross_product(b, a), -got_cross, 1e-9):
                    fails.append(f"cross 2d sign {a} × {b}")
            else:
                if not close(got_cross, expect_cross.tolist(), 1e-9):
                    fails.append(f"cross 3d {a} × {b}")
                elif not close(dot_product(got_cross, a), 0) or not close(dot_product(got_cross, b), 0):
                    fails.append(f"cross 3d perpendicular {a} × {b}")
    return fails


def run_doctests() -> bool:
    """运行 dimensions.py 的 doctest；返回是否全部通过。"""
    import dimensions

    result = doctest.testmod(dimensions, verbose=False, optionflags=doctest.REPORT_ONLY_FIRST_FAILURE)
    status = "PASS" if result.failed == 0 else "FAIL"
    print(f"{status} doctest {dimensions.__name__}: {result.attempted - result.failed}/{result.attempted}")
    return result.failed == 0


def main() -> None:
    ok = True
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL dimensions.py 尚未实现（NotImplementedError）")
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
            for fail in fails[:10]:
                print("FAIL parity: " + fail)
            print(f"parity failures: {len(fails)}")
        else:
            print("PASS parity: 200 组随机对拍与 numpy 一致")
    ok = run_doctests() and ok
    print("RESULT:", "PASS" if ok else "FAIL")
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
