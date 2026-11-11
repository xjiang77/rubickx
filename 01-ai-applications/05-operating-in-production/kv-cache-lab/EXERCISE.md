# Exercise — KV cache 概念练习（1.5.1）

> C5 约束：实现由本人手写。agent 可以讲解概念、评审代码、维护验证脚本，不得把完成的实现写进这个目录。

## 顺序

1. 读 `playground/prefill_decode.py` 的对照表，先把 prefill 和 decode 的 cache 长度分开。
2. 在 `kv_cache.py` 里手写下面的函数。类型标注和 Google docstring 已经在骨架里，`Examples` 是 doctest。
3. `python3 -m doctest kv_cache.py -v`
4. 从仓库根目录运行 `make test-kvcache`。

骨架状态下第 3、4 步失败是预期结果。不要加 `pytest.skip` 或 `xfail`。

## 目标

只用列表和整数，把笔记第 1、3、7 节里能手算的规则写成函数，并让一个小 cache 按同样的形状追加。

笔记路径：`02_Knowledge/01-ai-applications/05-operating-in-production/01-inference-and-serving/02 - AI - KV-Cache Deep Dive.md`。

本练习把这些闭式写成合同。私有 vault 在准备骨架时没有打开，若笔记里的某一组输入和下面不同，改 `fixtures.py` 里对应的字面量，不要改比较方式。

## 合同

`kv_cache.py` 不得 import numpy、torch。允许 `copy` 或自己写的副本。

### 形状（第 3 节）

轴序 `(layers, heads, seq, head_dim)`，没有 batch 维。玩具是 2 层、2 头、`head_dim=4`。

- `qkv_shapes(..., seq_len=5, phase="prefill")`：Q、K、V 都是 `(2, 2, 5, 4)`。
- `qkv_shapes(..., seq_len=6, phase="decode")`：Q 是 `(2, 2, 1, 4)`，K 和 V 是 `(2, 2, 6, 4)`。decode 的 K/V 指新 token 已经追加之后、查询要去看的整段 cache。

### token-forward（第 1 节）

`n` 是 prompt 长度，`m` 是新生成的 token 数。

- 无 cache：第 `k` 步（`k = 0 .. m-1`）重算长度 `n+k`。次数 `m*n + m*(m-1)/2`。
- 有 cache：prefill 的 `n` 次，加上 `m` 个新 token 各 1 次。次数 `n+m`。

| 例子 | n | m | 无 cache | 有 cache |
| --- | --- | --- | --- | --- |
| 短例，手算 4+5+6 | 4 | 3 | 15 | 7 |
| 同一闭式的对照 | 128 | 32 | 4592 | 160 |
| 5-token trace 再生成 1 个 | 5 | 1 | 5 | 6 |

### 字节（第 7 节）

`2 × layers × kv_heads × head_dim × seq_len × bytes_per_element`。

Llama 3 70B、GQA、FP16：80 层、8 个 KV 头、head_dim 128、每元素 2 字节。

- 1 个 token：327680 字节 = 320 KiB。笔记写「约 320 KB」，指的是这个数。
- 1K / 8K / 32K / 128K 按 1024 进位：335544320、2684354560、10737418240、42949672960 字节，也就是 320 MiB、2.5 GiB、10 GiB、40 GiB。

同一公式在玩具上：5 个 token 320 字节，6 个 token 384 字节。

### KVCache

`append` 接受 `[layer][head][position][head_dim]` 的 K 和 V，接到已有 position 后面。`get` 返回这两块的副本。空 cache 的 `seq_len` 是 0，层和头还在，position 列表为空。

形状不对时抛 `ValueError`，cache 保持调用前的长度和内容。修改传入的列表或 `get` 的返回值，都不能改 cache。

不要写注意力或整层 Transformer。

### 前缀复用

`prefix_reuse_hit` 只做规则：

- 缓存前缀非空，且请求 token 以这段前缀开头，id 逐个相同。
- model 字符串相同，adapter 字符串相同。

前缀末尾换成另一个 id（时间戳）则 miss。model 或 adapter 不同则 miss。请求比前缀更长可以命中；更短或前缀为空则 miss。

## 验证

```bash
make test-kvcache
```

验证器先跑上面这些字面量，再跑 `kv_cache.py` 和 `playground/prefill_decode.py` 的 doctest。playground 在骨架状态下可以通过；`kv_cache.py` 在写出实现之前失败，整个命令的退出码因此不是 0。
