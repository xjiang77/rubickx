# rubickx

[中文](./README.md) | [English](./README-en.md)

基于 [learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) 的多语言实现 — 从零构建 AI Agent 的实战课程。

上游项目提供了 Python 参考实现和三语文档，rubickx 在此基础上用 Go 重新实现全部 12 个课程，深入理解每个机制的设计细节。

## 项目结构

```
rubickx/
├── deps/learn-claude-code/    # 上游课程 (git submodule)
│   ├── agents/                # Python 参考实现
│   ├── docs/                  # 三语文档 (en/ja/zh)
│   ├── web/                   # Next.js 学习平台
│   └── skills/                # Skill 文件 (供 s05 使用)
├── go/                        # Go 实现
│   ├── s01-the-agent-loop/ ... s12-worktree-task-isolation/  # 12 个递进式课程
│   └── docs/                  # Go walkthrough 文档
│       ├── zh/                # 中文
│       └── en/                # English
├── tests/                     # 测试
├── skills -> deps/.../skills  # symlink (运行时兼容)
└── .github/workflows/         # CI
```

## 快速开始

```bash
# clone（含 submodule）
git clone --recurse-submodules https://github.com/xjiang77/rubickx.git
cd rubickx

# 如果已经 clone 但没带 --recurse-submodules
git submodule update --init --recursive
```

## Go 实现

```bash
# 配置环境
cp .env.example .env
# 编辑 .env，填入你的 API key

# 运行任意课程
make run S=01
```

每个课程都有对应的 walkthrough 文档，详见 [go/docs/zh/](go/docs/zh/)：

| 课程 | 主题 | 格言 | 文档 |
|------|------|------|------|
| s01 | Agent Loop | "One loop & Bash is all you need" | [walkthrough](go/docs/zh/s01-the-agent-loop.md) |
| s02 | Tool Use | "加一个工具，只加一个 handler" | [walkthrough](go/docs/zh/s02-tool-use.md) |
| s03 | Todo Write | "结构化状态，模型自己管理" | [walkthrough](go/docs/zh/s03-todo-write.md) |
| s04 | Subagent | "fork 一个子循环，隔离上下文" | [walkthrough](go/docs/zh/s04-subagent.md) |
| s05 | Skill Loading | "动态注入 system prompt" | [walkthrough](go/docs/zh/s05-skill-loading.md) |
| s06 | Context Compact | "上下文满了就压缩，循环不断" | [walkthrough](go/docs/zh/s06-context-compact.md) |
| s07 | Task System | "任务是持久化的 todo" | [walkthrough](go/docs/zh/s07-task-system.md) |
| s08 | Background Tasks | "后台执行，异步通知" | [walkthrough](go/docs/zh/s08-background-tasks.md) |
| s09 | Agent Teams | "多 agent 协作，共享 task list" | [walkthrough](go/docs/zh/s09-agent-teams.md) |
| s10 | Team Protocols | "shutdown / plan approval 协议" | [walkthrough](go/docs/zh/s10-team-protocols.md) |
| s11 | Autonomous Agents | "自治循环，自动发现并执行任务" | [walkthrough](go/docs/zh/s11-autonomous-agents.md) |
| s12 | Worktree Task Isolation | "git worktree 隔离并行任务" | [walkthrough](go/docs/zh/s12-worktree-task-isolation.md) |

## Web 学习平台

上游提供的交互式学习平台，包含可视化、模拟器和代码标注。

```bash
cd deps/learn-claude-code/web
npm install
npm run dev
```

## 测试

```bash
python tests/test_unit.py
```

## 未来计划

- 新增语言实现：直接在顶层创建 `rust/`, `ts/` 等目录
- 新增上游依赖：`git submodule add <url> deps/<name>`
