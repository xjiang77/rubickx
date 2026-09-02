# Managing data

管理数据。围绕访问模式设计数据存储、事务、并发与消息语义。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2093388974194872781)

## 当前状态

**已有局部实践**。已有 Data & Messaging patterns 和分布式语义实验，未覆盖完整数据治理。

## 主归属实践

本能力没有独立主归属实现。关联实验见下；待建设时仅保留说明，不生成空代码。

## 跨能力链接

- [Designing system architectures](../03-designing-system-architectures/README.md)
- [Grounding models with data](../../01-ai-applications/02-grounding-models-with-data/README.md)
- [Data & Messaging patterns](../03-designing-system-architectures/patterns/03-data-messaging-patterns)
- [分布式语义实验](../03-designing-system-architectures/systems-foundations/l4-distributed-semantics)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

没有独立验证 gate。关联课程的测试只证明其自身边界；本能力仍需补充具体任务与验收样本。

## 下一步

将事务、重复效果与数据质量问题放入具体业务负载验证。
