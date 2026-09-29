# rubickx

[中文](./README.md) | [English](./README-en.md)

Multi-language implementations of [learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) — a hands-on course for building AI agents from scratch.

The upstream project provides Python reference implementations and trilingual docs. rubickx re-implements all 12 sessions in Go to deeply understand the design of each mechanism.

## Project Structure

The top level follows the four skills in Andrew Ng's [AI Engineering Skills Map](https://www.deeplearning.ai/the-batch/the-ai-engineering-skills-map) (Aug 2026); `deps/`, `web/`, and `tests/` are supporting surfaces, not skills.

```
rubickx/
├── 01-ai-applications/
│   ├── 01-llm-foundations/
│   ├── 02-grounding-models-with-data/
│   ├── 03-building-agentic-systems/
│   │   └── agent-loop/
│   │   └── agent-loop-trpc/
│   ├── 04-evaluation-driven-development/
│   ├── 05-operating-in-production/
│   ├── 06-machine-learning-foundations/
│   │   └── nanochat/
├── 02-se-fundamentals/
│   ├── 01-building-full-stack-applications/
│   ├── 02-managing-data/
│   ├── 03-designing-system-architectures/
│   │   └── system-design/
│   │   └── patterns/
│   │   └── systems-foundations/
│   ├── 04-making-systems-secure-and-reliable/
│   │   └── network-security/
│   ├── 05-scaling-and-operating-in-production/
│   │   └── git-course/
│   ├── 06-algorithms-and-data-structures/
│   │   └── algo/
├── 03-coding-agents/
│   ├── 01-directing-the-workflow/ … 05-coding-agent-foundations/
├── 04-shaping-the-build/
│   ├── 01-driving-the-build-loop/ … 04-high-agency-ownership/
│   ├── 05-content-curation-decisions/
│   │   └── harness/
├── deps/  web/  tests/  .github/
└── Makefile  README.md  根配置
```

## Skills Map

The [capability catalog](web/capabilities.json) and pillar READMEs preserve all 20 Andrew Ng subskills from the four posts plus 2 Rubickx extensions (Algorithms and data structures, Content curation decisions). Nanochat remains a scaffold; placeholders do not imply coverage.

| Ng skill | Track directory | Content | Verification |
| --- | --- | --- | --- |
| Building and deploying AI applications | [`01-ai-applications/03-building-agentic-systems/agent-loop/`](01-ai-applications/03-building-agentic-systems/agent-loop/) | 12 Go agent sessions | `make check` |
| Building and deploying AI applications | [`01-ai-applications/03-building-agentic-systems/agent-loop-trpc/`](01-ai-applications/03-building-agentic-systems/agent-loop-trpc/) | The same 12 sessions on trpc-agent-go | `make check-trpc` |
| Building and deploying AI applications | [`01-ai-applications/06-machine-learning-foundations/nanochat/`](01-ai-applications/06-machine-learning-foundations/nanochat/) | LLM systems from scratch + micrograd | Per-system `test_impl.py` and `parity.py` (includes TODO scaffolds) |
| Software engineering fundamentals | [`02-se-fundamentals/05-scaling-and-operating-in-production/git-course/`](02-se-fundamentals/05-scaling-and-operating-in-production/git-course/) | Working tree, index, commits, and HEAD | `make verify-learning-git-gate1` |
| Software engineering fundamentals | [`02-se-fundamentals/06-algorithms-and-data-structures/algo/`](02-se-fundamentals/06-algorithms-and-data-structures/algo/) | Algorithms and data structures in five languages | `make -C 02-se-fundamentals/06-algorithms-and-data-structures/algo test` |
| Software engineering fundamentals | [`02-se-fundamentals/03-designing-system-architectures/patterns/`](02-se-fundamentals/03-designing-system-architectures/patterns/) | 42 engineering patterns with four-language contract tests | `make -C 02-se-fundamentals/03-designing-system-architectures/patterns verify` |
| Software engineering fundamentals | [`02-se-fundamentals/03-designing-system-architectures/system-design/`](02-se-fundamentals/03-designing-system-architectures/system-design/) | System-design components and HTTP / Redis lab | `make -C 02-se-fundamentals/03-designing-system-architectures/system-design test` |
| Software engineering fundamentals | [`02-se-fundamentals/04-making-systems-secure-and-reliable/network-security/`](02-se-fundamentals/04-making-systems-secure-and-reliable/network-security/) | 10 loopback network-security labs | `make -C 02-se-fundamentals/04-making-systems-secure-and-reliable/network-security verify` |
| Software engineering fundamentals | [`02-se-fundamentals/03-designing-system-architectures/systems-foundations/`](02-se-fundamentals/03-designing-system-architectures/systems-foundations/) | Go execution-model and distributed-semantics experiments | `cd 02-se-fundamentals/03-designing-system-architectures/systems-foundations && go test -race ./... && go vet ./...` |
| Using coding agents | [`03-coding-agents/01-directing-the-workflow/`](03-coding-agents/01-directing-the-workflow/) | Agent workflow, configuration, and verifier inventory | See the README verifier inventory |
| Shaping the build | [`04-shaping-the-build/05-content-curation-decisions/harness/`](04-shaping-the-build/05-content-curation-decisions/harness/) | Content-curation decisions: cases and grader | `make test-harness` |

