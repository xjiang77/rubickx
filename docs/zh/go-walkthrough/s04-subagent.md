# s04: Subagent — Go 逐行解读

`s01 > s02 > s03 > [ s04 ] s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> 源码: [`go/s04/main.go`](../../go/s04/main.go)

---

## s04 vs s03 的核心区别

s03 引入了 TodoManager 让模型管理进度。s04 引入了一个全新的架构概念: **子 agent (Subagent)**。

> **"Process isolation gives context isolation for free."**

核心问题: 长对话中, 探索性工具调用 (grep, find, cat) 的输出会不断膨胀上下文, 挤掉真正有用的信息。解决方案: 把探索任务委派给一个**独立上下文**的子 agent, 它完成后只返回摘要, 其完整对话历史被丢弃。

---

## 架构图

```
Parent agent                     Subagent
+------------------+             +------------------+
| messages=[...]   |             | messages=[]      |  <-- fresh context
|                  |  dispatch   |                  |
| tool: task       | ----------> | while tool_use:  |
|   prompt="..."   |             |   call tools     |
|   description="" |             |   append results |
|                  |  summary    |                  |
|   result = "..." | <---------- | return last text |
+------------------+             +------------------+
          |
Parent context stays clean.
Subagent context is discarded.
```

---

## 第 38-58 行: 全局变量 — 双 system prompt

```go
var (
    model          string
    workdir        string
    parentSystem   string                              // 父 agent 的 system prompt
    subagentSystem string                              // 子 agent 的 system prompt
    childTools     []anthropic.ToolUnionParam           // 子 agent 可用的工具
    parentTools    []anthropic.ToolUnionParam           // 父 agent 可用的工具
)
```

s04 首次出现了**两套配置**: 父和子有不同的 system prompt 和不同的工具集。

### 第 63-64 行 — 两个 system prompt

```go
parentSystem = fmt.Sprintf("You are a coding agent at %s. Use the task tool to delegate exploration or subtasks.", workdir)
subagentSystem = fmt.Sprintf("You are a coding subagent at %s. Complete the given task, then summarize your findings.", workdir)
```

- **父**: 被告知可以**委派**任务
- **子**: 被告知要**完成并总结**

---

## 第 67-116 行: 两套工具定义

### childTools (第 67-111 行)

子 agent 拿到 4 个基础工具: `bash`, `read_file`, `write_file`, `edit_file`。

**注意: 没有 `task` 工具** — 这防止了子 agent 递归生成更多子 agent。

### parentTools (第 113-127 行)

父 agent 拿到所有子工具 + `task`:

```go
parentTools = append(childTools, anthropic.ToolUnionParam{
    OfTool: &anthropic.ToolParam{
        Name:        "task",
        Description: anthropic.String("Spawn a subagent with fresh context. It shares the filesystem but not conversation history."),
        InputSchema: anthropic.ToolInputSchemaParam{
            Properties: map[string]interface{}{
                "prompt":      map[string]interface{}{"type": "string"},
                "description": map[string]interface{}{"type": "string", "description": "Short description of the task"},
            },
            Required: []string{"prompt"},
        },
    },
})
```

`task` 工具的 schema 很简单: 一个必填的 `prompt` (要做什么) 和一个可选的 `description` (简短描述, 用于日志)。

### dispatch map (第 130-135 行)

```go
dispatch = map[string]toolHandler{
    "bash":       handleBash,
    "read_file":  handleReadFile,
    "write_file": handleWriteFile,
    "edit_file":  handleEditFile,
}
```

**`task` 不在 dispatch map 中** — 它在 agent loop 里被特殊处理。这是因为 `task` 需要访问 `client` 来调用 API, 而普通工具不需要。

---

## 第 240-295 行: `runSubagent()` — s04 的核心新增

这是整个 s04 最重要的函数:

```go
func runSubagent(client *anthropic.Client, prompt string) string {
    // Fresh context — no parent history
    subMessages := []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
    }
```

**第一行就是关键**: `subMessages` 是一个全新的空消息列表, 只包含这次的 prompt。父 agent 的完整对话历史不会传入。

### 第 250-266 行 — 子 agent 的 agent loop

```go
    for i := 0; i < 30; i++ { // safety limit
        resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
            Model:     anthropic.Model(model),
            System:    []anthropic.TextBlockParam{{Text: subagentSystem}},  // 子专用 prompt
            Messages:  subMessages,
            Tools:     childTools,   // no "task" tool — prevents recursive spawning
            MaxTokens: 8000,
        })
