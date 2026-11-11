# KV cache 概念练习（topic 1.5.1）

CPU 上的形状、次数和规则练习。对应知识库 `02 - AI - KV-Cache Deep Dive.md` 的第 1、3、7 节，以及笔记里的跨请求复用三前提。正文仍是待核验，这里不把它标成已核验。

## 证明什么

手写实现通过 `make test-kvcache` 之后，只证明这些规则在固定数值上算对了：

- 第 3 节玩具（2 层、2 头、head_dim=4）在 5-token prefill 之后，以及 decode 到第 6 个 token 时，Q/K/V 的形状。
- 第 1 节的 token-forward 次数：无 cache 与有 cache。
- 第 7 节的字节公式，包括 Llama 3 70B FP16 单 token 327680 字节（约 320 KB），以及 1K/8K/32K/128K。
- 一个很小的 `KVCache.append` / `get`：沿 position 追加，形状不对就拒绝，并且不把调用方的列表接进 cache。
- 前缀复用：token 逐个相同、model 相同、adapter 相同才命中。前缀里换一个时间戳，或 model / adapter 不同，都是 miss。

## 不证明什么

- 不是 vLLM，不是 PagedAttention，也没有 GPU 页表。
- 不是 nanochat `systems/06-engine` 的 Inference Engine。那一组要等 01–05 重写之后由本人手写，本目录不动它。
- 不算注意力，不加载模型权重，不测延迟、量化或投机解码。
- 验证器在骨架状态下失败是预期结果。不得用 `pytest.skip` 或 `xfail` 把它变成退出码 0。

## 怎么跑

从仓库根目录：

```bash
make test-kvcache
```

这个目标不在 `make test` 里，也不进 CI。骨架状态下退出码非 0。

先看对照表（不依赖尚未写的函数）：

```bash
python3 01-ai-applications/05-operating-in-production/kv-cache-lab/playground/prefill_decode.py
```

要写的函数和可核对数字在 [EXERCISE.md](EXERCISE.md)。
