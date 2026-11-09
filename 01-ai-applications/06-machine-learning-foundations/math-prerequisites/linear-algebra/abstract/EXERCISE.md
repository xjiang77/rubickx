# Exercise — 抽象向量空间（1.6.5 / F 组）

> C5 约束：实现由本人手写。agent 可以讲解概念、评审代码、维护验证脚本，不得给出完成的实现。

对应 3Blue1Brown《Essence of Linear Algebra》第 16 集 Abstract vector spaces。A 组 `vectors/` 覆盖第 1–2 集。B 组 `matrices/`（第 3–5 集）在 draft PR #10，C 组 `properties/`（第 6–7 集）在 draft PR #11，D 组 `dimensions/`（第 8–11 集）在 draft PR #12，E 组 `eigen/`（第 12–15 集）在 draft PR #13。本目录不依赖那四组文件，也不改它们。

本仓没有这一篇知识正文。`web/topics.json` 里 1.6.5 只登记了正文 01 与 02，路径指向私有库 dragon-vault。下面的数字来自上述公开课。课页只把 x = −4 当作「在同一个输入上看函数值」的例子，没有指定是哪两个函数；和坐标一起出现的两个多项式是 `x^2 + 3x + 5` 与 `4x^7 − 5x^2`，本练习用它们在 x = −4 上的值。

`abstract.ipynb` 只留一句占位，随练习再补。本目录不建 ELI5 页，也不铺 playground。

## 学 → 玩 → 做 → 测

1. **学**：读本文件。函数可以相加、可以放缩；多项式的系数是一组坐标；导数保持加法与数乘；八条公理是一份清单。ELI5 页本轮不建。
2. **玩**：notebook 目前只有一句占位，没有画面。先把下面的手算算完，再写代码。
3. **做**：在 `abstract.py` 里手写函数体。签名、类型标注、Google 风格 docstring 与 Examples 已写好，实现时保留 doctest。只用 Python 列表、标量与 `abs` / `sum` / `zip` / `range`，不得 import numpy。
4. **测**：先跑 doctest，再跑总验证器。骨架状态下失败是预期结果，不得改成 skip。

## 目标

- 函数在固定采样点上的加法与数乘，就是对应输出相加、对应输出乘上同一个数。零函数的取值全是 0。
- 多项式是另一类向量。系数低次在前：`x^2 + 3x + 5` 写成 `[5, 3, 1]`。次数不同时按幂次对齐，缺的高次当作 0。
- 先把系数相加再取值，应与先取值再相加相同。数乘也一样。
- 导数把多项式送到多项式，并且保持加法与数乘。在 `1, x, x^2, ...` 这组基下，它由一张矩阵表示；列是各基函数的导数。
- 一个映射是线性的，当且仅当它保持加法、也保持数乘。清单只对拿来检查的有限个向量和标量负责。
- 一组自己定义的加法与数乘，要先通过八条公理，才谈得上用线性代数的结论。

## 手算

`f(x) = x^2 + 3x + 5`，`g(x) = 4x^7 − 5x^2`。

| 对象 | 坐标或取值 |
|---|---|
| f | `[5, 3, 1]`，f(−4) = 9 |
| g | `[0, 0, −5, 0, 0, 0, 0, 4]`，g(−4) = −65616 |
| f + g | `[5, 3, −4, 0, 0, 0, 0, 4]`，在 −4 处等于 −65607 |
| 2 · [1, 2, 3] | `[2, 4, 6]` |
| x^3 + 5x^2 + 4x + 5 | `[5, 4, 5, 1]` |
| 它的导数 | `[4, 10, 3]`，也就是 4 + 10x + 3x^2 |
| 同一个导数留在 4 维坐标里 | `[4, 10, 3, 0]` |
| (1/9) x^3 − x | `[0, −1, 0, 1/9]` |
| 它的导数 | `[−1, 0, 1/3]`，也就是 (1/3) x^2 − 1 |
| 4x^7 − 5x^2 的导数 | `[0, −10, 0, 0, 0, 0, 28]` |

次数低于 4 的导数矩阵：

