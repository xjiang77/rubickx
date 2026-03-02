# s09: Agent Teams — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > [ s09 ] s10 > s11 > s12`

> Source: [`go/s09/main.go`](../../s09/main.go)

---

## Core Difference: s09 vs s08

s08's background tasks can only run shell commands — no LLM, no decision-making ability. s09 upgrades background goroutines to **full agents**: each teammate has its own agent loop, tool set, inbox, and can autonomously execute complex tasks and communicate with other agents.

> **"Teammates that can talk to each other."**

Comparison with s04 subagent:
```
s04 Subagent:   spawn -> execute -> return summary -> discard (one-shot)
s09 Teammate:   spawn -> WORKING -> IDLE -> WORKING -> ... -> SHUTDOWN (persistent)
```

---

## Lines 53-56: Message Type Allowlist

```go
var validMsgTypes = map[string]bool{
    "message":                true,
    "broadcast":             true,
    "shutdown_request":      true,    // Used by s10
    "shutdown_response":     true,    // Used by s10
    "plan_approval_response": true,    // Used by s10
}
```

5 message types, but s09 only uses `message` and `broadcast`. The other three are reserved for s10 (permission control).

---

## Lines 59-67: MessageBus Structure

```go
type MessageBus struct {
    dir string        // .team/inbox/ directory
    mu  sync.Mutex    // Protects file reads/writes
}
```

Core communication component. Each teammate has a JSONL file (`alice.jsonl`); messages are appended on write, cleared on read (drain-on-read).

---

## Lines 82-102: `Send()` — Send Message

```go
func (bus *MessageBus) Send(sender, to, content, msgType string) string {
    if !validMsgTypes[msgType] {
        return fmt.Sprintf("Error: Invalid type '%s'", msgType)
    }
    msg := inboxMsg{
        Type:      msgType,
        From:      sender,
        Content:   content,
        Timestamp: float64(time.Now().UnixMilli()) / 1000.0,
    }
    data, _ := json.Marshal(msg)

    bus.mu.Lock()
    defer bus.mu.Unlock()

    path := filepath.Join(bus.dir, to+".jsonl")
    f, _ := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
    defer f.Close()
    f.Write(append(data, '\n'))
    return fmt.Sprintf("Sent %s to %s", msgType, to)
}
```

Key operations:
- `os.O_APPEND|os.O_CREATE`: append write, create file if it doesn't exist
- Each message serialized as one line of JSON + `\n`
- Mutex ensures concurrent writes from multiple goroutines to the same file don't interleave

---

## Lines 104-125: `ReadInbox()` — Read and Clear

```go
func (bus *MessageBus) ReadInbox(name string) []inboxMsg {
    bus.mu.Lock()
    defer bus.mu.Unlock()

    path := filepath.Join(bus.dir, name+".jsonl")
    data, _ := os.ReadFile(path)

    var messages []inboxMsg
    for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
        if line == "" { continue }
        var msg inboxMsg
        json.Unmarshal([]byte(line), &msg)
        messages = append(messages, msg)
    }

    // Drain: clear the file
    os.WriteFile(path, []byte(""), 0o644)
    return messages
}
```

**Drain-on-read** pattern: cleared immediately after reading. This guarantees:
1. Each message is consumed exactly once
2. No cursor or offset tracking needed

---

## Lines 127-136: `Broadcast()` — Broadcast

```go
func (bus *MessageBus) Broadcast(sender, content string, teammates []string) string {
    count := 0
    for _, name := range teammates {
        if name != sender {                    // Don't send to self
            bus.Send(sender, name, content, "broadcast")
            count++
        }
    }
    return fmt.Sprintf("Broadcast to %d teammates", count)
}
```

Iterates all teammates, skipping the sender.

---

## Lines 139-175: TeammateManager

```go
type memberConfig struct {
    Name   string `json:"name"`
    Role   string `json:"role"`
    Status string `json:"status"`  // idle | working | shutdown
}

type teamConfig struct {
    TeamName string         `json:"team_name"`
    Members  []memberConfig `json:"members"`
}

type TeammateManager struct {
    dir        string
    configPath string         // .team/config.json
    config     teamConfig
    mu         sync.Mutex
}
```

`config.json` is the team roster: records all teammates' names, roles, and statuses.

---

## Lines 199-220: `Spawn()` — Create/Restart Teammate

```go
func (tm *TeammateManager) Spawn(name, role, prompt string) string {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    member := tm.findMember(name)
    if member != nil {
        if member.Status != "idle" && member.Status != "shutdown" {
            return fmt.Sprintf("Error: '%s' is currently %s", name, member.Status)
        }
        member.Status = "working"    // Reactivate
    } else {
        tm.config.Members = append(tm.config.Members, memberConfig{...})
    }
    tm.saveConfig()

    go tm.teammateLoop(name, role, prompt)    // <- goroutine

    return fmt.Sprintf("Spawned '%s' (role: %s)", name, role)
}
```

Two paths:
- **New teammate**: added to `Members`, goroutine started
- **Restart**: existing but in `idle`/`shutdown`, status updated and relaunched

---

## Lines 223-280: `teammateLoop()` — Teammate's Full Agent Loop