## Getting Started

```bash
# Clone with submodule
git clone --recurse-submodules https://cnb.woa.com/kevinxjiang/rubickx.git
cd rubickx

# If already cloned without --recurse-submodules
git submodule update --init --recursive
```

## Learning Source Repos

The Rubickx homepage groups tracks and reference resources by the four skills, implemented in [web/index.html](web/index.html).

Current source-backed learning resources:

| Resource | Website | Upstream / Fork | Local Source | Status |
| --- | --- | --- | --- | --- |
| Learn Claude Code | <https://learn.shareai.run/> | <https://github.com/shareAI-lab/learn-claude-code> | `deps/learn-claude-code` | Already referenced as an upstream submodule |
| Learn Harness Engineering | <https://walkinglabs.github.io/learn-harness-engineering/zh/> | fork: <https://github.com/xjiang77/learn-harness-engineering>; upstream: <https://github.com/walkinglabs/learn-harness-engineering> | `deps/learn-harness-engineering` | Added as a fork submodule for future local changes |

## Engineering Patterns

[`02-se-fundamentals/03-designing-system-architectures/patterns/`](02-se-fundamentals/03-designing-system-architectures/patterns/) shares Designing system architectures with `system-design/` and `systems-foundations/`. Its 42-entry library contains 23 GoF, 6 reliability, 7 data and messaging, and 6 concurrency patterns. Each entry closes the loop from design judgment to four-language implementation, a shared fixture, and automated tests.

The first golden path is the [Adapter Pattern](02-se-fundamentals/03-designing-system-architectures/patterns/01-design-patterns/02-structural/01-adapter/NOTES.md); the full catalog follows the same behavior contract:

- stable target contract: `ChatClient`
- legacy adaptee with different deployment, prompt, stop-code, and error semantics
- Python `Protocol`, Go `interface`, Java `interface`, and a JavaScript structural runtime contract
- shared verification for request mapping, response/error normalization, and explicit unsupported-capability failures

```bash
make -C 02-se-fundamentals/03-designing-system-architectures/patterns setup
make -C 02-se-fundamentals/03-designing-system-architectures/patterns test-pattern PATTERN=gof.structural.adapter
make -C 02-se-fundamentals/03-designing-system-architectures/patterns verify
```

[`patterns/PROGRESS.md`](02-se-fundamentals/03-designing-system-architectures/patterns/PROGRESS.md) is the completion-status SSOT.

## Go Foundations

[`02-se-fundamentals/03-designing-system-architectures/systems-foundations/`](02-se-fundamentals/03-designing-system-architectures/systems-foundations/) turns systems mechanisms into deterministic experiments. L4 Distributed Semantics checks session histories, majority intersection, and at-least-once duplicate effects while explicitly avoiding claims that the set model proves Raft or Paxos correctness.

```bash
cd 02-se-fundamentals/03-designing-system-architectures/systems-foundations
go test -race ./...
go vet ./...
```

## Go Implementation

```bash
# Set up environment
cp .env.example .env
# Edit .env with your API key

# Run any session
make run S=01
```

See [01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/) for detailed walkthroughs of each session.

| Session | Topic | Walkthrough |
|---------|-------|-------------|
| s01 | Agent Loop | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s01-the-agent-loop.md) |
| s02 | Tool Use | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s02-tool-use.md) |
| s03 | Todo Write | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s03-todo-write.md) |
| s04 | Subagent | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s04-subagent.md) |
| s05 | Skill Loading | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s05-skill-loading.md) |
| s06 | Context Compact | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s06-context-compact.md) |
| s07 | Task System | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s07-task-system.md) |
| s08 | Background Tasks | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s08-background-tasks.md) |
| s09 | Agent Teams | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s09-agent-teams.md) |
| s10 | Team Protocols | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s10-team-protocols.md) |
| s11 | Autonomous Agents | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s11-autonomous-agents.md) |
| s12 | Worktree Task Isolation | [doc](01-ai-applications/03-building-agentic-systems/agent-loop/docs/en/s12-worktree-task-isolation.md) |

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
make harness-init RUN=04-shaping-the-build/05-content-curation-decisions/harness/runs/demo
make harness-init RUN=04-shaping-the-build/05-content-curation-decisions/harness/runs/git-only CASE=git-pro-book
make harness-grade RUN=04-shaping-the-build/05-content-curation-decisions/harness/runs/demo
```

See [harness/README.md](04-shaping-the-build/05-content-curation-decisions/harness/README.md) for the contract, scoring model, and case set.

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
