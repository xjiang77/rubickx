# rubickx

Multi-language implementations of [learn-claude-code](https://github.com/shareAI-lab/learn-claude-code) — a hands-on course for building AI agents from scratch.

## Project Structure

```
rubickx/
├── deps/learn-claude-code/    # Upstream course (git submodule)
│   ├── agents/                # Python reference implementations
│   ├── docs/                  # Trilingual documentation (en/ja/zh)
│   ├── web/                   # Next.js learning platform
│   └── skills/                # Skill files for s05
├── go/                        # Go implementation
│   ├── s01/ ... s12/          # Sessions 1-12
│   └── docs/zh/               # Go walkthrough docs (Chinese)
├── tests/                     # Test suite
├── skills -> deps/.../skills  # Symlink for runtime compatibility
└── .github/workflows/         # CI
```

## Getting Started

```bash
# Clone with submodule
git clone --recurse-submodules https://github.com/xjiang77/rubickx.git
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
cd go && go run ./s01
```

See [go/docs/zh/](go/docs/zh/) for detailed walkthroughs of each session.

## Web Learning Platform

```bash
cd deps/learn-claude-code/web
npm install
npm run dev
```

## Tests

```bash
python tests/test_unit.py
```
