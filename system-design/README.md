# system-design — 三语言系统设计组件板

用 **Python / Go / Java** 三种语言实现同一个系统设计核心组件，横向对比各语言的**并发模型、实现惯用法与现代最佳实践**。与 [`algo/`](../algo) 同构——`algo` 练算法，本板练系统设计组件。

> 设计理念与 rubickx 一致：**同一组件多语言重写，在差异里吃透每种语言的设计取向。**
> 为什么是这三种：Go（项目主语言 + 极轻并发）、Java（你的锚点 + JVM 并发体系）、Python（最快出原型 + GIL 这个绕不开的话题）。三者的并发故事差异最大，正是系统设计组件最该比的地方。

配套 vault：`dragon-vault/01_Projects/Interview_Prep/`（设计文档 [[01 - Rate Limiter]] + 笔记 [[Eng - Redis：架构、实现与高阶实战]]）。理论在 vault，三语言实现在这里。

## 目录结构（按系统并排）

```
system-design/
├── Makefile                 # make test / test-py / test-go / test-java
├── go.mod                   # 单一 Go module: rubickx/system-design
├── pytest.ini               # Python: pytest
├── PROGRESS.md              # 进度追踪表
└── systems/
    └── 01-rate-limiter/
        ├── NOTES.md         # ⭐ 三语言对比 + 最佳实践（中文，核心）
        ├── python/  token_bucket.py      + token_bucket_test.py
        ├── go/      tokenbucket.go        + tokenbucket_test.go   (package ratelimiter)
        └── java/    TokenBucket.java      + TokenBucketTest.java  (默认包，类名唯一)
```

每个系统先做最具代表性的**核心组件**（Rate Limiter → Token Bucket），三语言并排，`NOTES.md` 讲清**怎么写、为什么这么写、最佳实践**。

## 怎么跑测试

```bash
cd system-design
make setup        # 安装 Python(pytest)依赖
make test         # 跑全部三种语言
# 或单独跑：
make test-py      # pytest
make test-go      # go test ./...
make test-java    # javac + JUnit 5 console
```

### 工具链版本（最近实践）

| 语言 | 版本 | 测试框架 | 并发要点 |
|---|---|---|---|
| Python | 3.10+ | pytest | `threading.Lock`；GIL 不让复合操作原子，锁仍必需 |
| Go | 1.21+ | 标准库 `testing` | `sync.Mutex` / `sync/atomic`；`go test -race` 抓竞态 |
| Java | 17+(LTS) | JUnit 5.10 | `synchronized` / `ReentrantLock` / `AtomicLong` |

> 已在沙箱验证：**Python 测试通过**。Go / Java 需本机对应工具链（`go` / JDK 含 `javac`）。

## 几个工程决策（沿用 algo 的取舍）

- **按系统并排，而非按语言分目录**：核心诉求是"对比同一组件的三种实现"，并排最直观。
- **注入时钟做确定性测试**：三语言都把"当前时间"做成可注入参数（Python 传 callable、Go 传 `func() time.Time`、Java 传 `LongSupplier`），测试用假时钟推进，不靠 `sleep`。这是跨语言一致的可测试性最佳实践。
- **Java 默认包 + 唯一类名 + JUnit console**：和 algo 一致，避免 Maven/Gradle 的 `src/main/java` 强约定与"按系统并排"冲突。类名全局唯一（`TokenBucket`、后续 `SlidingWindow`…）。
- **Go 单 module**：`system-design/go.mod` 一个 module，每个 `go/` 子目录独立 package，`go test ./...` 一把梭。
