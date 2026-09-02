# 01 — Building and deploying AI applications

对应 Andrew Ng [AI Engineering Skills Map](https://www.deeplearning.ai/the-batch/the-ai-engineering-skills-map) 的第一项技能：理解 LLM、context engineering、agentic workflows、ML/DL 这些构件，并用 evals 与 error analysis 让不可预测的系统可控。

本目录放的是"自己从零把构件写一遍"的代码。

| Track | 内容 | 验证 |
| --- | --- | --- |
| `nanochat/` | LLM 全栈十个系统（tokenizer → data → model → optim → train → engine → SFT → RL → eval → tooluse）逐个重写，与参考实现 `nanochat-mlx` 对拍；`from-scratch/` 是 micrograd 与 addition-transformer 热身 | 每个系统目录内 `test_impl.py`（pytest）与 `parity.py` |
| `agent-loop/` | learn-claude-code 12 课的 Go 重写：agent loop、tool use、subagent、skill loading、context compact、task system、agent teams、worktree isolation | 根目录 `make check`、`make run S=01` |
| `agent-loop-trpc/` | 同 12 课用 trpc-agent-go 框架实现 | 根目录 `make check-trpc`、`make run-trpc S=01` |

每个 track 的规范、计划与实验记录在 vault（`04_Projects/Rubickx/`），代码与代码邻接的 spec / notes 在这里。
