# s01: Agent Loop — Go 逐行解读

`[ s01 ] s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> 源码: [`go/s01/main.go`](../../go/s01/main.go) (190 行)

---

## 第 1-19 行: 包声明与架构图

```go
// s01 - The Agent Loop
//
//	for stopReason == "tool_use" {
//	    response = LLM(messages, tools)
//	    execute tools
//	    append results
//	}
//
//	+----------+      +-------+      +---------+
//	|   User   | ---> |  LLM  | ---> |  Tool   |
//	|  prompt  |      |       |      | execute |
//	+----------+      +---+---+      +----+----+
//	                      ^               |
//	                      |   tool_result |
//	                      +---------------+
//	                      (loop continues)
package main
```

注释画出了整个 agent 的核心模式: 一个 **while 循环**, 只要模型返回 `tool_use` 就继续执行工具、回传结果。这就是 AI coding agent 的全部秘密。

---

## 第 21-33 行: 导入依赖

```go
import (
    "bufio"           // 逐行读取 stdin
    "context"         // 超时控制
    "encoding/json"   // 解析工具输入 JSON
    "fmt"             // 打印输出
    "os"              // 环境变量、工作目录
    "os/exec"         // 执行 shell 命令
    "strings"         // 字符串处理
    "time"            // 超时时间

    "github.com/anthropics/anthropic-sdk-go"         // Anthropic Go SDK
    "github.com/anthropics/anthropic-sdk-go/option"  // SDK 配置选项 (如 base URL)
)
```

标准库 + Anthropic 官方 Go SDK, 没有其他第三方依赖。

---

## 第 35-66 行: `init()` — 全局初始化

```go
var (
    model  string                        // 模型 ID
    system string                        // system prompt
    tools  []anthropic.ToolUnionParam    // 工具定义列表
)
```

### 第 41-45 行 — 读取模型 ID

```go
model = os.Getenv("MODEL_ID")
if model == "" {
    model = "claude-sonnet-4-20250514"   // 默认值兜底
}
```

### 第 47-48 行 — 构造 system prompt

```go
cwd, _ := os.Getwd()
system = fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.", cwd)
```

把当前工作目录嵌入 system prompt, 让模型知道自己在哪个路径下工作。`"Act, don't explain"` 指示模型直接行动, 不要啰嗦解释。

### 第 50-65 行 — 定义唯一的工具 `bash`

```go
tools = []anthropic.ToolUnionParam{
    {
        OfTool: &anthropic.ToolParam{
            Name:        "bash",
            Description: anthropic.String("Run a shell command."),
            InputSchema: anthropic.ToolInputSchemaParam{
                Properties: map[string]interface{}{
                    "command": map[string]interface{}{
                        "type": "string",          // 唯一参数: command, 字符串类型
                    },
                },
                Required: []string{"command"},      // 必填
            },
        },
    },
}
```

JSON Schema 格式的工具定义, 告诉模型"你有一个 `bash` 工具, 接受一个 `command` 字符串参数"。模型会根据这个 schema 生成合法的工具调用。

---

## 第 68-98 行: `runBash()` — 执行 shell 命令

### 第 69-74 行 — 危险命令拦截

```go
dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
for _, d := range dangerous {
    if strings.Contains(command, d) {
        return "Error: Dangerous command blocked"
    }
}
```

简单的黑名单机制, 防止模型执行破坏性命令。

### 第 76-77 行 — 120 秒超时上下文

```go
ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
defer cancel()
```

### 第 79-81 行 — 执行命令

```go
cmd := exec.CommandContext(ctx, "sh", "-c", command)   // 通过 sh -c 执行
cmd.Dir, _ = os.Getwd()                                // 在当前目录执行
output, err := cmd.CombinedOutput()                    // stdout + stderr 合并
```

### 第 83-97 行 — 处理结果, 4 种情况

```go
if ctx.Err() == context.DeadlineExceeded { return "Error: Timeout (120s)" }  // 超时
if err != nil && out == "" { return fmt.Sprintf("Error: %v", err) }          // 执行出错
if out == "" { return "(no output)" }                                         // 无输出
if len(out) > 50000 { return out[:50000] }                                   // 截断过长输出
return out
```

---

## 第 100-102 行: 工具输入结构体

```go
type bashInput struct {
    Command string `json:"command"`
}
```

用于把模型返回的 JSON `{"command": "ls -la"}` 反序列化为 Go 结构体。

---

## 第 104-154 行: `agentLoop()` — 核心循环 (最重要的函数)

```go
func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
    for {  // ← 无限循环, 靠 return 退出
```

### 第 107-113 行 — 调用 LLM

```go
resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
    Model:     anthropic.Model(model),
    System:    []anthropic.TextBlockParam{{Text: system}},
    Messages:  messages,       // 完整对话历史
    Tools:     tools,          // 告诉模型可用的工具
    MaxTokens: 8000,
})
```

每次循环都带上**完整的 messages 历史**, 模型根据上下文决定是调用工具还是直接回答。

### 第 119-129 行 — 将模型响应追加到 messages (构造 assistant turn)

```go
assistantBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(resp.Content))
for _, block := range resp.Content {
    switch block.Type {
    case "text":
        assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
    case "tool_use":
        assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))
    }
}
messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))
```

模型返回的 content 可能包含 `text` 和 `tool_use` 两种 block。需要把**响应类型** (`ContentBlock`) 转成**请求类型** (`ContentBlockParamUnion`), 才能放回 messages 历史。

### 第 131-134 行 — 退出条件

```go
if resp.StopReason != "tool_use" {
    return messages    // 模型没有调用工具 → 循环结束
}
```

这就是 agent loop 的**唯一退出条件**。如果 `StopReason` 是 `end_turn` (正常回答完毕) 或其他非 `tool_use` 值, 循环结束。

### 第 136-152 行 — 执行工具调用, 收集结果

```go
var results []anthropic.ContentBlockParamUnion
for _, block := range resp.Content {
    if block.Type == "tool_use" {
        var input bashInput
        _ = json.Unmarshal(block.Input, &input)           // 解析 JSON → bashInput
        fmt.Printf("\033[33m$ %s\033[0m\n", input.Command) // 黄色打印命令
        output := runBash(input.Command)                   // 执行命令
        if len(output) > 200 {
            fmt.Println(output[:200])                      // 屏幕只显示前200字符
        } else {
            fmt.Println(output)
        }
        results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
        //                                                  ^^^^^^^^      ^^^^^^  ^^^^^
        //                                                  关联ID        完整输出  isError=false
    }
}
messages = append(messages, anthropic.NewUserMessage(results...))
//                          ^^^^^^^^^^^^^^^^^^^^^^
//                          tool_result 以 user 角色追加
```

关键细节:

- `block.ID` 将工具结果与对应的工具调用**配对**
- 屏幕只显示 200 字符, 但回传给模型的是**完整输出** (最多 50000 字符)
- `tool_result` 必须以 **user 角色**追加到 messages (这是 Anthropic API 的规定)

---

## 第 156-190 行: `main()` — 入口与 REPL

### 第 157-161 行 — 创建 client

```go
opts := []option.RequestOption{}
if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
    opts = append(opts, option.WithBaseURL(baseURL))  // 支持自定义 API 地址
}
client := anthropic.NewClient(opts...)  // API key 自动从 ANTHROPIC_API_KEY 读取
```

### 第 163-164 行 — 初始化

```go
var messages []anthropic.MessageParam   // 空的对话历史
scanner := bufio.NewScanner(os.Stdin)   // 从 stdin 读取输入
```

### 第 166-189 行 — REPL 循环

```go
for {
    fmt.Print("\033[36ms01 >> \033[0m")     // 青色提示符
    if !scanner.Scan() { break }            // EOF → 退出
    query := strings.TrimSpace(scanner.Text())
    if query == "" || query == "q" || query == "exit" { break }

    // 追加用户输入
    messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))

    // 运行 agent loop (可能多轮工具调用)
    messages = agentLoop(&client, messages)

    // 打印最后一条 assistant 消息的文本部分
    if len(messages) > 0 {
        last := messages[len(messages)-1]
        for _, part := range last.Content {
            if part.OfText != nil {
                fmt.Println(part.OfText.Text)
            }
        }
    }
}
```

注意 `messages` 在多轮对话中**持续累积**, 这意味着 agent 有完整的对话记忆。

---

## 整体架构图

```
main()
  │
  ├── 创建 Anthropic client (读 API key + base URL)
  │
  └── REPL 循环
       │
       ├── 读取用户输入 → 追加到 messages
       │
       ├── agentLoop(messages)  ← 核心
       │    │
       │    └── for 循环 ─────────────────────┐
       │         │                             │
       │         ├── client.Messages.New()     │
       │         │   (发送 messages + tools)    │
       │         │                             │
       │         ├── 追加 assistant turn       │
       │         │                             │
       │         ├── StopReason != "tool_use"? │
       │         │   ├── YES → return ─────────┘
       │         │   └── NO ↓                  │
       │         │                             │
       │         ├── 遍历 tool_use blocks      │
       │         │   ├── json.Unmarshal        │
       │         │   ├── runBash(command)       │
       │         │   └── 收集 tool_result      │
       │         │                             │
       │         └── 追加 user turn (results) ─┘
       │
       └── 打印最终文本回答
```

**190 行代码, 一个完整的 AI coding agent。** 核心就是那个 `for` 循环 + `StopReason` 判断。
