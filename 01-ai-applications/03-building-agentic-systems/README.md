# Building agentic systems

构建 Agent 系统。把工具、记忆、任务与执行边界组合为可验证的 agent 工作流。

## 来源

[Andrew Ng 原文](https://x.com/AndrewYNg/status/2090840747738374568)

## 当前状态

**已有局部实践**。已有 12 课 Go 与 trpc 实现；编译与离线测试不证明线上质量。

## 主归属实践

- [agent-loop](agent-loop) — 12 课 Go agent 机制。
- [agent-loop-trpc](agent-loop-trpc) — 同 12 课的 trpc-agent-go 实现。

## 跨能力链接

- [Agent workflows](../../03-coding-agents/01-agent-workflows/README.md)
- [Evaluation-driven development](../04-evaluation-driven-development/README.md)

## 验证入口

从仓库根目录执行（Nanochat 命令在具体系统目录执行）：

- `make check`
- `make check-trpc`

## 下一步

为真实工具调用补任务级失败案例与评估证据。
