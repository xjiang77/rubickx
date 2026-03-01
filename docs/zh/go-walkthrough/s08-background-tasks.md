# s08: Background Tasks — Go 逐行解读

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > [ s08 ] s09 > s10 > s11 > s12`

> 源码: [`go/s08/main.go`](../../go/s08/main.go)

---

## s08 vs s07 的核心区别

s07 把任务持久化到磁盘, 但每个工具调用仍然是阻塞的 — `npm install` 跑 3 分钟, agent 就干等 3 分钟。s08 解决这个问题: **把慢操作丢到后台, agent 继续想下一步**。

> **"Fire and forget — the agent doesn't block while the command runs."**

Python 用 `threading.Thread(daemon=True)`, Go 用 `go func()` — goroutine 是 Go 的原生并发原语, 比线程更轻量。

```
Python:                         Go:
thread = threading.Thread(      go bg.execute(taskID, command)
    target=self._execute,
    args=(task_id, command),
    daemon=True)
thread.start()
```

---

## 第 47-60 行: 数据结构

```go
type bgTask struct {
    Status  string `json:"status"`  // running | completed | timeout | error
    Result  string `json:"result"`
    Command string `json:"command"`
}

type notification struct {
    TaskID  string `json:"task_id"`
    Status  string `json:"status"`
    Command string `json:"command"`
    Result  string `json:"result"`
}
```

两个结构体, 两个用途:
- `bgTask`: 完整记录, 存在 `tasks` map 中, `Result` 保留全文 (最多 50000 字符)
- `notification`: 轻量通知, 排入队列, `Result` 截断到 500 字符

---

## 第 63-69 行: BackgroundManager

```go
type BackgroundManager struct {
    tasks map[string]*bgTask    // 所有任务的完整记录
    queue []notification         // 待排空的通知队列
    mu    sync.Mutex             // 保护 tasks 和 queue
}
```

和 Python 版一样: `tasks` 字典 + `_notification_queue` + `_lock`。Go 用 `sync.Mutex` 替代 `threading.Lock()`。

---

## 第 77-92 行: `Run()` — 启动后台任务

```go
func (bg *BackgroundManager) Run(command string) string {
    taskID := randomID()                     // 8 字符 hex ID
    bg.mu.Lock()
    bg.tasks[taskID] = &bgTask{
        Status:  "running",
        Command: command,
    }
    bg.mu.Unlock()

    go bg.execute(taskID, command)            // ← 关键: goroutine, 不阻塞

    return fmt.Sprintf("Background task %s started: %s", taskID, cmdPreview)
}
```

`go bg.execute(...)` — 一个关键字就启动了并发执行。函数立即返回 task ID, agent 拿到 ID 后继续工作。

Python 版需要 4 行:
```python
thread = threading.Thread(target=self._execute, args=(task_id, command), daemon=True)
thread.start()
```

Go 版只需 1 行: `go bg.execute(taskID, command)`。Go 的 goroutine 没有 `daemon` 概念 — 主进程退出时所有 goroutine 自动终止。

---

## 第 95-135 行: `execute()` — goroutine 执行体

```go
func (bg *BackgroundManager) execute(taskID, command string) {
    ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
    defer cancel()

    cmd := exec.CommandContext(ctx, "sh", "-c", command)
    cmd.Dir = workdir
    output, err := cmd.CombinedOutput()

    var status, result string
    if ctx.Err() == context.DeadlineExceeded {
        status = "timeout"
    } else if err != nil {
        out := strings.TrimSpace(string(output))
        if out != "" {
            status = "completed"   // 非零退出码但有输出 → 仍算 completed
            result = out
        } else {
            status = "error"
        }
    } else {
        status = "completed"
        result = strings.TrimSpace(string(output))
    }

    bg.mu.Lock()
    defer bg.mu.Unlock()

    bg.tasks[taskID].Status = status
    bg.tasks[taskID].Result = result

    bg.queue = append(bg.queue, notification{
        TaskID: taskID, Status: status,
        Command: cmdPreview, Result: notifResult,   // 截断版
    })
}
```

和前台 `runBash()` 的区别:
1. **超时**: 300 秒 (前台 120 秒) — 后台允许跑更久
2. **加锁写入**: 结果写入 `tasks` + `queue`, 需要 mutex 保护
3. **双重存储**: `tasks[taskID].Result` 保留完整输出, `queue` 中截断到 500 字符

---

## 第 138-162 行: `Check()` — 查询状态

```go
func (bg *BackgroundManager) Check(taskID string) string {
    bg.mu.Lock()
    defer bg.mu.Unlock()

    if taskID != "" {
        t := bg.tasks[taskID]
        return fmt.Sprintf("[%s] %s\n%s", t.Status, cmdPreview, result)
    }

    // taskID 为空: 列出所有任务
    for tid, t := range bg.tasks {
        lines = append(lines, fmt.Sprintf("%s: [%s] %s", tid, t.Status, cmdPreview))
    }
}
```

两种模式:
- 指定 task ID → 返回详细信息 (状态 + 完整输出)
- 不指定 → 列出所有任务的摘要

---

## 第 164-173 行: `DrainNotifications()` — 排空队列

```go
func (bg *BackgroundManager) DrainNotifications() []notification {
    bg.mu.Lock()
    defer bg.mu.Unlock()

    notifs := make([]notification, len(bg.queue))
    copy(notifs, bg.queue)       // 复制出来
    bg.queue = bg.queue[:0]       // 清空队列 (保留底层数组)
    return notifs
}
```

这是 s08 的**核心接口**。Agent loop 每轮开头调用它, 把后台完成的结果注入对话。

`bg.queue[:0]` 而非 `nil`: 重用底层数组, 减少 GC 压力。

---

## 第 176-182 行: `randomID()` — 生成 8 位 hex ID

```go
func randomID() string {
    const chars = "0123456789abcdef"
    b := make([]byte, 8)
    for i := range b {
        b[i] = chars[rand.Intn(len(chars))]
    }
    return string(b)
}
```

Python 版用 `str(uuid.uuid4())[:8]`。Go 版手写, 效果一样: 产生类似 `407037ca` 的 ID。

---

## Agent Loop — s08 的关键变化

```go
func agentLoop(...) []anthropic.MessageParam {
    for {
        // ---- 新增: 排空通知队列 ----
        notifs := bg.DrainNotifications()
        if len(notifs) > 0 && len(messages) > 0 {
            // 格式化通知
            notifText := "[bg:abc123] completed: ..."

            // 注入为一对 user/assistant 消息
            messages = append(messages,
                anthropic.NewUserMessage(
                    anthropic.NewTextBlock("<background-results>\n" + notifText + "\n</background-results>"),
                ),
                anthropic.NewAssistantMessage(
                    anthropic.NewTextBlock("Noted background results."),
                ),
            )
        }

        // ---- 以下和 s02-s07 相同 ----
        resp, err := client.Messages.New(...)
    }
}
```

这是 s08 对 agent loop 的**唯一修改**: 在每次 LLM 调用前, 排空通知队列, 将后台完成的结果注入为一对 user/assistant 消息。

为什么要成对注入? Anthropic API 要求消息交替 (user/assistant/user/assistant...)。如果上一条是 assistant 消息, 不能直接追加另一条 user 消息后再追加 assistant 消息, 所以用一对来保持交替。

---

## Go vs Python: 并发模型对比

| 维度 | Python (threading) | Go (goroutine) |
|------|-------------------|----------------|
| 启动 | `Thread(target=fn, daemon=True).start()` | `go fn()` |
| 锁 | `threading.Lock()` | `sync.Mutex` |
| 内存 | 每线程 ~8MB 栈 | 每 goroutine ~2KB 栈 |
| GIL | 有 (CPU 密集型受限) | 无 |
| 守护 | `daemon=True` (主进程退出时自动终止) | 默认行为 |

Go 的 goroutine 天然适合这种"fire-and-forget"场景: 启动成本低, 不需要线程池, 也不受 GIL 限制。

---

## s07 → s08 变化对照表

| 维度 | s07 | s08 |
|------|-----|-----|
| 新增概念 | 持久化任务图 | **后台执行 + 通知队列** |
| 工具 | 8 个 (含 task_*) | **6 个** (含 background_run, check_background) |
| 执行方式 | 仅阻塞 | **阻塞 + 后台 goroutine** |
| 通知机制 | 无 | **每轮排空的 notification queue** |
| Agent Loop | 纯 dispatch | **新增: drain + inject 通知** |
| 并发安全 | sync.Mutex (TaskManager) | **sync.Mutex (BackgroundManager)** |

---

## 核心架构图

```
agentLoop() {
    for {
        ┌─ DrainNotifications() ──────────────┐
        │  bg.mu.Lock()                        │
        │  notifs = copy(bg.queue)             │
        │  bg.queue = bg.queue[:0]             │
        │  bg.mu.Unlock()                      │
        └──────────────────────────────────────┘
                │
                ▼ (如果有通知)
        messages += user: <background-results>
                    [bg:abc123] completed: ...
                    </background-results>
        messages += assistant: Noted background results.
                │
                ▼
        resp = LLM(messages, tools)
                │
                ├── tool: background_run("sleep 10")
                │   → bg.Run("sleep 10")
                │   → go bg.execute(taskID, "sleep 10")   ← goroutine
                │   → 立即返回 "Background task abc123 started"
                │
                ├── tool: bash("echo hello")
                │   → runBash("echo hello")                ← 阻塞
                │
                └── tool: check_background("abc123")
                    → bg.Check("abc123")
                    → "[running] sleep 10\n(running)"

        同时在后台:
        goroutine: bg.execute("abc123", "sleep 10")
            → exec.CommandContext("sh", "-c", "sleep 10")
            → 10 秒后完成
            → bg.mu.Lock()
            → bg.tasks["abc123"] = "completed"
            → bg.queue = append(queue, notification{...})
            → bg.mu.Unlock()
            → 下次 LLM 调用前会被 drain 出来
    }
}
```

**s08 的核心洞察**: Agent loop 本身保持单线程, 只有子进程 I/O 被并行化。通知队列是单线程和多 goroutine 之间的桥梁: 后台写入, 主循环排空。这样保证了对话状态的一致性 — LLM 永远看到的是一个有序的消息序列。
