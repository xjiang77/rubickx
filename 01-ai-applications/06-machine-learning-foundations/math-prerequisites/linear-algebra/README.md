# Linear algebra — 练习

对应知识库 `02_Knowledge/01-ai-applications/06-machine-learning-foundations/01-linear-algebra/`（topic 1.6.5）。一组概念一个练习目录，与知识正文同编号；每个目录按同一节奏分四段：

| 段 | 内容 | 位置 |
|---|---|---|
| 学 | ELI5 图解页，从知识正文派生；notebook 的 Markdown 单元格与之同屏对应 | `web/learn/06-machine-learning-foundations/<slug>.html`，`<目录>/<name>.ipynb` |
| 玩 | 能直接运行、有画面或交互的小程序，只看不写 | `<目录>/<name>.ipynb` 的演示单元格；同样的程序在 `<目录>/playground/` |
| 做 | 骨架函数由本人手写，不得引入 numpy 等库替代（C5） | `<目录>/<name>.py` |
| 测 | 验证器（骨架状态下预期失败，不得改为 skip）与题库 | `<目录>/test_<name>.py`，`web/quiz/` |

| 目录 | 对应正文 | 学 | 验证入口 |
|---|---|---|---|
| [vectors](vectors) | 02 - AI - 向量与空间：三种视角、线性组合与基 | [ELI5 页](../../../../web/learn/06-machine-learning-foundations/linear-algebra.html) | `make test-linalg` |

四段合在一个 notebook 里按顺序走（`vectors.ipynb`，JupyterLab 打开）。`demo.py` 用 numpy 复现正文中的示例，只供闭卷实现之后对照。notebook 提交前清除输出，交付前 Restart Kernel and Run All Cells 一次跑通。
