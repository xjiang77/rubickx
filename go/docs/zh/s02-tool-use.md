# s02: Tool Use — Go 逐行解读

`s01 > [ s02 ] s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > s12`

> 源码: [`go/s02-tool-use/main.go`](../../s02-tool-use/main.go) (351 行)

---

## s02 vs s01 的核心区别

> **"The loop didn't change at all. I just added tools."**

s01 只有 1 个工具 `bash`, s02 扩展到 4 个工具并引入了 **dispatch map 模式**。Agent loop 本身一行没动。

---

## 第 1-17 行: 包声明与架构图

```
User → LLM → Tool Dispatch { bash, read, write, edit }
              ↑                        |
              +---- tool_result -------+
```

跟 s01 的图一样, 唯一区别是右边从单个 `bash` 变成了一个 **dispatch 表**。

---

## 第 19-32 行: 导入

比 s01 多了一个:

```go
"path/filepath"    // s02 新增: 路径拼接、目录提取、安全解析
```

因为 `read_file`/`write_file`/`edit_file` 需要路径操作。

---

## 第 34-45 行: 全局变量 — s02 的两个新概念

```go
var (
    model   string
    system  string
    workdir string                        // s02 新增: 工作目录, 用于路径安全检查
    tools   []anthropic.ToolUnionParam
)

type toolHandler func(input json.RawMessage) string   // 统一的工具函数签名

var dispatch map[string]toolHandler                    // 名称 → 处理函数的映射
```

**`toolHandler` 类型**是 s02 的关键设计: 所有工具处理函数接受 `json.RawMessage` (原始 JSON) 返回 `string`。这样 dispatch map 可以统一存储所有工具。

Python 版用 `lambda **kw` 达到同样效果, Go 用统一的函数签名。

---

## 第 47-111 行: `init()` — 初始化

### 第 57-102 行 — 4 个工具的 JSON Schema 定义

| 工具名 | 参数 | 必填 | 功能 |
|--------|------|------|------|
| `bash` | `command: string` | command | 执行 shell 命令 |
| `read_file` | `path: string`, `limit: integer` | path | 读文件, 可选限制行数 |
| `write_file` | `path: string`, `content: string` | path, content | 写文件 |
| `edit_file` | `path: string`, `old_text: string`, `new_text: string` | 全部 | 精确文本替换 |

### 第 104-110 行 — Dispatch Map (s02 的核心新增)

```go
dispatch = map[string]toolHandler{
    "bash":       handleBash,
    "read_file":  handleReadFile,
    "write_file": handleWriteFile,
    "edit_file":  handleEditFile,
}
```

这就是 Python 版的 `TOOL_HANDLERS` 字典。当模型调用工具时, 通过名称查表, 找到对应处理函数。**新增工具只需要: (1) tools 数组加定义 (2) dispatch 表加映射 (3) 实现 handler**。

---

## 第 113-127 行: `safePath()` — 路径安全 (s02 全新)

```go
func safePath(p string) (string, error) {
    full := filepath.Join(workdir, p)        // 相对路径拼接到工作目录
    abs, err := filepath.Abs(full)           // 解析为绝对路径 (消除 ../)
    if err != nil {
        return "", fmt.Errorf("invalid path: %s", p)
    }
    if !strings.HasPrefix(abs, workdir) {    // 检查是否还在工作目录内
        return "", fmt.Errorf("path escapes workspace: %s", p)
    }
    return abs, nil
}
```

**防止路径穿越攻击**。如果模型尝试 `read_file("../../etc/passwd")`, `filepath.Abs` 会解析掉 `..`, 然后 `HasPrefix` 检查会拦截。对应 Python 版的 `safe_path()` + `is_relative_to()`。

---

## 第 129-165 行: Tool Handlers — JSON 反序列化层

```go
func handleBash(raw json.RawMessage) string {
    var input struct {
        Command string `json:"command"`
    }
    _ = json.Unmarshal(raw, &input)     // 从原始 JSON 解析出具体参数
    return runBash(input.Command)        // 调用实际实现
}
```

每个 handler 做同一件事: **把 `json.RawMessage` 反序列化为具体结构体, 然后调用实现函数**。

这是 Go 相比 Python 多出的一层。Python 可以直接 `handler(**block.input)` 用字典展开传参, Go 需要显式反序列化。4 个 handler 结构完全对称:

- `handleBash` → `runBash(command)`
- `handleReadFile` → `runRead(path, limit)`
- `handleWriteFile` → `runWrite(path, content)`
- `handleEditFile` → `runEdit(path, oldText, newText)`

