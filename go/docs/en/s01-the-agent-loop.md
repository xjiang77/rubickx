# s01: Agent Loop — Go Line-by-Line Walkthrough

`[ s01 ] s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> Source: [`go/s01-the-agent-loop/main.go`](../../s01-the-agent-loop/main.go) (190 lines)

---

## Lines 1-19: Package Declaration and Architecture Diagram

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

The comments illustrate the entire agent's core pattern: a **while loop** that keeps executing tools and feeding back results as long as the model returns `tool_use`. This is the entire secret of an AI coding agent.

---

## Lines 21-33: Imports

```go
import (
    "bufio"           // Read stdin line by line
    "context"         // Timeout control
    "encoding/json"   // Parse tool input JSON
    "fmt"             // Print output
    "os"              // Environment variables, working directory
    "os/exec"         // Execute shell commands
    "strings"         // String processing
    "time"            // Timeout duration

    "github.com/anthropics/anthropic-sdk-go"         // Anthropic Go SDK
    "github.com/anthropics/anthropic-sdk-go/option"  // SDK config options (e.g., base URL)
)
```

Standard library + official Anthropic Go SDK, no other third-party dependencies.

---

## Lines 35-66: `init()` — Global Initialization

```go
var (
    model  string                        // Model ID
    system string                        // System prompt
    tools  []anthropic.ToolUnionParam    // Tool definition list
)
```

### Lines 41-45 — Read Model ID

```go
model = os.Getenv("MODEL_ID")
if model == "" {
    model = "claude-sonnet-4-20250514"   // Default fallback
}
```

### Lines 47-48 — Construct System Prompt

```go
cwd, _ := os.Getwd()
system = fmt.Sprintf("You are a coding agent at %s. Use bash to solve tasks. Act, don't explain.", cwd)
```

Embeds the current working directory into the system prompt so the model knows which path it's operating in. `"Act, don't explain"` instructs the model to take action directly instead of being verbose.

### Lines 50-65 — Define the Single Tool: `bash`

```go
tools = []anthropic.ToolUnionParam{
    {
        OfTool: &anthropic.ToolParam{
            Name:        "bash",
            Description: anthropic.String("Run a shell command."),
            InputSchema: anthropic.ToolInputSchemaParam{
                Properties: map[string]interface{}{
                    "command": map[string]interface{}{
                        "type": "string",          // Single parameter: command, string type
                    },
                },
                Required: []string{"command"},      // Required
            },
        },
    },
}
```

A JSON Schema format tool definition telling the model "you have a `bash` tool that accepts a `command` string parameter." The model generates valid tool calls based on this schema.

---

## Lines 68-98: `runBash()` — Execute Shell Commands

### Lines 69-74 — Dangerous Command Interception

```go
dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
for _, d := range dangerous {
    if strings.Contains(command, d) {
        return "Error: Dangerous command blocked"
    }
}
```

A simple blocklist mechanism to prevent the model from executing destructive commands.

### Lines 76-77 — 120-Second Timeout Context

```go
ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
defer cancel()
```

### Lines 79-81 — Execute Command

```go
cmd := exec.CommandContext(ctx, "sh", "-c", command)   // Execute via sh -c
cmd.Dir, _ = os.Getwd()                                // Run in current directory
output, err := cmd.CombinedOutput()                    // Merge stdout + stderr
```

### Lines 83-97 — Handle Results, 4 Cases

```go
if ctx.Err() == context.DeadlineExceeded { return "Error: Timeout (120s)" }  // Timeout
if err != nil && out == "" { return fmt.Sprintf("Error: %v", err) }          // Execution error
if out == "" { return "(no output)" }                                         // No output
if len(out) > 50000 { return out[:50000] }                                   // Truncate long output
return out
```

---

## Lines 100-102: Tool Input Struct

```go
type bashInput struct {
    Command string `json:"command"`
}
```

Used to deserialize the JSON `{"command": "ls -la"}` returned by the model into a Go struct.

---

## Lines 104-154: `agentLoop()` — The Core Loop (Most Important Function)

```go
func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
    for {  // <- Infinite loop, exits via return
```

### Lines 107-113 — Call the LLM

```go
resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
    Model:     anthropic.Model(model),
    System:    []anthropic.TextBlockParam{{Text: system}},
    Messages:  messages,       // Full conversation history
    Tools:     tools,          // Tell the model available tools
    MaxTokens: 8000,
})
```

Each loop iteration carries the **complete messages history**; the model decides whether to call a tool or respond directly based on context.

### Lines 119-129 — Append Model Response to Messages (Construct Assistant Turn)

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

The model's response content may contain both `text` and `tool_use` blocks. We need to convert **response types** (`ContentBlock`) into **request types** (`ContentBlockParamUnion`) before putting them back into the messages history.

### Lines 131-134 — Exit Condition

```go
if resp.StopReason != "tool_use" {
    return messages    // Model didn't call a tool -> loop ends
}
```

This is the **only exit condition** of the agent loop. If `StopReason` is `end_turn` (finished answering normally) or any non-`tool_use` value, the loop ends.

### Lines 136-152 — Execute Tool Calls, Collect Results

```go
var results []anthropic.ContentBlockParamUnion
for _, block := range resp.Content {
    if block.Type == "tool_use" {
        var input bashInput
        _ = json.Unmarshal(block.Input, &input)           // Parse JSON -> bashInput
        fmt.Printf("\033[33m$ %s\033[0m\n", input.Command) // Print command in yellow
        output := runBash(input.Command)                   // Execute command
        if len(output) > 200 {
            fmt.Println(output[:200])                      // Display only first 200 chars on screen
        } else {
            fmt.Println(output)
        }
        results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
        //                                                  ^^^^^^^^      ^^^^^^  ^^^^^
        //                                                  correlation   full    isError=false
        //                                                  ID            output
    }
}
messages = append(messages, anthropic.NewUserMessage(results...))
//                          ^^^^^^^^^^^^^^^^^^^^^^
//                          tool_result appended as user role
```

Key details:

- `block.ID` **pairs** the tool result with its corresponding tool call
- Screen shows only 200 characters, but the **full output** (up to 50000 chars) is sent back to the model
- `tool_result` must be appended to messages with the **user role** (required by the Anthropic API)

---

## Lines 156-190: `main()` — Entry Point and REPL

### Lines 157-161 — Create Client

```go
opts := []option.RequestOption{}
if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
    opts = append(opts, option.WithBaseURL(baseURL))  // Support custom API endpoint
}
client := anthropic.NewClient(opts...)  // API key auto-read from ANTHROPIC_API_KEY
```

### Lines 163-164 — Initialization

```go
var messages []anthropic.MessageParam   // Empty conversation history
scanner := bufio.NewScanner(os.Stdin)   // Read input from stdin
```

### Lines 166-189 — REPL Loop

```go
for {
    fmt.Print("\033[36ms01 >> \033[0m")     // Cyan prompt
    if !scanner.Scan() { break }            // EOF -> exit
    query := strings.TrimSpace(scanner.Text())
    if query == "" || query == "q" || query == "exit" { break }

    // Append user input
    messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))

    // Run agent loop (may involve multiple tool call rounds)
    messages = agentLoop(&client, messages)

    // Print the text portion of the last assistant message
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

Note that `messages` **accumulates across multiple conversation turns**, meaning the agent has full conversation memory.

---

## Overall Architecture Diagram

```
main()
  |
  +-- Create Anthropic client (read API key + base URL)
  |
  +-- REPL loop
       |
       +-- Read user input -> append to messages
       |
       +-- agentLoop(messages)  <- core
       |    |
       |    +-- for loop ----------------------------+
       |         |                                   |
       |         +-- client.Messages.New()           |
       |         |   (send messages + tools)          |
       |         |                                   |
       |         +-- Append assistant turn           |
       |         |                                   |
       |         +-- StopReason != "tool_use"?       |
       |         |   +-- YES -> return --------------+
       |         |   +-- NO  |                       |
       |         |                                   |
       |         +-- Iterate tool_use blocks         |
       |         |   +-- json.Unmarshal              |
       |         |   +-- runBash(command)             |
       |         |   +-- Collect tool_result         |
       |         |                                   |
       |         +-- Append user turn (results) -----+
       |
       +-- Print final text response
```

**190 lines of code, a complete AI coding agent.** The core is just that `for` loop + `StopReason` check.
