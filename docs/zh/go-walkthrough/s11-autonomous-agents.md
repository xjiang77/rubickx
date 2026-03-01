# s11: Autonomous Agents — Go 逐行解读

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > [ s11 ] s12`

> 源码: [`go/s11/main.go`](../../go/s11/main.go)

---

## s11 vs s10 的核心区别

s10 的 teammate 做完分配的任务就停了, 领导要一个一个手动分配。s11 实现**自治**: teammate 做完活自己去看板找下一个。

> **"The agent finds work itself."**

```
s10 Teammate:  spawn → WORK → idle (等领导指派)
s11 Teammate:  spawn → WORK → IDLE (轮询) → auto-claim → WORK → ... → timeout → shutdown
```

三个新机制:
1. **Idle 轮询**: 每 5 秒检查收件箱 + 任务看板
2. **Auto-claim**: 发现未认领的 pending 任务, 自动抢下来
3. **Identity re-injection**: 上下文压缩后重新注入身份信息

---

## 第 54-55 行: 轮询常量

```go
const (
    pollInterval = 5  // 每 5 秒轮询一次
    idleTimeout  = 60 // 60 秒无活可干 → 自动关机
)
```

`60 / 5 = 12` 次轮询机会。12 次都没找到活 → shutdown。

---

## `scanUnclaimedTasks()` — 扫描任务看板

```go
func scanUnclaimedTasks() []map[string]interface{} {
    // 扫描 .tasks/task_*.json
    for _, name := range names {
        status, _ := task["status"].(string)
        owner, _ := task["owner"].(string)
        blockedBy, _ := task["blockedBy"].([]interface{})

        if status == "pending" && owner == "" && len(blockedBy) == 0 {
            unclaimed = append(unclaimed, task)
        }
    }
    return unclaimed
}
```

三个条件必须同时满足:
- `status == "pending"`: 未开始
- `owner == ""`: 没人认领
- `len(blockedBy) == 0`: 没有未完成的前置依赖

---

## `claimTask()` — 原子认领

```go
func claimTask(taskID int, owner string) string {
    claimMu.Lock()
    defer claimMu.Unlock()

    task["owner"] = owner
    task["status"] = "in_progress"
    os.WriteFile(path, out, 0o644)
}
```

`claimMu` 是独立的锁, 防止两个 teammate 同时抢同一个任务。

---

## `makeIdentityMessages()` — 身份重注入

```go
func makeIdentityMessages(name, role, teamName string) (anthropic.MessageParam, anthropic.MessageParam) {
    return anthropic.NewUserMessage(
            anthropic.NewTextBlock("<identity>You are 'name', role: ..., team: ...</identity>"),
        ),
        anthropic.NewAssistantMessage(
            anthropic.NewTextBlock("I am name. Continuing."),
        )
}
```

当 `len(messages) <= 3` 时 (上下文被压缩过), 插入到开头恢复身份。

---

## `loop()` — 自治生命周期 (核心函数)

```go
func (tm *TeammateManager) loop(name, role, prompt string) {
    messages := [...]

    for { // ← 外层无限循环: WORK → IDLE → WORK → ...

        // ======== WORK PHASE ========
        for round := 0; round < 50; round++ {
            inbox → shutdown_request? → return
            resp = LLM(messages, 10 tools)
            if "idle" tool called → break to IDLE
            if stop_reason != tool_use → break to IDLE
            execute tools
        }

        // ======== IDLE PHASE ========
        tm.setStatus(name, "idle")
        for p := 0; p < 12; p++ {   // 60s / 5s
            time.Sleep(5s)
            inbox? → resume WORK
            unclaimed tasks? → claim → identity re-inject → resume WORK
        }
        // 12 次都没活 → shutdown
    }
}
```

关键转换:
1. **WORK → IDLE**: `idle` 工具触发, 或 LLM 自然停止
2. **IDLE → WORK**: 收件箱有消息, 或抢到任务
3. **IDLE → SHUTDOWN**: 12 次轮询都没找到活
4. **任何时刻**: shutdown_request → 立即退出

---

## `idle` 工具 — 信号工具

```go
if block.Name == "idle" {
    idleRequested = true
    output = "Entering idle phase. Will poll for new tasks."
}
```

不执行任何操作, 只告诉 loop "我没活了"。LLM 主动调用它来进入空闲阶段。

---

## s10 → s11 变化对照表

| 维度 | s10 | s11 |
|------|-----|-----|
| 新增概念 | 请求-响应协议 | **自治生命周期 + 任务认领** |
| 工具 | Lead 12 / Teammate 8 | **Lead 14 / Teammate 10** (+idle, +claim_task) |
| 生命周期 | 做完就停 | **WORK → IDLE → poll → auto-claim → WORK** |
| 任务分配 | 领导指派 | **自组织: 自动扫描 + 认领** |
| 身份保持 | 仅 system prompt | **+ 压缩后 identity re-injection** |
| 超时 | 无 | **60 秒无活 → 自动关机** |
| 本地命令 | /team, /inbox | **+ /tasks** |

---

## 核心架构图

```
loop(name, role, prompt) {
    messages = [user: prompt]

    for {  // 无限循环

        ┌─── WORK PHASE ────────────────────────┐
        │ for round < 50:                        │
        │   inbox → shutdown_request? → return    │
        │   resp = LLM(messages, 10 tools)        │
        │   if "idle" tool → break                │
        │   if stop_reason != tool_use → break    │
        │   execute tools, append results         │
        └────────────────────────────────────────┘
                │
                ▼
        ┌─── IDLE PHASE ────────────────────────┐
        │ status = "idle"                        │
        │ for p < 12:  (60s / 5s)               │
        │   sleep(5s)                            │
        │   inbox? → inject, resume=true          │
        │   unclaimed tasks? → claim, resume=true │
        │   identity re-inject if compressed      │
        └────────────────────────────────────────┘
                │
                ├── resume=true  → "working" → WORK
                └── resume=false → "shutdown" → return
    }
}
```

**s11 的核心洞察**: 自治不只是"能自动做事", 更重要的是**能自己找事做**。Idle polling + task board scanning 实现了工作窃取模式 (work-stealing): 空闲的 agent 主动从共享队列中拿任务。这是从"命令式协作"到"自组织协作"的跨越。
