# nanochat — LLM from scratch (学习 track)

> rubickx 的第二条学习 track,与 `go/`(learn-claude-code 重写)平行。
> **方法论**:把 nanochat 全栈的每个系统**自己从零重写一遍**,再与参考实现(`scasella/nanochat-mlx`)对拍(parity),不照抄。

## 在 rubickx 里的位置(对齐既有约定)

| rubickx 约定 | go/ track | nanochat/ track |
|---|---|---|
| 顶层 track 目录 | `go/` | `nanochat/`(本目录) |
| 参考上游 submodule | `deps/learn-claude-code` | `deps/nanochat-mlx` |
| 编号课程/系统 | `go/s01…s12` | `nanochat/systems/01…10` |
| walkthrough 文档 | `go/docs/zh` | `nanochat/docs/zh` |

## 权威文档(canonical 在 vault)

规范 / 计划 / 决策日志维持在 Obsidian vault(项目不变量 C2:知识·计划·规范进 vault,源码进 rubickx)。本 track 的代码与代码邻接产物在这里;走读结论的 durable insight 晋升 vault `03_Slipbox/01_AI/`。

- SPEC(权威契约):vault `01_Projects/Nanochat/SPEC - nanochat Foundations Learning Plan.md`
- 实现决策日志:vault `01_Projects/Nanochat/implementation-notes.md`
- 实验记录(每个实验必落):vault `01_Projects/Nanochat/nanochat - Experiment Log.md`
- 母计划 04 / 地基课 05:同目录

## 目录结构

```
nanochat/
├── README.md            # 本文件
├── RUNBOOK.md           # 跑参考实现(nanochat-mlx)的命令
├── .gitignore
├── systems/             # ★ 每个系统自己重写一遍(01→10,按依赖序)
│   └── NN-slug/ = README · spec · impl.py · test_impl.py · parity.py · notes.md
├── from-scratch/        # 地基热身(纯打基础,非 nanochat 系统)
│   ├── micrograd/             # D2 / R3
│   └── addition-transformer/  # D3 / R6
├── docs/zh/             # walkthrough 文档(对齐 go/docs/zh)
├── experiments/         # 临时脚本/结果(结论回写 vault Log)
└── shared/              # 跨系统复用(保持薄)

../deps/nanochat-mlx/    # 参考实现(submodule,只读对照)
```

## 重写顺序(按依赖)→ SPEC 需求

`01-tokenizer → 02-data → 03-model → 04-optim → 05-train`(到此端到端训小模型)`→ 06-engine`(推理对话)`→ 07-sft → 09-eval`(微调+评测)`→ 08-rl → 10-tooluse`(进阶)。

| 系统 | 需求 | 阶段 |
|---|---|---|
| 01-tokenizer | R4 | Phase 1 |
| 03-model | R5 | Phase 1 |
| 04-optim / 05-train | R7 | Phase 1 |
| 02-data | R7(支撑) | Phase 1 |
| 06-engine | R9 | Phase 2 |
| 07-sft / 08-rl / 10-tooluse | R10 | Phase 2(+) |
| 09-eval | R11 | Phase 2 |

## 每个系统的工作流

`学/复习 → 走读(deps/nanochat-mlx) → 实践(impl.py 本人手写) → 验证(test_impl + parity 对拍) → 落 vault Log`。

> C5 学习诚实:`systems/*/impl.py` 与 `from-scratch/*` 必须本人手写,agent 不代写;agent 可讲概念、review、帮搭 parity 对拍框架。
