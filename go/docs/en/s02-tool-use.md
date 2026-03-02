# s02: Tool Use — Go Line-by-Line Walkthrough

`s01 > [ s02 ] s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> Source: [`go/s02/main.go`](../../s02/main.go) (351 lines)

---

## Core Difference: s02 vs s01

> **"The loop didn't change at all. I just added tools."**

s01 had only 1 tool `bash`; s02 expands to 4 tools and introduces the **dispatch map pattern**. The agent loop itself didn't change a single line.

---

## Lines 1-17: Package Declaration and Architecture Diagram

```
User -> LLM -> Tool Dispatch { bash, read, write, edit }
              ^                        |
              +---- tool_result -------+
```

Same diagram as s01, the only difference is the right side changed from a single `bash` to a **dispatch table**.

---

## Lines 19-32: Imports

One addition over s01:

```go
"path/filepath"    // New in s02: path joining, directory extraction, safe resolution
```

Needed because `read_file`/`write_file`/`edit_file` require path operations.

---

## Lines 34-45: Global Variables — Two New Concepts in s02

```go
var (
    model   string
    system  string
    workdir string                        // New in s02: working directory for path safety checks
    tools   []anthropic.ToolUnionParam
)

type toolHandler func(input json.RawMessage) string   // Unified tool function signature

var dispatch map[string]toolHandler                    // Name -> handler function mapping
```

**The `toolHandler` type** is the key design in s02: all tool handler functions accept `json.RawMessage` (raw JSON) and return `string`. This allows the dispatch map to uniformly store all tools.

The Python version achieves the same effect with `lambda **kw`; Go uses a unified function signature.

---

## Lines 47-111: `init()` — Initialization

### Lines 57-102 — JSON Schema Definitions for 4 Tools

| Tool Name | Parameters | Required | Function |
|-----------|-----------|----------|----------|
| `bash` | `command: string` | command | Execute shell command |
| `read_file` | `path: string`, `limit: integer` | path | Read file, optional line limit |
| `write_file` | `path: string`, `content: string` | path, content | Write file |
| `edit_file` | `path: string`, `old_text: string`, `new_text: string` | all | Exact text replacement |

### Lines 104-110 — Dispatch Map (Core Addition of s02)

```go
dispatch = map[string]toolHandler{
    "bash":       handleBash,
    "read_file":  handleReadFile,
    "write_file": handleWriteFile,
    "edit_file":  handleEditFile,
}
```

This is the Python version's `TOOL_HANDLERS` dictionary. When the model calls a tool, it looks up the corresponding handler function by name. **Adding a new tool only requires: (1) add definition to the tools array (2) add mapping to the dispatch table (3) implement the handler**.

---

## Lines 113-127: `safePath()` — Path Safety (Entirely New in s02)

```go
func safePath(p string) (string, error) {
    full := filepath.Join(workdir, p)        // Join relative path to working directory
    abs, err := filepath.Abs(full)           // Resolve to absolute path (eliminates ../)
    if err != nil {
        return "", fmt.Errorf("invalid path: %s", p)
    }
    if !strings.HasPrefix(abs, workdir) {    // Check if still within working directory
        return "", fmt.Errorf("path escapes workspace: %s", p)
    }
    return abs, nil
}
```

**Prevents path traversal attacks**. If the model tries `read_file("../../etc/passwd")`, `filepath.Abs` resolves away the `..`, then the `HasPrefix` check blocks it. Corresponds to Python's `safe_path()` + `is_relative_to()`.

---

## Lines 129-165: Tool Handlers — JSON Deserialization Layer

```go
func handleBash(raw json.RawMessage) string {
    var input struct {
        Command string `json:"command"`
    }
    _ = json.Unmarshal(raw, &input)     // Parse specific parameters from raw JSON
    return runBash(input.Command)        // Call actual implementation
}
```

Each handler does the same thing: **deserialize `json.RawMessage` into a specific struct, then call the implementation function**.

This is an extra layer compared to Python. Python can directly use `handler(**block.input)` with dictionary unpacking; Go requires explicit deserialization. All 4 handlers have perfectly symmetric structure:

- `handleBash` -> `runBash(command)`
- `handleReadFile` -> `runRead(path, limit)`
- `handleWriteFile` -> `runWrite(path, content)`
- `handleEditFile` -> `runEdit(path, oldText, newText)`

---

## Lines 167-198: `runBash()` — Identical to s01

No further discussion; exactly the same as s01.

---

## Lines 201-218: `runRead()` — Read File (New in s02)

```go
func runRead(path string, limit int) string {
    fp, err := safePath(path)                    // Safe path check
    if err != nil { return fmt.Sprintf("Error: %v", err) }

    data, err := os.ReadFile(fp)                 // Read entire file
    if err != nil { return fmt.Sprintf("Error: %v", err) }

    lines := strings.Split(string(data), "\n")
    if limit > 0 && limit < len(lines) {         // If line limit is specified
        lines = append(lines[:limit],
            fmt.Sprintf("... (%d more lines)", len(lines)-limit))
    }

    out := strings.Join(lines, "\n")
    if len(out) > 50000 { return out[:50000] }   // Truncation protection
    return out
}
```

Corresponds to Python's `run_read()`. The `limit` parameter is optional (not in the JSON Schema's required list), defaulting to 0 (read all).

---

## Lines 221-234: `runWrite()` — Write File (New in s02)

```go
func runWrite(path string, content string) string {
    fp, err := safePath(path)
    if err != nil { return ... }

    dir := filepath.Dir(fp)
    os.MkdirAll(dir, 0o755)           // Auto-create parent directories (Python's mkdir(parents=True))

    os.WriteFile(fp, []byte(content), 0o644)
    return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
}
```

---

## Lines 236-254: `runEdit()` — Exact Text Replacement (New in s02)

```go
func runEdit(path string, oldText string, newText string) string {
    fp, err := safePath(path)
    if err != nil { return ... }

    data, err := os.ReadFile(fp)
    content := string(data)

    if !strings.Contains(content, oldText) {       // Target text not found -> error
        return fmt.Sprintf("Error: Text not found in %s", path)
    }

    newContent := strings.Replace(content, oldText, newText, 1)  // Replace only first occurrence
    os.WriteFile(fp, []byte(newContent), 0o644)
    return fmt.Sprintf("Edited %s", path)
}
```

The 4th argument `1` in `strings.Replace(..., 1)` means **replace only the first match**, identical to Python's `str.replace(old, new, 1)`. This is a simplified version of Claude Code's Edit tool.

---

## Lines 256-306: `agentLoop()` — Nearly Identical to s01

The only difference is in **Lines 289-303**, the tool execution section:

```go
// s01 approach (hardcoded bash):
var input bashInput
_ = json.Unmarshal(block.Input, &input)
output := runBash(input.Command)

