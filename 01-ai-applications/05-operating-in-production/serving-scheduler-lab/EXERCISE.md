# Exercise — Serving 调度概念练习（1.5.2）

> C5 约束：实现由本人手写。agent 可以讲解概念、评审代码、维护验证脚本，不得把完成的实现写进这个目录。

## 顺序

1. 读 `playground/comparison.py` 的对照表，先把静态批、连续批和 overload 分开。
2. 在 `scheduler.py` 里手写下面的函数。类型标注和 Google docstring 已经在骨架里，`Examples` 是 doctest。
3. `python3 -m doctest scheduler.py -v`
4. 从仓库根目录运行 `make test-serving`。

骨架状态下第 3、4 步失败是预期结果。不要加 `pytest.skip` 或 `xfail`。

## 目标

只用列表和整数，把笔记里能手算的调度规则写成函数。时间是模拟步，不是墙钟。

笔记路径：`02_Knowledge/01-ai-applications/05-operating-in-production/01-inference-and-serving/03 - AI - LLM Serving Runtime：Scheduler、KV Cache 与 Backpressure.md`。

本练习把这些规则写成合同。私有 vault 在准备骨架时没有打开，正文仍是待核验。若笔记里的某一组输入和下面不同，改 `fixtures.py` 里对应的字面量，不要改比较方式。

## 参数

都在 `fixtures.py`，全是 int：

| 名字 | 值 | 含义 |
| --- | --- | --- |
| `TOKEN_BUDGET_PER_STEP` | 8 | 一步里 prefill 与 decode 的 token 之和 |
| `CHUNK_SIZE` | 4 | chunked prefill 每条请求每步最多拿走的 prefill token |
| `KV_CAPACITY` | 24 | 已预留 KV token 的上限 |
| `MAX_QUEUE` | 2 | 等待队列长度；达到这个数再来就拒绝 |

混合负载，都在 step 0 到达，列表顺序就是调度顺序：

| id | prompt | output |
| --- | --- | --- |
| `R_short` | 4 | 2 |
| `R_long_prefill` | 16 | 2 |
| `R_long_decode` | 4 | 8 |

## 合同

`scheduler.py` 不得 import numpy、torch，也不得读网络。`simulate` 要使用 `fixtures.TOKEN_BUDGET_PER_STEP` 和 `fixtures.CHUNK_SIZE`。

整数参数如果是 bool、不是 int，或小于下界，抛 `ValueError`。不要改调用方传入的 dict。

### 前缀复用

`prefill_work(prompt_len, shared_prefix)` 返回 `prompt_len - shared_prefix`。`shared_prefix` 要在 0 和 `prompt_len` 之间，含两端。

`decode_work(output_len)` 返回 `output_len`。前缀再长也不减少 decode。

第二个请求 prompt 16、共享前缀 4：prefill 是 12，decode 仍是 2。

### 背压

`classify_admission` 返回 `{"outcome": ..., "reason": ...}`。

- `live_kv + kv_need > capacity`：`outcome` 是 `overload`，`reason` 是 `kv`。相等可以准入。
- 否则如果 `max_queue` 不是 `None` 且 `queue_len >= max_queue`：`overload` / `queue`。
- 否则：`admit` / `ok`。
- `max_queue is None` 是无限队列：不检查队列，KV 闸门还在。两条都该拒绝时，`reason` 取 `kv`。

`screen_arrivals` 从左到右处理 `kv_needs`。已预留 KV 从 `live_kv` 起，队列长度从 0 起。admit 才把这份 `kv_need` 加进预留，并把队列长度加 1。overload 两个数都不变，后面的请求还能再试。

对照：

- live 16、need 8、capacity 24：admit。
- live 20、need 8：overload kv（20 + 8 = 28）。
- 队列长度已经是 2、need 4、KV 放得下：overload queue。
- 同样的队列长度、`max_queue=None`：admit。
- needs `[4, 4, 4]`、`max_queue=2`：admit、admit、overload queue。无限队列则三次都是 admit，队列长度变成 3。
- live 20、needs `[8, 4, 4, 4]`：第一个 8 被 KV 拒绝且不占名额；接着的 4 把预留补到 24；后面的 4 都是 overload kv。

