# Machine learning foundations

机器学习基础。通过模型与训练机制的从零实现理解机器学习基础。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2090840747738374568)

## 当前状态

**已有局部实践**。数学前置已有可运行核对；Nanochat 十系统与 from-scratch 练习仍是骨架，TODO 与 skip 不算能力完成。

## 主归属实践

- [math-prerequisites](math-prerequisites) — 导数、梯度与链式法则的数值核对（topic 1.6.0）。
- [nanochat](nanochat) — LLM 十系统与 from-scratch 练习骨架。

## 跨能力链接

- [LLM foundations](../01-llm-foundations/README.md)
- [Evaluation-driven development](../04-evaluation-driven-development/README.md)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

- `make test-math`
- `各系统目录 pytest test_impl.py；实现后 python parity.py`（TODO / skip 不证明实现完成。）

## 下一步

由本人按 RUNBOOK 完成 micrograd 与逐系统实现，再运行真实测试和对拍。
