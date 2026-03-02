# s08: Background Tasks — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > [ s08 ] s09 > s10 > s11 > s12`

> Source: [`go/s08/main.go`](../../s08/main.go)

---

## Core Difference: s08 vs s07

s07 persisted tasks to disk, but every tool call is still blocking — `npm install` takes 3 minutes and the agent just waits. s08 solves this: **offload slow operations to the background so the agent can think about the next step**.

> **"Fire and forget — the agent doesn't block while the command runs."**

Python uses `threading.Thread(daemon=True)`, Go uses `go func()` — goroutines are Go's native concurrency primitive, lighter than threads.

```
Python:                         Go:
thread = threading.Thread(      go bg.execute(taskID, command)
    target=self._execute,
    args=(task_id, command),
    daemon=True)
thread.start()
```

---

## Lines 47-60: Data Structures

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

Two structs, two purposes:
- `bgTask`: full record, stored in the `tasks` map, `Result` retains full text (up to 50000 chars)
- `notification`: lightweight notification, queued for draining, `Result` truncated to 500 chars

---

## Lines 63-69: BackgroundManager

```go
type BackgroundManager struct {
    tasks map[string]*bgTask    // Full records of all tasks
    queue []notification         // Notification queue pending drain
    mu    sync.Mutex             // Protects tasks and queue
}
```

Same as the Python version: `tasks` dict + `_notification_queue` + `_lock`. Go uses `sync.Mutex` instead of `threading.Lock()`.

---

## Lines 77-92: `Run()` — Start Background Task

```go
func (bg *BackgroundManager) Run(command string) string {
    taskID := randomID()                     // 8-char hex ID
    bg.mu.Lock()
    bg.tasks[taskID] = &bgTask{
        Status:  "running",
        Command: command,
    }
    bg.mu.Unlock()

    go bg.execute(taskID, command)            // <- Key: goroutine, non-blocking

    return fmt.Sprintf("Background task %s started: %s", taskID, cmdPreview)
}
```

`go bg.execute(...)` — a single keyword launches concurrent execution. The function immediately returns the task ID, and the agent continues working with the ID.

Python version needs 4 lines:
```python
thread = threading.Thread(target=self._execute, args=(task_id, command), daemon=True)
thread.start()
```

Go version needs just 1 line: `go bg.execute(taskID, command)`. Go goroutines have no `daemon` concept — all goroutines terminate automatically when the main process exits.

---

