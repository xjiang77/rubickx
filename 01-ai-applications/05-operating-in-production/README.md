# Operating in production

AI 生产运维。持续观察 AI 应用的质量、成本、延迟与行为变化。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2090840747738374568)

## 当前状态

**仅有骨架**。当前没有 AI 线上运维实践；通用 CI 不证明漂移检测、模型质量或成本控制。

topic 1.5.1 有一个 CPU 概念练习骨架：[kv-cache-lab](kv-cache-lab/README.md)。它只核对 KV cache 的形状、token-forward 次数、内存字节和前缀复用规则。

## 主归属实践

本能力没有独立主归属实现。kv-cache-lab 登记在 topic 1.5.1，不计入能力目录里的实践清单。它不是线上服务。

## 跨能力链接

- [Scaling and operating in production](../../02-se-fundamentals/05-scaling-and-operating-in-production/README.md)

## 验证入口

从仓库根目录执行：

- `make test-kvcache`（1.5.1 概念练习；骨架状态下预期失败；不计入 `make test`，也不进 CI）

没有独立的线上运维验证 gate。关联课程的测试只证明其自身边界；本能力仍需补充具体任务与验收样本。

## 下一步

选择真实 AI 服务后，定义观测指标、回归样本与响应流程。
