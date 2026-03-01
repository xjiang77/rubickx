# s03: TodoWrite — Go 逐行解读

`s01 > s02 > [ s03 ] s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> 源码: [`go/s03/main.go`](../../go/s03/main.go) (478 行)

---

## s03 vs s02 的核心区别

s02 的工具和 agent loop 结构全部保留。s03 新增两个概念: **TodoManager** (结构化状态) 和 **Nag Reminder** (催促注入)。

> **"没有计划的 agent 走哪算哪 — 先列步骤再动手, 完成率翻倍。"**

---

## 第 1-25 行: 包声明与架构图

```
User → LLM → Tools + todo
              ↑         |
              +--- tool_result
                    |
        +-----------+-----------+
        | TodoManager state     |
        | [ ] task A            |
        | [>] task B ← doing    |
        | [x] task C            |
        +-----------------------+
                    |
        if rounds_since_todo >= 3:
          inject <reminder>
```

两个核心新概念一目了然:

1. **TodoManager** — 模型可写入的结构化状态
2. **Nag Reminder** — 如果模型连续 3 轮不更新 todo, 自动注入催促消息

---

## 第 44-100 行: `TodoManager` — s03 的核心新增

### 类型定义

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

三种状态对应三种显示标记:

- `pending` → `[ ]`
- `in_progress` → `[>]`
- `completed` → `[x]`

### 第 60-85 行 — `Update()` 方法

模型每次调用 todo 工具时触发:

```go
func (t *TodoManager) Update(items []TodoItem) (string, error) {
    if len(items) > 20 {                                    // 1. 数量上限
        return "", fmt.Errorf("max 20 todos allowed")
    }
    inProgressCount := 0
    for i := range items {
        item := &items[i]
        if item.Text == "" {                                // 2. 文本不能为空
            return "", fmt.Errorf("item %s: text required", item.ID)
        }
        if item.ID == "" {                                  // 3. 自动补 ID
            item.ID = fmt.Sprintf("%d", i+1)
        }
        switch item.Status {
        case "pending", "in_progress", "completed":         // 4. 状态枚举校验
        default:
            return "", fmt.Errorf("item %s: invalid status '%s'", item.ID, item.Status)
        }
        if item.Status == "in_progress" {
            inProgressCount++
        }
    }
    if inProgressCount > 1 {                                // 5. 最多1个 in_progress
        return "", fmt.Errorf("only one task can be in_progress at a time")
    }
    t.Items = items                                         // 6. 全量替换
    return t.Render(), nil
}
```

关键设计: **全量替换而非增量更新**。模型每次调用 `todo` 工具都传入完整的 items 数组, 替换掉旧列表。这比增量 API (add/remove/update) 简单得多 — 模型只需要描述"我想要的最终状态", 不需要跟踪差异。

验证规则: **最多只能有 1 个 `in_progress` 任务**。这迫使模型聚焦 — 做完一件事再开始下一件。

### 第 88-100 行 — `Render()` 方法

将 todo 列表渲染为人类可读文本:

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

输出格式:

```
[x] #1: Create hello.txt with 'Hello World'
[x] #2: Read hello.txt
[>] #3: Edit 'World' to 'Go' in hello.txt

