# Rate Limiter · 四语言算法与系统链路对比笔记

> 配套理论：vault [[01 - Rate Limiter]]（RESHADED 设计）+ [[Eng - Redis：架构、实现与高阶实战]]（集中式版本）。
> `python/`、`go/`、`java/` 保留最小 Token Bucket baseline；[`lab/`](lab/) 将学习面扩展为 Python / Go / Java / JavaScript 的五算法真实执行，以及 Go-only HTTP / Redis / Delve 系统链路。

## 核心思路（语言无关）

令牌桶 = 一个按固定速率 `refillRate` 往里加令牌、上限 `capacity` 的桶；每个请求取走若干令牌，取得到放行、取不到拒绝。

实现上**不开后台线程定时加令牌**，而是**惰性补充（lazy refill）**：每次请求时，按"距上次的时间差 × 速率"一次性把欠的令牌补上（封顶 capacity），再判断够不够扣。一个公式搞定，无需定时器。

```
elapsed = now - last
tokens  = min(capacity, tokens + elapsed * refillRate)
if tokens >= n: tokens -= n; allow
else:           deny
```

这段“读 tokens → 算 → 写 tokens”是**复合操作**，并发下必须保护，否则就是 [[Eng - Redis：架构、实现与高阶实战]] 里那个 check-then-act 竞态。四种语言的差别，主要就在**怎么保护**、**哪里可能交错**和**怎么注入时间**。

## 四语言关键差异

| 维度 | Python | Go | Java | JavaScript |
|---|---|---|---|---|
| 并发保护 | `threading.Lock` | `sync.Mutex` | `synchronized` | 同步 `allow()` 不跨 turn；共享远端状态需原子 store |
| 主要竞态 | GIL 不让复合操作原子 | goroutine 真并行 | JMM 可见性与原子性 | read 与 write 之间出现 `await` |
| 时钟注入 | `Callable[[], float]` | `func() time.Time` | `LongSupplier` | injected millisecond clock |
| 测试 | pytest | `testing` + `-race` | JUnit 5 | `node:test` |

## 各语言要点与最佳实践

### Python
- **GIL ≠ 线程安全**。常见误解："Python 有 GIL，所以不用锁"。GIL 只保证**单条字节码**不被打断；`self._tokens -= n` 会编译成 `LOAD/SUB/STORE` 多条，线程可在中间切换 → 丢更新。所以**复合操作必须显式加锁**（`token_bucket_test.py` 的 `test_concurrent_safe` 正是证明：100 线程抢 1000 令牌，有锁才恰好 1000）。
- **`time.monotonic` 而非 `time.time`**：限流要的是"流逝了多久"，单调时钟不受系统时间回拨影响。
- **最佳实践**：用 `with self._lock:` 而非手动 `acquire/release`（异常安全）；类型注解 `Callable[[], float]` 让时钟可注入。
- **其它实现方式**：① 高并发 I/O 场景用 `asyncio` + `asyncio.Lock`（单线程事件循环，协程间无抢占，但 `await` 点仍要保护）；② 多进程限流（GIL 限制 CPU 并行）则状态要下沉到 Redis（见集中式版本）。

### Go
- **`sync.Mutex` 是默认答案**；`defer b.mu.Unlock()` 保证任何返回路径都解锁。
- **`go test -race` 是杀手锏**：编译期插桩，运行时抓数据竞态。`test_concurrent_safe` 配 `-race` 能直接证明实现无竞态——这是 Go 相对 Java/Python 在"并发正确性可验证性"上的最大优势。
- **最佳实践**：构造函数分 `New`（真实时钟）/`NewWithClock`（注入），符合 Go "用函数而非可选参数" 的惯例；`float64` 全程，避免整数令牌的精度问题。
- **其它实现方式**：① 纯计数型限流可用 `sync/atomic`（无锁，但令牌桶的"读-补-扣"是复合操作，纯 atomic 写不出来，得 CAS 循环，复杂度不值）；② `golang.org/x/time/rate` 是标准库级生产实现（基于令牌桶，`Allow/Wait/Reserve` 三种语义）——面试可提"轮子已有，但要会手写原理"；③ 用带缓冲 channel 当令牌池 + 后台 goroutine 定时投放，是"channel 即信号量"的 Go 味写法，但比惰性补充重。

### Java
- **`synchronized` 最简**；进阶用 `ReentrantLock`（支持 `tryLock` 超时、可中断、公平锁）。
- **`LongSupplier` 注入 `System::nanoTime`**：`nanoTime` 是单调的（对应 Go 的 monotonic / Python 的 monotonic）；**别用 `System.currentTimeMillis()`**（会受 NTP 回拨影响）。
- **最佳实践**：纯计数器场景优先 `AtomicLong`（无锁 CAS，见测试里的计数）；需要"限速 + 阻塞等待"用 `Semaphore`；生产里 Guava 的 `RateLimiter`（SmoothBursty/SmoothWarmingUp）或 Resilience4j 的 `RateLimiter` 是成熟轮子。
- **JMM 提醒**：若把 `tokens` 暴露成无锁读取，必须 `volatile` 才有可见性保证——本实现所有读写都在 `synchronized` 内，已隐含 happens-before，无需额外 `volatile`。

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

```bash
cd system-design
make test
make -C systems/01-rate-limiter/lab test-race
make -C systems/01-rate-limiter/lab verify
make -C systems/01-rate-limiter/lab verify-redis
make -C systems/01-rate-limiter/lab verify-debug
```

Lab 通过共享 fixtures 对比 `5 algorithms × 4 languages` 的 allow/deny、remaining 和 retry timing；语言自己的测试再覆盖并发模型与输入边界。

## Practice surface

- `cd lab && make dev`：在浏览器里逐步运行真实源码 trace。
- `cd lab && make verify-redis`：验证 Lua atomicity、TTL、shared quota 与 outage policy。
- `cd lab && make verify-debug`：通过真实 Delve DAP 查看 Go source line、stack 和 locals。
