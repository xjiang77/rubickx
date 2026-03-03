# s05: Skill Loading — Go 逐行解读

`s01 > s02 > s03 > s04 > [ s05 ] s06 | s07 > s08 > s09 > s10 > s11 > s12`

> 源码: [`go/s05-skill-loading/main.go`](../../s05-skill-loading/main.go)

---

## s05 vs s04 的核心区别

s04 引入了子 agent 来隔离上下文。s05 解决一个完全不同的问题: **如何高效地给模型注入专业知识**。

> **"Don't put everything in the system prompt. Load on demand."**

核心问题: 如果把所有知识都塞进 system prompt, token 成本高, 而且大部分知识在当前任务中用不到。解决方案: **两层注入**。

```
Layer 1 (便宜): skill 名称和简短描述放进 system prompt (~100 tokens/skill)
Layer 2 (按需): 模型调用 load_skill 时才注入完整内容 (~数千 tokens)
```

---

## 架构图

```
System prompt:                              tool_result:
+--------------------------------------+   +--------------------------------------+
| You are a coding agent.              |   | <skill name="code-review">           |
| Skills available:                    |   |   Full code review instructions...   |
|   - code-review: Perform thorough... |   |   Step 1: Security checks            |
|   - agent-builder: Build agents...   |   |   Step 2: Correctness                |
+--------------------------------------+   | </skill>                             |
        Layer 1: 名称 (~100 tokens)          Layer 2: 完整内容 (~数千 tokens)
        每次 API 调用都发送                    只在 load_skill 时发送一次
```

---

## 第 30-43 行: 导入

比 s04 多了两个:

```go
"regexp"    // 解析 YAML frontmatter 的正则
"sort"      // skill 名称排序, 确保输出确定性
```

---

## 第 45-146 行: `SkillLoader` — s05 的核心新增

### 第 47-55 行 — 数据结构

```go
type Skill struct {
    Name string
    Meta map[string]string   // frontmatter 的 key-value 对 (name, description, tags)
    Body string              // frontmatter 之后的完整内容
    Path string              // 文件路径, 用于调试
}

type SkillLoader struct {
    Skills map[string]*Skill  // name → Skill 的映射
}
```

每个 skill 由 `.md` 文件定义, 格式为:

```markdown
---
name: code-review
description: Perform thorough code reviews...
---

# Code Review Skill
(完整内容...)
```

### 第 58-67 行 — `NewSkillLoader()`: 支持两种目录布局

```go
func NewSkillLoader(dirs ...string) *SkillLoader {
    sl := &SkillLoader{Skills: make(map[string]*Skill)}
    for _, dir := range dirs {
        sl.loadDir(dir)
    }
    return sl
}
```

接受多个目录路径 (可变参数), 依次扫描。

### 第 69-94 行 — `loadDir()`: 两种文件布局

```go
func (sl *SkillLoader) loadDir(dir string) {
    entries, _ := os.ReadDir(dir)
    for _, e := range entries {
        if e.IsDir() {
            // 嵌套布局: skills/<name>/SKILL.md
            skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
            ...
        } else if strings.HasSuffix(e.Name(), ".md") {
            // 扁平布局: .skills/<name>.md
            name := strings.TrimSuffix(e.Name(), ".md")
            ...
        }
    }
}
```

- **扁平**: `.skills/git.md` → skill 名为 `git`
- **嵌套**: `skills/code-review/SKILL.md` → skill 名为 `code-review`

### 第 96-107 行 — `parseFrontmatter()`: 简易 YAML 解析

```go
var frontmatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)`)

