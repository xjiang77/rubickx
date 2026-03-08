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
├── deps/autoresearch-macos/   # autoresearch 的 macOS fork (git submodule)
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
git clone --recurse-submodules https://cnb.woa.com/kevinxjiang/rubickx.git
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


| 课程  | 主题                      | 格言                                | 文档                                                       |
| --- | ----------------------- | --------------------------------- | -------------------------------------------------------- |
| s01 | Agent Loop              | "One loop & Bash is all you need" | [walkthrough](go/docs/zh/s01-the-agent-loop.md)          |
| s02 | Tool Use                | "加一个工具，只加一个 handler"              | [walkthrough](go/docs/zh/s02-tool-use.md)                |
| s03 | Todo Write              | "结构化状态，模型自己管理"                    | [walkthrough](go/docs/zh/s03-todo-write.md)              |
| s04 | Subagent                | "fork 一个子循环，隔离上下文"                | [walkthrough](go/docs/zh/s04-subagent.md)                |
| s05 | Skill Loading           | "动态注入 system prompt"              | [walkthrough](go/docs/zh/s05-skill-loading.md)           |
| s06 | Context Compact         | "上下文满了就压缩，循环不断"                   | [walkthrough](go/docs/zh/s06-context-compact.md)         |
| s07 | Task System             | "任务是持久化的 todo"                    | [walkthrough](go/docs/zh/s07-task-system.md)             |
| s08 | Background Tasks        | "后台执行，异步通知"                       | [walkthrough](go/docs/zh/s08-background-tasks.md)        |
| s09 | Agent Teams             | "多 agent 协作，共享 task list"         | [walkthrough](go/docs/zh/s09-agent-teams.md)             |
| s10 | Team Protocols          | "shutdown / plan approval 协议"     | [walkthrough](go/docs/zh/s10-team-protocols.md)          |
| s11 | Autonomous Agents       | "自治循环，自动发现并执行任务"                  | [walkthrough](go/docs/zh/s11-autonomous-agents.md)       |
| s12 | Worktree Task Isolation | "git worktree 隔离并行任务"             | [walkthrough](go/docs/zh/s12-worktree-task-isolation.md) |


## Web 学习平台

上游提供的交互式学习平台，包含可视化、模拟器和代码标注。

```bash
cd deps/learn-claude-code/web
npm install
npm run dev
```

## autoresearch-macos

已接入 [miolini/autoresearch-macos](https://github.com/miolini/autoresearch-macos) 作为 `deps/autoresearch-macos` submodule。

```bash
# 安装 Python 依赖
make autoresearch-sync

# 首次准备数据与 tokenizer
make autoresearch-prepare

# 启动一次 5 分钟训练实验
make autoresearch-run

# 或者一条命令完成上述步骤
make autoresearch-start
```

## Harness Engineering

项目现在包含一套面向内容策展与课程化的 harness，目标不是测 toy prompt，而是评估这条真实工作流：

1. intake 经典学习材料
2. 判断是否值得沉淀
3. 生成中文摘要
4. 必要时转成 interactive course 或 blog post
5. 保留 trace 做 failure analysis

快速命令：

```bash
# 查看 case
make harness-list

# 初始化一个 run 目录
make harness-init RUN=harness/runs/demo

# 只跑一个 case
make harness-init RUN=harness/runs/git-only CASE=git-pro-book

# 填完输出后评分
make harness-grade RUN=harness/runs/demo
```

当前 harness 使用 deterministic fixtures 和 heuristic grader，细节见 [harness/README.md](harness/README.md)。

## 项目静态网页

仓库内的 [web/index.html](web/index.html) 是一个纯静态介绍页，适合直接部署到 GitHub Pages。

```bash
# 本地预览
python3 -m http.server 8000 -d web
```

线上地址：

- `https://xjiang77.github.io/rubickx/`

自动发布约定：

- 发布入口固定为 `web/`
- 只有 `main` 会触发正式发布
- `.github/workflows/pages.yml` 是唯一正式 Pages deploy 流程
- 发布前会先跑 `bash .github/scripts/check-pages.sh`

首次启用时需要在 GitHub 仓库设置里完成这一项：

- `Settings -> Pages -> Build and deployment -> Source` 设为 `GitHub Actions`

本地检查：

```bash
bash .github/scripts/check-pages.sh
```

故障排查：

- 如果 workflow 能跑但没有生成站点，先检查 Pages 的 `Source` 是否还是 `Deploy from a branch`，它必须改成 `GitHub Actions`
- 如果 `upload-pages-artifact` 或 deploy 失败，先检查 workflow 里的 artifact path 是否仍然是 `web`
- 如果 `Basic gate` 失败，通常是 `index.html` 引用了不存在的本地资源，或 `web/index.html` / `web/styles.css` / `web/favicon.svg` 缺失
- 站点运行在 project-site 路径 `/rubickx/` 下，新增资源时继续使用相对路径，不要写成依赖站点根路径 `/` 的绝对引用
