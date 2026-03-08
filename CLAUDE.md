# CLAUDE.md — rubickx

## Project Overview

rubickx is a progressive AI agent tutorial that re-implements the upstream `learn-claude-code` course (Python) in **Go**. It contains 12 sessions that incrementally build agent complexity — from a basic agent loop to multi-agent teams with git worktree isolation.

## Repository Structure

```
rubickx/
├── go/                        # Primary Go implementations (12 sessions)
│   ├── s01-the-agent-loop/    # Basic agent loop
│   ├── s02-tool-use/          # Tool dispatch (bash, read, write, edit)
│   ├── s03-todo-write/        # TodoManager state tracking
│   ├── s04-subagent/          # Subagent context isolation
│   ├── s05-skill-loading/     # Dynamic skill injection
│   ├── s06-context-compact/   # Token compression
│   ├── s07-task-system/       # Persistent task management
│   ├── s08-background-tasks/  # Async execution & notifications
│   ├── s09-agent-teams/       # Multi-agent collaboration
│   ├── s10-team-protocols/    # Shutdown/plan approval protocols
│   ├── s11-autonomous-agents/ # Self-discovery & execution loops
│   ├── s12-worktree-task-isolation/ # Git worktree parallel isolation
│   └── docs/{en,zh}/         # Per-session walkthroughs (line-by-line)
├── trpc-agent-go/             # Alternative tRPC-based Go implementation
├── deps/learn-claude-code/    # Upstream Python course (git submodule)
├── harness/                   # Content curation/evaluation harness
│   ├── cases/                 # Deterministic test cases
│   └── runs/                  # Harness run outputs
├── tests/                     # Python test suite
├── web/                       # Static GitHub Pages site
├── skills -> deps/.../skills  # Symlink to skill definitions
├── Makefile                   # Build/test orchestration
└── .env.example               # API key and model configuration
```

## Quick Start

```bash
cp .env.example .env           # Set ANTHROPIC_API_KEY and MODEL_ID
make setup                     # Init submodules + download Go/npm deps
make run S=01                  # Run session 01 REPL
```

## Key Commands

| Command | Description |
|---|---|
| `make run S=01` | Run a Go session REPL (01–12) |
| `make run-trpc S=01` | Run the tRPC variant |
| `make check` | Compile-check and `go vet` all Go sessions |
| `make check-trpc` | Compile-check and `go vet` all tRPC sessions |
| `make test` | Run full test suite (check → unit → harness → api) |
| `make test-unit` | Run offline Python unit tests only |
| `make test-harness` | Run harness tests only |
| `make test-api` | Run API connectivity test (needs API key) |
| `make harness-list` | List deterministic harness cases |
| `make harness-init RUN=... CASE=...` | Initialize a harness run |
| `make harness-grade RUN=...` | Grade a harness run |
| `make setup` | Full project setup |

## Tech Stack

- **Language**: Go 1.24+ (primary), Python 3.11 (tests/harness)
- **SDK**: `github.com/anthropics/anthropic-sdk-go` v1.26.0
- **Alternative**: `trpc.group/trpc-go/trpc-agent-go` v1.6.0 (in trpc-agent-go/)
- **Testing**: Python `unittest` (tests/), no Go test files
- **CI**: GitHub Actions (`.github/workflows/test.yml`, `pages.yml`)
- **Web**: Static HTML/CSS (GitHub Pages), Next.js upstream site

## Code Conventions

### Go Sessions

- Each session lives in `go/s{NN}-{name}/main.go`
- Architecture diagrams appear as block comments at the top of each file
- Code grows incrementally: s01 (~230 lines) → s12 (~1218 lines)
- Pattern: core loop → tool handlers → helpers → `init()` + `main()`
- Minimal dependencies — no web frameworks, just the Anthropic SDK + stdlib
- Environment loaded via `.env` file (ANTHROPIC_API_KEY, MODEL_ID, optional ANTHROPIC_BASE_URL)

### Formatting & Linting

- Go: standard `gofmt` formatting; `go vet ./...` in CI
- No explicit lint config files — rely on Go defaults
- Python tests: standard `unittest` conventions

### Commit Messages

- Follow conventional commits: `feat:`, `fix:`, `docs:`, `chore:`
- Feature branches prefixed with intent (e.g., `claude/`, `codex/`)
- PRs used for feature merging into `main`

## Environment Configuration

Required in `.env`:
- `ANTHROPIC_API_KEY` — API key from console.anthropic.com
- `MODEL_ID` — e.g. `claude-sonnet-4-6`

Optional:
- `ANTHROPIC_BASE_URL` — for compatible providers (MiniMax, GLM, Kimi, DeepSeek)

See `.env.example` for the full provider reference matrix with regional endpoints.

## CI/CD

- **test.yml**: Runs unit tests + harness tests (offline), session matrix tests (with API key), and web build verification
- **pages.yml**: Deploys `web/` to GitHub Pages on push to `main`
- Pages validation: `.github/scripts/check-pages.sh` ensures required files exist and assets use relative URLs

## Testing

- **Offline tests** (`make test-unit`): Test TodoManager, SkillLoader, token estimation, EventBus, TaskManager, WorktreeManager — no API key needed
- **Harness tests** (`make test-harness`): Verify deterministic case initialization and artifact contracts
- **API tests** (`make test-api`): Verify connectivity with real API key

Always run `make check` before committing Go changes to catch compilation errors and vet warnings.

## Harness System

The content harness (`harness/`) is a curation/evaluation pipeline:
1. Intake source material
2. Judge worthiness
3. Generate Chinese summary
4. Transform to course/blog format
5. Grade with partial credit (artifact contract, grounding, pedagogy, traceability)

Artifact contract per case: `result.json`, `summary_zh.md`, `trace.jsonl`

## Important Notes

- The `deps/learn-claude-code` directory is a **git submodule** — use `git submodule update --init --recursive` after cloning
- The `skills` symlink points into the submodule — do not break this link
- Go session binaries are gitignored (pattern: `go/s*/s*` excluding `*.go`)
- Documentation exists in both Chinese (`docs/zh/`) and English (`docs/en/`)
