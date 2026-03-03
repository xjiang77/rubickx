# s03: TodoWrite — Go Line-by-Line Walkthrough

`s01 > s02 > [ s03 ] s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> Source: [`go/s03-todo-write/main.go`](../../s03-todo-write/main.go) (478 lines)

---

## Core Difference: s03 vs s02

All tools and agent loop structure from s02 are retained. s03 adds two new concepts: **TodoManager** (structured state) and **Nag Reminder** (nudge injection).

> **"An agent without a plan wanders aimlessly — list the steps first, then act. Completion rate doubles."**

---

## Lines 1-25: Package Declaration and Architecture Diagram

```
User -> LLM -> Tools + todo
              ^         |
              +--- tool_result
                    |
        +-----------+-----------+
        | TodoManager state     |
        | [ ] task A            |
        | [>] task B <- doing   |
        | [x] task C            |
        +-----------------------+
                    |
        if rounds_since_todo >= 3:
          inject <reminder>
```

The two core new concepts at a glance:

1. **TodoManager** — Structured state writable by the model
2. **Nag Reminder** — Auto-injects a nudge if the model doesn't update todos for 3 consecutive rounds

---

## Lines 44-100: `TodoManager` — Core Addition of s03

### Type Definitions

```go
type TodoItem struct {
    ID     string `json:"id"`
    Text   string `json:"text"`
    Status string `json:"status"`   // "pending" | "in_progress" | "completed"
}

type TodoManager struct {
    Items []TodoItem
}
```

Three statuses map to three display markers:

- `pending` -> `[ ]`
- `in_progress` -> `[>]`
- `completed` -> `[x]`

### Lines 60-85 — `Update()` Method

Triggered each time the model calls the todo tool:

```go
func (t *TodoManager) Update(items []TodoItem) (string, error) {
    if len(items) > 20 {                                    // 1. Count limit
        return "", fmt.Errorf("max 20 todos allowed")
    }
    inProgressCount := 0
    for i := range items {
        item := &items[i]
        if item.Text == "" {                                // 2. Text cannot be empty
            return "", fmt.Errorf("item %s: text required", item.ID)
        }
        if item.ID == "" {                                  // 3. Auto-assign ID
            item.ID = fmt.Sprintf("%d", i+1)
        }
        switch item.Status {
        case "pending", "in_progress", "completed":         // 4. Status enum validation
        default:
            return "", fmt.Errorf("item %s: invalid status '%s'", item.ID, item.Status)
        }
        if item.Status == "in_progress" {
            inProgressCount++
        }
    }
    if inProgressCount > 1 {                                // 5. At most 1 in_progress
        return "", fmt.Errorf("only one task can be in_progress at a time")
    }
    t.Items = items                                         // 6. Full replacement
    return t.Render(), nil
}
```

Key design: **Full replacement rather than incremental updates**. Each time the model calls the `todo` tool, it passes the complete items array, replacing the old list. This is much simpler than an incremental API (add/remove/update) — the model only needs to describe "the final state I want" without tracking diffs.

Validation rule: **At most 1 `in_progress` task**. This forces the model to focus — finish one thing before starting the next.

### Lines 88-100 — `Render()` Method

Renders the todo list into human-readable text:

```go
func (t *TodoManager) Render() string {
    if len(t.Items) == 0 {
        return "No todos."
    }
    markers := map[string]string{
        "pending":     "[ ]",
        "in_progress": "[>]",
        "completed":   "[x]",
    }
    var lines []string
    done := 0
    for _, item := range t.Items {
        lines = append(lines, fmt.Sprintf("%s #%s: %s", markers[item.Status], item.ID, item.Text))
        if item.Status == "completed" {
            done++
        }
    }
    lines = append(lines, fmt.Sprintf("\n(%d/%d completed)", done, len(t.Items)))
    return strings.Join(lines, "\n")
}
```

Output format:

```
[x] #1: Create hello.txt with 'Hello World'
[x] #2: Read hello.txt
[>] #3: Edit 'World' to 'Go' in hello.txt