func parseFrontmatter(text string) (map[string]string, string) {
    match := frontmatterRe.FindStringSubmatch(text)
    if match == nil {
        return map[string]string{}, text
    }
    meta := map[string]string{}
    for _, line := range strings.Split(strings.TrimSpace(match[1]), "\n") {
        if idx := strings.Index(line, ":"); idx >= 0 {
            key := strings.TrimSpace(line[:idx])
            val := strings.TrimSpace(line[idx+1:])
            meta[key] = val
        }
    }
    return meta, strings.TrimSpace(match[2])
}
```

正则含义:
- `(?s)` — dotall 模式, `.` 匹配换行
- `^---\n(.*?)\n---\n(.*)` — 两个 `---` 之间是 frontmatter, 之后是 body
- 不使用完整的 YAML 解析器, 只做简单的 `key: value` 行分割

### 第 109-130 行 — `GetDescriptions()`: Layer 1

```go
func (sl *SkillLoader) GetDescriptions() string {
    // 排序确保每次输出一致
    names := make([]string, 0, len(sl.Skills))
    for name := range sl.Skills { names = append(names, name) }
    sort.Strings(names)

    var lines []string
    for _, name := range names {
        skill := sl.Skills[name]
        desc := skill.Meta["description"]
        line := fmt.Sprintf("  - %s: %s", name, desc)
        lines = append(lines, line)
    }
    return strings.Join(lines, "\n")
}
```

输出嵌入 system prompt, **每次 API 调用都发送**, 但 token 成本低。

### 第 132-143 行 — `GetContent()`: Layer 2

```go
func (sl *SkillLoader) GetContent(name string) string {
    skill, ok := sl.Skills[name]
    if !ok {
        return fmt.Sprintf("Error: Unknown skill '%s'. Available: %s", ...)
    }
    return fmt.Sprintf("<skill name=%q>\n%s\n</skill>", name, skill.Body)
}
```

返回 `<skill>` XML 标签包裹的完整内容。通过 `tool_result` 注入, **只发送一次**。

---

## 第 148-176 行: 全局初始化

### System Prompt 的动态构造

```go
skillLoader = NewSkillLoader(
    filepath.Join(workdir, ".skills"),   // Python 版约定
    filepath.Join(workdir, "skills"),    // rubickx 实际结构
)

system = fmt.Sprintf(`You are a coding agent at %s.
Use load_skill to access specialized knowledge before tackling unfamiliar topics.

Skills available:
%s`, workdir, skillLoader.GetDescriptions())
```

System prompt 在启动时**一次性构造**, skill 目录在启动时**一次性扫描**。

### `load_skill` 工具定义

```go
{OfTool: &anthropic.ToolParam{
    Name:        "load_skill",
    Description: anthropic.String("Load specialized knowledge by name."),
    InputSchema: anthropic.ToolInputSchemaParam{
        Properties: map[string]interface{}{
            "name": map[string]interface{}{"type": "string", "description": "Skill name to load"},
        },
        Required: []string{"name"},
    },
}},
```

极简 schema: 只有一个 `name` 字符串参数。

---

## 第 259-266 行: `handleLoadSkill`

```go
func handleLoadSkill(raw json.RawMessage) string {
    var input struct {
        Name string `json:"name"`
    }
    _ = json.Unmarshal(raw, &input)
    return skillLoader.GetContent(input.Name)
}
```

整个 handler 只有 5 行: 解析 name → 调用 `GetContent()` → 返回完整 skill 内容。

---

## Agent Loop — 和 s02 完全一样

s05 的 agent loop **没有任何变化**。`load_skill` 通过 dispatch map 正常分发, 不需要特殊逻辑。这体现了 s02 dispatch 模式的扩展性。

---

## s04 → s05 变化对照表

| 维度 | s04 | s05 |
|------|-----|-----|
| 新增概念 | Subagent | **两层 Skill 注入** |
| System Prompt | 固定文本 | **动态构造** (包含 skill 列表) |
| 工具 | 5 个 (含 task) | 5 个 (含 load_skill, 去掉 task) |
| 新数据结构 | 无 | `SkillLoader`, `Skill`, frontmatter 解析 |
| Agent Loop | task 特殊处理 | **回归 s02 的纯 dispatch** |
| 文件 I/O | 运行时 | **启动时扫描** skill 目录 |

---

## 核心架构图

```
启动时:
  NewSkillLoader(".skills/", "skills/")
    ├── 扫描目录, 发现 skill 文件
    ├── parseFrontmatter() 提取 meta + body
    └── Skills = {"code-review": {Meta, Body}, ...}

  system = "... Skills available:\n" + GetDescriptions()
    → "  - agent-builder: Build AI coding agents..."
    → "  - code-review: Perform thorough code reviews..."

运行时:
  agentLoop()
    for {
        resp = LLM(system, tools)     ← system 包含 Layer 1 (skill 名称)
        │
        tool_use: load_skill("code-review")
        │
        dispatch["load_skill"](input)
            → skillLoader.GetContent("code-review")
            → "<skill name=\"code-review\">\n完整内容...\n</skill>"
        │
        tool_result 回传给模型         ← Layer 2 (完整内容) 按需注入
    }
```

**s05 的核心洞察**: System prompt 是昂贵的 (每次 API 调用都发送), tool_result 是便宜的 (只出现一次)。把 skill 的**元数据**放 system prompt (让模型知道有什么可用), 把**完整内容**放 tool_result (模型需要时才加载)。这就是 Claude Code 的 Skill 工具的工作原理。
