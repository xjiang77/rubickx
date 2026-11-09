"""抽象向量空间 —— 本人手写 (C5)。下面只有骨架，实现留空。

函数在一组固定输入上的取值，以及多项式的系数，都用 list[float] 表示。
不得 import numpy 或其他数值库。
规格见 EXERCISE.md。数值示例取自 3Blue1Brown《Essence of Linear Algebra》
第 16 集：函数按点相加、按点放缩，多项式以 1, x, x^2, ... 为基，
导数是这些向量上的线性映射。
本仓没有这一篇知识正文（笔记在 dragon-vault；topics.json 只登记了正文 01 与 02）。

约定：多项式系数低次在前，下标 i 是 x^i 的系数。
课页上 x^2 + 3x + 5 的坐标是 [5, 3, 1]，4x^7 - 5x^2 的坐标是
[0, 0, -5, 0, 0, 0, 0, 4]。零多项式写成 [0]，不写空列表。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest abstract.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from collections.abc import Callable
from typing import TypeAlias

Vector: TypeAlias = list[float]
"""函数在一组固定输入上的取值。下标是采样点，不是幂次。int 也可传入。"""

Poly: TypeAlias = list[float]
"""多项式系数，低次在前。int 也可传入。末尾绝对值不超过 1e-12 的系数视为没有这一项。"""

Matrix: TypeAlias = list[list[float]]
"""矩阵，按行存放。M[i][j] 是第 i 行第 j 列。"""

AddFn: TypeAlias = Callable[[Vector, Vector], Vector]
"""两个向量相加。样本用 add_values，多项式用 poly_add。"""

ScaleFn: TypeAlias = Callable[[float, Vector], Vector]
"""标量乘向量。样本用 scale_values，多项式用 poly_scale。"""

CloseFn: TypeAlias = Callable[[Vector, Vector, float], bool]
"""在容差 tol 下比较两个结果。第三个参数是 tol。"""

MapFn: TypeAlias = Callable[[Vector], Vector]
"""把一个向量送进一个向量。线性映射的候选。"""


def add_values(f: Vector, g: Vector) -> Vector:
    """按点把两个函数的取值相加。

    第 16 集：(f + g)(x) = f(x) + g(x)。采样点已经排好，这里只加对应的输出。
    课页用 x = -4 说明「在同一个输入上相加」。x^2 + 3x + 5 在 -4 的值是 9，
    4x^7 - 5x^2 在 -4 的值是 -65616，两个输出相加是 -65607。

    Args:
        f: 第一个函数在各个采样点上的取值。
        g: 第二个函数的取值，与 f 等长。

    Returns:
        新列表，第 i 个分量是 f[i] + g[i]。不修改 f、g。

    Raises:
        ValueError: 两次取值的个数不同。

    Examples:
        >>> add_values([9], [-65616])
        [-65607]
        >>> add_values([1, 2], [3, -1])
        [4, 1]
        >>> sample = [9]; other = [-65616]; total = add_values(sample, other); sample[0] = 0; total
        [-65607]
        >>> add_values([1, 2], [3])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 两次取值的个数必须相同
    """
    raise NotImplementedError


def scale_values(c: float, f: Vector) -> Vector:
    """按点把函数的每个取值乘上标量 c。

    第 16 集把这和「每个坐标乘上同一个数」并列：2 · [x, y, z] = [2x, 2y, 2z]。
    乘上 0 得到同样多个 0，也就是这个采样网格上的零函数。

    Args:
        c: 标量。
        f: 函数取值，长度任意，可以是空列表。

    Returns:
        新列表，第 i 个分量是 c * f[i]。不修改 f。

    Examples:
        >>> scale_values(2, [1, 2, 3])
        [2, 4, 6]
        >>> scale_values(0, [9, -65616])
        [0, 0]
        >>> scale_values(-1, [])
        []
        >>> sample = [1, 2]; scaled = scale_values(3, sample); sample[0] = 0; scaled
        [3, 6]
    """
    raise NotImplementedError


def zero_values(n: int) -> Vector:
    """n 个采样点上的零函数。

    零向量把任何函数的取值留在原地：f + 0 = f。空网格的零函数是空列表。

    Args:
        n: 采样点个数，必须是非负 int。bool 不算。

    Returns:
        长度为 n 的新列表，每个分量都是 0。

    Raises:
        ValueError: n 不是非负 int。

    Examples:
        >>> zero_values(3)
        [0, 0, 0]
        >>> zero_values(0)
        []
        >>> zero_values(-1)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 点数必须是非负整数
        >>> zero_values(True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 点数必须是非负整数
    """
    raise NotImplementedError


def values_close(f: Vector, g: Vector, tol: float = 1e-9) -> bool:
    """两次取值是否在容差内相同。

    长度不同则返回 False，不把较短的列表用 0 补齐。
    每个分量用 abs(a - b) <= tol * max(1, abs(a), abs(b))。

    Args:
        f: 第一次取值。
        g: 第二次取值。
        tol: 相对容差。

    Returns:
        等长且每个分量都进了容差时返回 True，否则 False。

    Examples:
        >>> values_close([1, 0], [1.0, 0.0])
        True
        >>> values_close([1, 0], [1 + 1e-12, 0])
        True
        >>> values_close([1, 0], [1 + 1e-6, 0])
        False
        >>> values_close([1, 0], [1])
        False
    """
    raise NotImplementedError


def poly_add(p: Poly, q: Poly) -> Poly:
    """把两个多项式相加，也就是对应幂次的系数相加。

    次数不同时，缺的高次系数当作 0。这和固定网格上的 add_values 不一样：
    网格必须等长，多项式则按幂次对齐。
    结果去掉末尾绝对值不超过 1e-12 的系数。全部抵消时返回 [0]。

    Args:
        p: 第一个多项式，低次在前，不能是空列表。
        q: 第二个多项式，不能是空列表。

    Returns:
        新的系数列表。不修改 p、q。

    Raises:
        ValueError: 任一列表为空。

    Examples:
        >>> poly_add([5, 3, 1], [0, 0, -5, 0, 0, 0, 0, 4]) == [5, 3, -4, 0, 0, 0, 0, 4]
        True
        >>> poly_add([1, -1], [-1, 1]) == [0]
        True
        >>> poly_add([1, 0, 2], [0]) == [1, 0, 2]
        True
        >>> left = [5, 3, 1]; right = [0, 0, -5]; poly_add(left, right) == [5, 3, -4] and left == [5, 3, 1] and right == [0, 0, -5]
        True
        >>> poly_add([], [0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 多项式系数不能是空列表
    """
    raise NotImplementedError


def poly_scale(c: float, p: Poly) -> Poly:
    """每个系数乘上 c。

    c 为 0，或乘完后高次系数绝对值不超过 1e-12 时，去掉这些末项。
    零多项式仍写成 [0]。

    Args:
        c: 标量。
        p: 多项式系数，不能是空列表。

    Returns:
        新的系数列表。不修改 p。

    Raises:
        ValueError: p 是空列表。

    Examples:
        >>> poly_scale(2, [5, 3, 1]) == [10, 6, 2]
        True
        >>> poly_scale(0, [5, 3, 1]) == [0]
        True
        >>> poly_scale(1, [1, 0, 5e-13]) == [1]
        True
        >>> poly_scale(1, [1, 0, 5e-11]) == [1, 0, 5e-11]
        True
        >>> poly_scale(2, [])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 多项式系数不能是空列表
    """
    raise NotImplementedError


def zero_poly() -> Poly:
    """零多项式，也就是零函数。

    Returns:
        [0]。

    Examples:
        >>> zero_poly()
        [0]
        >>> poly_add([5, 3, 1], zero_poly()) == [5, 3, 1]
        True
    """
    raise NotImplementedError


def poly_eval(p: Poly, x: float) -> float:
    """计算多项式在 x 处的值。

    第 16 集在讲函数加法时把输入放在 x = -4。
    x^2 + 3x + 5 在这里等于 9，4x^7 - 5x^2 等于 -65616。

    Args:
        p: 多项式系数，低次在前，不能是空列表。
        x: 输入。

    Returns:
        p(x)。系数和 x 都是整数时，结果可以是 int。

    Raises:
        ValueError: p 是空列表。

    Examples:
        >>> poly_eval([5, 3, 1], -4) == 9
        True
        >>> poly_eval([0, 0, -5, 0, 0, 0, 0, 4], -4) == -65616
        True
        >>> poly_eval([5, 3, 1], 0) == 5
        True
        >>> poly_eval([0], 3) == 0
        True
        >>> poly_eval([], 0)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 多项式系数不能是空列表
    """
    raise NotImplementedError


def poly_values(p: Poly, xs: list[float]) -> Vector:
    """把多项式当成函数，读出它在每个输入上的取值。

    系数相加再取值，应与先取值再 add_values 相同。xs 为空时返回空列表。

    Args:
        p: 多项式系数，不能是空列表。
        xs: 输入列表，顺序就是返回值的顺序。

    Returns:
        新列表，第 i 个分量是 p(xs[i])。不修改 p、xs。

    Raises:
        ValueError: p 是空列表。

    Examples:
        >>> poly_values([5, 3, 1], [-4, 0]) == [9, 5]
        True
        >>> poly_values([1, 1], []) == []
        True
        >>> add_values(poly_values([5, 3, 1], [-4]), poly_values([0, 0, -5, 0, 0, 0, 0, 4], [-4]))
        [-65607]
        >>> poly_values([], [0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 多项式系数不能是空列表
    """
    raise NotImplementedError


def poly_close(p: Poly, q: Poly, tol: float = 1e-9) -> bool:
    """两个多项式是否表示同一个函数。

    较短的一边用高次 0 补齐后再比较，所以 [4, 10, 3] 与 [4, 10, 3, 0] 相同。
    分量的容差与 values_close 相同。空列表不是多项式，返回 False。

    Args:
        p: 第一个系数列表。
        q: 第二个系数列表。
        tol: 相对容差。

    Returns:
        补齐后每个分量都进了容差时返回 True，否则 False。

    Examples:
        >>> poly_close([4, 10, 3], [4, 10, 3, 0])
        True
        >>> poly_close([0], [0, 0, 0])
        True
        >>> poly_close([1, 2], [1, 3])
        False
        >>> poly_close([], [0])
        False
    """
    raise NotImplementedError


def poly_derivative(p: Poly) -> Poly:
    """对多项式求导，结果仍是多项式。

    第 16 集把导数当作函数空间上的线性映射。
    (1/9) x^3 - x 的导数是 (1/3) x^2 - 1。
    x^3 + 5x^2 + 4x + 5 的导数是 3x^2 + 10x + 4，坐标 [4, 10, 3]。
    基函数的导数是矩阵的列：1 的导数是 0，x 的导数是 1，x^2 的导数是 2x，
    x^3 的导数是 3x^2。
    常数的导数是零多项式。输入末尾的 0 不改变结果。

    Args:
        p: 多项式系数，低次在前，不能是空列表。

    Returns:
        导数的系数，已去掉末尾的 0。不修改 p。

    Raises:
        ValueError: p 是空列表。

    Examples:
        >>> poly_derivative([5, 4, 5, 1]) == [4, 10, 3]
        True
        >>> [round(c, 9) for c in poly_derivative([0, -1, 0, 1/9])] == [-1, 0, round(1/3, 9)]
        True
        >>> poly_derivative([0, 0, -5, 0, 0, 0, 0, 4]) == [0, -10, 0, 0, 0, 0, 28]
        True
        >>> poly_derivative([1]) == [0]
        True
        >>> poly_derivative([0, 1]) == [1]
        True
        >>> poly_derivative([0, 0, 1]) == [0, 2]
        True
        >>> poly_derivative([0, 0, 0, 1]) == [0, 0, 3]
        True
        >>> poly_derivative([7]) == [0]
        True
        >>> poly_derivative([0]) == [0]
        True
        >>> poly_derivative([1, 2, 0, 0]) == [2]
        True
        >>> poly_derivative([])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 多项式系数不能是空列表
    """
    raise NotImplementedError


def derivative_matrix(n: int) -> Matrix:
    """次数低于 n 的多项式坐标上，导数的前 n 行 n 列。

    第 16 集的导数矩阵是无穷的，副对角线上依次是 1, 2, 3, ...。
    这里取左上角 n×n：行 i、列 j（从 0 数）在 j = i + 1 时等于 j，其余为 0。
    n = 4 就是次数不超过 3 的坐标：

        [[0, 1, 0, 0],
         [0, 0, 2, 0],
         [0, 0, 0, 3],
         [0, 0, 0, 0]]

    Args:
        n: 坐标个数，必须是正整数。bool 不算。1 表示只保留常数。

    Returns:
        n 个新列表，每行也是新列表。

    Raises:
        ValueError: n 不是正整数。

    Examples:
        >>> derivative_matrix(4) == [[0, 1, 0, 0], [0, 0, 2, 0], [0, 0, 0, 3], [0, 0, 0, 0]]
        True
        >>> derivative_matrix(1) == [[0]]
        True
        >>> derivative_matrix(0)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 空间维数必须是正整数
        >>> derivative_matrix(True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 空间维数必须是正整数
    """
    raise NotImplementedError


def matvec(matrix: Matrix, vector: Vector) -> Vector:
    """矩阵乘向量，按行做点积。

    第 16 集用导数矩阵乘 [5, 4, 5, 1]，得到 [4, 10, 3, 0]：
    常数项只留下 1·4，x 的系数只留下 2·5，x^2 的系数只留下 3·1。
    与 poly_derivative 不同，这里保留空间的维数，最后的 0 不删掉。

    Args:
        matrix: 非空矩形，每行等长，至少有一列。
        vector: 长度等于列数。

    Returns:
        新列表，长度等于行数。不修改 matrix、vector，也不与它们共享内部列表。

    Raises:
        ValueError: 矩阵是空的、不是矩形，或向量长度不等于列数。

    Examples:
        >>> matvec([[0, 1, 0, 0], [0, 0, 2, 0], [0, 0, 0, 3], [0, 0, 0, 0]], [5, 4, 5, 1])
        [4, 10, 3, 0]
        >>> coords = [5, 4, 5, 1]; image = matvec([[0, 1, 0, 0], [0, 0, 2, 0], [0, 0, 0, 3], [0, 0, 0, 0]], coords); coords[0] = 0; image
        [4, 10, 3, 0]
        >>> matvec([[1, 2], [3]], [1, 0])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 矩阵必须是非空的矩形
        >>> matvec([[1, 2], [3, 4]], [1])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 向量长度必须等于矩阵的列数
    """
    raise NotImplementedError


def apply_on_basis(images: list[Poly], coords: list[float]) -> Poly:
    """用基的像和坐标还原线性映射的结果。

    第 16 集：线性映射完全由它把基送去的地方决定。
    坐标 [5, 4, 5, 1] 表示 5·1 + 4·x + 5·x^2 + 1·x^3。
    导数把这些基分别送到 0、1、2x、3x^2，再按同一组坐标相加，得到 4 + 10x + 3x^2。
    若 images[i] 就是第 i 个单项式本身，结果就是这个多项式。
    空的一组坐标表示空的线性组合，结果是零多项式。

    Args:
        images: 每个基向量的像，写成多项式。与 coords 等长。
        coords: 这个向量在这组基下的坐标。

    Returns:
        sum(coords[i] * images[i])，按 poly_scale 与 poly_add 相加，并去掉末尾的 0。
        不修改 images、coords。

    Raises:
        ValueError: 两边长度不同，或某个像是空列表。

    Examples:
        >>> apply_on_basis([[0], [1], [0, 2], [0, 0, 3]], [5, 4, 5, 1]) == [4, 10, 3]
        True
        >>> apply_on_basis([[1], [0, 1], [0, 0, 1]], [5, 3, 1]) == [5, 3, 1]
        True
        >>> apply_on_basis([], []) == [0]
        True
        >>> apply_on_basis([[1]], [])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 基的像与坐标个数必须相同
        >>> apply_on_basis([[]], [1])  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 多项式系数不能是空列表
    """
    raise NotImplementedError


def preserves_addition(
    transform: MapFn,
    u: Vector,
    v: Vector,
    add: AddFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """transform(u + v) 是否接近 transform(u) + transform(v)。

    这是第 16 集线性映射的第一条：L(v + w) = L(v) + L(w)。
    加法和比较都由调用方传入，因为采样网格和多项式的加法不一样。

    Args:
        transform: 待检查的映射。
        u: 第一个向量。
        v: 第二个向量。
        add: 这个空间的加法。
        close: 比较两个结果。调用时第三个参数是 tol。
        tol: 传给 close 的容差。

    Returns:
        close(transform(add(u, v)), add(transform(u), transform(v)), tol)。

    Examples:
        >>> preserves_addition(
        ...     lambda v: [2 * x for x in v],
        ...     [1, 2],
        ...     [3, -1],
        ...     lambda a, b: [x + y for x, y in zip(a, b)],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> preserves_addition(
        ...     lambda v: [x + 1 for x in v],
        ...     [9],
        ...     [-65616],
        ...     lambda a, b: [x + y for x, y in zip(a, b)],
        ...     lambda a, b, tol: a == b,
        ... )
        False
    """
    raise NotImplementedError


def preserves_scaling(
    transform: MapFn,
    scalar: float,
    v: Vector,
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """transform(scalar · v) 是否接近 scalar · transform(v)。

    这是第 16 集线性映射的第二条：L(s v) = s L(v)。

    Args:
        transform: 待检查的映射。
        scalar: 标量 s。
        v: 向量。
        scale: 这个空间的数乘。
        close: 比较两个结果。调用时第三个参数是 tol。
        tol: 传给 close 的容差。

    Returns:
        close(transform(scale(scalar, v)), scale(scalar, transform(v)), tol)。

    Examples:
        >>> preserves_scaling(
        ...     lambda v: [2 * x for x in v],
        ...     3,
        ...     [1, -2],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> preserves_scaling(
        ...     lambda v: [abs(x) for x in v],
        ...     -1,
        ...     [1, -2],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        False
    """
    raise NotImplementedError


def is_linear_on(
    transform: MapFn,
    vectors: list[Vector],
    scalars: list[float],
    add: AddFn,
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """在给定的向量和标量上，transform 是否同时保持加法与数乘。

    对 vectors 里每一对有序组合（包括一个向量配它自己）调用 preserves_addition，
    再对每个标量和每个向量调用 preserves_scaling。
    标量列表可以为空，这时只检查加法。这只覆盖拿来检查的有限样本，
    不是对整个函数空间的证明。
    导数在多项式上应通过。每个输出都加 1 的平移、以及把每个取值平方，都应失败。

    Args:
        transform: 待检查的映射。
        vectors: 至少要有一个向量。
        scalars: 用来检查数乘的标量，可以为空。
        add: 这个空间的加法。
        scale: 这个空间的数乘。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每一对加法和每一个数乘都保持时返回 True，否则 False。

    Raises:
        ValueError: vectors 是空的。add、scale 或 transform 自己抛出的异常原样传出。

    Examples:
        >>> is_linear_on(
        ...     lambda v: [2 * x for x in v],
        ...     [[1, 0], [3, -1]],
        ...     [2, 0, -1],
        ...     lambda a, b: [x + y for x, y in zip(a, b)],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> is_linear_on(
        ...     lambda v: [x + 1 for x in v],
        ...     [[9], [-65616]],
        ...     [],
        ...     lambda a, b: [x + y for x, y in zip(a, b)],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        False
        >>> is_linear_on(
        ...     poly_derivative,
        ...     [[5, 4, 5, 1], [0, -1, 0]],
        ...     [2, -1],
        ...     poly_add,
        ...     poly_scale,
        ...     poly_close,
        ... )
        True
        >>> is_linear_on(lambda v: v, [], [1], poly_add, poly_scale, poly_close)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 至少要有一个向量
    """
    raise NotImplementedError


def addition_is_commutative(
    vectors: list[Vector],
    add: AddFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """样本里的加法是否交换：u + v 与 v + u 接近。

    Args:
        vectors: 至少要有一个向量。只检查这些向量两两相加，不检查它们的和。
        add: 待检查的加法。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每一对都交换时返回 True，否则 False。

    Raises:
        ValueError: vectors 是空的。

    Examples:
        >>> addition_is_commutative(
        ...     [[9], [-65616]],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> addition_is_commutative(
        ...     [[9], [-65616]],
        ...     lambda u, v: list(u),
        ...     lambda a, b, tol: a == b,
        ... )
        False
        >>> addition_is_commutative([], lambda u, v: u, lambda a, b, tol: True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 至少要有一个向量
    """
    raise NotImplementedError


def addition_is_associative(
    vectors: list[Vector],
    add: AddFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """样本里的加法是否结合：(u + v) + w 与 u + (v + w) 接近。

    三层循环会把和再次送进 add，所以被检查的不只有原来的列表。

    Args:
        vectors: 至少要有一个向量。
        add: 待检查的加法。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每一组三个向量都结合时返回 True，否则 False。

    Raises:
        ValueError: vectors 是空的。

    Examples:
        >>> addition_is_associative(
        ...     [[1, 0], [0, 1]],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> addition_is_associative(
        ...     [[1, 0], [0, 1]],
        ...     lambda u, v: [a - b for a, b in zip(u, v)],
        ...     lambda a, b, tol: a == b,
        ... )
        False
        >>> addition_is_associative([], lambda u, v: u, lambda a, b, tol: True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 至少要有一个向量
    """
    raise NotImplementedError


def zero_is_identity(
    vectors: list[Vector],
    zero: Vector,
    add: AddFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """给定的 zero 是否是这些向量的加法单位元。

    两边都查：v + zero 与 zero + v 都要接近 v。

    Args:
        vectors: 至少要有一个向量。
        zero: 候选的零向量。
        add: 待检查的加法。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每个向量两侧都成立时返回 True，否则 False。

    Raises:
        ValueError: vectors 是空的。

    Examples:
        >>> zero_is_identity(
        ...     [[9], [-65616]],
        ...     [0],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> zero_is_identity(
        ...     [[9], [-65616]],
        ...     [1],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda a, b, tol: a == b,
        ... )
        False
    """
    raise NotImplementedError


def has_additive_inverse(
    vectors: list[Vector],
    zero: Vector,
    add: AddFn,
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """每个样本向量加上自己的 (-1) 倍之后，是否接近 zero。

    负向量取 scale(-1, v)，即使标量清单里没有 -1。
    v + (-v) 和 (-v) + v 都要接近 zero。

    Args:
        vectors: 至少要有一个向量。
        zero: 候选的零向量。
        add: 加法。
        scale: 数乘。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每个向量两侧都成立时返回 True，否则 False。

    Raises:
        ValueError: vectors 是空的。

    Examples:
        >>> has_additive_inverse(
        ...     [[9], [-65616]],
        ...     [0],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> has_additive_inverse(
        ...     [[9]],
        ...     [0],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [0] * len(v),
        ...     lambda a, b, tol: a == b,
        ... )
        False
    """
    raise NotImplementedError


def scale_distributes_over_addition(
    vectors: list[Vector],
    scalars: list[float],
    add: AddFn,
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """数乘是否对向量加法分配：a·(u + v) 接近 a·u + a·v。

    Args:
        vectors: 至少要有一个向量。
        scalars: 至少要有一个标量。
        add: 加法。
        scale: 数乘。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每个标量和每一对向量都成立时返回 True，否则 False。

    Raises:
        ValueError: vectors 或 scalars 是空的。

    Examples:
        >>> scale_distributes_over_addition(
        ...     [[1, 0], [0, 1]],
        ...     [2, -3],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> scale_distributes_over_addition(
        ...     [[1, 0], [0, 1]],
        ...     [2],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [c + x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        False
        >>> scale_distributes_over_addition([[1]], [], lambda u, v: u, lambda c, v: v, lambda a, b, tol: True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 至少要有一个标量
    """
    raise NotImplementedError


def scale_distributes_over_scalar_addition(
    vectors: list[Vector],
    scalars: list[float],
    add: AddFn,
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """数乘是否对标量加法分配：(a + b)·v 接近 a·v + b·v。

    Args:
        vectors: 至少要有一个向量。
        scalars: 至少要有一个标量。a、b 都从这里取，包括 a 和 b 相同的情形。
        add: 加法。
        scale: 数乘。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每一对标量和每个向量都成立时返回 True，否则 False。

    Raises:
        ValueError: vectors 或 scalars 是空的。

    Examples:
        >>> scale_distributes_over_scalar_addition(
        ...     [[1, -2]],
        ...     [2, 3],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> scale_distributes_over_scalar_addition(
        ...     [[1, -2]],
        ...     [2, 3],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [c * c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        False
    """
    raise NotImplementedError


def scale_is_compatible(
    vectors: list[Vector],
    scalars: list[float],
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """数乘是否配合：a·(b·v) 接近 (a·b)·v。

    Args:
        vectors: 至少要有一个向量。
        scalars: 至少要有一个标量。
        scale: 数乘。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每一对标量和每个向量都成立时返回 True，否则 False。

    Raises:
        ValueError: vectors 或 scalars 是空的。

    Examples:
        >>> scale_is_compatible(
        ...     [[1, -2]],
        ...     [2, 3],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> scale_is_compatible(
        ...     [[0, 0]],
        ...     [2, 3],
        ...     lambda c, v: [c + x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        False
        >>> scale_is_compatible([], [2], lambda c, v: v, lambda a, b, tol: True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 至少要有一个向量
    """
    raise NotImplementedError


def scale_by_one_is_identity(
    vectors: list[Vector],
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """乘上 1 是否把每个样本向量留在原地。

    这里的 1 不从标量清单里取，总是用数 1。

    Args:
        vectors: 至少要有一个向量。
        scale: 数乘。
        close: 比较两个结果。
        tol: 传给 close 的容差。

    Returns:
        每个向量都成立时返回 True，否则 False。

    Raises:
        ValueError: vectors 是空的。

    Examples:
        >>> scale_by_one_is_identity(
        ...     [[1, 2, 3]],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> scale_by_one_is_identity(
        ...     [[1, 2, 3]],
        ...     lambda c, v: [(c + 1) * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        False
    """
    raise NotImplementedError


def axioms_hold(
    vectors: list[Vector],
    scalars: list[float],
    zero: Vector,
    add: AddFn,
    scale: ScaleFn,
    close: CloseFn,
    tol: float = 1e-9,
) -> bool:
    """这组样本是否通过向量空间的八条公理。

    第 16 集把八条公理当作一份清单：加法与数乘先通过它，线性代数的结论才能用。
    这里只检查拿进来的向量和标量，不是对整个空间的证明。八条都要查：

    1. addition_is_commutative
    2. addition_is_associative
    3. zero_is_identity
    4. has_additive_inverse
    5. scale_distributes_over_addition
    6. scale_distributes_over_scalar_addition
    7. scale_is_compatible
    8. scale_by_one_is_identity

    任一返回 False，结果就是 False。

    Args:
        vectors: 至少要有一个向量。
        scalars: 至少要有一个标量。
        zero: 候选的零向量。
        add: 加法。
        scale: 数乘。
        close: 比较两个结果。
        tol: 传给上面八个函数的容差。

    Returns:
        八条都成立时返回 True，否则 False。

    Raises:
        ValueError: vectors 或 scalars 是空的。

    Examples:
        >>> axioms_hold(
        ...     [[1, 2], [3, -1]],
        ...     [2, -3],
        ...     [0, 0],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        True
        >>> axioms_hold(
        ...     [[1, 2], [3, -1]],
        ...     [2, -3],
        ...     [1, 0],
        ...     lambda u, v: [a + b for a, b in zip(u, v)],
        ...     lambda c, v: [c * x for x in v],
        ...     lambda a, b, tol: a == b,
        ... )
        False
        >>> axioms_hold([[5, 3, 1], [0, 0, -5, 0, 0, 0, 0, 4]], [2, -1], zero_poly(), poly_add, poly_scale, poly_close)
        True
        >>> axioms_hold([], [1], [0], lambda u, v: u, lambda c, v: v, lambda a, b, tol: True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 至少要有一个向量
        >>> axioms_hold([[1]], [], [0], lambda u, v: u, lambda c, v: v, lambda a, b, tol: True)  # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 至少要有一个标量
    """
    raise NotImplementedError
