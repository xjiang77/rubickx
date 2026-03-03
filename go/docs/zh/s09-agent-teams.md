# s09: Agent Teams — Go 逐行解读

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > [ s09 ] s10 > s11 > s12`

> 源码: [`go/s09-agent-teams/main.go`](../../s09-agent-teams/main.go)

---

## s09 vs s08 的核心区别

s08 的后台任务只能跑 shell 命令 — 没有 LLM, 没有决策能力。s09 把后台 goroutine 升级为**完整的 agent**: 每个 teammate 有自己的 agent loop、工具集、收件箱, 能自主执行复杂任务并与其他 agent 通信。

> **"Teammates that can talk to each other."**

对比 s04 subagent:
```
s04 Subagent:   spawn → execute → return summary → 销毁 (一次性)
s09 Teammate:   spawn → WORKING → IDLE → WORKING → ... → SHUTDOWN (持久)
```

---

## 第 53-56 行: 消息类型白名单

```go
var validMsgTypes = map[string]bool{
    "message":                true,
    "broadcast":             true,
    "shutdown_request":      true,    // s10 用
    "shutdown_response":     true,    // s10 用
    "plan_approval_response": true,    // s10 用
}
```

5 种消息类型, 但 s09 只实际用 `message` 和 `broadcast`。其余三种预留给 s10 (权限控制)。

---

## 第 59-67 行: MessageBus 结构

```go
type MessageBus struct {
    dir string        // .team/inbox/ 目录
    mu  sync.Mutex    // 保护文件读写
}
```

核心通信组件。每个 teammate 有一个 JSONL 文件 (`alice.jsonl`), 消息追加写入, 读取时清空 (drain-on-read)。

---

## 第 82-102 行: `Send()` — 发送消息

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

关键操作:
- `os.O_APPEND|os.O_CREATE`: 追加写入, 文件不存在则创建
- 每条消息序列化为一行 JSON + `\n`
- mutex 保证多 goroutine 并发写同一文件时不会交错

---

## 第 104-125 行: `ReadInbox()` — 读取并清空

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

    // Drain: 清空文件
    os.WriteFile(path, []byte(""), 0o644)
    return messages
}
```

**Drain-on-read** 模式: 读取后立即清空。这保证了:
1. 每条消息只被消费一次
2. 不需要游标或偏移量追踪

---

## 第 127-136 行: `Broadcast()` — 广播

```go
func (bus *MessageBus) Broadcast(sender, content string, teammates []string) string {
    count := 0
    for _, name := range teammates {
        if name != sender {                    // 不发给自己
            bus.Send(sender, name, content, "broadcast")
            count++
        }
    }
    return fmt.Sprintf("Broadcast to %d teammates", count)
}
```

遍历所有 teammate, 跳过发送者自己。

---

## 第 139-175 行: TeammateManager

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

`config.json` 是团队名册: 记录所有 teammate 的名字、角色、状态。

---

## 第 199-220 行: `Spawn()` — 创建/重启 teammate

```go
func (tm *TeammateManager) Spawn(name, role, prompt string) string {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    member := tm.findMember(name)
    if member != nil {
        if member.Status != "idle" && member.Status != "shutdown" {
            return fmt.Sprintf("Error: '%s' is currently %s", name, member.Status)
        }
        member.Status = "working"    // 重新激活
    } else {
        tm.config.Members = append(tm.config.Members, memberConfig{...})
    }
    tm.saveConfig()

    go tm.teammateLoop(name, role, prompt)    // ← goroutine

    return fmt.Sprintf("Spawned '%s' (role: %s)", name, role)
}
```

两种路径:
- **新 teammate**: 添加到 `Members`, 启动 goroutine
- **重启**: 已有但处于 `idle`/`shutdown`, 更新状态后重新启动

---

## 第 223-280 行: `teammateLoop()` — teammate 的完整 agent loop