// s02 approach (dispatch map):
handler, ok := dispatch[block.Name]        // Look up handler by name
var output string
if ok {
    output = handler(block.Input)          // Unified call interface
} else {
    output = fmt.Sprintf("Unknown tool: %s", block.Name)  // Unknown tool graceful degradation
}
```

This is the essence of s02: **the agent loop structure hasn't changed; tool execution simply went from hardcoded to table-driven dispatch**.

---

## Lines 317-351: `main()` — Identical to s01

Only difference is the prompt changed from `s01 >>` to `s02 >>`.

---

## s01 -> s02 Change Comparison

| Dimension | s01 | s02 |
|-----------|-----|-----|
| Tool count | 1 (bash) | 4 (bash + read + write + edit) |
| Tool execution | Hardcoded `runBash()` | `dispatch[name](input)` table lookup |
| Path safety | None | `safePath()` prevents path traversal |
| Agent Loop | `for { ... }` | **Completely unchanged** |
| New code | — | ~160 lines (tool definitions + handlers + implementations) |
| New concepts | — | dispatch map, toolHandler type, safePath |

---

## Overall Architecture Diagram

```
init()
  +-- tools = [ bash, read_file, write_file, edit_file ]   // Schema tells model what tools exist
  +-- dispatch = { "bash": handleBash, "read_file": ... }  // Name-to-function mapping

agentLoop()  <- same loop as s01
  for {
      resp = client.Messages.New(messages, tools)
      messages += assistant turn
      if resp.StopReason != "tool_use" -> return

      for each tool_use block:
          handler = dispatch[block.Name]   // <- s02's only change point
          output = handler(block.Input)
          results += tool_result
      messages += user turn (results)
  }
```
