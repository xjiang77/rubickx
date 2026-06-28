# Rate Limiter → Token Bucket · 三语言对比笔记

> 配套理论：vault [[01 - Rate Limiter]]（RESHADED 设计）+ [[Eng - Redis：架构、实现与高阶实战]]（集中式版本）。
> 本篇只聚焦**单机进程内**的 Token Bucket：在 Python / Go / Java 里怎么写、为什么这么写、最佳实践是什么。

## 核心思路（语言无关）

令牌桶 = 一个按固定速率 `refillRate` 往里加令牌、上限 `capacity` 的桶；每个请求取走若干令牌，取得到放行、取不到拒绝。

实现上**不开后台线程定时加令牌**，而是**惰性补充（lazy refill）**：每次请求时，按"距上次的时间差 × 速率"一次性把欠的令牌补上（封顶 capacity），再判断够不够扣。一个公式搞定，无需定时器。

```
elapsed = now - last
tokens  = min(capacity, tokens + elapsed * refillRate)
if tokens >= n: tokens -= n; allow
else:           deny
```

这段"读 tokens → 算 → 写 tokens"是**复合操作**，并发下必须保护，否则就是 [[Eng - Redis：架构、实现与高阶实战]] 里那个 check-then-act 竞态。三种语言的差别，主要就在**怎么保护**和**怎么注入时间**。

## 三语言关键差异

| 维度 | Python | Go | Java |
|---|---|---|---|
| 并发保护 | `threading.Lock` | `sync.Mutex` | `synchronized` 方法 |
| 为什么需要锁 | GIL 只保证单条字节码原子，`-=` 是多条 → 仍需锁 | 编译器/CPU 会重排，多 goroutine 真并行 → 必须锁 | JMM 下多线程可见性/原子性都无保证 → 必须锁 |
| 时钟注入 | 传 `Callable[[], float]` | 传 `func() time.Time` | 传 `LongSupplier`(纳秒) |
| 数值类型 | `float`(双精度) | `float64` | `double` |
| 取最小值 | `min()` 内建 | `min()` 内建(Go 1.21+) | `Math.min()` |
| 构造重载 | 默认参数 `now=time.monotonic` | 两个构造函数 `New`/`NewWithClock` | 两个构造函数重载 |
| 并发计数验证 | `threading.Lock` 累加 | `atomic.AddInt64` + `-race` | `AtomicLong` |

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

## 并发模型深差异（这才是三语言最该比的）

| | Python | Go | Java |
|---|---|---|---|
| 执行单元 | OS 线程（受 GIL 串行化 CPU） | goroutine（runtime 调度，~2KB 栈，几十万个） | OS 线程（~1MB 栈）/ 虚拟线程(Loom, 21+) |
| 真并行 | ❌ CPU 受 GIL；I/O 可并发 | ✅ 多核真并行 | ✅ 多核真并行 |
| 同步原语 | `threading.Lock/RLock/Semaphore` | `sync.Mutex/RWMutex`、channel、`atomic` | `synchronized`、`j.u.c.locks`、`atomic` |
| 竞态检测 | 无内建（靠测试/审查） | `go test -race` 一等公民 | 无内建（靠测试/FindBugs 类工具） |
| 设计取向 | 简单优先；CPU 密集靠多进程/C 扩展 | "用通信共享内存"(channel) 与"共享内存+锁"并存 | 成熟庞大的 `java.util.concurrent` 生态 |

> 一句话：**同一个令牌桶，三种语言的算法完全一样，差的全是"并发怎么保证正确"和"时间怎么拿"**。这正是用多语言重写来吃透语言差异的价值。

## 怎么验证

```bash
cd system-design
make test-py     # ✅ 沙箱已验证通过
make test-go     # 本机；并发安全建议 go test -race ./systems/01-rate-limiter/go
make test-java   # 本机（需 JDK 含 javac）
```

四个测试三语言一致：满桶突发、按时间补充、封顶、并发恰好放行 1000（线程安全证明）。

## 下一步

- 同目录补 **Sliding Window Counter**（O(1) 近似），同样三语言并排。
- 把 Token Bucket 接成一个最小 HTTP 中间件（Tier B mini-system），观察"加在请求关键路径"的延迟。
