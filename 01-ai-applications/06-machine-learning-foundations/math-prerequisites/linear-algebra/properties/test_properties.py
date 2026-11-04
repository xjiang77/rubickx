#!/usr/bin/env python3
"""变换的性质验证器：3Blue1Brown 第 6–7 集的数值示例 + numpy 随机对拍 + doctest。

运行（仓库根目录）：make test-linalg-properties
实现由本人手写（C5）。不得放宽容差、删除用例，或用库函数替换 properties.py 中的实现。
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

RANK1 = [[1, -2, 4], [-2, 4, -8], [5, -10, 20]]
"""第 7 集提问：三列共线，秩为 1。"""

SINGULAR = [[3, -1], [-9, 3]]
"""第 7 集：行列式为 0，没有逆。"""

SYSTEM2 = [[2, 2], [1, 3]]
"""第 7 集二元方程组的系数。"""

SYSTEM3 = [[2, 5, 3], [4, 0, 8], [1, 3, 0]]
"""第 7 集三元方程组的系数。"""


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


def matvec(M: list[list[float]], v: list[float]) -> list[float]:
    """验证器自用的矩阵乘向量，不代替 properties.py 里的实现。"""
    return [sum(M[i][j] * v[j] for j in range(len(v))) for i in range(len(M))]


def matmul(A: list[list[float]], B: list[list[float]]) -> list[list[float]]:
    """验证器自用的矩阵乘法。"""
    rows, inner, cols = len(A), len(B), len(B[0])
    return [[sum(A[i][k] * B[k][j] for k in range(inner)) for j in range(cols)] for i in range(rows)]


def cases() -> Iterator[Case]:
    """逐条产出第 6–7 集与 A 组衔接的数值用例。"""
    from properties import column_space, determinant, inverse_2x2, is_solvable, null_space, rank

    # 第 6 集：面积与方向
    yield "det scale 3 by 2 = 6", determinant([[3, 0], [0, 2]]) == 6
    yield "det shear = 1", determinant([[1, 1], [0, 1]]) == 1
    yield "det 90 deg = 1", determinant([[0, -1], [1, 0]]) == 1
    yield "det [[1,2],[3,4]] = -2", determinant([[1, 2], [3, 4]]) == -2
    yield "det [[1.5,1],[2,-2]] = -5", determinant([[1.5, 1], [2, -2]]) == -5
    yield "det singular lesson = 0", determinant(SINGULAR) == 0
    yield "det zero matrix = 0", determinant([[0, 0], [0, 0]]) == 0
    yield "det 1x1", determinant([[7]]) == 7
    yield "det 1x1 zero", determinant([[0]]) == 0
    yield "det yaw = 1", determinant([[0, 0, 1], [0, 1, 0], [-1, 0, 0]]) == 1
    flat = [[1.0, 0.0, 1.0], [0.5, 1.0, 1.5], [1.0, 0.0, 1.0]]
    yield "det flat parallelepiped = 0", determinant(flat) == 0
    volume = [[-1.0, 0.0, 0.05], [0.15, 1.0, 0.05], [0.0, 0.25, 1.0]]
    yield "det lesson parallelepiped", round(determinant(volume), 6) == -0.985625
    yield "det 3-eq system = 28", determinant(SYSTEM3) == 28
    shear, rot = [[1, 1], [0, 1]], [[0, -1], [1, 0]]
    yield "det(AB) = det A det B", determinant(matmul(shear, rot)) == determinant(shear) * determinant(rot)

    # 第 7 集：逆
    yield "inverse of 90 deg", close(inverse_2x2([[0, -1], [1, 0]]), [[0, 1], [-1, 0]])
    yield "inverse of shear", close(inverse_2x2([[1, 1], [0, 1]]), [[1, -1], [0, 1]])
    yield "inverse of 180 deg", close(inverse_2x2([[-1, 0], [0, -1]]), [[-1, 0], [0, -1]])
    yield "inverse of identity", close(inverse_2x2([[1, 0], [0, 1]]), [[1, 0], [0, 1]])
    yield "inverse of system2", close(inverse_2x2(SYSTEM2), [[3 / 4, -1 / 2], [-1 / 4, 1 / 2]])
    yield "inverse of scale", close(inverse_2x2([[3, 0], [0, 2]]), [[1 / 3, 0], [0, 1 / 2]])
    yield "singular has no inverse", inverse_2x2(SINGULAR) is None
    yield "zero matrix has no inverse", inverse_2x2([[0, 0], [0, 0]]) is None
    undone = matvec(inverse_2x2([[0, -1], [1, 0]]), matvec([[0, -1], [1, 0]], [1, 2]))
    yield "inverse undoes 90 deg", close(undone, [1, 2])
    solved = matvec(inverse_2x2(SYSTEM2), [-4, -1])
    yield "system2 solution via inverse", close(solved, [-2.5, 0.5])
    yield "system2 solution checks out", close(matvec(SYSTEM2, solved), [-4, -1])

    # 秩、列空间、零空间
    yield "rank scale = 2", rank([[3, 0], [0, 2]]) == 2
    yield "rank singular = 1", rank(SINGULAR) == 1
    yield "rank lesson question = 1", rank(RANK1) == 1
    yield "rank system3 = 3", rank(SYSTEM3) == 3
    plane = [[1, 0, 1], [0, 1, 1], [0, 0, 0]]
    yield "rank plane = 2", rank(plane) == 2
    tall = [[1, 0], [0, 1], [1, 1]]
    yield "rank tall span = 2", rank(tall) == 2
    yield "rank one row = 1", rank([[1, 2, 3]]) == 1
    yield "rank zero = 0", rank([[0, 0], [0, 0]]) == 0
    yield "columns of rank1", column_space(RANK1) == [[1, -2, 5]]
    yield "columns of tall", column_space(tall) == [[1, 0, 1], [0, 1, 1]]
    yield "columns of system2", column_space(SYSTEM2) == [[2, 1], [2, 3]]
    yield "columns of plane", column_space(plane) == [[1, 0, 0], [0, 1, 0]]
    yield "columns skip duplicate", column_space([[1, 2], [2, 4]]) == [[1, 2]]
    yield "columns skip zero column", column_space([[0, 1], [0, 0]]) == [[1, 0]]
    yield "columns of one row", column_space([[1, 2, 3]]) == [[1]]
    yield "columns of zero", column_space([[0, 0], [0, 0]]) == []
    yield "null of rank1", null_space(RANK1) == [[2, 1, 0], [-4, 0, 1]]
    yield "null of duplicate columns", null_space([[1, 2], [2, 4]]) == [[-2, 1]]
    yield "null of singular lesson", close(null_space(SINGULAR), [[1 / 3, 1]])
    yield "null of wide", null_space([[1, 0, 1], [0, 1, 1]]) == [[-1, -1, 1]]
    yield "null of plane", null_space(plane) == [[-1, -1, 1]]
    yield "null of one row", null_space([[1, 2, 3]]) == [[-2, 1, 0], [-3, 0, 1]]
    yield "null of full rank is empty", null_space(SYSTEM2) == []
    yield "null of tall is empty", null_space(tall) == []
    yield "null of zero matrix", null_space([[0, 0], [0, 0]]) == [[1, 0], [0, 1]]
    yield "null of zero column", null_space([[0, 1], [0, 0]]) == [[1, 0]]
    yield "rank1 kernel", all(close(matvec(RANK1, v), [0, 0, 0]) for v in null_space(RANK1))
    yield "rank matches basis count", all(
        rank(M) == len(column_space(M)) == len(M[0]) - len(null_space(M))
        for M in (RANK1, SINGULAR, SYSTEM2, SYSTEM3, plane, tall, [[0, 0], [0, 0]], [[1, 2, 3]])
    )

    # 是否有解
    yield "system2 solvable", is_solvable(SYSTEM2, [-4, -1]) is True
    yield "system3 solvable", is_solvable(SYSTEM3, [-3, 0, 2]) is True
    yield "singular on the line", is_solvable(SINGULAR, [1, -3]) is True
    yield "singular off the line", is_solvable(SINGULAR, [1, 0]) is False
    yield "singular zero is solvable", is_solvable(SINGULAR, [0, 0]) is True
    yield "rank1 on the line", is_solvable(RANK1, [2, -4, 10]) is True
    yield "rank1 off the line", is_solvable(RANK1, [1, 0, 0]) is False
    yield "tall [1,2,3] solvable", is_solvable(tall, [1, 2, 3]) is True
    yield "tall [1,2,4] not solvable", is_solvable(tall, [1, 2, 4]) is False
    yield "plane [1,2,0] solvable", is_solvable(plane, [1, 2, 0]) is True
    yield "plane [1,2,3] not solvable", is_solvable(plane, [1, 2, 3]) is False
    yield "one row any b", is_solvable([[1, 2, 3]], [5]) is True
    yield "zero matrix zero b", is_solvable([[0, 0], [0, 0]], [0, 0]) is True
    yield "zero matrix nonzero b", is_solvable([[0, 0], [0, 0]], [1, 0]) is False

    # 异常：必须抛出，不能 skip
    yield "det non-square", raises(ValueError, determinant, [[1, 2, 3], [4, 5, 6]])
    yield "det ragged", raises(ValueError, determinant, [[1, 0], [1]])
    yield "det empty", raises(ValueError, determinant, [])
    yield "det 4x4", raises(ValueError, determinant, [[1, 0, 0, 0], [0, 1, 0, 0], [0, 0, 1, 0], [0, 0, 0, 1]])
    yield "inverse rejects 3x3", raises(ValueError, inverse_2x2, [[1, 0, 0], [0, 1, 0], [0, 0, 1]])
    yield "inverse rejects row", raises(ValueError, inverse_2x2, [[1, 2]])
    yield "inverse ragged", raises(ValueError, inverse_2x2, [[1, 0], [1]])
    yield "rank ragged", raises(ValueError, rank, [[1, 2], [3]])
    yield "rank empty", raises(ValueError, rank, [])
    yield "column_space empty", raises(ValueError, column_space, [])
    yield "null_space ragged", raises(ValueError, null_space, [[1, 0], [1]])
    yield "is_solvable dim mismatch", raises(ValueError, is_solvable, [[1, 0], [0, 1]], [1, 2, 3])
    yield "is_solvable ragged", raises(ValueError, is_solvable, [[1, 0], [1]], [1, 0])


def parity(n: int = 200, seed: int = 1657) -> list[str] | None:
    """与 numpy 随机对拍 n 组；返回失败描述列表，未安装 numpy 时返回 None。"""
    try:
        import numpy as np
    except ImportError:
        return None
    from properties import column_space, determinant, inverse_2x2, is_solvable, null_space, rank

    rng = random.Random(seed)
    fails: list[str] = []

    def random_matrix(rows: int, cols: int) -> list[list[int]]:
        return [[rng.randint(-5, 5) for _ in range(cols)] for _ in range(rows)]

    def random_vector(k: int) -> list[int]:
        return [rng.randint(-5, 5) for _ in range(k)]

    def independent(vectors: list[list[float]]) -> bool:
        if not vectors:
            return True
        return int(np.linalg.matrix_rank(np.array(vectors, dtype=float))) == len(vectors)

    def spans(basis: list[list[float]], columns: list[list[float]]) -> bool:
        if not basis:
            return all(all(abs(x) <= 1e-8 for x in col) for col in columns)
        base = np.array(basis, dtype=float)
        base_rank = int(np.linalg.matrix_rank(base))
        for col in columns:
            stacked = np.vstack([base, np.array(col, dtype=float)])
            if int(np.linalg.matrix_rank(stacked)) != base_rank:
                return False
        return True

    for _ in range(n):
        for size in (1, 2, 3):
            matrix = random_matrix(size, size)
            expected_det = float(np.linalg.det(np.array(matrix, dtype=float)))
            if not close(determinant(matrix), expected_det, 1e-6):
                fails.append(f"determinant {matrix}")

        square = random_matrix(2, 2)
        got_inv = inverse_2x2(square)
        numpy_det = float(np.linalg.det(np.array(square, dtype=float)))
        if abs(numpy_det) < 1e-8:
            if got_inv is not None:
                fails.append(f"inverse_2x2 singular {square} returned {got_inv}")
        else:
            expected_inv = np.linalg.inv(np.array(square, dtype=float)).tolist()
            if got_inv is None or not close(got_inv, expected_inv, 1e-6):
                fails.append(f"inverse_2x2 {square}")

        rows, cols = rng.choice([(2, 2), (3, 3), (3, 2), (2, 3), (1, 3), (3, 1)])
        matrix = random_matrix(rows, cols)
        expected_rank = int(np.linalg.matrix_rank(np.array(matrix, dtype=float)))
        if rank(matrix) != expected_rank:
            fails.append(f"rank {matrix}: got {rank(matrix)}, expected {expected_rank}")
        basis = column_space(matrix)
        columns = [list(col) for col in zip(*matrix)]
        if len(basis) != expected_rank or not spans(basis, columns):
            fails.append(f"column_space {matrix}")
        elif any(not any(close(vec, col) for col in columns) for vec in basis):
            fails.append(f"column_space not an original column {matrix}")
        elif not independent(basis):
            fails.append(f"column_space dependent {matrix}")
        kernel = null_space(matrix)
        if len(kernel) != cols - expected_rank or not independent(kernel):
            fails.append(f"null_space count {matrix}")
        elif any(not close(matvec(matrix, vec), [0.0] * rows) for vec in kernel):
            fails.append(f"null_space not in kernel {matrix}")
        if rank(matrix) != len(basis) or rank(matrix) != cols - len(kernel):
            fails.append(f"rank-nullity {matrix}")

        target = random_vector(rows)
        augmented = [row + [value] for row, value in zip(matrix, target)]
        solvable = int(np.linalg.matrix_rank(np.array(augmented, dtype=float))) == expected_rank
        if is_solvable(matrix, target) is not solvable:
            fails.append(f"is_solvable {matrix} @ {target}")
    return fails


def run_doctests() -> bool:
    """运行 properties.py 的 doctest；返回是否全部通过。"""
    import properties

    result = doctest.testmod(properties, verbose=False, optionflags=doctest.REPORT_ONLY_FIRST_FAILURE)
    status = "PASS" if result.failed == 0 else "FAIL"
    print(f"{status} doctest {properties.__name__}: {result.attempted - result.failed}/{result.attempted}")
    return result.failed == 0


def main() -> None:
    ok = True
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL properties.py 尚未实现（NotImplementedError）")
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
