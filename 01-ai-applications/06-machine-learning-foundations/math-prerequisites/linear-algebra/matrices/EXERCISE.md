# Exercise — 变换与矩阵（1.6.5 / B 组）

> C5 约束：实现由本人手写。agent 可以讲解概念、评审代码、维护验证脚本，不得给出完成的实现。

对应 3Blue1Brown《Essence of Linear Algebra》第 3–5 集：线性变换、矩阵等于基向量的落点、矩阵乘向量、复合等于矩阵乘法、三维。A 组 `vectors/` 覆盖第 1–2 集，本目录不改那组实现。

本仓没有知识正文 03。`web/topics.json` 里 1.6.5 只登记了正文 01 与 02，路径指向私有库 dragon-vault；矩阵这一篇不在 rubickx 的任何目录。下面的数字来自上述公开课，并复用 A 组已经核对过的线性组合。

`matrices.ipynb` 只留一句占位，随练习再补。本目录不建 ELI5 页，也不铺 playground。

## 与 A 组的衔接

`linear_combination` 在系数已知时算出组合；`in_span` 问的是有没有这样的系数。矩阵把同一件事写成变换：第 j 列是第 j 个标准基向量的落点，向量的分量就是这些列的系数。

A 组有 `1·[1, 0, 1] + 2·[0, 1, 1] = [1, 2, 3]`（`in_span([1, 2, 3], [1, 0, 1], [0, 1, 1])` 为真）。把这两列排成矩阵，就是

`apply_matrix([[1, 0], [0, 1], [1, 1]], [1, 2]) == [1, 2, 3]`。

`solve_2x2` / `coords_in_basis` 负责找系数；`apply_matrix` 负责用系数算出落点。

## 顺序

1. 读本文件和 `matrices.py` 的 docstring。签名、类型标注、Google 风格 docstring 与 Examples 已写好，实现时保留 doctest。
2. 在 `matrices.py` 里手写函数体。只用 Python 列表、标量、`math`（旋转）与 `zip` / `sum` 一类基本工具，不得 import numpy。
3. 单独跑 doctest，再跑总验证器。骨架状态下失败是预期结果，不得改成 skip。

## 目标

- 线性变换由标准基向量的落点完全决定；这些落点排成矩阵的列。
- 矩阵乘向量 = 用向量分量对列做线性组合（第 3 集）。
- 先做 B 再做 A 的复合，等于乘积 A @ B，从右往左读（第 4 集）。
- 二维的缩放、逆时针旋转、水平剪切能写成上述矩阵。
- 三维同样：3×3 的三列是 î、ĵ、k̂ 的落点（第 5 集）。

## 合同

`matrices.py` 提供下列函数。矩阵是 `list[list[float]]`（按行），向量是 `list[float]`。返回新列表，不修改输入。

- `apply_matrix(M, v)`：m×n 矩阵乘 n 维向量，得到 m 维向量。空矩阵、各行长度不同、或 `len(v)` 与列数不同时抛出 `ValueError`。
- `compose(A, B)`：先做 B，再做 A，即 A @ B。A 为 m×n、B 为 n×p 时得到 m×p；内维对不上时抛出 `ValueError`。
- `scale_2d(sx, sy)`：返回 `[[sx, 0], [0, sy]]`。
- `rotate_2d(radians)`：绕原点逆时针旋转，返回 `[[cos θ, -sin θ], [sin θ, cos θ]]`。
- `shear_2d(k)`：水平剪切，x 变为 x + k·y，返回 `[[1, k], [0, 1]]`。k = 1 是第 4 集里的剪切。
- `apply_3x3(M, v)`：与 `apply_matrix` 相同，但 M 必须是 3×3、v 必须是三维，否则抛出 `ValueError`。
- `identity(n)`：n×n 单位矩阵。n 不是正整数时抛出 `ValueError`。

第 3 集要复现的数：矩阵 `[[1, 3], [-2, 0]]`（î→(1, −2)，ĵ→(3, 0)）把 (−1, 2) 送到 (5, 2)；逆时针 90° 的矩阵是 `[[0, -1], [1, 0]]`。

第 4 集要复现的数：剪切 `[[1, 1], [0, 1]]` 放在旋转 `[[0, -1], [1, 0]]` 之后，复合为 `[[1, -1], [1, 0]]`。另一例：先 `[[1, -2], [1, 0]]` 再 `[[0, 2], [1, 0]]`，复合为 `[[2, 0], [1, -2]]`。顺序反过来是另一块矩阵。

第 5 集要复现的数：绕 y 轴转 90° 的矩阵 `[[0, 0, 1], [0, 1, 0], [-1, 0, 0]]` 把 (3, 4, 5) 送到 (5, 4, −3)。课上另一问的矩阵 `[[0, 0.5, -0.5], [0, 0.5, 1], [1, 0, 0.5]]` 把 (1, 0, −2) 送到 (1, −2, 0)。

## 主验证器

```bash
make test-linalg-matrices
# 或
python3 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/matrices/test_matrices.py

python3 -m doctest 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/matrices/matrices.py -v
```

在 `matrices/` 目录下也可以直接 `python3 -m doctest matrices.py -v`。

`make test-linalg` 仍只跑 A 组 `vectors/`，本练习不改那个目标。B 组验证器先跑上面的数值（含异常必须抛出 `ValueError`），手写用例全部通过且安装了 numpy 时再做 200 组随机对拍，最后跑 `matrices.py` 的 doctest。骨架状态下预期失败，输出里应看到 `NotImplementedError`，而不是 skip。

实现完成后：doctest 与 `make test-linalg-matrices` 都应 PASS。
