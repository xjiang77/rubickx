# Evaluation-driven development

评估驱动开发。用评估、错误分析和回归检查推动 AI 应用迭代。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2090840747738374568)

## 当前状态

**仅有骨架**。Nanochat eval 尚为骨架；内容策展 harness 可供机制参考，但不代表通用 AI eval 已覆盖。

## 主归属实践

本能力没有独立主归属实现。关联实验见下；待建设时仅保留说明，不生成空代码。

## 跨能力链接

- [Machine learning foundations](../06-machine-learning-foundations/README.md)
- [Content curation decisions](../../04-shaping-the-build/05-content-curation-decisions/README.md)
- [Nanochat eval](../06-machine-learning-foundations/nanochat/systems/09-eval)
- [策展 harness（机制参考）](../../04-shaping-the-build/05-content-curation-decisions/harness)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

没有独立验证 gate。关联课程的测试只证明其自身边界；本能力仍需补充具体任务与验收样本。

## 下一步

先为一个 AI 任务建立数据集、判定规则与失败分析。
