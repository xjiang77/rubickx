# s06: Context Compact — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > s05 > [ s06 ] | s07 > s08 > s09 > s10 > s11 > s12`

> Source: [`go/s06/main.go`](../../s06/main.go)

---

## Core Difference: s06 vs s05

s05 solved "how to inject knowledge into the model." s06 solves the opposite problem: **how to make the model forget**.

> **"The agent can forget strategically and keep working forever."**

Core problem: the agent loop appends content to messages every iteration, causing unbounded context growth. Eventually it either hits the token limit or important information gets drowned out. Solution: **three-layer compression pipeline**.

```
Layer 1: microCompact   — Runs silently each iteration, replaces old tool_results with placeholders
Layer 2: autoCompact    — Triggers when tokens > 50000, LLM summarizes + replaces all history
Layer 3: compact tool   — Model calls explicitly, immediately executes Layer 2
```

---

## Architecture Diagram

```
Each loop iteration:
+------------------+
| Tool call result |
+------------------+
        |
        v
[Layer 1: microCompact]        (silent, runs every iteration)
  Replace tool_results older than the recent 3
  with "[Previous: used {tool_name}]"
        |
        v
[Check: tokens > 50000?]
   |               |
   no              yes
   |               |
   v               v
continue    [Layer 2: autoCompact]
              Save full transcript to .transcripts/
              Have LLM summarize conversation
              Replace all messages with [summary]
                    |
                    v
            [Layer 3: compact tool]
              Model calls compact -> immediately execute Layer 2
```

---

## Lines 47-55: Constants

```go
const (
    threshold  = 50000  // Token estimation threshold, triggers auto_compact when exceeded
    keepRecent = 3      // Keep the most recent 3 tool_results uncompressed
)
```

---

## Lines 136-140: `estimateTokens()` — Rough Token Estimation

```go
func estimateTokens(messages []anthropic.MessageParam) int {
    data, _ := json.Marshal(messages)
    return len(data) / 4
}
```

~4 characters ≈ 1 token, rough but practical.

---

## Lines 142-219: `microCompact()` — Layer 1

```go
func microCompact(messages []anthropic.MessageParam) {
    // 1. Collect positions of all tool_results (msgIdx, partIdx)
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

    // 2. Build tool_use_id -> tool_name mapping
    toolNameMap := map[string]string{}
    for _, msg := range messages {
        if msg.Role != "assistant" { continue }
        for _, part := range msg.Content {
            if part.OfToolUse != nil {
                toolNameMap[part.OfToolUse.ID] = part.OfToolUse.Name
            }
        }
    }

    // 3. Replace old results (keep last keepRecent)
    toClear := locs[:len(locs)-keepRecent]
    for _, loc := range toClear {
        tr := messages[loc.msgIdx].Content[loc.partIdx].OfToolResult
        // Only compress results longer than 100 characters
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

Key details:
- `contentLen > 100`: short results are not compressed
- Uses `ToolResultBlockParamContentUnion` (not `ContentBlockParamUnion`)
- Modifies messages slice elements directly (Go slice reference semantics)

---

## Lines 221-257: `autoCompact()` — Layer 2

```go
func autoCompact(messages []anthropic.MessageParam) []anthropic.MessageParam {
    // 1. Save full transcript to disk (JSONL format)
    transcriptPath := filepath.Join(transcriptDir,
        fmt.Sprintf("transcript_%d.jsonl", time.Now().Unix()))
    // ... write to file ...

    // 2. Have LLM summarize (separate API call)
    resp, err := globalClient.Messages.New(context.Background(), anthropic.MessageNewParams{
        Model: anthropic.Model(model),
        Messages: []anthropic.MessageParam{
            anthropic.NewUserMessage(anthropic.NewTextBlock(
                "Summarize this conversation for continuity..." + conversationText)),
        },
        MaxTokens: 2000,
    })

    // 3. Replace all history with 2 messages
    return []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(
            fmt.Sprintf("[Conversation compressed. Transcript: %s]\n\n%s",
                transcriptPath, summary))),
        anthropic.NewAssistantMessage(anthropic.NewTextBlock(
            "Understood. I have the context from the summary. Continuing.")),
    }
}
```

Three steps: save -> summarize -> replace. The original conversation can be recovered from the transcript file.

---

## Lines 306-347: `agentLoop()` — Three-Layer Compression Integration

```go
func agentLoop(messages *[]anthropic.MessageParam) {  // Note: pointer
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
                manualCompact = true                    // Flag it
                output = "Compressing..."
            }
        }

        if manualCompact {
            *messages = autoCompact(*messages)          // Layer 3
        }
    }
}
```

The signature changed from `[]MessageParam` to **`*[]MessageParam`** because `autoCompact` needs to **replace the entire slice**.

---

## s05 -> s06 Change Comparison

| Dimension | s05 | s06 |
|-----------|-----|-----|
| New concept | Two-layer Skill injection | **Three-layer context compression** |
| Tools | 5 (including load_skill) | 5 (including compact) |
| agentLoop signature | Returns messages | **`*[]MessageParam`** (in-place replacement) |
| LLM calls | Agent loop only | **Additional call**: for summarization |
| Disk I/O | Read at startup | **Write at runtime** transcript |
| Token management | None | `estimateTokens()` + threshold |

---

## Core Architecture Diagram

```
agentLoop(*messages)
  for {
  |   microCompact(*messages)                <- Layer 1: every iteration
  |     old tool_result -> "[Previous: used bash]"
  |
  |   estimateTokens(*messages) > 50000?
  |     YES -> autoCompact(*messages)          <- Layer 2: automatic
  |             +-- Save transcript to .transcripts/
  |             +-- LLM summarizes conversation
  |             +-- *messages = [summary, ack]
  |
  |   resp = LLM(system, *messages, tools)
  |   *messages += assistant turn
  |   if !tool_use -> return
  |
  |   for each tool_use:
  |     if "compact" -> manualCompact = true
  |     else -> dispatch[name](input)
  |   *messages += user turn (results)
  |
  |   manualCompact?
  |     YES -> autoCompact(*messages)          <- Layer 3: manual
  }
```

**s06's core insight**: An agent needs "strategic forgetting" to work indefinitely. The three-layer pipeline goes from light to heavy: micro-compact (zero cost, every iteration), auto-summarize (one extra API call), manual trigger (model proactively requests). This is a simplified version of Claude Code's context compression mechanism.
