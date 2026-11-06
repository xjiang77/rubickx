# Machine learning foundations

机器学习基础。通过模型与训练机制的从零实现理解机器学习基础。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2090840747738374568)

## 当前状态

**已有局部实践**。数学前置已有可运行核对；线性代数与 Nanochat 十系统、from-scratch 练习仍是骨架，TODO 与 skip 不算能力完成。

## 主归属实践

- [math-prerequisites](math-prerequisites) — 导数、梯度与链式法则的数值核对（topic 1.6.0）；[线性代数练习](math-prerequisites/linear-algebra)（topic 1.6.5）。
- [nanochat](nanochat) — LLM 十系统与 from-scratch 练习骨架。

## 跨能力链接

- [LLM foundations](../01-llm-foundations/README.md)
- [Evaluation-driven development](../04-evaluation-driven-development/README.md)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

- `make test-math`
- `make test-linalg`（A 组 vectors；不计入 `make test`）
- `make test-linalg-eigen`（E 组「坐标系变换」；骨架状态下预期失败；不计入 `make test`）
- `各系统目录 pytest test_impl.py；实现后 python parity.py`（TODO / skip 不证明实现完成。）

B 组「变换与矩阵」的入口是 `make test-linalg-matrices`（draft PR #10 的 `matrices/`）。C 组「变换的性质」的入口是 `make test-linalg-properties`（draft PR #11 的 `properties/`）。D 组「维度之间」的入口是 `make test-linalg-dimensions`（draft PR #12 的 `dimensions/`）。本分支不改那三组文件。上述线性代数目标都不计入 `make test`。

## 下一步

由本人按 RUNBOOK 完成 micrograd 与逐系统实现，再运行真实测试和对拍。