(2/3 completed)
```

This rendered result is sent back to the model as `tool_result`, letting it know the current progress.

---

## Lines 102-112: Global Variables

```go
var (
    model   string
    system  string
    workdir string
    tools   []anthropic.ToolUnionParam
    todo    = &TodoManager{}              // New in s03: global todo state
)
```

`todo` is a pointer with a lifetime spanning the entire conversation.

---

## Lines 118-121: System Prompt Changes

```go
system = fmt.Sprintf(`You are a coding agent at %s.
Use the todo tool to plan multi-step tasks. Mark in_progress before starting, completed when done.
Prefer tools over prose.`, workdir)
```

Compare with s02: `"Use tools to solve tasks. Act, don't explain."`

s03 adds **explicit todo usage instructions**: telling the model to use todo for planning multi-step tasks, mark `in_progress` before starting, and `completed` when done.

---

## Lines 159-182: `todo` Tool Schema Definition

```go
{OfTool: &anthropic.ToolParam{
    Name:        "todo",
    Description: anthropic.String("Update task list. Track progress on multi-step tasks."),
    InputSchema: anthropic.ToolInputSchemaParam{
        Properties: map[string]interface{}{
            "items": map[string]interface{}{
                "type": "array",
                "items": map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "id":     map[string]interface{}{"type": "string"},
                        "text":   map[string]interface{}{"type": "string"},
                        "status": map[string]interface{}{
                            "type": "string",
                            "enum": []string{"pending", "in_progress", "completed"},
                        },
                    },
                    "required": []string{"id", "text", "status"},
                },
            },
        },
        Required: []string{"items"},
    },
}},
```

The most complex schema among all tools: **nested array -> object**. The `enum` constraint ensures the model can only pass 3 valid statuses.

---

## Lines 223-235: `handleTodo` — The Todo Handler

```go
func handleTodo(raw json.RawMessage) string {
    var input struct {
        Items []TodoItem `json:"items"`        // Directly deserialize to []TodoItem
    }
    if err := json.Unmarshal(raw, &input); err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    result, err := todo.Update(input.Items)    // Call TodoManager.Update()
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    return result                               // Return rendered todo text
}
```

Note that `json.Unmarshal` error is **checked** here (unlike other handlers that ignore with `_`), because the todo schema is more complex and deserialization is more error-prone.

---

## Lines 289-311: `agentLoop()` — New Nag Reminder Mechanism

This is s03's **only modification** to the agent loop:

### Counter Initialization

```go
func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
    roundsSinceTodo := 0                       // Track rounds since last todo call
```

### Nudge Injection Logic

```go
    for {
        // Nag reminder: if 3+ rounds without todo update, inject nudge
        if roundsSinceTodo >= 3 && len(messages) > 0 {
            last := &messages[len(messages)-1]
            last.Content = append(
                []anthropic.ContentBlockParamUnion{
                    anthropic.NewTextBlock("<reminder>Update your todos.</reminder>"),
                },
                last.Content...,             // Original content shifted after
            )
        }
```

**How it works**:

1. `roundsSinceTodo` starts at 0
2. After each loop iteration, if the model called the `todo` tool -> reset to 0, otherwise +1
3. When the count reaches 3, insert `<reminder>Update your todos.</reminder>` at the **beginning of the last user message**
4. The model sees this reminder in the next API call and is "nudged" to update progress

### Counter Update

```go
        usedTodo := false
        for _, block := range resp.Content {
            if block.Type == "tool_use" {
                // ... dispatch ...
                if block.Name == "todo" {
                    usedTodo = true                // Record whether todo was used
                }
            }
        }
        if usedTodo {
            roundsSinceTodo = 0                    // Used -> reset counter
        } else {
            roundsSinceTodo++                      // Not used -> increment
        }
```

This is a **lightweight behavioral guidance**: doesn't change the system prompt, doesn't change tool definitions, just injects a small text snippet into the conversation flow to remind the model.

---

## s02 -> s03 Change Comparison

| Dimension | s02 | s03 |
|-----------|-----|-----|
| Tool count | 4 | 5 (+todo) |
| New concepts | dispatch map | TodoManager + nag reminder |
| Agent Loop | Pure loop | **+counter + reminder injection** |
| State management | Stateless (each tool executes independently) | **Stateful** (TodoManager persists across turns) |
| Model guidance | Single system prompt line | System prompt + nag reminder dual-layer guidance |
| New code | — | ~100 lines (TodoManager + handler + nag) |

---

## Core Architecture Diagram

```
agentLoop()
  roundsSinceTodo = 0
  |
  for {
  |   +--- roundsSinceTodo >= 3 ? ---+
  |   | YES: inject <reminder>       | NO: skip
  |   +------------------------------+
  |
  |   resp = client.Messages.New(messages, tools)
  |   messages += assistant turn
  |   if StopReason != "tool_use" -> return
  |
  |   for each tool_use:
  |       dispatch[name](input)
  |       if name == "todo":
  |           usedTodo = true --> TodoManager.Update(items)
  |                                  +-- validate (max 20, 1 in_progress)
  |                                  +-- replace items
  |                                  +-- return Render()
  |                                        "[ ] #1: ..."
  |                                        "[>] #2: ..."
  |                                        "(1/3 completed)"
  |
  |   roundsSinceTodo = usedTodo ? 0 : roundsSinceTodo + 1
  |   messages += user turn (results)
  }
```

**s03's core insight**: "The agent can track its own progress — and I can see it." TodoManager externalizes the model's thought process into observable structured data.