```go
func (tm *TeammateManager) teammateLoop(name, role, prompt string) {
    sysPrompt := fmt.Sprintf(
        "You are '%s', role: %s, at %s. Use send_message to communicate.",
        name, role, workdir,
    )
    tools := teammateTools()         // 6 tools (no spawn/broadcast)
    messages := []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
    }

    for i := 0; i < 50; i++ {       // Safety limit 50 rounds
        // Check inbox
        inbox := bus.ReadInbox(name)
        if len(inbox) > 0 {
            messages = append(messages,
                anthropic.NewUserMessage(anthropic.NewTextBlock("<inbox>...</inbox>")),
                anthropic.NewAssistantMessage(anthropic.NewTextBlock("Noted inbox messages.")),
            )
        }

        resp, _ := apiClient.Messages.New(...)
        // ... execute tools via execTool() ...

        if resp.StopReason != "tool_use" {
            break    // Task complete
        }
    }

    // Mark as idle
    member.Status = "idle"
    tm.saveConfig()
}
```

**The most important function in s09**. Each teammate is an independent agent:
1. Has its own `messages` (doesn't share the lead's conversation)
2. Has its own system prompt (includes role identity)
3. Has its own tool set (6 tools, 3 fewer than lead)
4. Checks inbox each round, injecting new messages into context
5. Goes back to `idle` after completion, can be spawned again

---

## Tool Sets: Lead vs Teammate

```
Lead (9 tools):                 Teammate (6 tools):
+-- bash                        +-- bash
+-- read_file                   +-- read_file
+-- write_file                  +-- write_file
+-- edit_file                   +-- edit_file
+-- spawn_teammate  <- exclusive +-- send_message
+-- list_teammates  <- exclusive +-- read_inbox
+-- send_message
+-- read_inbox
+-- broadcast       <- exclusive
```

Teammates can't spawn other teammates (prevents recursion) and can't broadcast (only the lead has a global view).

---

## Lead's Agent Loop — Inbox Injection

```go
func agentLoop(messages []anthropic.MessageParam) []anthropic.MessageParam {
    for {
        inbox := bus.ReadInbox("lead")
        if len(inbox) > 0 {
            messages = append(messages,
                anthropic.NewUserMessage(anthropic.NewTextBlock("<inbox>...</inbox>")),
                anthropic.NewAssistantMessage(anthropic.NewTextBlock("Noted inbox messages.")),
            )
        }
        resp, _ = apiClient.Messages.New(...)
        // ... dispatch via 9-tool map ...
    }
}
```

Similar to s08's `DrainNotifications()`, but draining JSONL inbox files instead of an in-memory queue.

---

## `/team` and `/inbox` — Local Commands

```go
if query == "/team" {
    fmt.Println(team.ListAll())
    continue
}
if query == "/inbox" {
    msgs := bus.ReadInbox("lead")
    fmt.Println(string(json.MarshalIndent(msgs)))
    continue
}
```

Shortcut commands that bypass the LLM, calling local functions directly. Useful for debugging.

---

## s08 -> s09 Change Comparison

| Dimension | s08 | s09 |
|-----------|-----|-----|
| New concept | Background shell execution | **Persistent agent team + message bus** |
| Tools | 6 | **9** (+spawn/send/read_inbox/broadcast/list) |
| Background unit | Goroutine runs shell | **Goroutine runs full agent loop** |
| Communication | notification queue (memory) | **JSONL inbox (disk)** |
| Persistence | None | **config.json + inbox/*.jsonl** |
| Lifecycle | One-shot | **idle -> working -> idle** |
| Role separation | Single agent | **lead (9 tools) + teammate (6 tools)** |

---

## Core Architecture Diagram

```
main()
  +-- NewMessageBus(".team/inbox/")
  +-- NewTeammateManager(".team/")
  +-- REPL loop
        |
        v
      agentLoop(messages)
        for {
            inbox = bus.ReadInbox("lead")   <- Check lead inbox
            resp = LLM(messages, 9 tools)
            |
            +-- spawn_teammate("alice", "coder", "fix bug")
            |     -> team.Spawn("alice", ...)
            |     -> config.json: alice=working
            |     -> go teammateLoop("alice", ...)
            |           |
            |           +-- for i < 50:
            |           |     inbox = bus.ReadInbox("alice")
            |           |     resp = LLM(alice.messages, 6 tools)
            |           |     +-- write_file -> runWrite(...)
            |           |     +-- send_message("lead", "done!")
            |           |           -> lead.jsonl << {"from":"alice",...}
            |           |
            |           +-- alice.status = "idle"
            |
            +-- send_message("alice", "new task")
            |     -> alice.jsonl << {"from":"lead",...}
            |
            +-- broadcast("status update")
                  -> alice.jsonl << {...}
                  -> bob.jsonl << {...}
        }
```

**s09's core insight**: The key to agent teams isn't "being able to start multiple agents" — it's the **communication mechanism**. The JSONL inbox provides:
1. **Asynchrony**: writer and reader don't need to be online simultaneously
2. **Persistence**: messages aren't lost due to context compression
3. **Decoupling**: agents communicate through the filesystem, no shared memory needed

This is the critical leap from "one agent does everything" to "multi-agent collaboration."
