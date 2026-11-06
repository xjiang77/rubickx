"""坐标系变换 —— 本人手写 (C5)。下面只有骨架，实现留空。

矩阵用 list[list[float]] 按行存放，向量用 list[float] 表示。只做 2×2。
不得 import numpy 或其他数值库。
规格见 EXERCISE.md。数值示例取自 3Blue1Brown《Essence of Linear Algebra》
第 13–15 集，并复用 A 组已经核对过的换基：
[4, 2] 在基 {[1, 1], [1, −1]} 下的坐标是 (3, 1)。
本仓没有这一篇知识正文（笔记在 dragon-vault；topics.json 只登记了正文 01 与 02）。

约定：change_of_basis_matrix 返回的 P，列是新基向量在旧坐标下的写法。
P 把新坐标映到旧坐标（v_old = P v_new）。逆矩阵把旧坐标映回新坐标。
matrix_in_basis(A_old, P) 是 P^{-1} A_old P，也就是同一个线性变换在新基下的矩阵。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest eigen.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from typing import TypeAlias

Vector: TypeAlias = list[float]
"""向量：有序的数字列表。与 vectors.py 相同；int 也可传入。"""

Matrix: TypeAlias = list[list[float]]
"""矩阵：按行存放。M[i][j] 是第 j 列的第 i 个分量。"""


def change_of_basis_matrix(basis_cols: list[Vector]) -> Matrix:
    """把新基向量（写成旧坐标）排成列，得到换基矩阵 P。

    P 把新坐标映到旧坐标：v_old = P v_new。
    第 13 集里，Jennifer 的基在我们的坐标下是 (2, 1) 与 (−1, 1)，
    所以 P 的两列就是这两个向量。她口中的坐标先乘 P，才是我们看到的向量。
    两列共线时它们不构成基。共线用行列式的绝对值判断，阈值 1e-9。

    Args:
        basis_cols: 恰好两个二维向量，都写成旧坐标。顺序就是 P 的列顺序。

    Returns:
        2×2 矩阵。不修改 basis_cols，也不与输入共享内部列表。
        分量是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: 不是恰好两个二维向量，或两列共线。

    Examples:
        第 13 集 Jennifer 与课页上的 Olive。A 组的基 {[1, 1], [1, −1]} 同样排成列。

        >>> change_of_basis_matrix([[2, 1], [-1, 1]]) == [[2, -1], [1, 1]]
        True
        >>> change_of_basis_matrix([[2, 3], [-3, 2]]) == [[2, -3], [3, 2]]
        True
        >>> change_of_basis_matrix([[1, 1], [1, -1]]) == [[1, 1], [1, -1]]
        True
        >>> change_of_basis_matrix([[1, 0], [0, 1]]) == [[1, 0], [0, 1]]
        True
        >>> change_of_basis_matrix([[1, 2], [2, 4]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 这两列共线，不构成基
        >>> change_of_basis_matrix([[1, 0]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 需要恰好两个二维基向量
        >>> change_of_basis_matrix([[1, 0, 0], [0, 1, 0]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 需要恰好两个二维基向量
    """
    raise NotImplementedError


def to_old_coords(P: Matrix, v_new: Vector) -> Vector:
    """把新坐标译成旧坐标：v_old = P v_new。

    P 的列必须是新基在旧坐标下的写法，与 change_of_basis_matrix 的返回值相同。
    第 13 集：Jennifer 说 (−1, 2)，我们看到的是 −1·(2, 1) + 2·(−1, 1) = (−4, 1)。

    Args:
        P: 2×2 换基矩阵，列是新基的旧坐标。
        v_new: 新坐标下的二维向量。

    Returns:
        旧坐标下的二维向量。不修改 P、v_new。
        分量是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: P 不是 2×2、v_new 不是二维，或 P 的行列式绝对值不超过 1e-9。

    Examples:
        >>> to_old_coords([[2, -1], [1, 1]], [-1, 2]) == [-4, 1]
        True
        >>> to_old_coords([[2, -1], [1, 1]], [2, -3]) == [7, -1]
        True
        >>> to_old_coords([[2, -3], [3, 2]], [-1, 2]) == [-8, 1]
        True
        >>> to_old_coords([[1, 1], [1, -1]], [3, 1]) == [4, 2]
        True
        >>> to_old_coords([[2, -1], [1, 1]], [0, 0]) == [0, 0]
        True
        >>> to_old_coords([[1, 2], [2, 4]], [1, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵不可逆
        >>> to_old_coords([[1, 0], [0, 1]], [1, 2, 3])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 向量必须是二维
    """
    raise NotImplementedError


def to_new_coords(P: Matrix, v_old: Vector) -> Vector:
    """把旧坐标译成新坐标：v_new = P^{-1} v_old。

    2×2 的逆是交换主对角线、另一条对角线变号，再除以行列式。
    第 13 集：我们的 (3, 2) 在 Jennifer 的基下是 (5/3, 1/3)。
    A 组：我们的 (4, 2) 在基 {[1, 1], [1, −1]} 下是 (3, 1)。

    Args:
        P: 2×2 换基矩阵，列是新基的旧坐标。
        v_old: 旧坐标下的二维向量。

    Returns:
        新坐标下的二维向量。不修改 P、v_old。
        分量是 int 还是 float 取决于实现，比较时用 ==；有舍入时先 round。

    Raises:
        ValueError: P 不是 2×2、v_old 不是二维，或 P 的行列式绝对值不超过 1e-9。

    Examples:
        >>> got = to_new_coords([[2, -1], [1, 1]], [3, 2])
        >>> [round(x, 9) for x in got] == [round(5/3, 9), round(1/3, 9)]
        True
        >>> to_new_coords([[2, -3], [3, 2]], [-5, -1]) == [-1, 1]
        True
        >>> to_new_coords([[1, 1], [1, -1]], [4, 2]) == [3, 1]
        True
        >>> to_new_coords([[1, 0], [0, 1]], [3, -2]) == [3, -2]
        True
        >>> to_new_coords([[1, 2], [2, 4]], [3, 2])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵不可逆
        >>> to_new_coords([[1, 0, 0], [0, 1, 0]], [1, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵必须是 2×2
    """
    raise NotImplementedError


def matrix_in_basis(A_old: Matrix, P: Matrix) -> Matrix:
    """同一个线性变换在新基下的矩阵：P^{-1} A_old P。

    先用 P 把新坐标译成旧坐标，再用 A_old，最后用 P^{-1} 译回新坐标。
    第 13 集：逆时针转 90° 在我们这里是 [[0, −1], [1, 0]]。
    换成 Jennifer 的基之后，列是 (1/3, 5/3) 与 (−2/3, −1/3)。
    若 P 的列是特征向量，结果是对角矩阵，对角元是对应的特征值。

    Args:
        A_old: 2×2，旧坐标下的线性变换。
        P: 2×2 换基矩阵，列是新基的旧坐标，必须可逆。

    Returns:
        2×2 矩阵。不修改 A_old、P。
        分量是 int 还是 float 取决于实现，比较时用 ==；有舍入时先 round。

    Raises:
        ValueError: 任一矩阵不是 2×2，或 P 的行列式绝对值不超过 1e-9。

    Examples:
        >>> got = matrix_in_basis([[0, -1], [1, 0]], [[2, -1], [1, 1]])
        >>> [[round(x, 9) for x in row] for row in got] == [
        ...     [round(1/3, 9), round(-2/3, 9)],
        ...     [round(5/3, 9), round(-1/3, 9)],
        ... ]
        True
        >>> matrix_in_basis([[3, 1], [0, 2]], [[1, 1], [0, -1]]) == [[3, 0], [0, 2]]
        True
        >>> matrix_in_basis([[3, 1], [0, 2]], [[1, 2], [2, 4]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵不可逆
        >>> matrix_in_basis([[1, 0]], [[1, 0], [0, 1]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵必须是 2×2
    """
    raise NotImplementedError


def is_eigenvector(A: Matrix, v: Vector, tol: float = 1e-9) -> bool:
    """v 是否是 A 的特征向量：非零，且 Av 落在 v 张成的直线上。

    零向量使 Av = λv 对每个 λ 都成立，但按定义它不是特征向量，这里返回 False。
    平行用二维叉积判断：
    abs(Av[0]·v[1] − Av[1]·v[0]) <= tol · max(1, |v[0]|+|v[1]|, |Av[0]|+|Av[1]|)。
    特征值可以是负数或 0。Av = 0 且 v 非零时，0 是特征值，结果为 True。

    Args:
        A: 2×2 矩阵。
        v: 二维向量。
        tol: 浮点容差，也用来判断 v 是不是零向量。

    Returns:
        是特征向量返回 True，否则 False。

    Raises:
        ValueError: A 不是 2×2，或 v 不是二维。

    Examples:
        第 14 集：[[3, 1], [0, 2]] 把 x 轴拉长 3 倍，把 (1, −1) 拉长 2 倍。
        (1, 1) 会被转出自己的直线。零向量返回 False。
        剪切只留住 x 轴。逆时针转 90° 没有实特征向量。

        >>> is_eigenvector([[3, 1], [0, 2]], [1, 0])
        True
        >>> is_eigenvector([[3, 1], [0, 2]], [1, -1])
        True
        >>> is_eigenvector([[3, 1], [0, 2]], [-1, 1])
        True
        >>> is_eigenvector([[3, 1], [0, 2]], [1, 1])
        False
        >>> is_eigenvector([[3, 1], [0, 2]], [0, 0])
        False
        >>> is_eigenvector([[1, 1], [0, 1]], [1, 0])
        True
        >>> is_eigenvector([[1, 1], [0, 1]], [0, 1])
        False
        >>> is_eigenvector([[0, -1], [1, 0]], [1, 0])
        False
        >>> is_eigenvector([[2, 0], [0, 2]], [3, -4])
        True
        >>> is_eigenvector([[1, 0], [0, 0]], [0, 1])
        True
        >>> is_eigenvector([[3, 1], [0, 2]], [1, 2, 3])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 向量必须是二维
    """
    raise NotImplementedError


def eigenvalue_of(A: Matrix, v: Vector, tol: float = 1e-9) -> float:
    """已知 v 是特征向量时，读出特征值 λ，使 Av = λv。

    λ = (Av · v) / (v · v)。v 是零向量或不是特征向量时抛出 ValueError，
    不返回一个凑出来的数。同一条直线上的非零倍数给出同一个 λ。

    Args:
        A: 2×2 矩阵。
        v: 二维特征向量。
        tol: 传给 is_eigenvector 的容差。

    Returns:
        特征值。是 int 还是 float 取决于实现，比较时用 ==；无理数先 round。

    Raises:
        ValueError: 形状不对，或 v 不是特征向量（含零向量）。

    Examples:
        >>> eigenvalue_of([[3, 1], [0, 2]], [1, 0]) == 3
        True
        >>> eigenvalue_of([[3, 1], [0, 2]], [1, -1]) == 2
        True
        >>> eigenvalue_of([[3, 1], [0, 2]], [-1, 1]) == 2
        True
        >>> eigenvalue_of([[3, 1], [0, 2]], [2, 0]) == 3
        True
        >>> eigenvalue_of([[1, 1], [0, 1]], [1, 0]) == 1
        True
        >>> eigenvalue_of([[2, 0], [0, 2]], [3, -4]) == 2
        True
        >>> eigenvalue_of([[-1, 0], [0, 2]], [1, 0]) == -1
        True
        >>> eigenvalue_of([[-1, 0], [0, 2]], [0, 1]) == 2
        True
        >>> eigenvalue_of([[2, 2], [1, 3]], [1, 1]) == 4
        True
        >>> eigenvalue_of([[2, 2], [1, 3]], [2, -1]) == 1
        True
        >>> eigenvalue_of([[1, 0], [0, 0]], [0, 1]) == 0
        True
        >>> round(eigenvalue_of([[0, 1], [1, 1]], [(-1 + 5**0.5) / 2, 1]), 9) == round((1 + 5**0.5) / 2, 9)
        True
        >>> eigenvalue_of([[3, 1], [0, 2]], [0, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: v 不是特征向量
        >>> eigenvalue_of([[3, 1], [0, 2]], [1, 1])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: v 不是特征向量
        >>> eigenvalue_of([[0, -1], [1, 0]], [1, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: v 不是特征向量
    """
    raise NotImplementedError


def characteristic_poly_2x2(A: Matrix) -> tuple[float, float, float]:
    """2×2 的特征多项式 det(λI − A)，按 λ²、λ、常数项返回系数。

    [[a, b], [c, d]] 的系数是 (1, −(a+d), ad−bc)，也就是 λ² − (迹)λ + 行列式。
    2×2 时 det(A − λI) 与 det(λI − A) 是同一个多项式。
    第 15 集：[[3, 1], [4, 1]] 化成 λ² − 4λ − 1，系数 (1, −4, −1)。
    第 14 集的 90° 旋转得到 λ² + 1，没有实根。

    Args:
        A: 2×2 矩阵。

    Returns:
        (二次项, 一次项, 常数项)。二次项是 1。
        是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: A 不是 2×2。

    Examples:
        >>> characteristic_poly_2x2([[3, 1], [4, 1]]) == (1, -4, -1)
        True
        >>> characteristic_poly_2x2([[3, 1], [0, 2]]) == (1, -5, 6)
        True
        >>> characteristic_poly_2x2([[0, -1], [1, 0]]) == (1, 0, 1)
        True
        >>> characteristic_poly_2x2([[1, 1], [0, 1]]) == (1, -2, 1)
        True
        >>> characteristic_poly_2x2([[0, 1], [1, 1]]) == (1, -1, -1)
        True
        >>> characteristic_poly_2x2([[0, 1], [1, 0]]) == (1, 0, -1)
        True
        >>> characteristic_poly_2x2([[1, 2, 3], [4, 5, 6]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵必须是 2×2
    """
    raise NotImplementedError


def eigenvalues_from_mean_product(mean: float, product: float) -> tuple[float, float]:
    """由平均数 m 与乘积 p 恢复两个实数：m ± sqrt(m² − p)。

    第 15 集的快速技巧只依赖这两件事：特征值的平均数等于对角元的平均数，
    乘积等于行列式。先写 m ± sqrt(m² − …)，再把 p 填进根号。
    返回值从大到小。m² − p < −1e-9 时没有实数，抛出 ValueError。
    介于 −1e-9 和 0 之间的判别式当成 0。

    Args:
        mean: 两个数的平均数。
        product: 两个数的乘积。

    Returns:
        (较大的数, 较小的数)。相等时两个分量相同。
        是 int 还是 float 取决于实现，比较时用 ==；无理数先 round。

    Raises:
        ValueError: 判别式小于 −1e-9。

    Examples:
        第 15 集：平均数 7、乘积 40 得到 10 与 4。
        [[3, 1], [4, 1]] 的平均数是 2、乘积是 −1，得到 2 ± sqrt(5)。
        90° 旋转的平均数是 0、乘积是 1，判别式是 −1。

        >>> eigenvalues_from_mean_product(7, 40) == (10, 4)
        True
        >>> larger, smaller = eigenvalues_from_mean_product(2, -1)
        >>> round(larger, 9) == round(2 + 5**0.5, 9)
        True
        >>> round(smaller, 9) == round(2 - 5**0.5, 9)
        True
        >>> eigenvalues_from_mean_product(5, 9) == (9, 1)
        True
        >>> eigenvalues_from_mean_product(0, -1) == (1, -1)
        True
        >>> eigenvalues_from_mean_product(1, 1) == (1, 1)
        True
        >>> eigenvalues_from_mean_product(0, 1)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 没有实数解
    """
    raise NotImplementedError


def eigenvalues_2x2(A: Matrix) -> tuple[float, float]:
    """用第 15 集的均值—乘积技巧读出 2×2 的两个实特征值。

    m = (a+d)/2，p = ad−bc，再调用 eigenvalues_from_mean_product(m, p)。
    顺序是从大到小。重根返回两个相同的数，不要求矩阵可对角化。
    没有实特征值时抛出 ValueError，不返回复数。

    Args:
        A: 2×2 矩阵。

    Returns:
        (较大特征值, 较小特征值)。
        是 int 还是 float 取决于实现，比较时用 ==；无理数先 round。

    Raises:
        ValueError: A 不是 2×2，或没有实特征值。

    Examples:
        >>> eigenvalues_2x2([[3, 1], [0, 2]]) == (3, 2)
        True
        >>> eigenvalues_2x2([[8, 4], [2, 6]]) == (10, 4)
        True
        >>> eigenvalues_2x2([[2, 7], [1, 8]]) == (9, 1)
        True
        >>> eigenvalues_2x2([[3, 2], [2, 3]]) == (5, 1)
        True
        >>> larger, smaller = eigenvalues_2x2([[3, 1], [4, 1]])
        >>> round(larger, 9) == round(2 + 5**0.5, 9)
        True
        >>> round(smaller, 9) == round(2 - 5**0.5, 9)
        True
        >>> hi, lo = eigenvalues_2x2([[2, 3], [2, 4]])
        >>> round(hi, 9) == round(3 + 7**0.5, 9)
        True
        >>> round(lo, 9) == round(3 - 7**0.5, 9)
        True
        >>> phi, psi = eigenvalues_2x2([[0, 1], [1, 1]])
        >>> round(phi, 9) == round((1 + 5**0.5) / 2, 9)
        True
        >>> round(psi, 9) == round((1 - 5**0.5) / 2, 9)
        True
        >>> eigenvalues_2x2([[0, 1], [1, 0]]) == (1, -1)
        True
        >>> eigenvalues_2x2([[1, 1], [0, 1]]) == (1, 1)
        True
        >>> eigenvalues_2x2([[-2, 1], [0, -2]]) == (-2, -2)
        True
        >>> eigenvalues_2x2([[2, 0], [0, 2]]) == (2, 2)
        True
        >>> eigenvalues_2x2([[1, 0], [0, 0]]) == (1, 0)
        True
        >>> eigenvalues_2x2([[0, -1], [1, 0]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 没有实特征值
        >>> eigenvalues_2x2([[2, 1], [-1, 2]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 没有实特征值
    """
    raise NotImplementedError


def diagonalize_2x2(A: Matrix, tol: float = 1e-9) -> tuple[Matrix, Matrix]:
    """把可对角化的 2×2 写成 A = P D P^{-1}，返回 (P, D)。

    D 的对角元是特征值，从大到小，非对角元是 0。
    P 的列是对应的特征向量，刻度按下述规则取定，使 doctest 只有一个答案：

    1. 两个特征值之差的绝对值不超过 tol，并且两条非对角元的绝对值也不超过 tol：
       这是标量矩阵。P 取单位矩阵，D 取 λI。第 14 集「整个平面放大 2 倍」就是这种。
    2. 特征值重了，但不是标量矩阵：不能对角化，抛出 ValueError。
       剪切 [[1, 1], [0, 1]] 与 [[−2, 1], [0, −2]] 都是这种。
    3. 没有实特征值：抛出 ValueError。90° 旋转是这种。
    4. 两个实特征值不同：对每个 λ，取 A−λI 第一条绝对值超过 tol 的行 [u, v]，
       特征向量先写成 [−v, u]。若第一个绝对值超过 tol 的分量是负的，整条乘 −1。
       两个分量离整数都不超过 1e-8 时，取整后再除以它们的最大公因数。

    第 14 集开头的 [[3, 1], [0, 2]] 因此得到列 (1, 0) 与 (1, −1)，
    D = diag(3, 2)。课页上斐波那契矩阵的向量是同一直线的另一刻度，不是这个代表元。

    Args:
        A: 2×2 矩阵。
        tol: 判断重根、零行、标量矩阵非对角元，以及零向量的容差。

    Returns:
        (P, D)，两个新的 2×2 矩阵。不修改 A。
        分量是 int 还是 float 取决于实现，比较时用 ==；无理数先 round。

    Raises:
        ValueError: A 不是 2×2，没有实特征值，或实特征值不足以对角化。

    Examples:
        >>> P, D = diagonalize_2x2([[3, 1], [0, 2]])
        >>> P == [[1, 1], [0, -1]]
        True
        >>> D == [[3, 0], [0, 2]]
        True
        >>> P, D = diagonalize_2x2([[-1, 0], [0, 2]])
        >>> P == [[0, 1], [1, 0]]
        True
        >>> D == [[2, 0], [0, -1]]
        True
        >>> diagonalize_2x2([[2, 0], [0, 2]]) == ([[1, 0], [0, 1]], [[2, 0], [0, 2]])
        True
        >>> diagonalize_2x2([[1, 0], [0, 0]]) == ([[1, 0], [0, 1]], [[1, 0], [0, 0]])
        True
        >>> diagonalize_2x2([[0, 1], [1, 0]]) == ([[1, 1], [1, -1]], [[1, 0], [0, -1]])
        True
        >>> P, D = diagonalize_2x2([[0, 1], [1, 1]])
        >>> [round(x, 9) for x in P[0]] == [1, 1]
        True
        >>> round(P[1][0], 9) == round((1 + 5**0.5) / 2, 9)
        True
        >>> round(P[1][1], 9) == round((1 - 5**0.5) / 2, 9)
        True
        >>> round(D[0][0], 9) == round((1 + 5**0.5) / 2, 9)
        True
        >>> round(D[1][1], 9) == round((1 - 5**0.5) / 2, 9)
        True
        >>> diagonalize_2x2([[1, 1], [0, 1]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 不能对角化
        >>> diagonalize_2x2([[0, -1], [1, 0]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 不能对角化
        >>> diagonalize_2x2([[-2, 1], [0, -2]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 不能对角化
        >>> diagonalize_2x2([[1, 2, 3], [4, 5, 6]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵必须是 2×2
    """
    raise NotImplementedError
