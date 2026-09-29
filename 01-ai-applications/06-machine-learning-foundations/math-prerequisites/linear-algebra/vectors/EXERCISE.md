# Exercise — 向量与空间（1.6.5 / 正文 02）

> C5 约束：实现由本人手写。agent 可以讲解概念、评审代码、维护验证脚本，不得给出完成的实现。

## 顺序

主线是 notebook `vectors.ipynb`（JupyterLab 打开，从上到下执行），它与 ELI5 页 `web/learn/06-machine-learning-foundations/linear-algebra.html` 同屏对应；每一屏四段：

1. **学**：该屏的说明与公式（Markdown 单元格）。
2. **玩**：运行演示——字符画板、命中目标、像素字母混合、attention 玩具（代码单元格，只看不写；同样的程序也可在 `playground/` 下直接运行）。
3. **做**：在 `vectors.py` 中手写该屏要求的函数，回到 notebook 重新执行「做」单元格确认状态。
4. **测**：该屏的验证单元格；骨架状态下失败是预期结果。

全部通过后运行总验证器 `make test-linalg`，再做题库中的选择题。交付前用 Restart Kernel and Run All Cells 确认从头能跑通；提交前清除输出。

## 目标

只用 Python 列表与标量运算，实现向量的加法与数乘、线性组合、共线判定、二维方程组求解、换基坐标、张成判定与三向量线性无关判定，并用它们复现正文 02 的全部数值示例，包括 attention 输出作为 value 向量加权平均的实例。

## 合同

- `vectors.py` 提供下列函数；向量用 `list[float]` 表示，不得 import numpy 或其他数值库。
  - `add(v, w)`、`scale(c, v)`、`linear_combination(coeffs, vectors)`
  - `is_collinear(v, w, tol=1e-9)`：二者是否共线；零向量与任何向量共线。
  - `solve_2x2(v, w, t)`：求 `(a, b)` 使 `a·v + b·w = t`（v、w、t 为二维）；v 与 w 共线时返回 `None`。
  - `coords_in_basis(t, b1, b2)`：t 在基 `{b1, b2}` 下的坐标；b1、b2 共线时抛出 `ValueError`。
  - `in_span(t, v, w, tol=1e-9)`：三维向量 t 是否在不共线的 v、w 的张成内。
  - `is_independent_3(v1, v2, v3, tol=1e-9)`：三个三维向量是否线性无关。
  - `weighted_sum(weights, vectors)`：权重非负且和为 1 时返回线性组合，否则抛出 `ValueError`。
- 允许的工具：`abs`、`sum`、`zip`、基本算术。浮点比较使用容差。

## 主验证器

```bash
make test-linalg   # 或 python3 01-ai-applications/06-machine-learning-foundations/math-prerequisites/linear-algebra/vectors/test_vectors.py
```

验证器用正文 02 的数值（[7, 4] = 2·[3, 1] + 1·[1, 2]，[8, 1] = 2·[1, 2] + 3·[2, −1]，[4, 2] 在基 {[1, 1], [1, −1]} 下坐标 (3, 1)，[1, 2, 3] 在 span{[1, 0, 1], [0, 1, 1]} 内而 [1, 2, 4] 不在，[1, 1, 2] = [1, 0, 1] + [0, 1, 1]，attention 实例 [0.9, 0.4]）逐项检查；安装了 numpy 时另做 200 组随机对拍。骨架状态下验证器预期失败，不得标记 skip。

## 与学习的连接

先闭卷实现，再运行 `demo.py` 对照 numpy 的写法。正文 02 的「运算示例」与「推理示例」是每个函数的规格说明；实现时若发现正文的条件说得不够（例如零向量、容差），记入知识库该篇的「适用条件与局限」。