### 一步里怎么排

`simulate(requests, policy)` 的 `policy` 只能是 `static`、`chunked`、`monolithic`。

请求字段是 `id`、`prompt`、`output`、`arrival`。`prompt >= 0`，`output >= 1`，`arrival >= 0`，`id` 是非空字符串且不重复。空列表返回空的 completions、ttft、steps。

步号从 0 开始。一步的 `tokens` 等于这一步的 `prefill + decode`，并且不超过 8。一条请求一步最多 decode 1 个 token。本步刚把 prefill 做完的请求，下一步才能 decode。完成步是 decode 减到 0 的那一步。TTFT 是第一次 decode 的那一步。

三种策略：

- **chunked**：先给 prefill 已经为 0、且已经到达的请求各 decode 1 个，按列表顺序，直到 budget 用完。然后按列表顺序，每条还在 prefill 的请求最多拿一个 chunk：`min(4, 剩余 prefill, 剩余 budget)`。本步每条最多一个 chunk。
- **monolithic**：decode 阶段与 chunked 相同。prefill 只给一条：剩余 prefill 最大的；一样大就取列表里更靠前的。它不受 chunk 限制，可以拿走本步剩下的全部 budget。
- **static**：维护一个当前批。某一步开始时，如果上一批都已经完成，就用「这一步开始时已经到达、还没完成」的请求组成下一批。批次开着的时候，后到的请求不能插进来。批里只要还有 prefill，就只给列表里第一条没做完的 prefill，它可以拿走整个 budget；同一步不开始下一条的 prefill，也不 decode。大家都 prefill 完了，才按列表顺序 decode。某条在本步 decode 完，下一批最早也要到下一步才开始。

`prompt == 0` 的请求没有 prefill，从它第一次能被调度的那一步直接 decode。

### 写死的结果

混合负载：

| 策略 | R_short 完成 | R_long_prefill 完成 | R_long_decode 完成 | R_short TTFT |
| --- | --- | --- | --- | --- |
| static | 5 | 5 | 11 | 4 |
| monolithic | 4 | 3 | 11 | 3 |
| chunked | 2 | 5 | 10 | 1 |

短请求的完成步是 2 < 4 < 5。长 prefill 在 monolithic 下第 3 步完成，在 chunked 下第 5 步完成：它少占 budget 之后，自己会晚一点结束。三种策略做完的 token 总数都是 36。

chunked 的混合负载，逐步是：

| step | 发生的事 | prefill | decode | tokens |
| --- | --- | --- | --- | --- |
| 0 | R_short 4，R_long_prefill 4；R_long_decode 没额度 | 8 | 0 | 8 |
| 1 | R_short decode 1；R_long_prefill 4；R_long_decode 3 | 7 | 1 | 8 |
| 2 | R_short 完成；R_long_prefill 4；R_long_decode 的 prefill 做完 | 5 | 1 | 6 |
| 3 | R_long_decode 开始 decode；R_long_prefill 的 prefill 做完 | 4 | 1 | 5 |
| 4 | 两条都在 decode | 0 | 2 | 2 |
| 5 | R_long_prefill 完成 | 0 | 2 | 2 |
| 6–10 | 只剩 R_long_decode，第 10 步完成 | 0 | 1 | 1 |

下一批：`R_a` 在 step 0 到达，`R_b` 在 step 1 到达，两者都是 prompt 4、output 2。静态批要等 `R_a` 在 step 2 完成，`R_b` 的 prefill 从 step 3 开始，`R_b` 在 step 5 完成。chunked 让 `R_b` 在 step 1 就做 prefill，`R_b` 在 step 3 完成。

每一步的完整账在 `fixtures.py`。

## 验证

```bash
make test-serving
```

验证器先核对 fixtures 和对照表，再跑上面这些字面量，然后跑 `scheduler.py` 和 `playground/comparison.py` 的 doctest。playground 在骨架状态下可以通过；`scheduler.py` 在写出实现之前失败，整个命令的退出码因此不是 0。
