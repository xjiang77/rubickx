# foundations —— 系统基础前置（GP0）的代码实践

配套 vault：`[[Session Log - GP0 系统基础前置]]`（学习阶梯 L0–L5）。
每一阶的概念都尽量配一个**可跑、可验证**的 Go 实验，用 Go 机制把抽象概念落到能 `go run` / `go test -race` 的代码上，并标注 **Java 锚点**。

模块路径：`github.com/xjiang77/rubickx/go`，所以这里的实验都在 `go/foundations/` 下。

## 阶梯 → 实验对照

| 阶 | 概念 | 实验目录 | 怎么验证 | 状态 |
|----|------|----------|----------|------|
| L0 | 执行模型：goroutine、并发、数据竞态 | `l0-execution-model/` | `go run -race .` 看竞态被抓；`go test -race .` 看 Mutex/Atomic 全绿 | ✅ |
| L1 | 内存与数据结构 | `l1-memory-structures/`（待建） | benchmark 值 vs 指针、map/slice | ⬜ |
| L2 | 并发正确性深化：mutex/atomic/channel | `l2-concurrency/`（待建） | `-race` + bench atomic vs mutex vs channel | ⬜ |
| L3 | 网络与延迟：RTT、QPS、pipeline | `l3-network/`（待建） | 最小 TCP echo + 批处理对比 | ⬜ |
| L4 | 分布式语义：session、replica、quorum、isolation、duplicate effect | `l4-distributed-semantics/` | deterministic history / partition / access-set / delivery tests | ✅ |
| L5 | 系统设计框架 | 各 system-design 实现 | `go test` | ⬜ |

## 跑法（在 `go/` 目录下）

```bash
cd go
go run        ./foundations/l0-execution-model      # 看现象（含丢更新）
go run  -race ./foundations/l0-execution-model      # 竞态检测器抓 UnsafeCounter
go test -race ./foundations/l0-execution-model      # Mutex/Atomic 应全绿
go test -bench . ./foundations/l0-execution-model   # atomic vs mutex
go test -race ./foundations/l4-distributed-semantics # session/replica/quorum/isolation/idempotency
go vet ./foundations/...
```

## Java 锚点速查

| Go | Java | 说明 |
|----|------|------|
| goroutine | 线程 / 虚拟线程(Loom) | goroutine 初始栈 ~2KB，由 runtime(GMP) 调度，几十万个没问题 |
| `sync.Mutex` | `synchronized` / `ReentrantLock` | 互斥锁，保护临界区 |
| `sync/atomic` | `AtomicInteger` / `AtomicLong` | 无锁原子操作（CAS/FAA 指令） |
| `go test -race` | （无内建对应；async-profiler 之外靠工具） | 编译期插桩，运行时抓数据竞态 |

> L0 的 race demo 对应 Vault `Redis：契约、状态与运维` 中 command atomicity vs client-side GET-then-SET：
> Redis 通过单 shard execution owner 让单条 command 串行生效；Go 进程内用 Mutex / atomic 保护同一 read-modify-write。两者都不自动提供跨 key、durability 或分布式事务保证。