```

三个与父 agent loop 的区别:

1. `System` 用 `subagentSystem` 而非 `parentSystem`
2. `Tools` 用 `childTools` 而非 `parentTools` (没有 `task`)
3. `for i < 30` 而非 `for {}` — 安全上限防止子 agent 无限循环

### 第 275-285 行 — 提取摘要并丢弃上下文

```go
        if resp.StopReason != "tool_use" {
            // Extract final text and return — child context discarded
            var texts []string
            for _, block := range resp.Content {
                if block.Type == "text" && block.Text != "" {
                    texts = append(texts, block.Text)
                }
            }
            if len(texts) == 0 {
                return "(no summary)"
            }
            return strings.Join(texts, "")
        }
```

当子 agent 停止调用工具时, **只提取最后一轮的文本内容**作为摘要返回。`subMessages` (子 agent 的完整对话历史) 在函数返回后被 Go 的垃圾回收器自动释放。

这就是 "context isolation" 的实现: 子 agent 可能执行了 10 次 grep、读了 20 个文件, 产生了大量工具输出, 但父 agent 只会收到一段简洁的摘要。

### 第 288-301 行 — 子 agent 的工具执行

```go
        var results []anthropic.ContentBlockParamUnion
        for _, block := range resp.Content {
            if block.Type == "tool_use" {
                handler, ok := dispatch[block.Name]
                var output string
                if ok {
                    output = handler(block.Input)
                } else {
                    output = fmt.Sprintf("Unknown tool: %s", block.Name)
                }
                if len(output) > 50000 {
                    output = output[:50000]
                }
                results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
            }
        }
        subMessages = append(subMessages, anthropic.NewUserMessage(results...))
```

子 agent 使用和父 agent **完全相同的 dispatch map** 来执行工具。它们共享文件系统 — 子 agent 创建的文件, 父 agent 也能看到。

---

## 第 307-350 行: `agentLoop()` — 父 agent loop 的变化

主要变化在工具执行部分:

```go
        for _, block := range resp.Content {
            if block.Type == "tool_use" {
                var output string
                if block.Name == "task" {                        // task 特殊处理
                    var input struct {
                        Prompt      string `json:"prompt"`
                        Description string `json:"description"`
                    }
                    _ = json.Unmarshal(block.Input, &input)
                    desc := input.Description
                    if desc == "" {
                        desc = "subtask"
                    }
                    fmt.Printf("> task (%s): %s\n", desc, truncate(input.Prompt, 80))
                    output = runSubagent(client, input.Prompt)   // 生成子 agent
                } else {
                    handler, ok := dispatch[block.Name]          // 普通工具走 dispatch
                    if ok {
                        output = handler(block.Input)
                    } else {
                        output = fmt.Sprintf("Unknown tool: %s", block.Name)
                    }
                }
                fmt.Printf("  %s\n", truncate(output, 200))
                results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
            }
        }
```

`task` 工具被**内联处理**而非放入 dispatch map, 因为它需要 `client` 参数。当模型调用 `task` 时:

1. 解析 `prompt` 和 `description`
2. 打印日志 `> task (description): prompt...`
3. 调用 `runSubagent()` 生成子 agent
4. 子 agent 的摘要作为 `tool_result` 返回给父模型

---

## s03 → s04 变化对照表

| 维度 | s03 | s04 |
|------|-----|-----|
| 新增概念 | TodoManager | **Subagent (子 agent)** |
| System Prompt | 1 个 | 2 个 (parent + subagent) |
| 工具集 | 1 套 (所有工具给所有人) | **2 套** (child 没有 task) |
| 上下文管理 | 所有操作共享同一 messages | **子 agent 独立上下文, 结束后丢弃** |
| 递归防护 | 无需 | `childTools` 不含 `task` |
| 安全限制 | 无 | `for i < 30` 防无限循环 |
| TodoManager | 有 | 移除 (s04 聚焦子 agent 概念) |

---

## 核心架构图

```
main()
  └── agentLoop(parent)
       for {
           resp = LLM(parentSystem, parentTools)
           │
           for each tool_use:
           │   ├── name == "task" ?
           │   │   YES → runSubagent(prompt)
           │   │          │
           │   │          ├── subMessages = [prompt]        ← fresh context
           │   │          ├── for i < 30:
           │   │          │     resp = LLM(subagentSystem, childTools)
           │   │          │     if !tool_use → extract text → return summary
           │   │          │     execute tools via dispatch
           │   │          │     append results
           │   │          └── return "(hit limit)" if 30 rounds
           │   │
           │   └── NO → dispatch[name](input)               ← normal tool
           │
           results → messages
       }
```

**s04 的核心洞察**: 子 agent 和父 agent 共享**文件系统**但不共享**上下文**。子 agent 可以大量探索 (grep, find, cat), 产生的噪音全部留在子 agent 的上下文中, 只有精炼的摘要返回给父 agent。这就是 Claude Code 的 Agent 工具的工作原理。
