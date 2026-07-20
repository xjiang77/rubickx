# rubickx

[中文](./README.md) | [English](./README-en.md)

Multi-language implementations of [learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) — a hands-on course for building AI agents from scratch.

The upstream project provides Python reference implementations and trilingual docs. rubickx re-implements all 12 sessions in Go to deeply understand the design of each mechanism.

## Project Structure

```
rubickx/
├── deps/learn-claude-code/    # Learn Claude Code upstream course (git submodule)
│   ├── agents/                # Python reference implementations
│   ├── docs/                  # Trilingual documentation (en/ja/zh)
│   ├── web/                   # Next.js learning platform
│   └── skills/                # Skill files for s05
├── deps/learn-harness-engineering/  # Learn Harness Engineering fork (git submodule)
├── algo/                      # Algorithms and data structures
├── system-design/             # Runnable system-design implementations and labs
├── patterns/                  # Four-language engineering patterns with shared contract tests
├── go/                        # Go implementation
│   ├── s01-the-agent-loop/ ... s12-worktree-task-isolation/  # 12 progressive sessions
│   ├── foundations/           # Runnable systems-mechanism experiments
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

## Learning Source Repos

The Rubickx homepage is now a learning repo / website index, implemented in [web/index.html](web/index.html).

Current source-backed learning resources:

| Resource | Website | Upstream / Fork | Local Source | Status |
| --- | --- | --- | --- | --- |
| Learn Claude Code | <https://learn.shareai.run/> | <https://github.com/shareAI-lab/learn-claude-code> | `deps/learn-claude-code` | Already referenced as an upstream submodule |
| Learn Harness Engineering | <https://walkinglabs.github.io/learn-harness-engineering/zh/> | fork: <https://github.com/xjiang77/learn-harness-engineering>; upstream: <https://github.com/walkinglabs/learn-harness-engineering> | `deps/learn-harness-engineering` | Added as a fork submodule for future local changes |

## Engineering Patterns

[`patterns/`](patterns/) sits beside `algo/` and `system-design/`. Its 42-entry library contains 23 GoF, 6 reliability, 7 data and messaging, and 6 concurrency patterns. Each entry closes the loop from design judgment to four-language implementation, a shared fixture, and automated tests.

The first golden path is the [Adapter Pattern](patterns/01-design-patterns/02-structural/01-adapter/NOTES.md); the full catalog follows the same behavior contract:

- stable target contract: `ChatClient`
- legacy adaptee with different deployment, prompt, stop-code, and error semantics
- Python `Protocol`, Go `interface`, Java `interface`, and a JavaScript structural runtime contract
- shared verification for request mapping, response/error normalization, and explicit unsupported-capability failures

```bash
make -C patterns setup
make -C patterns test-pattern PATTERN=gof.structural.adapter
make -C patterns verify
```

[`patterns/PROGRESS.md`](patterns/PROGRESS.md) is the completion-status SSOT.

## Go Foundations

[`go/foundations/`](go/foundations/) turns systems mechanisms into deterministic experiments. L4 Distributed Semantics checks session histories, majority intersection, and at-least-once duplicate effects while explicitly avoiding claims that the set model proves Raft or Paxos correctness.

```bash
cd go
go test -race ./foundations/...
go vet ./foundations/...
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

## Harness Engineering

The repo now includes a harness for the actual rubickx content loop:

1. intake a classic learning resource
2. decide whether it should be curated
3. write a Chinese summary
4. transform it into an interactive course or blog post when appropriate
5. keep a trace for failure analysis

Quick commands:

```bash
make harness-list
make harness-init RUN=harness/runs/demo
make harness-init RUN=harness/runs/git-only CASE=git-pro-book
make harness-grade RUN=harness/runs/demo
```

See [harness/README.md](harness/README.md) for the contract, scoring model, and case set.

## Project Landing Page

[web/index.html](web/index.html) is now a plain static landing page for the project, intended for GitHub Pages deployment.

```bash
python3 -m http.server 8000 -d web
```

Live URL:

- `https://xjiang77.github.io/rubickx/`

Deployment contract:

- `web/` is the only Pages artifact root
- `main` is the only branch that triggers the production site deploy
- `.github/workflows/pages.yml` is the only production Pages workflow
- deployment is gated by `bash .github/scripts/check-pages.sh`

One-time repository setting:

- `Settings -> Pages -> Build and deployment -> Source` must be set to `GitHub Actions`

Local check:

```bash
bash .github/scripts/check-pages.sh
```

Troubleshooting:

- If the workflow succeeds but no site is published, verify that Pages `Source` is set to `GitHub Actions` instead of `Deploy from a branch`
- If artifact upload or deploy fails, verify that the workflow still uploads the `web` directory
- If `Basic gate` fails, `index.html` is usually referencing a missing local asset, or one of `web/index.html`, `web/styles.css`, `web/favicon.svg` is missing
- The site is served under the project-site path `/rubickx/`, so future assets must continue to use relative URLs instead of root-based `/...` references
