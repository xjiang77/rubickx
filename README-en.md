# rubickx

[中文](./README.md) | [English](./README-en.md)

Multi-language implementations of [learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) — a hands-on course for building AI agents from scratch.

The upstream project provides Python reference implementations and trilingual docs. rubickx re-implements all 12 sessions in Go to deeply understand the design of each mechanism.

## Project Structure

```
rubickx/
├── deps/learn-claude-code/    # Upstream course (git submodule)
│   ├── agents/                # Python reference implementations
│   ├── docs/                  # Trilingual documentation (en/ja/zh)
│   ├── web/                   # Next.js learning platform
│   └── skills/                # Skill files for s05
├── go/                        # Go implementation
│   ├── s01-the-agent-loop/ ... s12-worktree-task-isolation/  # 12 progressive sessions
│   └── docs/                  # Go walkthrough docs
│       ├── zh/                # 中文
│       └── en/                # English
├── tests/                     # Test suite
├── skills -> deps/.../skills  # Symlink for runtime compatibility
└── .github/workflows/         # CI
```

## Getting Started

```bash
# Clone with submodule
git clone --recurse-submodules https://cnb.woa.com/kevinxjiang/rubickx.git
cd rubickx

# If already cloned without --recurse-submodules
git submodule update --init --recursive
```

## Go Implementation

```bash
# Set up environment
cp .env.example .env
# Edit .env with your API key

# Run any session
make run S=01
```

See [go/docs/en/](go/docs/en/) for detailed walkthroughs of each session.

| Session | Topic | Walkthrough |
|---------|-------|-------------|
| s01 | Agent Loop | [doc](go/docs/en/s01-the-agent-loop.md) |
| s02 | Tool Use | [doc](go/docs/en/s02-tool-use.md) |
| s03 | Todo Write | [doc](go/docs/en/s03-todo-write.md) |
| s04 | Subagent | [doc](go/docs/en/s04-subagent.md) |
| s05 | Skill Loading | [doc](go/docs/en/s05-skill-loading.md) |
| s06 | Context Compact | [doc](go/docs/en/s06-context-compact.md) |
| s07 | Task System | [doc](go/docs/en/s07-task-system.md) |
| s08 | Background Tasks | [doc](go/docs/en/s08-background-tasks.md) |
| s09 | Agent Teams | [doc](go/docs/en/s09-agent-teams.md) |
| s10 | Team Protocols | [doc](go/docs/en/s10-team-protocols.md) |
| s11 | Autonomous Agents | [doc](go/docs/en/s11-autonomous-agents.md) |
| s12 | Worktree Task Isolation | [doc](go/docs/en/s12-worktree-task-isolation.md) |

## Web Learning Platform

Interactive learning platform from upstream with visualizations, simulators, and code annotations.

```bash
cd deps/learn-claude-code/web
npm install
npm run dev
```

## Tests

```bash
python tests/test_unit.py
```

## Future Plans

- Add new language implementations: create `rust/`, `ts/`, etc. at the top level
