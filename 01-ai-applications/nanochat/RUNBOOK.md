# RUNBOOK — 跑参考实现 nanochat-mlx

> 参考实现(只读对照):`../deps/nanochat-mlx/`(submodule)。
> 命令名取自上游 README 的 2026-06 快照(implementation-notes #8)。**以仓库内实际 README 为准**;若脚本名变了,在 vault implementation-notes 记一笔。

## 0. 环境(M2 Ultra · MLX)

```bash
cd deps/nanochat-mlx
uv sync                 # 或按其 README 装依赖
```

## 1. S1 · 跑通 d4 全栈(R1)

```bash
# 一键 quickstart(若上游提供),d4 尺寸:
uv run python -m scripts.quickstart --depth 4
# 或分步:train -> sft -> chat
uv run python -m scripts.train  --depth 4
uv run python -m scripts.sft    --depth 4
uv run python -m scripts.chat   --depth 4     # 进对话,验证 d4 能聊
```

验收 R1:d4 能对话 + vault Log 有首条执行记录(command / observation / friction / next)。

## 2. 模型转换(如需从 HF 权重)

```bash
uv run python -m scripts.convert_from_hf ...
```

## 3. depth sweep(R8,Week 3)

```bash
for d in 4 6 8; do uv run python -m scripts.train --depth $d; done
# 收 d4/d6/d8 曲线 -> vault scaling writeup(D5)
```

## 注意

- 本机只跑 d4–d20 尺寸用于**学习**;真·GPT-2 CORE 基准需云端(Phase 5,不在本 SPEC)。
- 每次运行的 command + 观察 + 失败,**必落 vault Experiment Log**(C6),否则视为未完成。
