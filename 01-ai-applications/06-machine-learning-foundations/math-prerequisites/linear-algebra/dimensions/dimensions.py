"""维度之间 —— 本人手写 (C5)。下面只有骨架，实现留空。

矩阵用 list[list[float]] 按行存放，向量用 list[float] 表示。
不得 import numpy 或其他数值库。
规格见 EXERCISE.md。数值示例取自 3Blue1Brown《Essence of Linear Algebra》
第 8–11 集，并复用 A 组已经核对过的张成：
1·[1, 0, 1] + 2·[0, 1, 1] = [1, 2, 3]。
本仓没有这一篇知识正文（笔记在 dragon-vault；topics.json 只登记了正文 01 与 02）。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest dimensions.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from typing import TypeAlias

Vector: TypeAlias = list[float]
"""向量：有序的数字列表。与 vectors.py 相同；int 也可传入。"""

Matrix: TypeAlias = list[list[float]]
"""矩阵：按行存放。M[i][j] 是第 j 列（第 j 个输入基向量的落点）的第 i 个分量。"""


def apply_between(M: Matrix, v: Vector) -> Vector:
    """用 m×n 矩阵把 n 维向量送到 m 维。

    第 j 列是第 j 个标准基向量的落点，因此
    apply_between(M, v) = v[0]·列0 + v[1]·列1 + ...。
    行数是输出维数，列数是输入维数。方阵是同维的特例，公式相同。
    1×n 的结果仍是长度为 1 的列表，不是标量；那个标量是 dot_product。

    Args:
        M: m×n 矩阵，按行存放。
        v: n 维向量。

    Returns:
        m 维向量。不修改 M、v。
        分量是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: 矩阵为空、某行长度为 0、各行长度不同，或 len(v) 与列数不同。

    Examples:
        第 8 集：î 落到 (2, −1, −2)，ĵ 落到 (0, 1, 1)，这是 2 维到 3 维。
        画面上的另一块 2×3 把三维基向量送到平面。[1, 1, 1] 的像是三列之和。
        第 9 集：î 落到 1、ĵ 落到 −2 时，(4, 3) 落到 −2，输出是长度为 1 的列表。
        A 组：1·[1, 0, 1] + 2·[0, 1, 1] = [1, 2, 3]。

        >>> apply_between([[2, 0], [-1, 1], [-2, 1]], [1, 0]) == [2, -1, -2]
        True
        >>> apply_between([[2, 0], [-1, 1], [-2, 1]], [0, 1]) == [0, 1, 1]
        True
        >>> apply_between([[2, 0], [-1, 1], [-2, 1]], [1, 1]) == [2, 0, -1]
        True
        >>> apply_between([[2, 0], [-1, 1], [-2, 1]], [0, 0]) == [0, 0, 0]
        True
        >>> apply_between([[3, 1, 4], [1, 5, 9]], [1, 0, 0]) == [3, 1]
        True
        >>> apply_between([[3, 1, 4], [1, 5, 9]], [0, 1, 0]) == [1, 5]
        True
        >>> apply_between([[3, 1, 4], [1, 5, 9]], [0, 0, 1]) == [4, 9]
        True
        >>> apply_between([[3, 1, 4], [1, 5, 9]], [1, 1, 1]) == [8, 15]
        True
        >>> apply_between([[1, -2]], [4, 3]) == [-2]
        True
        >>> apply_between([[1, 2]], [1, 0]) == [1]
        True
        >>> apply_between([[1, 2]], [0, 1]) == [2]
        True
        >>> apply_between([[1, 0], [0, 1], [1, 1]], [1, 2]) == [1, 2, 3]
        True
        >>> apply_between([[2], [-1], [-2]], [3]) == [6, -3, -6]
        True
        >>> apply_between([[1, 0], [0, 1]], [3, -2]) == [3, -2]
        True
        >>> apply_between([[2, 0], [-1, 1], [-2, 1]], [1, 2, 3])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 向量维数与矩阵列数不一致
        >>> apply_between([[1, 2], [3]], [1, 2])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵各行长度必须相同且不能为空
        >>> apply_between([], [1])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵各行长度必须相同且不能为空
    """
    raise NotImplementedError


def dot_product(v: Vector, w: Vector) -> float:
    """对应分量相乘再相加。

    几何上，这是把其中一个向量投影到另一个所在直线后，用投影长度乘另一向量的长度。
    两条向量垂直时结果为 0；指向大体相反时为负。顺序不影响结果。
    与 1×n 矩阵乘向量是同一次计算，但这里返回标量，不是长度为 1 的列表。

    Args:
        v: 向量。
        w: 与 v 等长的向量。

    Returns:
        标量。是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: 任一向量为空，或两者长度不同。

    Examples:
        第 9 集：把坐标配对相乘再相加。(1, 2)·(3, 4) = 1·3 + 2·4。
        (6, 2, 8, 3)·(1, 8, 5, 3) 同样。î 落到 1、ĵ 落到 −2 时，(4, 3) 的点积是 −2。

        >>> dot_product([1, 2], [3, 4]) == 11
        True
        >>> dot_product([3, 4], [1, 2]) == 11
        True
        >>> dot_product([2, 4], [3, 4]) == 22
        True
        >>> dot_product([6, 2, 8, 3], [1, 8, 5, 3]) == 71
        True
        >>> dot_product([1, -2], [4, 3]) == -2
        True
        >>> dot_product([1, 2], [2, -1]) == 0
        True
        >>> dot_product([0, 0], [5, 7]) == 0
        True
        >>> dot_product([3, 4], [3, 4]) == 25
        True
        >>> dot_product([1, 0, 1], [0, 1, 1]) == 1
        True
        >>> dot_product([1, 2, 3], [1, 0, 1]) == 4
        True
        >>> dot_product([1, 2], [3])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 两个向量必须等长且不能为空
        >>> dot_product([], [])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 两个向量必须等长且不能为空
    """
    raise NotImplementedError


def project(v: Vector, direction: Vector, tol: float = 1e-9) -> Vector:
    """把 v 正交投影到 direction 张成的一维子空间。

    返回的是直线上的向量，不是有符号长度。公式是
    ((v · direction) / (direction · direction)) · direction。
    direction 换成它的非零倍数，直线不变，结果相同。
    v 与 direction 垂直时得到零向量。零向量投影到任意非零方向上仍是零向量。
    点积等于「这条投影向量再与 direction 做点积」。

    Args:
        v: 被投影的向量。
        direction: 张成那条直线的方向，不能是零向量。
        tol: 判断 direction 是否为零。direction 与自身的点积不超过 tol 时视为零。

    Returns:
        与 v 等长的向量，落在 direction 所在直线上。不修改 v、direction。
        分量是 int 还是 float 取决于实现，比较时用 ==；有舍入误差时先 round。

    Raises:
        ValueError: 任一向量为空、长度不同，或 direction 与自身的点积不超过 tol。

    Examples:
        第 9 集的 (1, 2) 与 (3, 4)：点积是 11，投影向量再点回去仍是 11。
        (4, 3) 用 (1, −2) 做点积得到 −2；落在直线上的向量是 (−2/5, 4/5)。
        A 组：[1, 2, 3] 投到 [1, 0, 1] 上得到 [2, 0, 2]。

        >>> project([4, 3], [1, 0]) == [4, 0]
        True
        >>> project([3, 4], [6, 8]) == [3, 4]
        True
        >>> project([3, 4], [-6, -8]) == [3, 4]
        True
        >>> project([1, 2], [2, -1]) == [0, 0]
        True
        >>> project([0, 0], [3, 4]) == [0, 0]
        True
        >>> project([1, 2, 3], [1, 0, 1]) == [2, 0, 2]
        True
        >>> onto = project([1, 2], [3, 4])
        >>> [round(x, 9) for x in onto] == [round(33/25, 9), round(44/25, 9)]
        True
        >>> round(dot_product(onto, [3, 4]), 9) == 11
        True
        >>> onto_line = project([4, 3], [1, -2])
        >>> [round(x, 9) for x in onto_line] == [round(-2/5, 9), round(4/5, 9)]
        True
        >>> project([1, 2], [0, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 方向向量不能为零
        >>> project([1, 2, 3], [1, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 两个向量必须等长且不能为空
    """
    raise NotImplementedError


def dual_vector(landings: Vector) -> Vector:
    """从「标准基落在数轴上的位置」得到对偶向量。

    第 9 集：到一维的线性变换由 î、ĵ、… 各自落到的数完全决定。
    这些数排成 1×n 矩阵；把矩阵立起来，就是与之做点积的那个向量。
    因此对偶向量的分量就是 landings 本身。返回新列表，不修改输入。

    Args:
        landings: 各标准基向量落到的数，至少有一个。

    Returns:
        与 landings 等长的新列表。分量是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: landings 为空。

    Examples:
        第 9 集：î→1、ĵ→−2 的对偶向量是 [1, −2]，再与 (4, 3) 点积得到 −2。
        第 8 集末尾的数轴例子：î→1、ĵ→2，对偶向量是 [1, 2]。

        >>> dual_vector([1, -2]) == [1, -2]
        True
        >>> dual_vector([1, 2]) == [1, 2]
        True
        >>> dot_product(dual_vector([1, -2]), [4, 3]) == -2
        True
        >>> apply_between([dual_vector([1, -2])], [4, 3]) == [-2]
        True
        >>> dual_vector([])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 落点列表不能为空
    """
    raise NotImplementedError


def cross_product(v: Vector, w: Vector) -> Vector | float:
    """二维返回有符号面积，三维返回同时垂直于二者的向量。

    二维：v1·w2 − v2·w1，等于以二者为列的行列式。交换顺序变号；共线时为 0。
    三维：右手系，î × ĵ = k̂。分量是
    [v2·w3 − v3·w2, v3·w1 − v1·w3, v1·w2 − v2·w1]。
    结果垂直于 v 和 w，长度等于二者张成的平行四边形面积。交换顺序乘以 −1。
    第 11 集：这个向量与任意 x 的点积，等于以 x、v、w 为列的行列式。

    Args:
        v: 二维或三维向量。
        w: 与 v 等长的二维或三维向量。

    Returns:
        二维时是标量，三维时是向量。不修改 v、w。
        是 int 还是 float 取决于实现，比较时用 ==。

    Raises:
        ValueError: 长度不是 2 或 3，或两者长度不同。

    Examples:
        二维单位正方形的有符号面积是 1；第 6 集那个 2×3 矩形的面积是 6，顺序反过来变号。
        三维里 î × ĵ = k̂，ĵ × î = −k̂。第 8 集的两列 (2, −1, −2) 与 (0, 1, 1) 的叉积是 (1, −2, 2)。

        >>> cross_product([1, 0], [0, 1]) == 1
        True
        >>> cross_product([0, 1], [1, 0]) == -1
        True
        >>> cross_product([3, 0], [0, 2]) == 6
        True
        >>> cross_product([2, -1], [4, -2]) == 0
        True
        >>> cross_product([1, 0, 0], [0, 1, 0]) == [0, 0, 1]
        True
        >>> cross_product([0, 1, 0], [1, 0, 0]) == [0, 0, -1]
        True
        >>> cross_product([0, 1, 0], [0, 0, 1]) == [1, 0, 0]
        True
        >>> cross_product([1, 0, 0], [0, 0, 1]) == [0, -1, 0]
        True
        >>> cross_product([2, -1, -2], [0, 1, 1]) == [1, -2, 2]
        True
        >>> cross_product([1, 2, 3], [2, 4, 6]) == [0, 0, 0]
        True
        >>> cross_product([1, 2, 3, 4], [5, 6, 7, 8])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 叉积只接受等长的二维或三维向量
        >>> cross_product([1, 2], [3, 4, 5])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 叉积只接受等长的二维或三维向量
        >>> cross_product([1], [2])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 叉积只接受等长的二维或三维向量
    """
    raise NotImplementedError
