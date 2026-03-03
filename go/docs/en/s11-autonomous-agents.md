# s11: Autonomous Agents — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > [ s11 ] s12`

> Source: [`go/s11-autonomous-agents/main.go`](../../s11-autonomous-agents/main.go)

---

## Core Difference: s11 vs s10

s10's teammates stop after completing their assigned task, and the lead must assign work one by one. s11 implements **autonomy**: teammates find the next task from the board on their own when they finish.

> **"The agent finds work itself."**

```
s10 Teammate:  spawn -> WORK -> idle (wait for lead assignment)
s11 Teammate:  spawn -> WORK -> IDLE (polling) -> auto-claim -> WORK -> ... -> timeout -> shutdown
```

Three new mechanisms:
1. **Idle polling**: check inbox + task board every 5 seconds
2. **Auto-claim**: discover unclaimed pending tasks, automatically grab them
3. **Identity re-injection**: re-inject identity information after context compression

---

## Lines 54-55: Polling Constants

```go
const (
    pollInterval = 5  // Poll every 5 seconds
    idleTimeout  = 60 // 60 seconds with nothing to do -> auto-shutdown
)
```

`60 / 5 = 12` polling opportunities. 12 polls with no work found -> shutdown.

---

## `scanUnclaimedTasks()` — Scan the Task Board

```go
func scanUnclaimedTasks() []map[string]interface{} {
    // Scan .tasks/task_*.json
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

Three conditions must all be met:
- `status == "pending"`: not started
- `owner == ""`: nobody claimed it
- `len(blockedBy) == 0`: no unfinished upstream dependencies

---

## `claimTask()` — Atomic Claim

```go
func claimTask(taskID int, owner string) string {
    claimMu.Lock()
    defer claimMu.Unlock()

    task["owner"] = owner
    task["status"] = "in_progress"
    os.WriteFile(path, out, 0o644)
}
```

`claimMu` is a separate lock preventing two teammates from claiming the same task simultaneously.

---

## `makeIdentityMessages()` — Identity Re-injection

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

When `len(messages) <= 3` (context has been compressed), inserted at the beginning to restore identity.

---

## `loop()` — Autonomous Lifecycle (Core Function)

```go
func (tm *TeammateManager) loop(name, role, prompt string) {
    messages := [...]

    for { // <- Outer infinite loop: WORK -> IDLE -> WORK -> ...

        // ======== WORK PHASE ========
        for round := 0; round < 50; round++ {
            inbox -> shutdown_request? -> return
            resp = LLM(messages, 10 tools)
            if "idle" tool called -> break to IDLE
            if stop_reason != tool_use -> break to IDLE
            execute tools
        }

        // ======== IDLE PHASE ========
        tm.setStatus(name, "idle")
        for p := 0; p < 12; p++ {   // 60s / 5s
            time.Sleep(5s)
            inbox? -> resume WORK
            unclaimed tasks? -> claim -> identity re-inject -> resume WORK
        }
        // 12 polls with no work -> shutdown
    }
}
```

Key transitions:
1. **WORK -> IDLE**: `idle` tool triggered, or LLM naturally stops
2. **IDLE -> WORK**: inbox has messages, or task claimed
3. **IDLE -> SHUTDOWN**: 12 polls all found nothing
4. **Any moment**: shutdown_request -> exit immediately

---

## `idle` Tool — Signal Tool

```go
if block.Name == "idle" {
    idleRequested = true
    output = "Entering idle phase. Will poll for new tasks."
}
```

Doesn't execute any operation — just tells the loop "I have nothing to do." The LLM proactively calls it to enter the idle phase.

---

## s10 -> s11 Change Comparison

| Dimension | s10 | s11 |
|-----------|-----|-----|
| New concept | Request-response protocols | **Autonomous lifecycle + task claiming** |
| Tools | Lead 12 / Teammate 8 | **Lead 14 / Teammate 10** (+idle, +claim_task) |
| Lifecycle | Stop when done | **WORK -> IDLE -> poll -> auto-claim -> WORK** |
| Task assignment | Lead assigns | **Self-organizing: auto-scan + claim** |
| Identity preservation | System prompt only | **+ post-compression identity re-injection** |
| Timeout | None | **60 seconds idle -> auto-shutdown** |
| Local commands | /team, /inbox | **+ /tasks** |

---

## Core Architecture Diagram

```
loop(name, role, prompt) {
    messages = [user: prompt]

    for {  // Infinite loop

        +--- WORK PHASE -----------------------+
        | for round < 50:                       |
        |   inbox -> shutdown_request? -> return |
        |   resp = LLM(messages, 10 tools)      |
        |   if "idle" tool -> break             |
        |   if stop_reason != tool_use -> break |
        |   execute tools, append results       |
        +---------------------------------------+
                |
                v
        +--- IDLE PHASE -----------------------+
        | status = "idle"                       |
        | for p < 12:  (60s / 5s)              |
        |   sleep(5s)                           |
        |   inbox? -> inject, resume=true       |
        |   unclaimed tasks? -> claim, resume=true |
        |   identity re-inject if compressed    |
        +---------------------------------------+
                |
                +-- resume=true  -> "working" -> WORK
                +-- resume=false -> "shutdown" -> return
    }
}
```

**s11's core insight**: Autonomy isn't just "being able to do things automatically" — more importantly, it's **being able to find things to do**. Idle polling + task board scanning implements a work-stealing pattern: idle agents proactively grab tasks from a shared queue. This is the leap from "imperative collaboration" to "self-organizing collaboration."
