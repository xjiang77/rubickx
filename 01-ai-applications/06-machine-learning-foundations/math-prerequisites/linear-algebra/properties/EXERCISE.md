# Exercise — 变换的性质（1.6.5 / C 组）

> C5 约束：实现由本人手写。agent 可以讲解概念、评审代码、维护验证脚本，不得给出完成的实现。

对应 3Blue1Brown《Essence of Linear Algebra》第 6–7 集：行列式、逆矩阵、秩、列空间、零空间，以及 Ax = b 什么时候有解。A 组 `vectors/` 覆盖第 1–2 集。B 组 `matrices/`（变换与矩阵，第 3–5 集）在 draft PR #10，本目录不依赖那组文件，也不改它。

本仓没有这一篇知识正文。`web/topics.json` 里 1.6.5 只登记了正文 01 与 02，路径指向私有库 dragon-vault。下面的数字来自上述公开课，并复用 A 组已经核对过的张成。

`properties.ipynb` 只留一句占位，随练习再补。本目录不建 ELI5 页，也不铺 playground。

## 与 A、B 组的衔接

A 组问「目标在不在某几列的张成里」。B 组把列写成「标准基向量的落点」，矩阵乘向量就是用这些列做线性组合。这一组给那件事起名字，并补上缩放面积的倍数：

- 行列式是面积或体积的缩放倍数；变成 0 就是被压扁，于是没有逆。
- 列空间是所有能落到的输出，秩是它的维数。
- Ax = b 有解，当且仅当 b 在列空间里。
- 零空间是被送到原点的那些输入。

A 组有 `in_span([1, 2, 3], [1, 0, 1], [0, 1, 1])` 为真，`in_span([1, 2, 4], ...)` 为假。把这两列排成矩阵，就是

`is_solvable([[1, 0], [0, 1], [1, 1]], [1, 2, 3])` 为真，`[1, 2, 4]` 为假。

第 6 集的剪切 `[[1, 1], [0, 1]]` 与逆时针 90° `[[0, -1], [1, 0]]` 就是 B 组里的那两块矩阵，行列式都是 1。剪切的逆是向左剪 `[[1, -1], [0, 1]]`，旋转的逆是顺时针 90° `[[0, 1], [-1, 0]]`。

## 顺序

1. 读本文件和 `properties.py` 的 docstring。签名、类型标注、Google 风格 docstring 与 Examples 已写好，实现时保留 doctest。
2. 在 `properties.py` 里手写函数体。只用 Python 列表、标量与 `abs` / `sum` / `zip` / `range`，不得 import numpy。
3. 单独跑 doctest，再跑总验证器。骨架状态下失败是预期结果，不得改成 skip。

## 目标

- 行列式表示线性变换把面积（2 维）或体积（3 维）放大了多少；负数表示方向翻转。
- 2×2 在行列式不为 0 时有逆，为 0 时没有。
- 秩是列空间的维数。第 7 集那块 3×3 提问矩阵的秩是 1。
- 列空间的基用从左到右选出的原列，不改分量。
- 零空间的基用行最简形的自由变量特解，约定见下。
- Ax = b 有没有解，看 b 在不在列空间里。本练习不要求把 x 求出来。

## 合同

`properties.py` 提供下列函数。矩阵是 `list[list[float]]`（按行），向量是 `list[float]`。返回新列表，不修改输入。整数结果与等值浮点通过 `==` 视为相同（`2 == 2.0`）。

- `determinant(M)`：1×1、2×2 或 3×3 的行列式。2×2 为 ad − bc；3×3 按第一行余子式展开。空矩阵、参差行、非方阵，或阶数不是 1、2、3 时抛出 `ValueError`。
- `inverse_2x2(M, tol=1e-9)`：2×2 的逆；`abs(det) <= tol` 时返回 `None`。不是 2×2 时抛出 `ValueError`。
- `rank(M, tol=1e-9)`：列空间维数。与 `len(column_space(M, tol))` 相同，也等于列数减去 `len(null_space(M, tol))`。
- `column_space(M, tol=1e-9)`：从左到右，若该列在容差内已落在已选列的张成中（零列永远如此），就跳过；否则把这一列原样放进基。
- `null_space(M, tol=1e-9)`：行最简形（主元为 1，主元列其余为 0）里，按列下标从小到大，每次令一个自由变量为 1、其余自由变量为 0，回代出主元变量。只有零向量时返回 `[]`。
- `is_solvable(M, b, tol=1e-9)`：b 是否在列空间里。`len(b)` 不等于行数时抛出 `ValueError`。

空矩阵与参差行在上述每个函数里都抛出 `ValueError`。

第 6 集要复现的数：`[[3, 0], [0, 2]]` 的行列式是 6；剪切 `[[1, 1], [0, 1]]` 是 1；`[[1, 2], [3, 4]]` 是 −2；`[[1.5, 1], [2, -2]]` 是 −5。被压扁的平行六面体 `[[1, 0, 1], [0.5, 1, 1.5], [1, 0, 1]]` 是 0。课上另一块带小数的 3×3，行列式四舍五入到 6 位是 −0.985625。

第 7 集要复现的数：`[[3, -1], [-9, 3]]` 没有逆。二元方程组 `[[2, 2], [1, 3]]` 把某个 x 送到 `(−4, −1)`，逆是 `[[3/4, -1/2], [-1/4, 1/2]]`。三元方程组 `[[2, 5, 3], [4, 0, 8], [1, 3, 0]]` 的行列式是 28，所以 `(-3, 0, 2)` 有解。秩的提问矩阵 `[[1, -2, 4], [-2, 4, -8], [5, -10, 20]]` 秩为 1，列空间基是 `[[1, -2, 5]]`，零空间基是 `[[2, 1, 0], [-4, 0, 1]]`。

## 主验证器

```bash
make test-linalg-properties
# 或
python3 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/properties/test_properties.py

python3 -m doctest 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/properties/properties.py -v
```

在 `properties/` 目录下也可以直接 `python3 -m doctest properties.py -v`。

`make test-linalg` 仍只跑 A 组 `vectors/`。B 组用 `make test-linalg-matrices`（draft PR #10，本分支没有该目标）。C 组验证器先跑上面的数值（含异常必须抛出 `ValueError`），手写用例全部通过且安装了 numpy 时再做 200 组随机对拍，最后跑 `properties.py` 的 doctest。骨架状态下预期失败，输出里应看到 `NotImplementedError`，而不是 skip。

实现完成后：doctest 与 `make test-linalg-properties` 都应 PASS。
