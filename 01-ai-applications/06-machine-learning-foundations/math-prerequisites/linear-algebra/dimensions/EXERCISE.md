# Exercise — 维度之间（1.6.5 / D 组）

> C5 约束：实现由本人手写。agent 可以讲解概念、评审代码、维护验证脚本，不得给出完成的实现。

对应 3Blue1Brown《Essence of Linear Algebra》第 8–11 集：非方阵是维数之间的线性变换、点积与对偶、投影到一条过原点的直线、叉积。A 组 `vectors/` 覆盖第 1–2 集。B 组 `matrices/`（第 3–5 集）在 draft PR #10，C 组 `properties/`（第 6–7 集）在 draft PR #11。本目录不依赖那两组文件，也不改它们。

本仓没有这一篇知识正文。`web/topics.json` 里 1.6.5 只登记了正文 01 与 02，路径指向私有库 dragon-vault。下面的数字来自上述公开课，并复用 A 组已经核对过的张成。

`dimensions.ipynb` 只留一句占位，随练习再补。本目录不建 ELI5 页，也不铺 playground。

## 与 A 组的衔接

A 组问目标在不在某几列的张成里。矩阵把同一件事写成变换：第 j 列是第 j 个标准基向量的落点，输入向量的分量是这些列的系数。列数是输入维数，行数是输出维数，所以矩阵可以不是方阵。

A 组有 `1·[1, 0, 1] + 2·[0, 1, 1] = [1, 2, 3]`（`in_span([1, 2, 3], [1, 0, 1], [0, 1, 1])` 为真）。把这两列排成矩阵，就是从 2 维到 3 维：

`apply_between([[1, 0], [0, 1], [1, 1]], [1, 2]) == [1, 2, 3]`。

`[1, 2, 4]` 不在这两列的张成里，所以它不是这块 3×2 矩阵的输出。本练习不求那个输入，只把已知系数送进去。

## 顺序

1. 读本文件和 `dimensions.py` 的 docstring。签名、类型标注、Google 风格 docstring 与 Examples 已写好，实现时保留 doctest。
2. 在 `dimensions.py` 里手写函数体。只用 Python 列表、标量与 `abs` / `sum` / `zip` / `range`，不得 import numpy。
3. 单独跑 doctest，再跑总验证器。骨架状态下失败是预期结果，不得改成 skip。

## 目标

- 非方阵乘向量会改变维数：3×2 从平面到空间，2×3 从空间到平面，1×n 从 n 维到数轴。
- 点积是对应分量相乘再相加。它同时也是「投影长度乘上另一向量的长度」。
- 到一维的线性变换对偶于一个向量：对标准基的取值就是这个向量的分量，做变换等于与它点积。
- 用点积把向量正交投影到一条过原点的直线上，得到的是直线上的向量。
- 二维叉积是有符号面积；三维叉积是同时垂直于二者的向量，右手系，î × ĵ = k̂。

## 证明什么

- 第 8 集那块 3×2：î 落到 (2, −1, −2)，ĵ 落到 (0, 1, 1)，于是 (1, 1) 落到 (2, 0, −1)。画面上的 2×3 `[[3, 1, 4], [1, 5, 9]]` 把三个标准基向量送到两列坐标；(1, 1, 1) 的像是三列之和 (8, 15)。
- 第 9 集：(1, 2)·(3, 4) = 11；(6, 2, 8, 3)·(1, 8, 5, 3) = 71。î→1、ĵ→−2 时，(4, 3) 落到 −2。这个 −2 是点积。落在直线上的向量是 (−2/5, 4/5)，不是 −2。
- 投影是用点积做的：`(v · d) / (d · d) · d`。把 (1, 2) 投到 (3, 4) 上，再与 (3, 4) 点积，仍得 11。
- 第 10–11 集：二维 `[1, 0] × [0, 1] = 1`，交换顺序得 −1。三维 `[1, 0, 0] × [0, 1, 0] = [0, 0, 1]`。第 8 集的两列叉积是 `[1, -2, 2]`，它与两列的点积都是 0，与自身的点积是 9（面积的平方）。

## 不证明什么

- 不证明一般行列式、逆、秩、列空间或零空间。那是 C 组。
- 不证明矩阵复合。那是 B 组。这里只做矩阵乘向量。
- 不把向量投影到高于一维的子空间，也不做最小二乘。
- 不做 4 维及以上的叉积。二维结果是标量，三维结果是向量。
- 不求 Ax = b 的解。A 组已经会在二维里找系数。

## 合同

`dimensions.py` 提供下列函数。矩阵是 `list[list[float]]`（按行），向量是 `list[float]`。返回新列表，不修改输入。整数结果与等值浮点通过 `==` 视为相同（`2 == 2.0`）。

- `apply_between(M, v)`：m×n 矩阵乘 n 维向量，得到 m 维向量。方阵也接受。1×n 的结果是长度为 1 的列表。空矩阵、零长度的行、参差行，或 `len(v)` 与列数不同时抛出 `ValueError`。
- `dot_product(v, w)`：对应分量相乘再相加，返回标量。任一为空或长度不同时抛出 `ValueError`。
- `project(v, direction, tol=1e-9)`：v 到 span{direction} 的正交投影向量。direction 与自身的点积不超过 tol 时抛出 `ValueError`。长度不同或有空向量时同样抛出。
- `dual_vector(landings)`：对偶向量，分量就是这些落点；返回新列表。landings 为空时抛出 `ValueError`。
- `cross_product(v, w)`：二维返回 `v1·w2 − v2·w1`；三维返回上面的向量。长度不是 2 或 3，或两者长度不同时抛出 `ValueError`。

第 8 集要复现的数：`[[2, 0], [-1, 1], [-2, 1]]` 把 `[1, 0]`、`[0, 1]`、`[1, 1]` 分别送到 `[2, -1, -2]`、`[0, 1, 1]`、`[2, 0, -1]`。2×3 `[[3, 1, 4], [1, 5, 9]]` 把 `[1, 1, 1]` 送到 `[8, 15]`。

第 9 集要复现的数：`dot_product([1, 2], [3, 4]) == 11`，`dot_product([6, 2, 8, 3], [1, 8, 5, 3]) == 71`，`dot_product([1, -2], [4, 3]) == -2`。`project([4, 3], [1, -2])` 四舍五入后是 `[-2/5, 4/5]`。

第 10–11 集要复现的数：`cross_product([3, 0], [0, 2]) == 6`，`cross_product([1, 0, 0], [0, 1, 0]) == [0, 0, 1]`，`cross_product([2, -1, -2], [0, 1, 1]) == [1, -2, 2]`。

## 主验证器

```bash
make test-linalg-dimensions
# 或
python3 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/dimensions/test_dimensions.py

python3 -m doctest 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/dimensions/dimensions.py -v
```

在 `dimensions/` 目录下也可以直接 `python3 -m doctest dimensions.py -v`。

`make test-linalg` 仍只跑 A 组 `vectors/`。B 组用 `make test-linalg-matrices`（draft PR #10），C 组用 `make test-linalg-properties`（draft PR #11）。这两个目标本分支都没有。D 组验证器先跑上面的数值（含异常必须抛出 `ValueError`），手写用例全部通过且安装了 numpy 时再做 200 组随机对拍，最后跑 `dimensions.py` 的 doctest。骨架状态下预期失败，输出里应看到 `NotImplementedError`，而不是 skip。

实现完成后：doctest 与 `make test-linalg-dimensions` 都应 PASS。
