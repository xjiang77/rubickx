# s04: Subagent — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > [ s04 ] s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> Source: [`go/s04/main.go`](../../s04/main.go)

---

## Core Difference: s04 vs s03

s03 introduced TodoManager for progress tracking. s04 introduces an entirely new architectural concept: **Subagent**.

> **"Process isolation gives context isolation for free."**

Core problem: in long conversations, exploratory tool calls (grep, find, cat) continuously bloat the context, crowding out truly useful information. Solution: delegate exploration tasks to a subagent with an **independent context** — it returns only a summary when done, and its full conversation history is discarded.

---

## Architecture Diagram

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

## Lines 38-58: Global Variables — Dual System Prompts

```go
var (
    model          string
    workdir        string
    parentSystem   string                              // Parent agent's system prompt
    subagentSystem string                              // Subagent's system prompt
    childTools     []anthropic.ToolUnionParam           // Tools available to subagent
    parentTools    []anthropic.ToolUnionParam           // Tools available to parent agent
)
```

s04 is the first time **two sets of configurations** appear: parent and child have different system prompts and different tool sets.

### Lines 63-64 — Two System Prompts

```go
parentSystem = fmt.Sprintf("You are a coding agent at %s. Use the task tool to delegate exploration or subtasks.", workdir)
subagentSystem = fmt.Sprintf("You are a coding subagent at %s. Complete the given task, then summarize your findings.", workdir)
```

- **Parent**: told it can **delegate** tasks
- **Child**: told to **complete and summarize**

---

## Lines 67-116: Two Tool Sets

### childTools (Lines 67-111)

The subagent gets 4 basic tools: `bash`, `read_file`, `write_file`, `edit_file`.

**Note: no `task` tool** — this prevents the subagent from recursively spawning more subagents.

### parentTools (Lines 113-127)

The parent agent gets all child tools + `task`:

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

The `task` tool has a simple schema: one required `prompt` (what to do) and an optional `description` (short label for logging).

### Dispatch Map (Lines 130-135)

```go
dispatch = map[string]toolHandler{
    "bash":       handleBash,
    "read_file":  handleReadFile,
    "write_file": handleWriteFile,
    "edit_file":  handleEditFile,
}
```

**`task` is not in the dispatch map** — it's handled specially in the agent loop. This is because `task` needs access to `client` for API calls, while normal tools don't.

---

## Lines 240-295: `runSubagent()` — Core Addition of s04

This is the most important function in all of s04:

```go
func runSubagent(client *anthropic.Client, prompt string) string {
    // Fresh context — no parent history
    subMessages := []anthropic.MessageParam{
        anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
    }
```

**The first line is key**: `subMessages` is a brand-new empty message list containing only this prompt. The parent agent's full conversation history is not passed in.

### Lines 250-266 — Subagent's Agent Loop

```go
    for i := 0; i < 30; i++ { // safety limit
        resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
            Model:     anthropic.Model(model),
            System:    []anthropic.TextBlockParam{{Text: subagentSystem}},  // Child-specific prompt
            Messages:  subMessages,
            Tools:     childTools,   // no "task" tool — prevents recursive spawning
            MaxTokens: 8000,
        })
```

Three differences from the parent agent loop:

1. `System` uses `subagentSystem` instead of `parentSystem`
2. `Tools` uses `childTools` instead of `parentTools` (no `task`)
3. `for i < 30` instead of `for {}` — safety limit prevents infinite subagent loops

### Lines 275-285 — Extract Summary and Discard Context

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

When the subagent stops calling tools, **only the text content from the last round** is extracted as a summary to return. `subMessages` (the subagent's full conversation history) is automatically freed by Go's garbage collector after the function returns.

This is the implementation of "context isolation": the subagent may have executed 10 greps, read 20 files, and produced massive tool output, but the parent agent only receives a concise summary.

### Lines 288-301 — Subagent Tool Execution

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

The subagent uses the **exact same dispatch map** as the parent agent for tool execution. They share the filesystem — files created by the subagent are visible to the parent agent.

---

## Lines 307-350: `agentLoop()` — Changes to Parent Agent Loop

Main change is in the tool execution section:

```go
        for _, block := range resp.Content {
            if block.Type == "tool_use" {
                var output string
                if block.Name == "task" {                        // task handled specially
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
                    output = runSubagent(client, input.Prompt)   // Spawn subagent
                } else {
                    handler, ok := dispatch[block.Name]          // Normal tools via dispatch
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

The `task` tool is **handled inline** rather than placed in the dispatch map because it needs the `client` parameter. When the model calls `task`:

1. Parse `prompt` and `description`
2. Print log `> task (description): prompt...`
3. Call `runSubagent()` to spawn a subagent
4. The subagent's summary is returned to the parent model as `tool_result`

---

## s03 -> s04 Change Comparison

| Dimension | s03 | s04 |
|-----------|-----|-----|
| New concept | TodoManager | **Subagent** |
| System Prompt | 1 | 2 (parent + subagent) |
| Tool sets | 1 set (all tools for everyone) | **2 sets** (child lacks task) |
| Context management | All operations share one messages | **Subagent has independent context, discarded when done** |
| Recursion protection | Not needed | `childTools` excludes `task` |
| Safety limit | None | `for i < 30` prevents infinite loops |
| TodoManager | Present | Removed (s04 focuses on subagent concept) |

---

## Core Architecture Diagram

```
main()
  +-- agentLoop(parent)
       for {
           resp = LLM(parentSystem, parentTools)
           |
           for each tool_use:
           |   +-- name == "task" ?
           |   |   YES -> runSubagent(prompt)
           |   |          |
           |   |          +-- subMessages = [prompt]        <- fresh context
           |   |          +-- for i < 30:
           |   |          |     resp = LLM(subagentSystem, childTools)
           |   |          |     if !tool_use -> extract text -> return summary
           |   |          |     execute tools via dispatch
           |   |          |     append results
           |   |          +-- return "(hit limit)" if 30 rounds
           |   |
           |   +-- NO -> dispatch[name](input)               <- normal tool
           |
           results -> messages
       }
```

**s04's core insight**: The subagent and parent agent share the **filesystem** but not the **context**. The subagent can explore extensively (grep, find, cat), and all the noise stays in the subagent's context — only a refined summary returns to the parent agent. This is how Claude Code's Agent tool works.
