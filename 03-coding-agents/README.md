# 03 — Using coding agents

对应 Skills Map 第三项技能：有 agent 如何工作的心智模型，会管理它的 context、在计划与执行之间取舍、给它 verifier 让它自己闭环、写 spec、编排多个 agent、避开碰生产环境这类坑，并且持续试新工具。

这项技能在本仓库里的证据不是新代码，而是"这个仓库怎么被 agent 做出来"的配置、hook 与 verifier。配置文件必须留在根目录（Claude Code、Codex、pi 都按根路径读取），本目录只做清单。

## Agent 配置（根目录）

| 文件 | 用途 | 是否进 Git |
| --- | --- | --- |
| `.claude/settings.json` | Claude Code 项目设置；`PostToolUse` hook 调 `.claude/metadata-updater.sh`，把 `gh pr create` / `git checkout -b` / `gh pr merge` 的结果写进 agent-orchestrator 的 session metadata | 否（含本机绝对路径） |
| `.claude/settings.local.json` | 个人 permissions allowlist | 否 |
| `.codex/config.toml`、`.codex/hooks.json` | Codex 的 MCP server 与同一个 PostToolUse hook | 否（含 credential） |
| `.mcp.json` / `.mcp.json.example` | MCP server 配置；example 的 Authorization 留空 | 只进 example |
| `.pi/extensions/obs-jsonl.ts` | pi 的本地 observability 扩展，把 agent 事件写成 JSONL；日志在 `.pi/obs/`（gitignored） | 是 |
| `agent-orchestrator.yaml` | agent-orchestrator 的项目定义：worktree 隔离、tmux runtime、commit 约定 | 否 |
| `.vscode/launch.json` | Go 测试的 Delve 调试入口 | 是 |

## Verifier 清单

agent 改完代码后能自己跑、自己判断对错的命令。给 agent 的每个任务都应指向其中至少一条。

| 范围 | 命令 | 判定 |
| --- | --- | --- |
| Go agent 课程 | `make check`、`make check-trpc` | 全部 session 可编译且 `go vet` 通过 |
| 根级 Python | `make test-unit`、`make test-harness` | 退出码 0 |
| Git 课程（`02-se-fundamentals/git-course/`） | `make test-learning-git`、`make verify-learning-git-gate1` | vitest / Playwright 全绿 |
| SE fundamentals | `make -C 02-se-fundamentals/<track> test` 或 `verify` | 见 `02-se-fundamentals/README.md` |
| nanochat | 各系统目录 `pytest test_impl.py` 与 `python parity.py` | 测试通过，parity 在容差内 |
| 发布面 | `bash .github/scripts/check-pages.sh` | Pages basic gate passed |
| CI | `.github/workflows/test.yml` | 全部 job 绿 |

## Spec 约定

- `01-ai-applications/nanochat/systems/*/spec.md`：每个系统的 IO 契约与验收标准，实现前先写。
- `02-se-fundamentals/patterns/*/NOTES.md` 与 `catalog.json`：每个 pattern 的判断与 contract，四语言实现必须满足同一 contract。
- `02-se-fundamentals/system-design/systems/*/DESIGN.md`：系统设计的取舍记录。
- `04-shaping-the-build/harness/cases/*.json`：给 agent 的 deterministic task spec。

## 工作约定（来自 agent-orchestrator.yaml）

- push 前先跑测试。
- Conventional commits：`feat:` `fix:` `chore:` `docs:` `refactor:` `test:`。
- commit message 说明 WHY，不只是 WHAT。
