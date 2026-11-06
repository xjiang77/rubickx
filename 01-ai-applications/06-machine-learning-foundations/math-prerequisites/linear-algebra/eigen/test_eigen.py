#!/usr/bin/env python3
"""坐标系变换验证器：3Blue1Brown 第 13–15 集的数值示例 + numpy 随机对拍 + doctest。

运行（仓库根目录）：make test-linalg-eigen
实现由本人手写（C5）。不得放宽容差、删除用例，或用库函数替换 eigen.py 中的实现。
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

JENNIFER = [[2, -1], [1, 1]]
"""第 13 集：列是 Jennifer 的基 (2, 1) 与 (−1, 1)。"""

OLIVE = [[2, -3], [3, 2]]
"""第 13 集课页：列是 Olive 的基 (2, 3) 与 (−3, 2)。"""

A_BASIS = [[1, 1], [1, -1]]
"""A 组：基 {[1, 1], [1, −1]}。"""

ROT90 = [[0, -1], [1, 0]]
"""逆时针转 90°。"""

OPENING = [[3, 1], [0, 2]]
"""第 14 集开头：x 轴拉长 3 倍，(1, −1) 拉长 2 倍。"""

SHEAR = [[1, 1], [0, 1]]
"""第 14 集的剪切。只有特征值 1，不能对角化。"""

FIB = [[0, 1], [1, 1]]
"""第 14 集幂的谜题。特征值是 (1±√5)/2。"""

PHI = (1 + 5**0.5) / 2
PSI = (1 - 5**0.5) / 2


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
    """逐条产出第 13–15 集与 A 组衔接的数值用例。"""
    from eigen import (
        change_of_basis_matrix,
        characteristic_poly_2x2,
        diagonalize_2x2,
        eigenvalue_of,
        eigenvalues_2x2,
        eigenvalues_from_mean_product,
        is_eigenvector,
        matrix_in_basis,
        to_new_coords,
        to_old_coords,
    )

    # 第 13 集：换基
    cols = [[2, 1], [-1, 1]]
    P = change_of_basis_matrix(cols)
    yield "Jennifer basis matrix", P == JENNIFER
    cols[0][0] = 99
    yield "change_of_basis does not alias columns", P == JENNIFER
    yield "Olive basis matrix", change_of_basis_matrix([[2, 3], [-3, 2]]) == OLIVE
    yield "A basis matrix", change_of_basis_matrix([[1, 1], [1, -1]]) == A_BASIS
    yield "standard basis matrix", change_of_basis_matrix([[1, 0], [0, 1]]) == [[1, 0], [0, 1]]

    spoken = [-1, 2]
    ours = to_old_coords(JENNIFER, spoken)
    spoken[0] = 0
    yield "Jennifer (-1, 2) -> (-4, 1)", ours == [-4, 1]
    yield "to_old does not alias the vector", spoken == [0, 2]
    yield "Jennifer (2, -3) -> (7, -1)", to_old_coords(JENNIFER, [2, -3]) == [7, -1]
    yield "Olive (-1, 2) -> (-8, 1)", to_old_coords(OLIVE, [-1, 2]) == [-8, 1]
    yield "A coords back to (4, 2)", to_old_coords(A_BASIS, [3, 1]) == [4, 2]
    yield "origin stays origin", to_old_coords(JENNIFER, [0, 0]) == [0, 0]

    yield "our (3, 2) in Jennifer coords", close(to_new_coords(JENNIFER, [3, 2]), [5 / 3, 1 / 3])
    yield "our (-5, -1) in Olive coords", to_new_coords(OLIVE, [-5, -1]) == [-1, 1]
    yield "A group (4, 2) -> (3, 1)", to_new_coords(A_BASIS, [4, 2]) == [3, 1]
    yield "standard basis keeps (3, -2)", to_new_coords([[1, 0], [0, 1]], [3, -2]) == [3, -2]
    back = to_old_coords(JENNIFER, to_new_coords(JENNIFER, [3, 2]))
    yield "new then old is identity", close(back, [3, 2])

    rotated = matrix_in_basis(ROT90, JENNIFER)
    yield "90 degree rotation in Jennifer basis", close(rotated, [[1 / 3, -2 / 3], [5 / 3, -1 / 3]])
    copied_rot = [row[:] for row in ROT90]
    copied_p = [row[:] for row in JENNIFER]
    matrix_in_basis(copied_rot, copied_p)
    yield "matrix_in_basis does not mutate", copied_rot == ROT90 and copied_p == JENNIFER
    eigen_p = [[1, 1], [0, -1]]
    yield "opening matrix in its eigenbasis", matrix_in_basis(OPENING, eigen_p) == [[3, 0], [0, 2]]

    # 第 14 集：特征向量
    yield "x-axis is an eigenvector", is_eigenvector(OPENING, [1, 0]) is True
    yield "(1, -1) is an eigenvector", is_eigenvector(OPENING, [1, -1]) is True
    yield "(-1, 1) is the same line", is_eigenvector(OPENING, [-1, 1]) is True
    yield "(1, 1) is knocked off its span", is_eigenvector(OPENING, [1, 1]) is False
    yield "zero vector is not an eigenvector", is_eigenvector(OPENING, [0, 0]) is False
    yield "eigenvalue 3", eigenvalue_of(OPENING, [1, 0]) == 3
    yield "eigenvalue 2", eigenvalue_of(OPENING, [1, -1]) == 2
    yield "opposite vector same eigenvalue", eigenvalue_of(OPENING, [-1, 1]) == 2
    yield "scaled x-axis still 3", eigenvalue_of(OPENING, [2, 0]) == 3
    yield "shear keeps i-hat", is_eigenvector(SHEAR, [1, 0]) is True and eigenvalue_of(SHEAR, [1, 0]) == 1
    yield "shear moves j-hat", is_eigenvector(SHEAR, [0, 1]) is False
    yield "90 degree rotation has no eigenvector", is_eigenvector(ROT90, [1, 0]) is False
    yield "scaling by 2: every nonzero vector", eigenvalue_of([[2, 0], [0, 2]], [3, -4]) == 2
    yield "negative eigenvalue", eigenvalue_of([[-1, 0], [0, 2]], [1, 0]) == -1
    yield "positive partner", eigenvalue_of([[-1, 0], [0, 2]], [0, 1]) == 2
    fixed = [[2, 2], [1, 3]]
    yield "lambda 1 stays fixed", eigenvalue_of(fixed, [2, -1]) == 1
    yield "the other eigenvalue is 4", eigenvalue_of(fixed, [1, 1]) == 4
    yield "quiz eigenvalues 5 and 1", eigenvalues_2x2([[3, 2], [2, 3]]) == (5, 1)
    fib_v1 = [(-1 + 5**0.5) / 2, 1]
    fib_v2 = [(-1 - 5**0.5) / 2, 1]
    yield "Fibonacci lesson vector, larger", is_eigenvector(FIB, fib_v1) is True
    yield "Fibonacci lesson eigenvalue", close(eigenvalue_of(FIB, fib_v1), PHI)
    yield "Fibonacci lesson vector, smaller", close(eigenvalue_of(FIB, fib_v2), PSI)

    # agent 选定：课程页没有特征值为 0 的矩阵。投影到 x 轴。
    onto_x = [[1, 0], [0, 0]]
    yield "eigenvalue 0 on the y-axis", is_eigenvector(onto_x, [0, 1]) is True
    yield "that eigenvalue is 0", eigenvalue_of(onto_x, [0, 1]) == 0

    # 第 14–15 集：多项式与快速技巧
    yield "chapter 15 polynomial", characteristic_poly_2x2([[3, 1], [4, 1]]) == (1, -4, -1)
    yield "opening polynomial", characteristic_poly_2x2(OPENING) == (1, -5, 6)
    yield "rotation polynomial", characteristic_poly_2x2(ROT90) == (1, 0, 1)
    yield "shear polynomial", characteristic_poly_2x2(SHEAR) == (1, -2, 1)
    yield "Fibonacci polynomial", characteristic_poly_2x2(FIB) == (1, -1, -1)
    yield "Pauli x polynomial", characteristic_poly_2x2([[0, 1], [1, 0]]) == (1, 0, -1)
    yield "mean 7 product 40", eigenvalues_from_mean_product(7, 40) == (10, 4)
    yield "mean 5 product 9", eigenvalues_from_mean_product(5, 9) == (9, 1)
    yield "mean 0 product -1", eigenvalues_from_mean_product(0, -1) == (1, -1)
    hi, lo = eigenvalues_from_mean_product(2, -1)
    yield "2 ± sqrt(5) from mean and product", close([hi, lo], [2 + 5**0.5, 2 - 5**0.5])
    yield "opening eigenvalues", eigenvalues_2x2(OPENING) == (3, 2)
    yield "10 and 4", eigenvalues_2x2([[8, 4], [2, 6]]) == (10, 4)
    yield "9 and 1", eigenvalues_2x2([[2, 7], [1, 8]]) == (9, 1)
    pair = eigenvalues_2x2([[3, 1], [4, 1]])
    yield "2 ± sqrt(5)", close(list(pair), [2 + 5**0.5, 2 - 5**0.5])
    quiz = eigenvalues_2x2([[2, 3], [2, 4]])
    yield "3 ± sqrt(7)", close(list(quiz), [3 + 7**0.5, 3 - 7**0.5])
    fib_eigs = eigenvalues_2x2(FIB)
    yield "golden ratio eigenvalues", close(list(fib_eigs), [PHI, PSI])
    yield "Pauli x eigenvalues", eigenvalues_2x2([[0, 1], [1, 0]]) == (1, -1)
    yield "Pauli z eigenvalues", eigenvalues_2x2([[1, 0], [0, -1]]) == (1, -1)
    yield "shear repeated eigenvalue", eigenvalues_2x2(SHEAR) == (1, 1)
    yield "defective negative shear", eigenvalues_2x2([[-2, 1], [0, -2]]) == (-2, -2)
    yield "scale-by-2 eigenvalues", eigenvalues_2x2([[2, 0], [0, 2]]) == (2, 2)
    yield "projection eigenvalues", eigenvalues_2x2(onto_x) == (1, 0)

    # 对角化：刻度按 EXERCISE 的规则钉死
    P_open, D_open = diagonalize_2x2(OPENING)
    yield "diagonalize opening P", P_open == [[1, 1], [0, -1]]
    yield "diagonalize opening D", D_open == [[3, 0], [0, 2]]
    yield "diagonalize does not mutate", OPENING == [[3, 1], [0, 2]]
    signed = [[-1, 0], [0, 2]]
    yield "diagonalize diag(-1, 2)", diagonalize_2x2(signed) == ([[0, 1], [1, 0]], [[2, 0], [0, -1]])
    yield "diagonalize scale by 2", diagonalize_2x2([[2, 0], [0, 2]]) == (
        [[1, 0], [0, 1]],
        [[2, 0], [0, 2]],
    )
    yield "diagonalize projection", diagonalize_2x2(onto_x) == ([[1, 0], [0, 1]], [[1, 0], [0, 0]])
    yield "diagonalize Pauli x", diagonalize_2x2([[0, 1], [1, 0]]) == (
        [[1, 1], [1, -1]],
        [[1, 0], [0, -1]],
    )
    yield "diagonalize symmetric quiz", diagonalize_2x2([[3, 2], [2, 3]]) == (
        [[1, 1], [1, -1]],
        [[5, 0], [0, 1]],
    )
    yield "diagonalize 10 and 4", diagonalize_2x2([[8, 4], [2, 6]]) == (
        [[2, 1], [1, -1]],
        [[10, 0], [0, 4]],
    )
    yield "diagonalize 9 and 1", diagonalize_2x2([[2, 7], [1, 8]]) == (
        [[1, 7], [1, -1]],
        [[9, 0], [0, 1]],
    )
    yield "diagonalize lambda-1 example", diagonalize_2x2(fixed) == (
        [[1, 2], [1, -1]],
        [[4, 0], [0, 1]],
    )
    fib_p, fib_d = diagonalize_2x2(FIB)
    yield "Fibonacci canonical P", close(fib_p, [[1, 1], [PHI, PSI]])
    yield "Fibonacci diagonal", close(fib_d, [[PHI, 0], [0, PSI]])
    yield "eigenbasis matrix matches D", close(matrix_in_basis(FIB, fib_p), fib_d)

    # 异常：必须抛出，不能 skip
    yield "collinear basis", raises(ValueError, change_of_basis_matrix, [[1, 2], [2, 4]])
    yield "one basis vector", raises(ValueError, change_of_basis_matrix, [[1, 0]])
    yield "3d basis vectors", raises(ValueError, change_of_basis_matrix, [[1, 0, 0], [0, 1, 0]])
    yield "empty basis", raises(ValueError, change_of_basis_matrix, [])
    singular = [[1, 2], [2, 4]]
    yield "to_old singular", raises(ValueError, to_old_coords, singular, [1, 0])
    yield "to_new singular", raises(ValueError, to_new_coords, singular, [3, 2])
    yield "to_old bad vector", raises(ValueError, to_old_coords, JENNIFER, [1])
    yield "to_new bad matrix", raises(ValueError, to_new_coords, [[1, 0, 0], [0, 1, 0]], [1, 0])
    yield "matrix_in_basis singular", raises(ValueError, matrix_in_basis, OPENING, singular)
    yield "matrix_in_basis ragged", raises(ValueError, matrix_in_basis, [[1, 0]], JENNIFER)
    yield "is_eigenvector bad shape", raises(ValueError, is_eigenvector, OPENING, [1, 2, 3])
    yield "eigenvalue of zero", raises(ValueError, eigenvalue_of, OPENING, [0, 0])
    yield "eigenvalue of a non-eigenvector", raises(ValueError, eigenvalue_of, OPENING, [1, 1])
    yield "eigenvalue of a rotation", raises(ValueError, eigenvalue_of, ROT90, [1, 0])
    yield "polynomial bad shape", raises(ValueError, characteristic_poly_2x2, [[1, 2, 3], [4, 5, 6]])
    yield "mean-product no real", raises(ValueError, eigenvalues_from_mean_product, 0, 1)
    yield "rotation eigenvalues", raises(ValueError, eigenvalues_2x2, ROT90)
    yield "quiz matrix has no real eigenvalues", raises(ValueError, eigenvalues_2x2, [[2, 1], [-1, 2]])
    yield "eigenvalues bad shape", raises(ValueError, eigenvalues_2x2, [])
    yield "shear does not diagonalize", raises(ValueError, diagonalize_2x2, SHEAR)
    yield "negative shear does not diagonalize", raises(ValueError, diagonalize_2x2, [[-2, 1], [0, -2]])
    yield "rotation does not diagonalize", raises(ValueError, diagonalize_2x2, ROT90)
    yield "quiz rotation-like does not diagonalize", raises(ValueError, diagonalize_2x2, [[2, 1], [-1, 2]])
    yield "diagonalize bad shape", raises(ValueError, diagonalize_2x2, [[1, 2, 3], [4, 5, 6]])


def _det(M: list[list[float]]) -> float:
    return M[0][0] * M[1][1] - M[0][1] * M[1][0]


def _matmul(A: list[list[float]], B: list[list[float]]) -> list[list[float]]:
    return [
        [
            A[0][0] * B[0][0] + A[0][1] * B[1][0],
            A[0][0] * B[0][1] + A[0][1] * B[1][1],
        ],
        [
            A[1][0] * B[0][0] + A[1][1] * B[1][0],
            A[1][0] * B[0][1] + A[1][1] * B[1][1],
        ],
    ]


def _inv(M: list[list[float]]) -> list[list[float]]:
    d = _det(M)
    return [[M[1][1] / d, -M[0][1] / d], [-M[1][0] / d, M[0][0] / d]]


def parity(n: int = 200, seed: int = 1315) -> list[str] | None:
    """与 numpy 随机对拍 n 组；返回失败描述列表，未安装 numpy 时返回 None。"""
    try:
        import numpy as np
    except ImportError:
        return None
    from eigen import (
        change_of_basis_matrix,
        characteristic_poly_2x2,
        diagonalize_2x2,
        eigenvalue_of,
        eigenvalues_2x2,
        is_eigenvector,
        matrix_in_basis,
        to_new_coords,
        to_old_coords,
    )

    rng = random.Random(seed)
    fails: list[str] = []

    def random_entry() -> int:
        return rng.randint(-5, 5)

    def random_matrix() -> list[list[int]]:
        return [[random_entry(), random_entry()], [random_entry(), random_entry()]]

    def invertible_matrix() -> list[list[int]]:
        matrix = random_matrix()
        while abs(_det(matrix)) < 1:
            matrix = random_matrix()
        return matrix

    for _ in range(n):
        basis = invertible_matrix()
        columns = [[basis[0][0], basis[1][0]], [basis[0][1], basis[1][1]]]
        P = change_of_basis_matrix([col[:] for col in columns])
        if not close(P, basis):
            fails.append(f"change_of_basis {columns}")
            continue
        vector = [random_entry(), random_entry()]
        old = to_old_coords([row[:] for row in P], vector[:])
        expect_old = (np.array(P, dtype=float) @ np.array(vector, dtype=float)).tolist()
        if not close(old, expect_old, 1e-6):
            fails.append(f"to_old {P} {vector}")
        new = to_new_coords([row[:] for row in P], expect_old)
        if not close(new, vector, 1e-6):
            fails.append(f"to_new {P} {expect_old}")

        A = random_matrix()
        changed = matrix_in_basis([row[:] for row in A], [row[:] for row in P])
        expect = _matmul(_matmul(_inv(P), A), P)
        numpy_expect = np.linalg.inv(np.array(P, dtype=float)) @ np.array(A, dtype=float) @ np.array(P, dtype=float)
        if not close(changed, expect, 1e-6) or not close(changed, numpy_expect.tolist(), 1e-6):
            fails.append(f"matrix_in_basis {A} {P}")

        coeff = characteristic_poly_2x2(A)
        trace = A[0][0] + A[1][1]
        det = _det(A)
        if not close(list(coeff), [1, -trace, det]):
            fails.append(f"characteristic poly {A}")

        eigvals = np.linalg.eigvals(np.array(A, dtype=float))
        # 重根时 numpy 可能带上约 1e-8 的虚部。整数 2×2 若真有复特征值，虚部至少约 0.8。
        real = bool(np.all(np.isclose(eigvals.imag, 0, atol=1e-6)))
        if real:
            ordered = tuple(sorted((float(eigvals[0].real), float(eigvals[1].real)), reverse=True))
            try:
                got = eigenvalues_2x2(A)
            except ValueError:
                fails.append(f"eigenvalues raised on real pair {A} {ordered}")
            else:
                if not close(list(got), list(ordered), 1e-6):
                    fails.append(f"eigenvalues {A}: got {got}, numpy {ordered}")
        else:
            if not raises(ValueError, eigenvalues_2x2, A):
                fails.append(f"eigenvalues should reject complex {A}")

        distinct = real and abs(float(eigvals[0].real) - float(eigvals[1].real)) > 1e-6
        scalar = abs(A[0][1]) == 0 and abs(A[1][0]) == 0 and A[0][0] == A[1][1]
        diagonalizable = distinct or scalar
        if diagonalizable:
            try:
                factors, diagonal = diagonalize_2x2([row[:] for row in A])
            except ValueError:
                fails.append(f"diagonalize raised on {A}")
            else:
                if not close(diagonal[0][1], 0) or not close(diagonal[1][0], 0):
                    fails.append(f"D not diagonal {A} {diagonal}")
                rebuilt = _matmul(_matmul(factors, diagonal), _inv(factors))
                if abs(_det(factors)) <= 1e-8 or not close(rebuilt, A, 1e-5):
                    fails.append(f"P D Pinv != A for {A}")
                for index, column in enumerate(([factors[0][0], factors[1][0]], [factors[0][1], factors[1][1]])):
                    if not is_eigenvector(A, column):
                        fails.append(f"column {index} of {factors} not eigenvector of {A}")
                    elif not close(eigenvalue_of(A, column), diagonal[index][index], 1e-6):
                        fails.append(f"eigenvalue mismatch {A}")
        elif not raises(ValueError, diagonalize_2x2, A):
            fails.append(f"diagonalize should reject {A}")

        if real and distinct:
            values, vectors = np.linalg.eig(np.array(A, dtype=float))
            for column in range(2):
                vector = vectors[:, column].real.tolist()
                if not is_eigenvector(A, vector):
                    fails.append(f"numpy eigenvector rejected {A} {vector}")
                elif not close(eigenvalue_of(A, vector), float(values[column].real), 1e-5):
                    fails.append(f"numpy eigenvalue mismatch {A}")
        if is_eigenvector(A, [0.0, 0.0]):
            fails.append(f"zero vector accepted as eigenvector of {A}")
        elif not raises(ValueError, eigenvalue_of, A, [0.0, 0.0]):
            fails.append(f"eigenvalue_of accepted the zero vector of {A}")
    return fails


def run_doctests() -> bool:
    """运行 eigen.py 的 doctest；返回是否全部通过。"""
    import eigen

    result = doctest.testmod(eigen, verbose=False, optionflags=doctest.REPORT_ONLY_FIRST_FAILURE)
    status = "PASS" if result.failed == 0 else "FAIL"
    print(f"{status} doctest {eigen.__name__}: {result.attempted - result.failed}/{result.attempted}")
    return result.failed == 0


def main() -> None:
    ok = True
    try:
        for name, passed in cases():
            print(("PASS " if passed else "FAIL ") + name)
            ok = ok and passed
    except NotImplementedError:
        print("FAIL eigen.py 尚未实现（NotImplementedError）")
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