---

## 第 167-198 行: `runBash()` — 与 s01 完全相同

不再赘述, 和 s01 一模一样。

---

## 第 201-218 行: `runRead()` — 读文件 (s02 新增)

```go
func runRead(path string, limit int) string {
    fp, err := safePath(path)                    // 安全路径检查
    if err != nil { return fmt.Sprintf("Error: %v", err) }

    data, err := os.ReadFile(fp)                 // 读取整个文件
    if err != nil { return fmt.Sprintf("Error: %v", err) }

    lines := strings.Split(string(data), "\n")
    if limit > 0 && limit < len(lines) {         // 如果指定了行数限制
        lines = append(lines[:limit],
            fmt.Sprintf("... (%d more lines)", len(lines)-limit))
    }

    out := strings.Join(lines, "\n")
    if len(out) > 50000 { return out[:50000] }   // 截断保护
    return out
}
```

对应 Python 的 `run_read()`。`limit` 参数可选 (JSON Schema 里不在 required 中), 默认为 0 (读全部)。

---

## 第 221-234 行: `runWrite()` — 写文件 (s02 新增)

```go
func runWrite(path string, content string) string {
    fp, err := safePath(path)
    if err != nil { return ... }

    dir := filepath.Dir(fp)
    os.MkdirAll(dir, 0o755)           // 自动创建父目录 (对应 Python 的 mkdir(parents=True))

    os.WriteFile(fp, []byte(content), 0o644)
    return fmt.Sprintf("Wrote %d bytes to %s", len(content), path)
}
```

---

## 第 236-254 行: `runEdit()` — 精确文本替换 (s02 新增)

```go
func runEdit(path string, oldText string, newText string) string {
    fp, err := safePath(path)
    if err != nil { return ... }

    data, err := os.ReadFile(fp)
    content := string(data)

    if !strings.Contains(content, oldText) {       // 找不到目标文本 → 报错
        return fmt.Sprintf("Error: Text not found in %s", path)
    }

    newContent := strings.Replace(content, oldText, newText, 1)  // 只替换第一处
    os.WriteFile(fp, []byte(newContent), 0o644)
    return fmt.Sprintf("Edited %s", path)
}
```

`strings.Replace(..., 1)` 的第 4 个参数 `1` 表示**只替换第一处匹配**, 和 Python 的 `str.replace(old, new, 1)` 完全一致。这是 Claude Code 的 Edit 工具的简化版。

---

## 第 256-306 行: `agentLoop()` — 和 s01 几乎一样

唯一的区别在**第 289-303 行**, 工具执行部分:

```go
// s01 的写法 (硬编码 bash):
var input bashInput
_ = json.Unmarshal(block.Input, &input)
output := runBash(input.Command)

// s02 的写法 (dispatch map):
handler, ok := dispatch[block.Name]        // 通过名称查找处理函数
var output string
if ok {
    output = handler(block.Input)          // 统一调用接口
} else {
    output = fmt.Sprintf("Unknown tool: %s", block.Name)  // 未知工具优雅降级
}
```

这就是 s02 的精髓: **agent loop 的结构没有变化, 只是工具执行从硬编码变成了查表分发**。

---

## 第 317-351 行: `main()` — 和 s01 完全一样

唯一区别是提示符从 `s01 >>` 变成 `s02 >>`。

---

## s01 → s02 变化对照表

| 维度 | s01 | s02 |
|------|-----|-----|
| 工具数量 | 1 个 (bash) | 4 个 (bash + read + write + edit) |
| 工具执行 | 硬编码 `runBash()` | `dispatch[name](input)` 查表分发 |
| 路径安全 | 无 | `safePath()` 防路径穿越 |
| Agent Loop | `for { ... }` | **完全不变** |
| 新增代码量 | — | ~160 行 (工具定义+handler+实现) |
| 新增概念 | — | dispatch map, toolHandler 类型, safePath |

---

## 整体架构图

```
init()
  ├── tools = [ bash, read_file, write_file, edit_file ]   // Schema 告诉模型有哪些工具
  └── dispatch = { "bash": handleBash, "read_file": ... }  // 名称到函数的映射

agentLoop()  ← 和 s01 一模一样的循环
  for {
      resp = client.Messages.New(messages, tools)
      messages += assistant turn
      if resp.StopReason != "tool_use" → return

      for each tool_use block:
          handler = dispatch[block.Name]   // ← s02 唯一的变化点
          output = handler(block.Input)
          results += tool_result
      messages += user turn (results)
  }
```
