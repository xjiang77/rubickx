# system-design — 四语言系统设计组件板

用 **Python / Go / Java / JavaScript** 四种语言实现同一个系统设计核心组件，横向对比各语言的**并发模型、实现惯用法与现代最佳实践**。与 [`algo/`](../algo) 同构——`algo` 练算法，本板练系统设计组件。

> 设计理念与 rubickx 一致：**同一组件多语言重写，在差异里吃透每种语言的设计取向。**
> 四种语言给出四种并发视角：Go 的 goroutine、Java 的 JVM 并发体系、Python 的 GIL 与显式锁、JavaScript 的 event loop 与跨 `await` 竞态。算法相同，正确性边界不同。

配套 Vault 知识层：[Rate Limiting：策略、状态与失败语义](https://github.com/xjiang7712/dragon-vault/blob/main/02_Knowledge/02_Engineering/08%20-%20Components/02%20-%20Eng%20-%20Rate%20Limiting%EF%BC%9A%E7%AD%96%E7%95%A5%E3%80%81%E7%8A%B6%E6%80%81%E4%B8%8E%E5%A4%B1%E8%B4%A5%E8%AF%AD%E4%B9%89.md) 与 [Redis：契约、状态与运维](https://github.com/xjiang7712/dragon-vault/blob/main/02_Knowledge/02_Engineering/08%20-%20Components/01%20-%20Redis/00%20-%20Eng%20-%20Redis%EF%BC%9A%E5%A5%91%E7%BA%A6%E3%80%81%E7%8A%B6%E6%80%81%E4%B8%8E%E8%BF%90%E7%BB%B4.md)。稳定判断在 Vault，四语言实现、动态完成度与可运行 Lab 在本仓库；进度只以 [`PROGRESS.md`](PROGRESS.md) 为准。

Rate Limiter 的 repo-adjacent 系统设计镜像见 [`DESIGN.md`](systems/01-rate-limiter/DESIGN.md)，实现比较见 [`NOTES.md`](systems/01-rate-limiter/NOTES.md)。

## Rate Limiter Lab

[`systems/01-rate-limiter/lab/`](systems/01-rate-limiter/lab/) 是本板的第一个完整 practice surface：五种经典算法、四语言真实 semantic trace、Go HTTP/Redis 链路和 Delve DAP 调试都收在一个本地 Web App 中。

```bash
cd systems/01-rate-limiter/lab
make setup
make dev
# http://127.0.0.1:8080
```

## 目录结构（按系统并排）

```
system-design/
├── Makefile                 # make test / test-py / test-go / test-go-race / test-java / test-js
├── go.mod                   # 单一 Go module: rubickx/system-design
├── pytest.ini               # Python: pytest
├── PROGRESS.md              # 进度追踪表
└── systems/
    └── 01-rate-limiter/
        ├── DESIGN.md        # 系统设计的 repo-adjacent 镜像
        ├── NOTES.md         # ⭐ 四语言对比 + 最佳实践（中文，核心）
        ├── python/  token_bucket.py      + token_bucket_test.py
        ├── go/      tokenbucket.go        + tokenbucket_test.go   (package ratelimiter)
        ├── java/    TokenBucket.java      + TokenBucketTest.java  (默认包，类名唯一)
        └── lab/     四语言五算法 Web Lab + Go system path
```

每个系统先做最具代表性的**核心组件**，四语言并排，`NOTES.md` 讲清**怎么写、为什么这么写、最佳实践**；成熟组件再进入可交互 Lab。

## 怎么跑测试

```bash
cd system-design
make setup        # 安装 Python(pytest)依赖
make test         # 跑全部四种语言
# 或单独跑：
make test-py      # pytest
make test-go      # go test ./...
make test-go-race # go test -race ./...
make test-java    # javac + JUnit 5 console
make test-js      # Node.js built-in node:test
```

### 工具链版本（最近实践）

| 语言 | 版本 | 测试框架 | 并发要点 |
|---|---|---|---|
| Python | 3.10+ | pytest | `threading.Lock`；GIL 不让复合操作原子，锁仍必需 |
| Go | 1.21+ | 标准库 `testing` | `sync.Mutex` / `sync/atomic`；`go test -race` 抓竞态 |
| Java | 17+(LTS) | JUnit 5.10 | `synchronized` / `ReentrantLock` / `AtomicLong` |
| JavaScript | Node.js 24 | `node:test` | 同步 turn 内不交错；跨 `await` 的复合操作仍有竞态 |

> 当前 Lab 已在本机完成四语言测试、`go test -race`、UI production build、真实 Redis、HTTP E2E 与 Delve DAP 验证；复现入口见 [`lab/README.md`](systems/01-rate-limiter/lab/README.md)。

## 几个工程决策（沿用 algo 的取舍）

- **按系统并排，而非按语言分目录**：核心诉求是“对比同一组件的四种实现”，并排最直观。
- **注入时钟做确定性测试**：四语言都把“当前时间”做成依赖；测试推进 fake clock，不靠 `sleep`。这是跨语言一致的可测试性最佳实践。
- **Java 默认包 + 唯一类名 + JUnit console**：和 algo 一致，避免 Maven/Gradle 的 `src/main/java` 强约定与"按系统并排"冲突。类名全局唯一（`TokenBucket`、后续 `SlidingWindow`…）。
- **Go 单 module**：`system-design/go.mod` 一个 module，每个 `go/` 子目录独立 package，`go test ./...` 一把梭。
