"""变换的性质 —— 本人手写 (C5)。下面只有骨架，实现留空。

矩阵用 list[list[float]] 按行存放，向量用 list[float] 表示。
不得 import numpy 或其他数值库。
规格见 EXERCISE.md。数值示例取自 3Blue1Brown《Essence of Linear Algebra》
第 6–7 集，并复用 A 组已经核对过的张成：
1·[1, 0, 1] + 2·[0, 1, 1] = [1, 2, 3]，而 [1, 2, 4] 不在这个张成里。
本仓没有这一篇知识正文（笔记在 dragon-vault；topics.json 只登记了正文 01 与 02）。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest properties.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from typing import TypeAlias

Vector: TypeAlias = list[float]
"""向量：有序的数字列表。与 vectors.py 相同；int 也可传入。"""

Matrix: TypeAlias = list[list[float]]
"""矩阵：按行存放。M[i][j] 是第 j 列的第 i 个分量。"""


def determinant(M: Matrix) -> float:
    """方阵的行列式：面积（2 维）或体积（3 维）的缩放倍数，含方向。

    2×2 的公式是 ad − bc。3×3 按第一行余子式展开。
    1×1 的行列式就是那个数。非方阵，以及 4×4 及以上，不在本函数的范围内。

    Args:
        M: 1×1、2×2 或 3×3 方阵。

    Returns:
        行列式。正数表示方向不变，负数表示方向翻转，0 表示被压扁。
        结果是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: 矩阵为空、各行长度不同、不是方阵，或阶数不是 1、2、3。

    Examples:
        第 6 集：î 放到 (3, 0)、ĵ 放到 (0, 2)，单位正方形变成 2×3 矩形，面积 6。
        剪切不改面积。[[1, 2], [3, 4]] 把方向翻过来，行列式是 −2。

        >>> determinant([[3, 0], [0, 2]]) == 6
        True
        >>> determinant([[1, 1], [0, 1]]) == 1
        True
        >>> determinant([[0, -1], [1, 0]]) == 1
        True
        >>> determinant([[1, 2], [3, 4]]) == -2
        True
        >>> determinant([[1.5, 1], [2, -2]]) == -5
        True
        >>> determinant([[3, -1], [-9, 3]]) == 0
        True
        >>> determinant([[0, 0], [0, 0]]) == 0
        True
        >>> determinant([[7]]) == 7
        True
        >>> determinant([[0, 0, 1], [0, 1, 0], [-1, 0, 0]]) == 1
        True
        >>> determinant([[1.0, 0.0, 1.0], [0.5, 1.0, 1.5], [1.0, 0.0, 1.0]]) == 0
        True
        >>> round(determinant([[-1.0, 0.0, 0.05], [0.15, 1.0, 0.05], [0.0, 0.25, 1.0]]), 6) == -0.985625
        True
        >>> determinant([[2, 5, 3], [4, 0, 8], [1, 3, 0]]) == 28
        True
        >>> determinant([[1, 2, 3], [4, 5, 6]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 只接受 1×1、2×2 或 3×3 方阵
    """
    raise NotImplementedError


def inverse_2x2(M: Matrix, tol: float = 1e-9) -> Matrix | None:
    """2×2 矩阵的逆。行列式的绝对值不超过 tol 时没有逆，返回 None。

    逆把变换倒着放：先做 M 再做逆，回到恒等。
    [[a, b], [c, d]] 在 det = ad − bc 不为 0 时，逆是
    (1/det) · [[d, −b], [−c, a]]。

    Args:
        M: 2×2 矩阵。
        tol: 判断 det 是否为 0 的容差。

    Returns:
        2×2 逆矩阵；不可逆时返回 None。不修改 M。
        分量是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: M 不是 2×2（含空矩阵、参差行、其他阶数）。

    Examples:
        第 7 集：逆时针 90° 的逆是顺时针 90°；向右剪切的逆是向左剪切；
        180° 旋转的逆是它自己。[[3, −1], [−9, 3]] 的行列式为 0，没有逆。

        >>> inverse_2x2([[0, -1], [1, 0]]) == [[0, 1], [-1, 0]]
        True
        >>> inverse_2x2([[1, 1], [0, 1]]) == [[1, -1], [0, 1]]
        True
        >>> inverse_2x2([[-1, 0], [0, -1]]) == [[-1, 0], [0, -1]]
        True
        >>> inverse_2x2([[1, 0], [0, 1]]) == [[1, 0], [0, 1]]
        True
        >>> inverse_2x2([[2, 2], [1, 3]]) == [[3/4, -1/2], [-1/4, 1/2]]
        True
        >>> got = [[round(x, 9) for x in row] for row in inverse_2x2([[3, 0], [0, 2]])]
        >>> got == [[round(1/3, 9), 0.0], [0.0, 0.5]]
        True
        >>> print(inverse_2x2([[3, -1], [-9, 3]]))
        None
        >>> print(inverse_2x2([[0, 0], [0, 0]]))
        None
        >>> inverse_2x2([[1, 0, 0], [0, 1, 0], [0, 0, 1]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 只接受 2×2 矩阵
    """
    raise NotImplementedError


def rank(M: Matrix, tol: float = 1e-9) -> int:
    """列空间的维数，也就是输出空间的维数。

    满秩指秩等于列数。零矩阵的秩是 0。
    与 len(column_space(M, tol)) 相同，也等于列数减去 len(null_space(M, tol))。

    Args:
        M: 矩阵，行数和列数至少为 1，各行等长。练习中的例子不超过 3×3。
        tol: 判断一个列是否已落在已选列张成内的容差。

    Returns:
        非负整数。

    Raises:
        ValueError: 矩阵为空或各行长度不同。

    Examples:
        第 7 集的提问：三列共线，秩为 1。压到一张平面上是秩 2，不是秩 1。

        >>> rank([[3, 0], [0, 2]]) == 2
        True
        >>> rank([[3, -1], [-9, 3]]) == 1
        True
        >>> rank([[1, -2, 4], [-2, 4, -8], [5, -10, 20]]) == 1
        True
        >>> rank([[2, 5, 3], [4, 0, 8], [1, 3, 0]]) == 3
        True
        >>> rank([[1, 0, 1], [0, 1, 1], [0, 0, 0]]) == 2
        True
        >>> rank([[1, 0], [0, 1], [1, 1]]) == 2
        True
        >>> rank([[1, 2, 3]]) == 1
        True
        >>> rank([[0, 0], [0, 0]]) == 0
        True
        >>> rank([[1, 2], [3]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵各行长度必须相同且不能为空
    """
    raise NotImplementedError


def column_space(M: Matrix, tol: float = 1e-9) -> list[Vector]:
    """列空间的基：从左到右选出的主元列，分量保持原样。

    某一列在容差 tol 内已落在更左侧已选列的张成中时，不进入基。
    零列永远不进入基。零矩阵返回空列表。
    不要换成单位化后的向量，也不要改用行。
    第 7 集那块秩 1 矩阵的第一列是 [1, −2, 5]，第一行才是 [1, −2, 4]。

    Args:
        M: 矩阵，行数和列数至少为 1，各行等长。
        tol: 判断线性相关的容差。

    Returns:
        基向量列表，按被选中的列从左到右排列。不修改 M。

    Raises:
        ValueError: 矩阵为空或各行长度不同。

    Examples:
        >>> column_space([[1, -2, 4], [-2, 4, -8], [5, -10, 20]]) == [[1, -2, 5]]
        True
        >>> column_space([[1, 0], [0, 1], [1, 1]]) == [[1, 0, 1], [0, 1, 1]]
        True
        >>> column_space([[2, 2], [1, 3]]) == [[2, 1], [2, 3]]
        True
        >>> column_space([[1, 0, 1], [0, 1, 1], [0, 0, 0]]) == [[1, 0, 0], [0, 1, 0]]
        True
        >>> column_space([[1, 2], [2, 4]]) == [[1, 2]]
        True
        >>> column_space([[0, 1], [0, 0]]) == [[1, 0]]
        True
        >>> column_space([[1, 2, 3]]) == [[1]]
        True
        >>> column_space([[0, 0], [0, 0]]) == []
        True
        >>> column_space([[1, 0], [2]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵各行长度必须相同且不能为空
    """
    raise NotImplementedError


def null_space(M: Matrix, tol: float = 1e-9) -> list[Vector]:
    """零空间的基：行最简形里，每个自由变量取 1、其余自由变量取 0 的特解。

    行最简形把主元化成 1，主元列的其他位置化成 0。自由变量按列下标从小到大
    各做一次。只含零向量时返回空列表（满列秩）。零矩阵返回这组标准基。
    这组基是约定好的，不是任意一组张成向量：[[1, 2], [2, 4]] 的基是
    [[−2, 1]]，不是 [[2, −1]] 或其他倍数。

    Args:
        M: 矩阵，行数和列数至少为 1，各行等长。
        tol: 消元时把绝对值不超过 tol 的数当成 0。

    Returns:
        基向量列表。分量是 int 还是 float 取决于实现，比较时用 ==。
        不修改 M。

    Raises:
        ValueError: 矩阵为空或各行长度不同。

    Examples:
        第 7 集秩 1 矩阵的列都是 [1, −2, 5] 的倍数，零空间是一张平面。
        [[3, −1], [−9, 3]] 的特解带有 1/3，先 round 再比较。

        >>> null_space([[1, -2, 4], [-2, 4, -8], [5, -10, 20]]) == [[2, 1, 0], [-4, 0, 1]]
        True
        >>> null_space([[1, 2], [2, 4]]) == [[-2, 1]]
        True
        >>> got = [[round(x, 9) for x in v] for v in null_space([[3, -1], [-9, 3]])]
        >>> got == [[round(1/3, 9), 1.0]]
        True
        >>> null_space([[1, 0, 1], [0, 1, 1]]) == [[-1, -1, 1]]
        True
        >>> null_space([[1, 0, 1], [0, 1, 1], [0, 0, 0]]) == [[-1, -1, 1]]
        True
        >>> null_space([[1, 2, 3]]) == [[-2, 1, 0], [-3, 0, 1]]
        True
        >>> null_space([[2, 2], [1, 3]]) == []
        True
        >>> null_space([[1, 0], [0, 1], [1, 1]]) == []
        True
        >>> null_space([[0, 0], [0, 0]]) == [[1, 0], [0, 1]]
        True
        >>> null_space([[0, 1], [0, 0]]) == [[1, 0]]
        True
        >>> null_space([[1, 0], [1]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵各行长度必须相同且不能为空
    """
    raise NotImplementedError


def is_solvable(M: Matrix, b: Vector, tol: float = 1e-9) -> bool:
    """Ax = b 是否有解：b 是否落在列空间里。

    方阵满秩时，每一个维数相符的 b 都有解。秩不足时，只有落在列空间里的 b 有解。
    零向量总有解。这里只判断有没有解，不返回 x。

    Args:
        M: 系数矩阵，行数和列数至少为 1，各行等长。
        b: 右侧向量，维数必须等于 M 的行数。
        tol: 与 column_space 相同的容差。

    Returns:
        有解返回 True，否则 False。

    Raises:
        ValueError: 矩阵为空、各行长度不同，或 len(b) 不等于行数。

    Examples:
        第 7 集：[[2, 2], [1, 3]] 作用后能落到 (−4, −1)；三元方程组同样有解。
        [[3, −1], [−9, 3]] 只能落到 [1, −3] 这条线上，[1, 0] 到不了。
        A 组： [1, 2, 3] 在 span{[1, 0, 1], [0, 1, 1]} 内，[1, 2, 4] 不在。

        >>> is_solvable([[2, 2], [1, 3]], [-4, -1])
        True
        >>> is_solvable([[2, 5, 3], [4, 0, 8], [1, 3, 0]], [-3, 0, 2])
        True
        >>> is_solvable([[3, -1], [-9, 3]], [1, -3])
        True
        >>> is_solvable([[3, -1], [-9, 3]], [1, 0])
        False
        >>> is_solvable([[3, -1], [-9, 3]], [0, 0])
        True
        >>> is_solvable([[1, -2, 4], [-2, 4, -8], [5, -10, 20]], [2, -4, 10])
        True
        >>> is_solvable([[1, -2, 4], [-2, 4, -8], [5, -10, 20]], [1, 0, 0])
        False
        >>> is_solvable([[1, 0], [0, 1], [1, 1]], [1, 2, 3])
        True
        >>> is_solvable([[1, 0], [0, 1], [1, 1]], [1, 2, 4])
        False
        >>> is_solvable([[1, 2, 3]], [5])
        True
        >>> is_solvable([[0, 0], [0, 0]], [0, 0])
        True
        >>> is_solvable([[0, 0], [0, 0]], [1, 0])
        False
        >>> is_solvable([[1, 0], [0, 1]], [1, 2, 3])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: b 的维数必须等于矩阵行数
    """
    raise NotImplementedError
