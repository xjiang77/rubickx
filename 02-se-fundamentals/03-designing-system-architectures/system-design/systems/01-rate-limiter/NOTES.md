# Rate Limiter · 四语言算法与系统链路对比笔记

> 配套知识：[Rate Limiting：策略、状态与失败语义](https://github.com/xjiang7712/dragon-vault/blob/main/02_Knowledge/02_Engineering/08%20-%20Components/02%20-%20Eng%20-%20Rate%20Limiting%EF%BC%9A%E7%AD%96%E7%95%A5%E3%80%81%E7%8A%B6%E6%80%81%E4%B8%8E%E5%A4%B1%E8%B4%A5%E8%AF%AD%E4%B9%89.md) + [Redis：契约、状态与运维](https://github.com/xjiang7712/dragon-vault/blob/main/02_Knowledge/02_Engineering/08%20-%20Components/01%20-%20Redis/00%20-%20Eng%20-%20Redis%EF%BC%9A%E5%A5%91%E7%BA%A6%E3%80%81%E7%8A%B6%E6%80%81%E4%B8%8E%E8%BF%90%E7%BB%B4.md)。本文件解释四语言表达与 lab 证据，动态完成度只以 [`../../PROGRESS.md`](../../PROGRESS.md) 为准。
> `python/`、`go/`、`java/` 保留最小 Token Bucket baseline；[`lab/`](lab/) 将学习面扩展为 Python / Go / Java / JavaScript 的五算法真实执行，以及 Go-only HTTP / Redis / Delve 系统链路。
> 系统级 RESHADED 设计的 repo-adjacent 镜像见 [`DESIGN.md`](./DESIGN.md)。

## 共同契约

- `capacity` / `limit` / `window` 必须为正数，Token Bucket 的 `refillRate` 可为 0 但不能为负；非法配置 fail fast：Python `ValueError`、Go `panic`、Java `IllegalArgumentException`。
- `allow(n)` / `AllowN(n)` 的 cost 必须大于 0。`n <= 0` 返回 `false`，且不得 refill、rollover 或改变任何状态。
- 拒绝请求不扣 token、不增加 counter。
- 注入 monotonic clock 做确定性测试；若测试时钟回拨，按 `elapsed = 0` 处理，而且不能把内部时间基准倒退。
- 一次判断涉及“读状态 → 计算 → 写状态”，必须在同一个临界区完成；只保护写操作仍会有 check-then-act 竞态。

## Token Bucket

`capacity` 决定最多能积攒多少 burst，`refillRate` 决定长期平均吞吐。实现使用 lazy refill，不开后台线程：

```text
elapsed = max(0, now - lastRefill)
tokens  = min(capacity, tokens + elapsed * refillRate)
allow   = tokens >= n
if allow: tokens -= n
```

这段“读 tokens → 算 → 写 tokens”是**复合操作**，并发下必须保护；单进程使用语言同步原语，共享 Redis state 使用单 command 或同 slot script/function。四种语言的差别，主要就在**怎么保护**、**哪里可能交错**和**怎么注入时间**。

## 四语言关键差异

| 维度 | Python | Go | Java | JavaScript |
|---|---|---|---|---|
| 并发保护 | `threading.Lock` | `sync.Mutex` | `synchronized` | 同步 `allow()` 不跨 turn；共享远端状态需原子 store |
| 主要竞态 | GIL 不让复合操作原子 | goroutine 真并行 | JMM 可见性与原子性 | read 与 write 之间出现 `await` |
| 时钟注入 | `Callable[[], float]` | `func() time.Time` | `LongSupplier` | injected millisecond clock |
| 测试 | pytest | `testing` + `-race` | JUnit 5 | `node:test` |

### Sliding Window Counter baseline 补充

`python/`、`go/`、`java/` 中的本地 baseline 还包含 Sliding Window Counter：状态固定为 `previousCount`、`currentCount`、`currentWindowStart`，空间复杂度 O(1)。它用上一窗口的线性权重近似当前滑动窗口内的请求量：

```text
weight   = 1 - elapsedInCurrent / window
estimate = currentCount + previousCount * weight
allow    = estimate + n <= limit
if allow: currentCount += n
```

- 跨一个窗口时，旧 `currentCount` 转为 `previousCount`，新 `currentCount = 0`。
- 跨两个及以上窗口时，previous/current 都归零。
- 恰好切窗时 previous 权重为 100%，避免 Fixed Window 边界两侧形成 `2N` 突刺。
- 这是线性加权近似，不等价于保存每个请求时间戳的精确 Sliding Window Log。

<!-- 以下语言级细节同时适用于 Token Bucket 与 Sliding Window Counter baseline。 -->

只有 `now >= lastRefill` 时才推进 `lastRefill`。因此 clock rollback 不会凭空增减令牌，也不会让回拨后的低时间戳成为新基准。`Tokens()` 只暴露最近一次 materialized 状态用于测试/诊断，不承诺调用时自动 refill，不能当实时余额 API。

Token Bucket 允许“空闲积攒、瞬时消费”：满桶时可立刻放行 `capacity`，之后按 `refillRate` 恢复。这是它适合 API gateway、任务调度和允许短 burst 流量的原因。

## Sliding Window Counter

状态固定为 `previousCount`、`currentCount`、`currentWindowStart`，空间复杂度 O(1)。它用上一窗口的线性权重近似当前滑动窗口内的请求量：

```text
weight   = 1 - elapsedInCurrent / window
estimate = currentCount + previousCount * weight
allow    = estimate + n <= limit
if allow: currentCount += n
```

### Rollover 与边界

- 仍在当前窗口：不 rollover，直接按 elapsed 计算权重。
- 跨一个窗口：旧 `currentCount` 转为 `previousCount`，新 `currentCount = 0`，窗口起点前进一个 window。
- 跨两个及以上窗口：previous/current 都归零，旧请求已经完全离开估算范围。
- 恰好切窗：新窗口 elapsed 为 0，previous 权重为 100%。若上一窗口刚好打满，边界后的请求会被拒绝，而不是像 Fixed Window 一样在边界两侧各放行 N、形成 `2N` 突刺。
- 随着新窗口推进，previous 权重从 1 线性降到 0；半窗口时权重恰好为 50%。这是近似，不等价于保存每个请求时间戳的精确 Sliding Window Log。

## 三语言实现差异

| 维度 | Python | Go | Java |
|---|---|---|---|
| 并发保护 | `threading.Lock` + `with` | `sync.Mutex` + `defer Unlock` | `synchronized` 方法 |
| 真实 monotonic clock | `time.monotonic`（秒） | `time.Now` / `time.Duration` | `System.nanoTime`（纳秒） |
| 测试时钟注入 | `Callable[[], float]` | `func() time.Time` | `LongSupplier` |
| 数值 | `float` | `float64` | `double` |
| 并发验收 | Barrier + 加锁计数 | start channel + `atomic` + `-race` | `CountDownLatch` + `AtomicLong` |

- **Python**：GIL 只保证单条 bytecode 层面的执行，不会让 refill/estimate/扣减这组复合操作原子；仍需 `Lock`。
- **Go**：多个 goroutine 可真并行，`Mutex` 是这类小临界区的直接选择；`go test -race` 同时验证测试路径不存在 data race。
- **Java**：`synchronized` 同时提供 mutual exclusion 与 JMM happens-before；状态不需要再加 `volatile`。只有需要超时、公平锁或可中断获取时才值得换 `ReentrantLock`。

## 怎么选

| 维度 | Token Bucket | Sliding Window Counter |
|---|---|---|
| 核心语义 | 平均速率 + 可积攒 burst | 最近一个 window 的近似配额 |
| 状态量 | tokens + last refill | previous/current + window start |
| 空间 | O(1) | O(1) |
| 边界行为 | burst 由 capacity 明确定义 | 平滑 Fixed Window 的 `2N` 边界突刺 |
| 精确度 | 对“速率”语义精确 | 线性加权近似；不如 Sliding Window Log 精确 |
| 优先场景 | API gateway、下载/任务整形、允许短 burst | 登录/发送/风控配额，更在意任意窗口公平性 |

一句话：业务说“每秒稳定处理 R，允许短时突发 C”选 Token Bucket；业务说“任意 W 秒内不要明显超过 N，且不接受固定窗口边界翻倍”选 Sliding Window Counter。若审计要求逐请求精确，再考虑 Sliding Window Log，并接受 O(N) 状态成本。

### JavaScript

- **event loop ≠ 通用原子性**：不含 `await` 的同步 `allow()` 会在一个 turn 内完成，别为了单线程内的同步状态机械加 mutex。
- **`await` 是明确的交错 seam**：把 read 与 write 分在两个 `await` 之间，另一个 task 就能观察旧值并造成 lost update。Lab 包含 deterministic race case，不用概率性 `sleep` 证明它。
- **进程边界更重要**：Node cluster、多实例和其它语言进程完全不共享 event loop。共享 quota 要在 Redis 中用 `INCR`/Lua 原子执行，不能把“JavaScript 单线程”误当 distributed lock。
- **最佳实践**：Node runner 使用 plain ESM、injected millisecond clock 与内建 `node:test`，不把 UI 的 TypeScript toolchain 泄漏进算法实现。

## 并发模型深差异（这才是四语言最该比的）

| | Python | Go | Java | JavaScript |
|---|---|---|---|---|
| 执行单元 | OS 线程（受 GIL 串行化 CPU） | goroutine（runtime 调度） | OS/virtual thread | event-loop task / worker |
| 真并行 | CPU 默认受 GIL | 多核 | 多核 | 单 loop 否；worker/多进程是 |
| 同步原语 | `Lock/RLock/Semaphore` | `Mutex`、channel、`atomic` | `synchronized`、`j.u.c` | 避免跨 `await` RMW；远端原子 store |
| 竞态验证 | 测试/审查 | `go test -race` | 测试/分析工具 | deterministic interleaving test |
| 设计取向 | 简单优先；CPU 密集靠多进程/C 扩展 | "用通信共享内存"(channel) 与"共享内存+锁"并存 | 成熟庞大的 `java.util.concurrent` 生态 | 单进程保持同步状态简单；跨进程把原子性下沉到 shared store |

> 一句话：**同一个限流算法，四种语言的数学规则相同，差的是“状态在哪、何时交错、怎样证明正确”**。这正是多语言重写的价值。

## 怎么验证

baseline 测试覆盖满额后拒绝、multi-token/cost、拒绝不改变计数、时钟回拨、确定性时间推进和同步起跑的并发请求；Sliding Window Counter 另覆盖恰好切窗、半窗口 50% 权重与跨两个窗口清零。

```bash
cd system-design
make test
make -C systems/01-rate-limiter/lab test-race
make -C systems/01-rate-limiter/lab verify
make -C systems/01-rate-limiter/lab verify-redis
make -C systems/01-rate-limiter/lab verify-debug
make test-go-race
```

Lab 通过共享 fixtures 对比 `5 algorithms × 4 languages` 的 allow/deny、remaining 和 retry timing；语言自己的测试再覆盖并发模型与输入边界。

## Practice surface

- `cd lab && make dev`：在浏览器里逐步运行真实源码 trace。
- `cd lab && make verify-redis`：验证 Lua atomicity、TTL、shared quota 与 outage policy。
- `cd lab && make verify-debug`：通过真实 Delve DAP 查看 Go source line、stack 和 locals。

三语言 baseline 可用于脱稿检查：先讲两个公式与状态，再画出 SWC rollover，解释边界为什么不是 `2N`，最后用 burst、公平性、精确度和内存成本完成选型。
