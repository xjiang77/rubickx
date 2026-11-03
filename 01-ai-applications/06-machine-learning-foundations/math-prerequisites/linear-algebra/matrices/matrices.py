"""变换与矩阵 —— 本人手写 (C5)。下面只有骨架，实现留空。

矩阵用 list[list[float]] 按行存放：第 j 列是第 j 个标准基向量的落点。
向量用 list[float] 表示。不得 import numpy 或其他数值库；旋转可用标准库 math。
规格见 EXERCISE.md。数值示例取自 3Blue1Brown《Essence of Linear Algebra》
第 3–5 集；与 A 组 in_span 衔接的数字来自 vectors.py。
本仓没有知识正文 03（笔记在 dragon-vault）。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest matrices.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from typing import TypeAlias

Vector: TypeAlias = list[float]
"""向量：有序的数字列表。与 vectors.py 相同；int 也可传入。"""

Matrix: TypeAlias = list[list[float]]
"""矩阵：按行存放。M[i][j] 是第 j 个基向量落点的第 i 个分量。"""


def apply_matrix(M: Matrix, v: Vector) -> Vector:
    """用矩阵乘向量：结果是各列按 v 的分量做线性组合。

    第 j 列是第 j 个标准基向量的落点，因此
    apply_matrix(M, v) = v[0]·列0 + v[1]·列1 + ...。
    这与 A 组 linear_combination 是同一件事：系数就是 v 的分量。

    Args:
        M: m×n 矩阵，按行存放，n 列对应 n 个基向量的落点。
        v: n 维向量。

    Returns:
        m 维向量。不修改 M、v。

    Raises:
        ValueError: 矩阵为空、各行长度不同，或 len(v) 与列数不同。

    Examples:
        第 3 集：î 落到 (1, −2)、ĵ 落到 (3, 0)，向量 (−1, 2) 落到 (5, 2)。
        结果可能是 int 或 float，所以用 == 比较：

        >>> apply_matrix([[1, 3], [-2, 0]], [-1, 2]) == [5, 2]
        True
        >>> apply_matrix([[0, -1], [1, 0]], [1, 0]) == [0, 1]   # 逆时针 90°
        True
        >>> apply_matrix([[1, 0], [0, 1], [1, 1]], [1, 2]) == [1, 2, 3]  # A 组 in_span
        True
        >>> apply_matrix([[1, 3], [-2, 0]], [0, 0]) == [0, 0]    # 零向量
        True
        >>> apply_matrix([[1, 0], [0, 1]], [1, 2, 3])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 向量维数与矩阵列数不一致
    """
    raise NotImplementedError


def compose(A: Matrix, B: Matrix) -> Matrix:
    """复合：先做 B，再做 A。结果等于矩阵乘积 A @ B（从右往左读）。

    Args:
        A: 后做的变换，形状 m×n。
        B: 先做的变换，形状 n×p。

    Returns:
        m×p 矩阵。不修改 A、B。对任意向量 v（在形状合法时），
        apply_matrix(compose(A, B), v) 与
        apply_matrix(A, apply_matrix(B, v)) 相同。

    Raises:
        ValueError: 任一矩阵为空或各行长度不同，或 A 的列数不等于 B 的行数。

    Examples:
        第 4 集：先逆时针转 90°，再做把 ĵ 送到 (1, 1) 的剪切。
        î 最终落在 (1, 1)，ĵ 最终落在 (−1, 0)。交换顺序得到另一块矩阵。

        >>> compose([[1, 1], [0, 1]], [[0, -1], [1, 0]]) == [[1, -1], [1, 0]]
        True
        >>> compose([[0, -1], [1, 0]], [[1, 1], [0, 1]]) == [[0, -1], [1, 1]]
        True
        >>> compose([[0, 2], [1, 0]], [[1, -2], [1, 0]]) == [[2, 0], [1, -2]]
        True
        >>> compose([[2]], [[3]]) == [[6]]
        True
        >>> compose([[1, 0], [0, 1]], [[1, 0, 0]])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: A 的列数必须等于 B 的行数
    """
    raise NotImplementedError


def scale_2d(sx: float, sy: float) -> Matrix:
    """二维缩放：x 方向乘 sx，y 方向乘 sy。

    Args:
        sx: x 方向的缩放倍数。
        sy: y 方向的缩放倍数。

    Returns:
        2×2 矩阵 [[sx, 0], [0, sy]]。î 落到 (sx, 0)，ĵ 落到 (0, sy)。

    Examples:
        >>> scale_2d(2, 3) == [[2, 0], [0, 3]]
        True
        >>> scale_2d(1, 1) == [[1, 0], [0, 1]]
        True
        >>> scale_2d(0, 0) == [[0, 0], [0, 0]]
        True
    """
    raise NotImplementedError


def rotate_2d(radians: float) -> Matrix:
    """绕原点逆时针旋转，角用弧度。

    Args:
        radians: 逆时针转角。0 是恒等；pi/2 是第 3 集的 90°。

    Returns:
        2×2 矩阵 [[cos θ, −sin θ], [sin θ, cos θ]]。

    Examples:
        余弦在 π/2 附近不是精确的 0，先 round 再用 == 比较
        （−0.0 与 0.0 在 == 下相等）：

        >>> import math
        >>> [[round(x, 9) for x in row] for row in rotate_2d(0)] == [[1.0, 0.0], [0.0, 1.0]]
        True
        >>> [[round(x, 9) for x in row] for row in rotate_2d(math.pi / 2)] == [[0.0, -1.0], [1.0, 0.0]]
        True
        >>> [[round(x, 9) for x in row] for row in rotate_2d(math.pi)] == [[-1.0, 0.0], [0.0, -1.0]]
        True
    """
    raise NotImplementedError


def shear_2d(k: float) -> Matrix:
    """水平剪切：x 变成 x + k·y，y 不变。

    Args:
        k: 剪切系数。k = 1 是第 4 集里跟在旋转后面的那次剪切。

    Returns:
        2×2 矩阵 [[1, k], [0, 1]]。î 留在 (1, 0)，ĵ 落到 (k, 1)。

    Examples:
        >>> shear_2d(1) == [[1, 1], [0, 1]]
        True
        >>> shear_2d(0) == [[1, 0], [0, 1]]
        True
        >>> shear_2d(-1) == [[1, -1], [0, 1]]
        True
    """
    raise NotImplementedError


def apply_3x3(M: Matrix, v: Vector) -> Vector:
    """三维线性变换：3×3 矩阵乘三维向量。

    列仍是 î、ĵ、k̂ 的落点，算法与 apply_matrix 相同，并检查形状。

    Args:
        M: 3×3 矩阵。
        v: 三维向量。

    Returns:
        三维向量。不修改 M、v。

    Raises:
        ValueError: M 不是 3×3，或 v 不是三维。

    Examples:
        第 5 集：绕 y 轴转 90°，î→(0, 0, −1)，ĵ 不动，k̂→(1, 0, 0)。
        (3, 4, 5) 落到 (5, 4, −3)。课上的另一问：给定矩阵与 (1, 0, −2)。

        >>> Y = [[0, 0, 1], [0, 1, 0], [-1, 0, 0]]
        >>> apply_3x3(Y, [3, 4, 5]) == [5, 4, -3]
        True
        >>> L = [[0, 0.5, -0.5], [0, 0.5, 1], [1, 0, 0.5]]
        >>> apply_3x3(L, [1, 0, -2]) == [1, -2, 0]
        True
        >>> apply_3x3([[1, 0], [0, 1]], [1, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: M 必须是 3×3 矩阵且 v 为三维向量
    """
    raise NotImplementedError


def identity(n: int) -> Matrix:
    """n 维恒等变换：每个标准基向量留在原地。

    Args:
        n: 维数，正整数。

    Returns:
        n×n 矩阵，对角线为 1，其余为 0。

    Raises:
        ValueError: n 不是正整数。

    Examples:
        >>> identity(1) == [[1]]
        True
        >>> identity(2) == [[1, 0], [0, 1]]
        True
        >>> identity(0)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: n 必须是正整数
    """
    raise NotImplementedError