```go
func (tm *TeammateManager) teammateLoop(name, role, prompt string) {
    sysPrompt := fmt.Sprintf(
        "You are '%s', role: %s, at %s. Use send_message to communicate.",
        name, role, workdir,
    )
    tools := teammateTools()         // 6 个工具 (无 spawn/broadcast)
    messages := []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
    }

    for i := 0; i < 50; i++ {       // 安全上限 50 轮
        // 检查收件箱
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
            break    // 任务完成
        }
    }

    // 标记为 idle
    member.Status = "idle"
    tm.saveConfig()
}
```

**s09 最重要的函数**。每个 teammate 是独立的 agent:
1. 有自己的 `messages` (不共享 lead 的对话)
2. 有自己的 system prompt (包含角色身份)
3. 有自己的工具集 (6 个, 比 lead 少 3 个)
4. 每轮检查收件箱, 将新消息注入上下文
5. 完成后状态回 `idle`, 可被再次 spawn

---

## 工具集: Lead vs Teammate

```
Lead (9 tools):                 Teammate (6 tools):
├── bash                        ├── bash
├── read_file                   ├── read_file
├── write_file                  ├── write_file
├── edit_file                   ├── edit_file
├── spawn_teammate  ← 独有     ├── send_message
├── list_teammates  ← 独有     └── read_inbox
├── send_message
├── read_inbox
└── broadcast       ← 独有
```

Teammate 不能 spawn 其他 teammate (防止递归), 不能 broadcast (只有 lead 有全局视角)。

---

## Lead 的 Agent Loop — 收件箱注入

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

和 s08 的 `DrainNotifications()` 类似, 但排空的是 JSONL 收件箱而非内存队列。

---

## `/team` 和 `/inbox` — 本地命令

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

不经过 LLM 的快捷命令, 直接调用本地函数。方便调试。

---

## s08 → s09 变化对照表

| 维度 | s08 | s09 |
|------|-----|-----|
| 新增概念 | 后台 shell 执行 | **持久化 agent 团队 + 消息总线** |
| 工具 | 6 个 | **9 个** (+spawn/send/read_inbox/broadcast/list) |
| 后台单元 | goroutine 跑 shell | **goroutine 跑完整 agent loop** |
| 通信 | notification queue (内存) | **JSONL 收件箱 (磁盘)** |
| 持久化 | 无 | **config.json + inbox/*.jsonl** |
| 生命周期 | 一次性 | **idle → working → idle** |
| 角色分离 | 单一 agent | **lead (9 tools) + teammate (6 tools)** |

---

## 核心架构图

```
main()
  ├── NewMessageBus(".team/inbox/")
  ├── NewTeammateManager(".team/")
  └── REPL loop
        │
        ▼
      agentLoop(messages)
        for {
            inbox = bus.ReadInbox("lead")   ← 检查 lead 收件箱
            resp = LLM(messages, 9 tools)
            │
            ├── spawn_teammate("alice", "coder", "fix bug")
            │     → team.Spawn("alice", ...)
            │     → config.json: alice=working
            │     → go teammateLoop("alice", ...)
            │           │
            │           ├── for i < 50:
            │           │     inbox = bus.ReadInbox("alice")
            │           │     resp = LLM(alice.messages, 6 tools)
            │           │     ├── write_file → runWrite(...)
            │           │     └── send_message("lead", "done!")
            │           │           → lead.jsonl << {"from":"alice",...}
            │           │
            │           └── alice.status = "idle"
            │
            ├── send_message("alice", "new task")
            │     → alice.jsonl << {"from":"lead",...}
            │
            └── broadcast("status update")
                  → alice.jsonl << {...}
                  → bob.jsonl << {...}
        }
```

**s09 的核心洞察**: Agent 团队的关键不是"能启动多个 agent", 而是**通信机制**。JSONL 收件箱提供了:
1. **异步**: 写入者和读取者不需要同时在线
2. **持久化**: 消息不会因为上下文压缩而丢失
3. **解耦**: agent 之间通过文件系统通信, 不需要共享内存

这是从"单 agent 做所有事"到"多 agent 协作"的关键跨越。
