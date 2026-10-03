"""向量与空间 —— 本人手写 (C5)。下面只有骨架，实现留空。

向量用 list[float] 表示。不得 import numpy 或其他数值库。
规格见 EXERCISE.md；数值示例见知识库正文 02。

每个函数的 Examples 是可执行的 doctest，实现后运行：

    python3 -m doctest vectors.py -v

骨架状态下 doctest 报 NotImplementedError 是预期结果。
"""

from typing import TypeAlias

Vector: TypeAlias = list[float]
"""向量：有序的数字列表。int 也可传入（类型检查器把 int 视为 float 的子类型）。"""


def add(v: Vector, w: Vector) -> Vector:
    """按分量相加。

    Args:
        v: 第一个向量。
        w: 第二个向量，与 v 等长。

    Returns:
        新列表，第 i 个分量为 v[i] + w[i]；不修改 v、w。

    Examples:
        >>> add([1, 2], [3, -1])
        [4, 1]
        >>> add([3, 0], [0, -2])      # 3·î + (−2)·ĵ
        [3, -2]
    """
    return [x + y for x, y in zip(v, w)]


def scale(c: float, v: Vector) -> Vector:
    """每个分量乘以标量 c。

    Args:
        c: 标量。
        v: 向量，维数任意。

    Returns:
        新列表，第 i 个分量为 c * v[i]；不修改 v。

    Examples:
        >>> scale(2, [1, 0])
        [2, 0]
        >>> scale(-1.5, [2, -4])
        [-3.0, 6.0]
        >>> scale(0.5, [255, 0, 0])   # 0.5 倍红色，三维同样适用
        [127.5, 0.0, 0.0]
    """
    return [c * x for x in v]


def linear_combination(coeffs: list[float], vectors: list[Vector]) -> Vector:
    """线性组合 sum(c_i * v_i)。

    Args:
        coeffs: 系数 [c_1, ..., c_n]。
        vectors: 向量 [v_1, ..., v_n]，与 coeffs 等长，各向量维数相同。

    Returns:
        c_1·v_1 + ... + c_n·v_n，维数与各 v_i 相同。

    Examples:
        结果可能是 int 或 float，取决于实现，所以用 == 比较：

        >>> linear_combination([2, 1], [[3, 1], [1, 2]]) == [7, 4]      # 2·v + 1·w
        True
        >>> linear_combination([2, -3], [[1, 2], [3, -1]]) == [-7, 7]
        True
    """
    result = [0] * len(vectors[0])
    for c, v in zip(coeffs, vectors):
        result = add(result, scale(c, v))
    return result


def is_collinear(v: Vector, w: Vector, tol: float = 1e-9) -> bool:
    """v 与 w 是否共线：存在 k 使 w = k·v 或 v = k·w。

    Args:
        v: 第一个向量。
        w: 第二个向量，与 v 等长。
        tol: 浮点容差。

    Returns:
        共线返回 True，否则 False。零向量与任何向量共线。

    Examples:
        >>> is_collinear([3, 1], [6, 2])      # w = 2·v
        True
        >>> is_collinear([2, -4], [-3, 6])    # w = −1.5·v
        True
        >>> is_collinear([3, 1], [1, 2])
        False
        >>> is_collinear([0, 0], [5, 7])      # 零向量
        True
    """
    n = len(v)
    for i in range(n):
        for j in range(i + 1, n):
            if abs(v[i] * w[j] - v[j] * w[i]) > tol:
                return False
    return True


def solve_2x2(v: Vector, w: Vector, t: Vector) -> tuple[float, float] | None:
    """求 (a, b) 使 a·v + b·w = t。

    Args:
        v: 二维向量。
        w: 二维向量。
        t: 二维目标向量。

    Returns:
        元组 (a, b)；v 与 w 共线（方程无唯一解）时返回 None。

    Examples:
        >>> solve_2x2([3, 1], [1, 2], [7, 4])    # 2·[3, 1] + 1·[1, 2] = [7, 4]
        (2.0, 1.0)
        >>> solve_2x2([1, 2], [2, -1], [8, 1])   # 2·[1, 2] + 3·[2, −1] = [8, 1]
        (2.0, 3.0)
        >>> print(solve_2x2([3, 1], [6, 2], [7, 4]))   # [6, 2] = 2·[3, 1]，共线
        None
    """
    d = v[0] * w[1] - v[1] * w[0]
    if abs(d) <= 1e-9:
        return None
    a = (t[0] * w[1] - t[1] * w[0]) / d
    b = (v[0] * t[1] - v[1] * t[0]) / d
    return a, b

