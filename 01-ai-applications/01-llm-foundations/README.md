# LLM foundations

LLM 基础。理解 token、模型结构、上下文与推理行为。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2090840747738374568)

## 当前状态

**仅有骨架**。Nanochat tokenizer 与 model 提供学习契约，核心实现仍待本人完成。

## 主归属实践

本能力没有独立主归属实现。关联实验见下；待建设时仅保留说明，不生成空代码。

## 跨能力链接

- [Machine learning foundations](../06-machine-learning-foundations/README.md)
- [Nanochat tokenizer](../06-machine-learning-foundations/nanochat/systems/01-tokenizer)
- [Nanochat model](../06-machine-learning-foundations/nanochat/systems/03-model)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

没有独立验证 gate。关联课程的测试只证明其自身边界；本能力仍需补充具体任务与验收样本。

## 下一步

从 tokenizer 的 IO 契约与最小测试开始。
