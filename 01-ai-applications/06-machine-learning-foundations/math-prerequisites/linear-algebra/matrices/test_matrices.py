#!/usr/bin/env python3
"""变换与矩阵验证器：3Blue1Brown 第 3–5 集的数值示例 + numpy 随机对拍 + doctest。

运行（仓库根目录）：make test-linalg-matrices
实现由本人手写（C5）。不得放宽容差、删除用例，或用库函数替换 matrices.py 中的实现。
骨架状态下本脚本预期失败；不得改为 skip。
"""
import doctest
import math
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


def close(a: Any, b: Any, tol: float = 1e-7) -> bool:
    """数值或数值序列是否在相对容差内相等。

    Examples:
        >>> close([[1.0, 0.0], [0.0, 1.0]], [[1, 0], [0, 1]])
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
    """逐条产出第 3–5 集与 A 组衔接的数值用例。"""
    from matrices import (apply_3x3, apply_matrix, compose, identity, rotate_2d, scale_2d,
                          shear_2d)

    # 第 3 集：列 = 基向量落点
    yield "ch3: [-1,2] -> [5,2]", close(apply_matrix([[1, 3], [-2, 0]], [-1, 2]), [5, 2])
    yield "ch3: 90 deg CCW [1,0] -> [0,1]", close(apply_matrix([[0, -1], [1, 0]], [1, 0]), [0, 1])
    yield "ch3: 90 deg CCW [1,1] -> [-1,1]", close(apply_matrix([[0, -1], [1, 0]], [1, 1]), [-1, 1])
    yield "zero vector stays zero", close(apply_matrix([[1, 3], [-2, 0]], [0, 0]), [0, 0])
    yield "identity leaves [3,-2]", close(apply_matrix(identity(2), [3, -2]), [3, -2])

    # 与 A 组 in_span 衔接：1·[1,0,1] + 2·[0,1,1] = [1,2,3]
    yield "span bridge [1,2] -> [1,2,3]", close(apply_matrix([[1, 0], [0, 1], [1, 1]], [1, 2]), [1, 2, 3])

    # 二维构造
    yield "scale_2d(2, 3)", close(scale_2d(2, 3), [[2, 0], [0, 3]])
    yield "scale_2d(1, 1) is identity", close(scale_2d(1, 1), identity(2))
    yield "scale_2d(0, 0)", close(scale_2d(0, 0), [[0, 0], [0, 0]])
    yield "apply scale [1,2] -> [2,1]", close(apply_matrix(scale_2d(2, 0.5), [1, 2]), [2, 1])
    yield "shear_2d(1)", close(shear_2d(1), [[1, 1], [0, 1]])
    yield "shear_2d(0) is identity", close(shear_2d(0), identity(2))
    yield "shear_2d(-1)", close(shear_2d(-1), [[1, -1], [0, 1]])
    yield "apply shear [1,1] -> [2,1]", close(apply_matrix(shear_2d(1), [1, 1]), [2, 1])
    yield "rotate 0", close(rotate_2d(0), identity(2))
    yield "rotate 90", close(rotate_2d(math.pi / 2), [[0, -1], [1, 0]])
    yield "rotate 180", close(rotate_2d(math.pi), [[-1, 0], [0, -1]])
    yield "apply rotate [1,0] -> [0,1]", close(apply_matrix(rotate_2d(math.pi / 2), [1, 0]), [0, 1])

    # 第 4 集：先 B 后 A
    shear = [[1, 1], [0, 1]]
    rot = [[0, -1], [1, 0]]
    yield "shear after rotation", close(compose(shear, rot), [[1, -1], [1, 0]])
    yield "rotation after shear", close(compose(rot, shear), [[0, -1], [1, 1]])
    yield "ch4: M2 after M1", close(compose([[0, 2], [1, 0]], [[1, -2], [1, 0]]), [[2, 0], [1, -2]])
    yield "compose([[2]], [[3]])", close(compose([[2]], [[3]]), [[6]])
    yield "constructors match literal compose", close(
        compose(shear_2d(1), rotate_2d(math.pi / 2)), [[1, -1], [1, 0]])
    yield "successive apply matches compose", close(
        apply_matrix(compose(shear, rot), [1, 0]),
        apply_matrix(shear, apply_matrix(rot, [1, 0])),
    )
    yield "rot then shear on [1,0] -> [1,1]", close(apply_matrix(compose(shear, rot), [1, 0]), [1, 1])

    # 第 5 集
    yaw = [[0, 0, 1], [0, 1, 0], [-1, 0, 0]]
    lesson = [[0, 0.5, -0.5], [0, 0.5, 1], [1, 0, 0.5]]
    roll = [[0, -1, 0], [1, 0, 0], [0, 0, 1]]
    yield "yaw [3,4,5] -> [5,4,-3]", close(apply_3x3(yaw, [3, 4, 5]), [5, 4, -3])
    yield "lesson [1,0,-2] -> [1,-2,0]", close(apply_3x3(lesson, [1, 0, -2]), [1, -2, 0])
    yield "roll [1,0,0] -> [0,1,0]", close(apply_3x3(roll, [1, 0, 0]), [0, 1, 0])
    yield "apply_3x3 matches apply_matrix", close(apply_3x3(yaw, [3, 4, 5]), apply_matrix(yaw, [3, 4, 5]))
    yield "identity(1)", close(identity(1), [[1]])
    yield "identity(3) leaves [1,2,3]", close(apply_3x3(identity(3), [1, 2, 3]), [1, 2, 3])

    # 异常：必须抛出，不能 skip
    yield "apply dim mismatch", raises(ValueError, apply_matrix, [[1, 0], [0, 1]], [1, 2, 3])
    yield "ragged matrix", raises(ValueError, apply_matrix, [[1, 0], [1]], [1, 0])
    yield "empty matrix", raises(ValueError, apply_matrix, [], [1])
    yield "compose inner mismatch", raises(ValueError, compose, [[1, 0], [0, 1]], [[1, 0, 0]])
    yield "apply_3x3 rejects 2x2", raises(ValueError, apply_3x3, [[1, 0], [0, 1]], [1, 0])
    yield "apply_3x3 rejects short vector", raises(ValueError, apply_3x3, identity(3), [1, 2])
    yield "identity(0)", raises(ValueError, identity, 0)


def parity(n: int = 200, seed: int = 1653) -> list[str] | None:
    """与 numpy 随机对拍 n 组；返回失败描述列表，未安装 numpy 时返回 None。"""
    try:
        import numpy as np
    except ImportError:
        return None
    from matrices import apply_3x3, apply_matrix, compose, identity, rotate_2d, scale_2d, shear_2d

    rng = random.Random(seed)
    fails: list[str] = []

    def random_vector(k: int) -> list[int]:
        return [rng.randint(-5, 5) for _ in range(k)]

    def random_matrix(rows: int, cols: int) -> list[list[int]]:
        return [random_vector(cols) for _ in range(rows)]

    for _ in range(n):
        rows, cols = rng.choice([(2, 2), (3, 2), (2, 3), (3, 3)])
        matrix = random_matrix(rows, cols)
        vector = random_vector(cols)
        expected = (np.array(matrix, dtype=float) @ np.array(vector, dtype=float)).tolist()
        if not close(apply_matrix(matrix, vector), expected, 1e-9):
            fails.append(f"apply_matrix {matrix} @ {vector}")

        left_cols = rng.choice([2, 3])
        left = random_matrix(rng.choice([2, 3]), left_cols)
        right = random_matrix(left_cols, rng.choice([2, 3]))
        expected_prod = (np.array(left, dtype=float) @ np.array(right, dtype=float)).tolist()
        if not close(compose(left, right), expected_prod, 1e-9):
            fails.append("compose random case")

        theta = rng.uniform(-math.pi, math.pi)
        cosine, sine = math.cos(theta), math.sin(theta)
        if not close(rotate_2d(theta), [[cosine, -sine], [sine, cosine]], 1e-12):
            fails.append(f"rotate_2d {theta}")

        sx, sy = rng.uniform(-3, 3), rng.uniform(-3, 3)
        if not close(scale_2d(sx, sy), [[sx, 0.0], [0.0, sy]]):
            fails.append(f"scale_2d {sx},{sy}")

        factor = rng.uniform(-3, 3)
        if not close(shear_2d(factor), [[1.0, factor], [0.0, 1.0]]):
            fails.append(f"shear_2d {factor}")

        dim = rng.randint(1, 5)
        if not close(identity(dim), np.eye(dim).tolist()):
            fails.append(f"identity {dim}")

        cube = random_matrix(3, 3)
        vector3 = random_vector(3)
        expected3 = (np.array(cube, dtype=float) @ np.array(vector3, dtype=float)).tolist()
        if not close(apply_3x3(cube, vector3), expected3, 1e-9):
            fails.append("apply_3x3 random case")
        if not close(apply_3x3(cube, vector3), apply_matrix(cube, vector3), 1e-9):
            fails.append("apply_3x3 disagrees with apply_matrix")

        square_a, square_b = random_matrix(2, 2), random_matrix(2, 2)
        probe = random_vector(2)
        via_product = apply_matrix(compose(square_a, square_b), probe)
        via_steps = apply_matrix(square_a, apply_matrix(square_b, probe))
        if not close(via_product, via_steps, 1e-9):
            fails.append("compose is not apply-B-then-A")
    return fails


def run_doctests() -> bool:
    """运行 matrices.py 的 doctest；返回是否全部通过。"""
    import matrices

    result = doctest.testmod(matrices, verbose=False, optionflags=doctest.REPORT_ONLY_FIRST_FAILURE)
    status = "PASS" if result.failed == 0 else "FAIL"
    print(f"{status} doctest {matrices.__name__}: {result.attempted - result.failed}/{result.attempted}")
    return result.failed == 0


def main() -> None:
    ok = True
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL matrices.py 尚未实现（NotImplementedError）")
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