def coords_in_basis(t: Vector, b1: Vector, b2: Vector) -> tuple[float, float]:
    """t 在基 {b1, b2} 下的坐标 (a, b)，即 a·b1 + b·b2 = t。

    Args:
        t: 二维向量。
        b1: 第一个基向量，二维。
        b2: 第二个基向量，二维。

    Returns:
        元组 (a, b)。

    Raises:
        ValueError: b1 与 b2 共线，不构成基。

    Examples:
        >>> coords_in_basis([4, 2], [1, 1], [1, -1])   # 3·[1, 1] + 1·[1, −1] = [4, 2]
        (3.0, 1.0)
        >>> coords_in_basis([3, -2], [1, 0], [0, 1])   # 标准基下坐标即分量
        (3.0, -2.0)
        >>> coords_in_basis([1, 1], [1, 2], [2, 4])    # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: b1 与 b2 共线，不构成基
    """
    result = solve_2x2(b1, b2, t)
    if result is None:
        raise ValueError("b1 与 b2 共线，不构成基")
    return result


def in_span(t: Vector, v: Vector, w: Vector, tol: float = 1e-9) -> bool:
    """t 是否在 span{v, w} 内，即是否存在 a、b 使 a·v + b·w = t。

    Args:
        t: 三维向量。
        v: 三维向量。
        w: 三维向量，与 v 不共线。
        tol: 浮点容差。

    Returns:
        在张成内返回 True，否则 False。

    Examples:
        >>> in_span([1, 2, 3], [1, 0, 1], [0, 1, 1])   # 1·v + 2·w
        True
        >>> in_span([1, 2, 4], [1, 0, 1], [0, 1, 1])   # 第三个分量对不上
        False
        >>> in_span([3, 6, 1], [1, 2, 0], [2, 4, 1])   # 1·v + 1·w；v、w 的前两个分量共线
        True
        >>> in_span([3, 7, 1], [1, 2, 0], [2, 4, 1])
        False
    """
    raise NotImplementedError


def is_independent_3(v1: Vector, v2: Vector, v3: Vector, tol: float = 1e-9) -> bool:
    """三个三维向量是否线性无关。

    Args:
        v1: 三维向量。
        v2: 三维向量。
        v3: 三维向量。
        tol: 浮点容差。

    Returns:
        线性无关返回 True；任一向量可由其余两个线性表出时返回 False。

    Examples:
        >>> is_independent_3([1, 0, 1], [0, 1, 1], [1, 1, 2])   # v3 = v1 + v2
        False
        >>> is_independent_3([2, 0, 0], [0, 3, 0], [4, 6, 0])   # 都在 xy 平面上
        False
        >>> is_independent_3([1, 0, 0], [0, 1, 0], [0, 0, 1])   # 标准基
        True
    """
    raise NotImplementedError


def weighted_sum(weights: list[float], vectors: list[Vector]) -> Vector:
    """加权平均：权重合法时返回 linear_combination(weights, vectors)。

    Args:
        weights: 权重，每个都非负，总和为 1（容差 1e-9），例如 softmax 的输出。
        vectors: 向量，与 weights 等长。

    Returns:
        sum(weights[i] * vectors[i])。

    Raises:
        ValueError: 有权重为负，或权重和与 1 的差超过 1e-9。

    Examples:
        浮点结果带舍入误差（0.9 可能算成 0.8999999999999999），所以先取整再展示：

        >>> out = weighted_sum([0.7, 0.2, 0.1], [[1, 0], [0, 1], [2, 2]])   # attention 输出
        >>> [round(x, 9) for x in out]
        [0.9, 0.4]
        >>> weighted_sum([0.5, 0.7], [[1, 0], [0, 1]])     # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 权重和不为 1
        >>> weighted_sum([1.5, -0.5], [[1, 0], [0, 1]])    # doctest: +IGNORE_EXCEPTION_DETAIL
        Traceback (most recent call last):
        ValueError: 权重不能为负
    """
    raise NotImplementedError
