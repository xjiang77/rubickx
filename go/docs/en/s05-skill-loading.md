# s05: Skill Loading — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > [ s05 ] s06 | s07 > s08 > s09 > s10 > s11 > s12`

> Source: [`go/s05/main.go`](../../s05/main.go)

---

## Core Difference: s05 vs s04

s04 introduced subagents for context isolation. s05 tackles a completely different problem: **how to efficiently inject specialized knowledge into the model**.

> **"Don't put everything in the system prompt. Load on demand."**

Core problem: if all knowledge is stuffed into the system prompt, token cost is high and most knowledge is irrelevant to the current task. Solution: **two-layer injection**.

```
Layer 1 (cheap): skill names and short descriptions in the system prompt (~100 tokens/skill)
Layer 2 (on-demand): full content injected only when model calls load_skill (~thousands of tokens)
```

---

## Architecture Diagram

```
System prompt:                              tool_result:
+--------------------------------------+   +--------------------------------------+
| You are a coding agent.              |   | <skill name="code-review">           |
| Skills available:                    |   |   Full code review instructions...   |
|   - code-review: Perform thorough... |   |   Step 1: Security checks            |
|   - agent-builder: Build agents...   |   |   Step 2: Correctness                |
+--------------------------------------+   | </skill>                             |
        Layer 1: names (~100 tokens)          Layer 2: full content (~thousands of tokens)
        Sent with every API call              Sent only once via load_skill
```

---

## Lines 30-43: Imports

Two additions over s04:

```go
"regexp"    // Regex for parsing YAML frontmatter
"sort"      // Sort skill names for deterministic output
```

---

## Lines 45-146: `SkillLoader` — Core Addition of s05

### Lines 47-55 — Data Structures

```go
type Skill struct {
    Name string
    Meta map[string]string   // Frontmatter key-value pairs (name, description, tags)
    Body string              // Full content after frontmatter
    Path string              // File path for debugging
}

type SkillLoader struct {
    Skills map[string]*Skill  // name -> Skill mapping
}
```

Each skill is defined by an `.md` file with the format:

```markdown
---
name: code-review
description: Perform thorough code reviews...
---

# Code Review Skill
(full content...)
```

### Lines 58-67 — `NewSkillLoader()`: Supports Two Directory Layouts

```go
func NewSkillLoader(dirs ...string) *SkillLoader {
    sl := &SkillLoader{Skills: make(map[string]*Skill)}
    for _, dir := range dirs {
        sl.loadDir(dir)
    }
    return sl
}
```

Accepts multiple directory paths (variadic), scanning each in order.

### Lines 69-94 — `loadDir()`: Two File Layouts

```go
func (sl *SkillLoader) loadDir(dir string) {
    entries, _ := os.ReadDir(dir)
    for _, e := range entries {
        if e.IsDir() {
            // Nested layout: skills/<name>/SKILL.md
            skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
            ...
        } else if strings.HasSuffix(e.Name(), ".md") {
            // Flat layout: .skills/<name>.md
            name := strings.TrimSuffix(e.Name(), ".md")
            ...
        }
    }
}
```

- **Flat**: `.skills/git.md` -> skill named `git`
- **Nested**: `skills/code-review/SKILL.md` -> skill named `code-review`

### Lines 96-107 — `parseFrontmatter()`: Simple YAML Parsing

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

Regex breakdown:
- `(?s)` — dotall mode, `.` matches newlines
- `^---\n(.*?)\n---\n(.*)` — content between two `---` is frontmatter, everything after is body
- No full YAML parser used, just simple `key: value` line splitting

### Lines 109-130 — `GetDescriptions()`: Layer 1

```go
func (sl *SkillLoader) GetDescriptions() string {
    // Sort for consistent output each time
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

Output is embedded in the system prompt, **sent with every API call**, but token cost is low.

### Lines 132-143 — `GetContent()`: Layer 2

```go
func (sl *SkillLoader) GetContent(name string) string {
    skill, ok := sl.Skills[name]
    if !ok {
        return fmt.Sprintf("Error: Unknown skill '%s'. Available: %s", ...)
    }
    return fmt.Sprintf("<skill name=%q>\n%s\n</skill>", name, skill.Body)
}
```

Returns the full content wrapped in `<skill>` XML tags. Injected via `tool_result`, **sent only once**.

---

## Lines 148-176: Global Initialization

### Dynamic System Prompt Construction

```go
skillLoader = NewSkillLoader(
    filepath.Join(workdir, ".skills"),   // Python version convention
    filepath.Join(workdir, "skills"),    // rubickx actual structure
)

system = fmt.Sprintf(`You are a coding agent at %s.
Use load_skill to access specialized knowledge before tackling unfamiliar topics.

Skills available:
%s`, workdir, skillLoader.GetDescriptions())
```

The system prompt is **constructed once at startup**; the skill directory is **scanned once at startup**.

### `load_skill` Tool Definition

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

Minimal schema: just one `name` string parameter.

---

## Lines 259-266: `handleLoadSkill`

```go
func handleLoadSkill(raw json.RawMessage) string {
    var input struct {
        Name string `json:"name"`
    }
    _ = json.Unmarshal(raw, &input)
    return skillLoader.GetContent(input.Name)
}
```

The entire handler is just 5 lines: parse name -> call `GetContent()` -> return full skill content.

---

## Agent Loop — Identical to s02

s05's agent loop has **no changes whatsoever**. `load_skill` dispatches normally through the dispatch map with no special logic needed. This demonstrates the extensibility of the s02 dispatch pattern.

---

## s04 -> s05 Change Comparison

| Dimension | s04 | s05 |
|-----------|-----|-----|
| New concept | Subagent | **Two-layer Skill injection** |
| System Prompt | Static text | **Dynamically constructed** (includes skill list) |
| Tools | 5 (including task) | 5 (including load_skill, task removed) |
| New data structures | None | `SkillLoader`, `Skill`, frontmatter parsing |
| Agent Loop | task handled specially | **Returns to s02's pure dispatch** |
| File I/O | Runtime | **Startup-time** skill directory scan |

---

## Core Architecture Diagram

```
At startup:
  NewSkillLoader(".skills/", "skills/")
    +-- Scan directories, discover skill files
    +-- parseFrontmatter() extracts meta + body
    +-- Skills = {"code-review": {Meta, Body}, ...}

  system = "... Skills available:\n" + GetDescriptions()
    -> "  - agent-builder: Build AI coding agents..."
    -> "  - code-review: Perform thorough code reviews..."

At runtime:
  agentLoop()
    for {
        resp = LLM(system, tools)     <- system contains Layer 1 (skill names)
        |
        tool_use: load_skill("code-review")
        |
        dispatch["load_skill"](input)
            -> skillLoader.GetContent("code-review")
            -> "<skill name=\"code-review\">\nFull content...\n</skill>"
        |
        tool_result sent back to model  <- Layer 2 (full content) injected on-demand
    }
```

**s05's core insight**: The system prompt is expensive (sent with every API call); tool_result is cheap (appears only once). Put skill **metadata** in the system prompt (so the model knows what's available), and **full content** in tool_result (loaded only when the model needs it). This is how Claude Code's Skill tool works.