## Lines 95-135: `execute()` — Goroutine Execution Body

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
            status = "completed"   // Non-zero exit code but has output -> still completed
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
        Command: cmdPreview, Result: notifResult,   // Truncated version
    })
}
```

Differences from foreground `runBash()`:
1. **Timeout**: 300 seconds (foreground is 120 seconds) — background tasks are allowed to run longer
2. **Locked writes**: results written to `tasks` + `queue` need mutex protection
3. **Dual storage**: `tasks[taskID].Result` retains full output, `queue` truncates to 500 chars

---

## Lines 138-162: `Check()` — Query Status

```go
func (bg *BackgroundManager) Check(taskID string) string {
    bg.mu.Lock()
    defer bg.mu.Unlock()

    if taskID != "" {
        t := bg.tasks[taskID]
        return fmt.Sprintf("[%s] %s\n%s", t.Status, cmdPreview, result)
    }

    // Empty taskID: list all tasks
    for tid, t := range bg.tasks {
        lines = append(lines, fmt.Sprintf("%s: [%s] %s", tid, t.Status, cmdPreview))
    }
}
```

Two modes:
- With task ID -> return detailed info (status + full output)
- Without -> list summaries of all tasks

---

## Lines 164-173: `DrainNotifications()` — Drain the Queue

```go
func (bg *BackgroundManager) DrainNotifications() []notification {
    bg.mu.Lock()
    defer bg.mu.Unlock()

    notifs := make([]notification, len(bg.queue))
    copy(notifs, bg.queue)       // Copy out
    bg.queue = bg.queue[:0]       // Clear queue (retain underlying array)
    return notifs
}
```

This is s08's **core interface**. The agent loop calls it at the beginning of each iteration, injecting completed background results into the conversation.

`bg.queue[:0]` instead of `nil`: reuses the underlying array, reducing GC pressure.

---

## Lines 176-182: `randomID()` — Generate 8-Digit Hex ID

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

Python version uses `str(uuid.uuid4())[:8]`. Go version is hand-written, same effect: produces IDs like `407037ca`.

---

## Agent Loop — Key Changes in s08

```go
func agentLoop(...) []anthropic.MessageParam {
    for {
        // ---- New: drain notification queue ----
        notifs := bg.DrainNotifications()
        if len(notifs) > 0 && len(messages) > 0 {
            // Format notifications
            notifText := "[bg:abc123] completed: ..."

            // Inject as a user/assistant message pair
            messages = append(messages,
                anthropic.NewUserMessage(
                    anthropic.NewTextBlock("<background-results>\n" + notifText + "\n</background-results>"),
                ),
                anthropic.NewAssistantMessage(
                    anthropic.NewTextBlock("Noted background results."),
                ),
            )
        }

        // ---- Below is identical to s02-s07 ----
        resp, err := client.Messages.New(...)
    }
}
```

This is s08's **only modification** to the agent loop: before each LLM call, drain the notification queue and inject completed background results as a user/assistant message pair.

Why inject as a pair? The Anthropic API requires alternating messages (user/assistant/user/assistant...). If the last message was from the assistant, you can't directly append another user message followed by an assistant message — so a pair maintains the alternation.

---

## Go vs Python: Concurrency Model Comparison

| Dimension | Python (threading) | Go (goroutine) |
|-----------|-------------------|----------------|
| Start | `Thread(target=fn, daemon=True).start()` | `go fn()` |
| Lock | `threading.Lock()` | `sync.Mutex` |
| Memory | ~8MB stack per thread | ~2KB stack per goroutine |
| GIL | Yes (CPU-bound limited) | No |
| Daemon | `daemon=True` (auto-terminate on main exit) | Default behavior |

Go's goroutines are naturally suited for this "fire-and-forget" scenario: low startup cost, no thread pool needed, and no GIL limitation.

---

## s07 -> s08 Change Comparison

| Dimension | s07 | s08 |
|-----------|-----|-----|
| New concept | Persistent task graph | **Background execution + notification queue** |
| Tools | 8 (including task_*) | **6** (including background_run, check_background) |
| Execution | Blocking only | **Blocking + background goroutines** |
| Notification | None | **Per-iteration drain notification queue** |
| Agent Loop | Pure dispatch | **New: drain + inject notifications** |
| Concurrency safety | sync.Mutex (TaskManager) | **sync.Mutex (BackgroundManager)** |

---

## Core Architecture Diagram

```
agentLoop() {
    for {
        +- DrainNotifications() ----------------------+
        |  bg.mu.Lock()                               |
        |  notifs = copy(bg.queue)                    |
        |  bg.queue = bg.queue[:0]                    |
        |  bg.mu.Unlock()                             |
        +---------------------------------------------+
                |
                v (if notifications exist)
        messages += user: <background-results>
                    [bg:abc123] completed: ...
                    </background-results>
        messages += assistant: Noted background results.
                |
                v
        resp = LLM(messages, tools)
                |
                +-- tool: background_run("sleep 10")
                |   -> bg.Run("sleep 10")
                |   -> go bg.execute(taskID, "sleep 10")   <- goroutine
                |   -> Immediately returns "Background task abc123 started"
                |
                +-- tool: bash("echo hello")
                |   -> runBash("echo hello")                <- blocking
                |
                +-- tool: check_background("abc123")
                    -> bg.Check("abc123")
                    -> "[running] sleep 10\n(running)"

        Meanwhile in background:
        goroutine: bg.execute("abc123", "sleep 10")
            -> exec.CommandContext("sh", "-c", "sleep 10")
            -> Completes after 10 seconds
            -> bg.mu.Lock()
            -> bg.tasks["abc123"] = "completed"
            -> bg.queue = append(queue, notification{...})
            -> bg.mu.Unlock()
            -> Will be drained before next LLM call
    }
}
```

**s08's core insight**: The agent loop itself stays single-threaded; only subprocess I/O is parallelized. The notification queue bridges the single thread and multiple goroutines: background writes, main loop drains. This ensures conversation state consistency — the LLM always sees an ordered message sequence.
