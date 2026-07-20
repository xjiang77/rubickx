# Rate Limiter Lab

一个只服务于学习的本地 Rate Limiter Web Lab：用同一组 deterministic scenarios 横向运行 **Python / Go / Java / JavaScript** 的真实实现，再沿 semantic trace 逐步观察窗口、桶和队列如何变化。完整 HTTP、Redis 与真实 debugger 链路保持 Go-only，让算法对比和系统设计边界彼此清楚。

## Quick start

Prerequisites: Python 3.10+, Go 1.21+, JDK 17+, and Node.js 24. `make verify-redis` additionally requires a local `redis-server` executable; Docker Compose is only the long-running Redis option for `make dev`. JavaScript runners are plain native ESM and use only `node:test`; TypeScript stays inside the React UI.

```bash
cd system-design/systems/01-rate-limiter/lab
make setup
make dev
```

打开 <http://127.0.0.1:8080>。核心 Lab 不依赖 Redis。交互体验 Redis 场景时可启动长期容器；独立验收则直接运行会自启临时 `redis-server` 的 verifier：

```bash
docker compose up -d redis  # interactive UI
make verify-redis           # isolated end-to-end verification
```

页头的 `EN / 中文` 只切换 UI chrome；scenario catalog、trace、源码、HTTP 数据、状态 token 与既有 ARIA 文案始终保持英文。选择会持久化到 `localStorage["rl-lab-uiLang"]`，合法值仅为 `en | zh`，无效值或 storage 不可用时安全回退英文。

默认视觉保持全直角。需要对比 soft-lines 圆角方案时，用同一开发入口显式开启；该开关不进入 UI，也不会持久化：

```bash
VITE_SOFT_LINES=true make dev
```

## Mental model

```text
React UI
   │ RunRequest / TraceEvent / DebugSnapshot
   ▼
Go Lab Server
   ├── Go runner (in-process)
   ├── Python runner ─┐
   ├── Java runner   ├── controlled JSON Lines subprocess
   └── JS runner ────┘

Go /demo middleware ── MemoryStore
                    └─ RedisStore ── Lua (atomic INCRBY + PEXPIRE)

Go Debug UI ── DelveSessionManager ── dlv dap ── controlled harness ── real algorithms.go
```

`LanguageRunner` 是四语言行为 seam；调用方只知道固定 `RunRequest` 和 `RunResponse`。算法自己通过 observer 发出 `stepId`，server 再把 `@step:<id>` anchor 映射到真实源码行。UI 的动画来自这次执行产生的 trace，不保存另一份“教学版算法”。

时间统一为 integer milliseconds；数值输出遵循 [`fixtures/ROUNDING_POLICY.md`](fixtures/ROUNDING_POLICY.md) 的六位 IEEE-754 quantization，shared fixtures 因而可以直接比较四语言结果。

## Algorithms

| Algorithm | What it makes visible | Main trade-off |
| --- | --- | --- |
| Fixed Window | window rollover and count | O(1), boundary burst |
| Sliding Window Log | timestamp eviction | exact, O(n) memory |
| Sliding Window Counter | weighted previous window | O(1), approximate |
| Token Bucket | lazy refill and burst budget | burst-friendly |
| Leaky Bucket | queued water and steady drain | smooth output |

Core scenarios run in all four languages: `steady-traffic`, `burst-capacity`, `window-boundary`, `exactness-vs-memory`, `smoothed-output`, `concurrent-callers`, `clock-input-safety`.

System scenarios run through Go: `policy-composition`, `http-contract`, `local-vs-shared`, `redis-atomicity`, `redis-outage`, `hot-key-sharding`, `multi-region-quota`.

`local-vs-shared` exposes Replica A/B: Memory keeps an independent quota per replica while Redis shares the same key. `hot-key-sharding` returns stable `hash(key)%4` routing evidence, so repeating one key visibly concentrates on one shard without inventing fake performance numbers. `multi-region-quota` remains an explicitly conceptual trade-off exercise.

JavaScript's concurrency case deliberately separates two facts:

- A synchronous `allow()` completes within one event-loop turn.
- A read-modify-write split across `await` points is not atomic. Multiple processes require a shared atomic store such as Redis; the event loop is not a distributed lock.

Semantic runners intentionally use a deterministic event order so traces can rewind exactly; they do not pretend to be an OS thread scheduler. Shared fixtures interleave `alice` and `bob` for all five algorithms to prove per-key isolation. Actual wall-clock contention is tested at the concurrency seams that own it: the existing Python/Go/Java Token Bucket baselines, JavaScript's limiter-specific cross-`await` over-admission case, and the Go Memory/Redis stores (including `go test -race` and atomic Lua).

## API

- `GET /api/health`
- `GET /api/catalog`
- `POST /api/runs`
- `POST /api/debug/sessions`
- `POST /api/debug/sessions/{id}/commands`
- `DELETE /api/debug/sessions/{id}`
- `GET /demo/{endpoint}?store=memory|redis&limit=...&window_ms=...&failure=fail-open|fail-closed`（client key 走 `X-RateLimit-Key`）

`GET /api/catalog` 同时返回每个 scenario 的 `brief`：当前 policy 的展示覆盖、traffic 摘要，以及可机器校验的 core admissions 或 system conditional cases。UI 用它在执行前展示 `POLICY / TRAFFIC / EXPECTED / LESSON`，不会为了预览而隐藏调用 `/api/runs`，也不会在浏览器中复制一套限流算法。

The server accepts only catalogued algorithms and a bounded request timeline. It never compiles or executes user-supplied source code. The debugger launches a fixed controlled harness with the validated selection, then pauses inside the real `server/algorithms.go` implementation.

## Verification

```bash
make test             # four languages + Go server + UI
make test-race        # Go concurrency instrumentation
make build            # production UI embedded in the Go binary
make verify           # start local binary and exercise public HTTP paths
make verify-redis     # starts an ephemeral real local redis-server
make verify-debug     # requires dlv
```

The strongest completion check is the real browser workflow: switch the same burst fixture through four languages, step and rewind its trace, enter Go Debug and inspect a real local variable, then exercise Memory and Redis `/demo` paths including outage policy.

## Boundaries

- Local learning tool, not a production control plane.
- No auth, cloud deployment, multi-user sessions, arbitrary code execution or JavaScript DAP.
- Multi-region is a decision scenario, not a simulated cluster.
- Redis is required for final verification, not for normal startup.