```text
[[0, 1, 0, 0],
 [0, 0, 2, 0],
 [0, 0, 0, 3],
 [0, 0, 0, 0]]
```

基的像 `[0]`、`[1]`、`[0, 2]`、`[0, 0, 3]` 配上坐标 `[5, 4, 5, 1]`，仍得到 `[4, 10, 3]`。

## 证明什么

- 第 16 集的点值加法：在 x = −4，9 + (−65616) = −65607。同一对数乘 2，得到 `[2, 4, 6]` 那种「每个输出乘同一个数」。
- 零函数：采样网格上是一串 0；多项式里是 `[0]`。加上它，向量不动；乘上 0，得到零。
- 两个多项式的系数相加，再在 −4、0、1、2 上取值，与先取值再 `add_values` 相同。
- 导数的两条等式，在课页的两个多项式和标量 9、0、−1、2 上成立。每个输出加 1，以及把每个输出换成绝对值或平方，不成立。
- 八条公理对普通采样、对课页上的两个多项式都成立。验证器另外准备了只打破其中一条的自制运算，`axioms_hold` 必须和八个单独判断的合取一致。

## 不在本练习里的事

- 不证明无穷维空间上的内积、特征函数。课页只提到这两个名字。
- 不证明八条公理对所有函数成立。函数只检查给进来的有限样本。
- 不把「对加法和数乘封闭」单独列成一条。函数的返回类型已经要求结果仍是列表。
- 不做一般的矩阵乘法练习。那是 B 组。这里的 `matvec` 只用来把导数矩阵乘到一条系数上。
- 不从 `vectors.py` 导入。采样的等长加法与多项式的按幂次对齐是两套运算。

## 合同

`abstract.py` 提供下列函数。采样与多项式都是 `list[float]`（int 也可以）。返回新列表，不修改输入，也不与输入共享内部列表。整数结果与等值浮点通过 `==` 视为相同（`2 == 2.0`）。

采样（固定网格上的函数值）：

- `add_values(f, g)`：对应分量相加。长度不同抛出 `ValueError`。
- `scale_values(c, f)`：每个分量乘 c。空列表可以。
- `zero_values(n)`：n 个 0。n 必须是非负 int，bool 与浮点都抛出 `ValueError`。0 个点得到 `[]`。
- `values_close(f, g, tol=1e-9)`：长度不同返回 False。否则每个分量满足 `abs(a − b) <= tol * max(1, abs(a), abs(b))`。

多项式（低次在前）：

- `poly_add`、`poly_scale`：按幂次对齐。空列表抛出 `ValueError`。结果去掉末尾绝对值不超过 `1e-12` 的系数；什么都不剩时返回 `[0]`。`[1, 0, 2]` 这种中间的 0 要保留。
- `zero_poly()`：返回 `[0]`。
- `poly_eval(p, x)`、`poly_values(p, xs)`：Horner 或等价的幂次求和。`xs` 可以为空。
- `poly_close(p, q, tol=1e-9)`：较短的一边用高次 0 补齐，再用和 `values_close` 相同的容差。空列表返回 False。
- `poly_derivative(p)`：`x^i` 的系数乘 i 后移到 `x^{i−1}`。常数的导数是 `[0]`。

导数矩阵：

- `derivative_matrix(n)`：n×n。行 i、列 j 从 0 数，`j == i + 1` 时取值 j，否则 0。n 不是正整数时抛出 `ValueError`。
- `matvec(matrix, vector)`：按行点积。矩阵必须是非空矩形，向量长度必须等于列数，否则 `ValueError`。高次上的 0 保留，因此 `matvec(derivative_matrix(4), [5, 4, 5, 1]) == [4, 10, 3, 0]`，而 `poly_derivative` 得到 `[4, 10, 3]`。
- `apply_on_basis(images, coords)`：`sum coords[i] * images[i]`，用 `poly_scale` 与 `poly_add`。长度不同，或某个像是空列表，抛出 `ValueError`。两边都空时返回 `[0]`。

线性：

