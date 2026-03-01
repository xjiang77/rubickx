# s06: Context Compact — Go 逐行解读

`s01 > s02 > s03 > s04 > s05 > [ s06 ] | s07 > s08 > s09 > s10 > s11 > s12`

> 源码: [`go/s06/main.go`](../../go/s06/main.go)

---

## s06 vs s05 的核心区别

s05 解决了"怎么给模型注入知识"。s06 解决相反的问题: **怎么让模型遗忘**。

> **"The agent can forget strategically and keep working forever."**

核心问题: agent loop 每轮都往 messages 里追加内容, 上下文无限增长。最终要么撞到 token 上限, 要么重要信息被淹没。解决方案: **三层压缩管线**。

```
Layer 1: microCompact   — 每轮静默执行, 用占位符替换旧 tool_result
Layer 2: autoCompact    — token > 50000 时触发, LLM 总结 + 替换全部历史
Layer 3: compact 工具   — 模型主动调用, 立即执行 Layer 2
```

---

## 架构图

```
每轮循环:
+------------------+
| Tool call result |
+------------------+
        |
        v
[Layer 1: microCompact]        (静默, 每轮执行)
  把最近 3 个之前的 tool_result
  替换为 "[Previous: used {tool_name}]"
        |
        v
[Check: tokens > 50000?]
   |               |
   no              yes
   |               |
   v               v
continue    [Layer 2: autoCompact]
              保存完整 transcript 到 .transcripts/
              让 LLM 总结对话
              用 [summary] 替换全部 messages
                    |
                    v
            [Layer 3: compact 工具]
              模型调用 compact → 立即执行 Layer 2
```

---

## 第 47-55 行: 常量

```go
const (
    threshold  = 50000  // token 估算阈值, 超过就触发 auto_compact
    keepRecent = 3      // 保留最近 3 个 tool_result 不压缩
)
```

---

## 第 136-140 行: `estimateTokens()` — 粗略 token 估算

```go
func estimateTokens(messages []anthropic.MessageParam) int {
    data, _ := json.Marshal(messages)
    return len(data) / 4
}
```

~4 个字符 ≈ 1 个 token, 粗略但实用。

---

## 第 142-219 行: `microCompact()` — Layer 1

```go
func microCompact(messages []anthropic.MessageParam) {
    // 1. 收集所有 tool_result 的位置 (msgIdx, partIdx)
    var locs []toolResultLoc
    for i, msg := range messages {
        if msg.Role != "user" { continue }
        for j, part := range msg.Content {
            if part.OfToolResult != nil {
                locs = append(locs, toolResultLoc{i, j})
            }
        }
    }
    if len(locs) <= keepRecent { return }

    // 2. 构建 tool_use_id → tool_name 映射
    toolNameMap := map[string]string{}
    for _, msg := range messages {
        if msg.Role != "assistant" { continue }
        for _, part := range msg.Content {
            if part.OfToolUse != nil {
                toolNameMap[part.OfToolUse.ID] = part.OfToolUse.Name
            }
        }
    }

    // 3. 替换旧结果 (保留最后 keepRecent 个)
    toClear := locs[:len(locs)-keepRecent]
    for _, loc := range toClear {
        tr := messages[loc.msgIdx].Content[loc.partIdx].OfToolResult
        // 只压缩超过 100 字符的结果
        if contentLen > 100 {
            toolName := toolNameMap[tr.ToolUseID]
            tr.Content = []anthropic.ToolResultBlockParamContentUnion{
                {OfText: &anthropic.TextBlockParam{
                    Text: fmt.Sprintf("[Previous: used %s]", toolName),
                }},
            }
        }
    }
}
```

关键细节:
- `contentLen > 100`: 短结果不压缩
- 用 `ToolResultBlockParamContentUnion` (不是 `ContentBlockParamUnion`)
- 直接修改 messages slice 元素 (Go slice 引用语义)

---

## 第 221-257 行: `autoCompact()` — Layer 2

```go
func autoCompact(messages []anthropic.MessageParam) []anthropic.MessageParam {
    // 1. 保存完整 transcript 到磁盘 (JSONL 格式)
    transcriptPath := filepath.Join(transcriptDir,
        fmt.Sprintf("transcript_%d.jsonl", time.Now().Unix()))
    // ... 写入文件 ...

    // 2. 让 LLM 总结 (独立的 API 调用)
    resp, err := globalClient.Messages.New(context.Background(), anthropic.MessageNewParams{
        Model: anthropic.Model(model),
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(
                "Summarize this conversation for continuity..." + conversationText)),
        },
        MaxTokens: 2000,
    })

    // 3. 用 2 条消息替换全部历史
    return []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(
            fmt.Sprintf("[Conversation compressed. Transcript: %s]\n\n%s",
                transcriptPath, summary))),
        anthropic.NewAssistantMessage(anthropic.NewTextBlock(
            "Understood. I have the context from the summary. Continuing.")),
    }
}
```

三步: 保存 → 总结 → 替换。原始对话可从 transcript 文件恢复。

---

## 第 306-347 行: `agentLoop()` — 三层压缩集成

```go
func agentLoop(messages *[]anthropic.MessageParam) {  // 注意: 指针
    for {
        microCompact(*messages)                        // Layer 1

        if estimateTokens(*messages) > threshold {
            *messages = autoCompact(*messages)          // Layer 2
        }

        resp, err := globalClient.Messages.New(...)
        // ...

        manualCompact := false
        for _, block := range resp.Content {
            if block.Name == "compact" {
                manualCompact = true                    // 标记
                output = "Compressing..."
            }
        }

        if manualCompact {
            *messages = autoCompact(*messages)          // Layer 3
        }
    }
}
```

签名从 `[]MessageParam` 变为 **`*[]MessageParam`**, 因为 `autoCompact` 需要**替换整个 slice**。

---

## s05 → s06 变化对照表

| 维度 | s05 | s06 |
|------|-----|-----|
| 新增概念 | 两层 Skill 注入 | **三层上下文压缩** |
| 工具 | 5 个 (含 load_skill) | 5 个 (含 compact) |
| agentLoop 签名 | 返回 messages | **`*[]MessageParam`** (就地替换) |
| LLM 调用 | 仅 agent loop | **额外调用**: 总结用 |
| 磁盘 I/O | 启动时读 | **运行时写** transcript |
| Token 管理 | 无 | `estimateTokens()` + 阈值 |

---

## 核心架构图

```
agentLoop(*messages)
  for {
  │   microCompact(*messages)                ← Layer 1: 每轮
  │     旧 tool_result → "[Previous: used bash]"
  │
  │   estimateTokens(*messages) > 50000?
  │     YES → autoCompact(*messages)          ← Layer 2: 自动
  │             ├── 保存 transcript 到 .transcripts/
  │             ├── LLM 总结对话
  │             └── *messages = [summary, ack]
  │
  │   resp = LLM(system, *messages, tools)
  │   *messages += assistant turn
  │   if !tool_use → return
  │
  │   for each tool_use:
  │     if "compact" → manualCompact = true
  │     else → dispatch[name](input)
  │   *messages += user turn (results)
  │
  │   manualCompact?
  │     YES → autoCompact(*messages)          ← Layer 3: 手动
  }
```

**s06 的核心洞察**: Agent 需要"战略性遗忘"才能无限工作。三层管线从轻到重: 微压缩 (零成本, 每轮), 自动总结 (一次额外 API 调用), 手动触发 (模型主动请求)。这就是 Claude Code 的上下文压缩机制的简化版。