(2/3 completed)
```

这个渲染结果会作为 `tool_result` 回传给模型, 让它知道当前进度。

---

## 第 102-112 行: 全局变量

```go
var (
    model   string
    system  string
    workdir string
    tools   []anthropic.ToolUnionParam
    todo    = &TodoManager{}              // s03 新增: 全局 todo 状态
)
```

`todo` 是一个指针, 生命周期跨越整个对话。

---

## 第 118-121 行: System Prompt 的变化

```go
system = fmt.Sprintf(`You are a coding agent at %s.
Use the todo tool to plan multi-step tasks. Mark in_progress before starting, completed when done.
Prefer tools over prose.`, workdir)
```

对比 s02: `"Use tools to solve tasks. Act, don't explain."`

s03 新增了**明确的 todo 使用指令**: 告诉模型用 todo 来规划多步任务, 开始前标 `in_progress`, 完成后标 `completed`。

---

## 第 159-182 行: `todo` 工具的 Schema 定义

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

这是所有工具中最复杂的 schema: **嵌套的 array → object**。`enum` 约束确保模型只能传 3 种有效状态。

---

## 第 223-235 行: `handleTodo` — todo 的 handler

```go
func handleTodo(raw json.RawMessage) string {
    var input struct {
        Items []TodoItem `json:"items"`        // 直接反序列化为 []TodoItem
    }
    if err := json.Unmarshal(raw, &input); err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    result, err := todo.Update(input.Items)    // 调用 TodoManager.Update()
    if err != nil {
        return fmt.Sprintf("Error: %v", err)
    }
    return result                               // 返回渲染后的 todo 文本
}
```

注意这里 `json.Unmarshal` 的错误被**检查**了 (不像其他 handler 用 `_` 忽略), 因为 todo 的 schema 更复杂, 反序列化更容易出错。

---

## 第 289-311 行: `agentLoop()` — 新增 nag reminder 机制

这是 s03 对 agent loop **唯一的修改**:

### 计数器初始化

```go
func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
    roundsSinceTodo := 0                       // 追踪距上次 todo 调用的轮数
```

### 催促注入逻辑

```go
    for {
        // Nag reminder: 如果 3+ 轮没更新 todo, 注入催促
        if roundsSinceTodo >= 3 && len(messages) > 0 {
            last := &messages[len(messages)-1]
            last.Content = append(
                []anthropic.ContentBlockParamUnion{
                    anthropic.NewTextBlock("<reminder>Update your todos.</reminder>"),
                },
                last.Content...,             // 原有内容后移
            )
        }
```

**工作原理**:

1. `roundsSinceTodo` 初始为 0
2. 每轮循环后, 如果模型调用了 `todo` 工具 → 重置为 0, 否则 +1
3. 当计数达到 3, 在**最后一条 user 消息的开头**插入 `<reminder>Update your todos.</reminder>`
4. 模型在下一轮 API 调用时会看到这个 reminder, 被"催促"更新进度

### 计数器更新

```go
        usedTodo := false
        for _, block := range resp.Content {
            if block.Type == "tool_use" {
                // ... dispatch ...
                if block.Name == "todo" {
                    usedTodo = true                // 记录是否使用了 todo
                }
            }
        }
        if usedTodo {
            roundsSinceTodo = 0                    // 用了 → 重置计数
        } else {
            roundsSinceTodo++                      // 没用 → 计数+1
        }
```

这是一种**轻量级的行为引导**: 不改变 system prompt, 不改变工具定义, 只是在对话流中注入一小段文本来提醒模型。

---

## s02 → s03 变化对照表

| 维度 | s02 | s03 |
|------|-----|-----|
| 工具数量 | 4 | 5 (+todo) |
| 新增概念 | dispatch map | TodoManager + nag reminder |
| Agent Loop | 纯循环 | **+计数器 + reminder 注入** |
| 状态管理 | 无状态 (每次工具独立执行) | **有状态** (TodoManager 跨轮次持久化) |
| 模型引导 | system prompt 一句话 | system prompt + nag reminder 双层引导 |
| 新增代码量 | — | ~100 行 (TodoManager + handler + nag) |

---

## 核心架构图

```
agentLoop()
  roundsSinceTodo = 0
  │
  for {
  │   ┌─── roundsSinceTodo >= 3 ? ───┐
  │   │ YES: inject <reminder>        │ NO: skip
  │   └───────────────────────────────┘
  │
  │   resp = client.Messages.New(messages, tools)
  │   messages += assistant turn
  │   if StopReason != "tool_use" → return
  │
  │   for each tool_use:
  │       dispatch[name](input)
  │       if name == "todo":
  │           usedTodo = true ──→ TodoManager.Update(items)
  │                                  ├── validate (max 20, 1 in_progress)
  │                                  ├── replace items
  │                                  └── return Render()
  │                                        "[ ] #1: ..."
  │                                        "[>] #2: ..."
  │                                        "(1/3 completed)"
  │
  │   roundsSinceTodo = usedTodo ? 0 : roundsSinceTodo + 1
  │   messages += user turn (results)
  }
```

**s03 的核心洞察**: "Agent 可以追踪自己的进度 — 而且我能看到它。" TodoManager 让模型的思维过程外化为可观察的结构化数据。