- `preserves_addition(transform, u, v, add, close, tol=1e-9)`
- `preserves_scaling(transform, scalar, v, scale, close, tol=1e-9)`
- `is_linear_on(transform, vectors, scalars, add, scale, close, tol=1e-9)`：`vectors` 的每一个有序对，以及每一个标量配每一个向量。`scalars` 可以为空，这时只查加法。`vectors` 为空则抛出 `ValueError`。运算或 transform 抛出的异常原样传出。

八条公理，每条一个函数，最后由 `axioms_hold` 做合取。`vectors` 为空抛出 `ValueError`。需要标量的那几条，在 `scalars` 为空时也抛出 `ValueError`。比较都把 `tol` 传给 `close`。

1. `addition_is_commutative(vectors, add, close, tol=1e-9)`
2. `addition_is_associative(vectors, add, close, tol=1e-9)`
3. `zero_is_identity(vectors, zero, add, close, tol=1e-9)`：`v + zero` 与 `zero + v` 都要接近 v。
4. `has_additive_inverse(vectors, zero, add, scale, close, tol=1e-9)`：负向量用 `scale(-1, v)`，两侧相加都要接近 zero。
5. `scale_distributes_over_addition(...)`
6. `scale_distributes_over_scalar_addition(...)`
7. `scale_is_compatible(vectors, scalars, scale, close, tol=1e-9)`：`a·(b·v)` 接近 `(a·b)·v`。
8. `scale_by_one_is_identity(vectors, scale, close, tol=1e-9)`：数 1 不从标量清单里取。

`axioms_hold(vectors, scalars, zero, add, scale, close, tol=1e-9)` 在向量或标量为空时抛出 `ValueError`，否则返回上面八条是否全为真。

第 16 集要复现的数：`add_values([9], [-65616]) == [-65607]`，`poly_add([5, 3, 1], [0, 0, -5, 0, 0, 0, 0, 4]) == [5, 3, -4, 0, 0, 0, 0, 4]`，`poly_derivative([5, 4, 5, 1]) == [4, 10, 3]`，`poly_derivative([0, -1, 0, 1/9])` 在容差内等于 `[-1, 0, 1/3]`，`matvec(derivative_matrix(4), [5, 4, 5, 1]) == [4, 10, 3, 0]`，`apply_on_basis([[0], [1], [0, 2], [0, 0, 3]], [5, 4, 5, 1]) == [4, 10, 3]`。

### 实现时要单独记住的选择

- 采样必须等长。多项式按幂次补 0。把多项式的末尾 0 再接到采样后面，对采样来说是另一个向量；对多项式来说还是同一个函数。`poly_close` 认它们相同，`values_close` 不认。
- 去掉的是绝对值不超过 `1e-12` 的**末尾**系数。`5e-11` 保留，`5e-13` 去掉。
- `is_linear_on` 不会自动把 0 放进标量。要检查 `L(0) = 0`，调用时自己放进 0；它也已经包含在「乘上 0」这一条里。
- 负向量固定用 `(-1) · v`，即使标量清单里没有 −1。
- 交换律只看拿进来的那些向量，不看它们相加之后的新向量。结合律会把和再次送进加法，所以一份只改写「和再加一次」的运算，可以只打破结合律。

## 主验证器

```bash
make test-linalg-abstract
# 或
python3 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/abstract/test_abstract.py

python3 -m doctest 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/abstract/abstract.py -v
```

在 `abstract/` 目录下也可以直接 `python3 -m doctest abstract.py -v`。

`make test-linalg` 仍只跑 A 组 `vectors/`。B 组用 `make test-linalg-matrices`（draft PR #10），C 组用 `make test-linalg-properties`（draft PR #11），D 组用 `make test-linalg-dimensions`（draft PR #12），E 组用 `make test-linalg-eigen`（draft PR #13）。这四个目标本分支都没有。F 组验证器先跑上面的数值（含异常必须抛出 `ValueError`），手写用例全部通过且安装了 numpy 时再做 200 组随机对拍，最后跑 `abstract.py` 的 doctest。骨架状态下预期失败，输出里应看到 `NotImplementedError`，而不是 skip。

实现完成后：doctest 与 `make test-linalg-abstract` 都应 PASS。
