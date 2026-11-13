# Serving 调度概念练习（topic 1.5.2）

CPU 上的步数练习。对应知识库 `03 - AI - LLM Serving Runtime：Scheduler、KV Cache 与 Backpressure.md` 里能手算的调度规则。正文仍是待核验，这里不把它标成已核验。

## 证明什么

手写实现通过 `make test-serving` 之后，只证明这些规则在固定整数上算对了：

- 同一组混合负载下，静态批、整段 prefill、chunked prefill 的完成步和 TTFT 步。
- 一步里的 prefill token 与 decode token 之和不超过 `TOKEN_BUDGET_PER_STEP`。
- 新请求会让活 KV 超过 `KV_CAPACITY`，或等待队列已满 `MAX_QUEUE` 时，返回 `overload`；无限队列会把第三次请求也收进来。
- 长 prefill 一次占满 budget 时，短请求更晚完成；chunked prefill 让短请求更早完成。
- 共享长度为 4 的前缀时，第二个请求的 prefill 是 `prompt_len - 4`，decode 不变。

## 不证明什么

- 不是 vLLM，不是 PagedAttention，也没有 GPU。
- 不是 nanochat `systems/06-engine` 的 Inference Engine。本目录不动它。
- 不算注意力，不加载模型，不测墙钟延迟，也不做 Gateway 路由或重试。
- 验证器在骨架状态下失败是预期结果。不得用 `pytest.skip` 或 `xfail` 把它变成退出码 0。

## 怎么跑

从仓库根目录：

```bash
make test-serving
```

这个目标不在 `make test` 里，也不进 CI。骨架状态下退出码非 0。

先看对照表（不依赖尚未写的函数）：

```bash
python3 01-ai-applications/05-operating-in-production/serving-scheduler-lab/playground/comparison.py
```

要写的函数和可核对数字在 [EXERCISE.md](EXERCISE.md)。
