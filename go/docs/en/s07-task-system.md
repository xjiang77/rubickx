# s07: Task System — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > s05 > s06 | [ s07 ] s08 > s09 > s10 > s11 > s12`

> Source: [`go/s07-task-system/main.go`](../../s07-task-system/main.go)

---

## Core Difference: s07 vs s06

s06 solved "what to do when the context is too long." s07 solves the opposite problem: **what happens when state is lost after compression**.

> **"State that survives compression — because it's outside the conversation."**

Core problem: s03's TodoManager only lives in memory (`[]TodoItem`); once context compression (s06) runs, the checklist is gone. Also, a flat checklist has no dependency relationships — it can't distinguish what's ready from what's blocked. Solution: **a persistent task graph**.

```
s03 TodoManager (memory)           s07 TaskManager (disk)
+-------------------+           .tasks/
| []TodoItem        |             task_1.json  {"status":"completed"}
| Lost on restart/  |             task_2.json  {"blockedBy":[1]}
|   compression     |             task_3.json  {"blockedBy":[2]}
| No dependencies   |           Survives forever, has dependency graph
+-------------------+
```

---

## Lines 42-54: Task Data Structure

```go
type Task struct {
    ID          int    `json:"id"`
    Subject     string `json:"subject"`
    Description string `json:"description"`
    Status      string `json:"status"`      // pending | in_progress | completed
    BlockedBy   []int  `json:"blockedBy"`   // Upstream deps (who blocks me)
    Blocks      []int  `json:"blocks"`      // Downstream deps (who I block)
    Owner       string `json:"owner"`       // Reserved for multi-agent
}
```

Key fields:
- `BlockedBy`: upstream dependency list. `[1, 3]` means tasks 1 and 3 must complete before the current task is unblocked
- `Blocks`: downstream dependency list. Maintained bidirectionally for easy querying of "who gets unblocked when I finish"

---

## Lines 57-62: TaskManager Structure

```go
type TaskManager struct {
    dir    string      // .tasks/ directory path
    nextID int         // Next available ID
    mu     sync.Mutex  // Concurrency protection
}
```

Has `sync.Mutex` compared to the Python version, preparing for s08 (multi-goroutine background execution).

---

## Lines 64-72: `NewTaskManager()` — Initialization

```go
func NewTaskManager(dir string) *TaskManager {
    _ = os.MkdirAll(dir, 0o755)           // Ensure directory exists
    tm := &TaskManager{dir: dir}
    tm.nextID = tm.maxID() + 1            // Scan existing files, continue numbering
    return tm
}
```

`maxID()` scans `.tasks/task_*.json` files to find the maximum ID. Called once at startup to ensure IDs don't repeat.

---

## Lines 86-99: `load()` and `save()` — File I/O

```go
func (tm *TaskManager) load(taskID int) (*Task, error) {
    path := filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", taskID))
    data, err := os.ReadFile(path)
    // ...
    var task Task
    json.Unmarshal(data, &task)
    return &task, nil
}

func (tm *TaskManager) save(task *Task) error {
    data, _ := json.MarshalIndent(task, "", "  ")
    path := filepath.Join(tm.dir, fmt.Sprintf("task_%d.json", task.ID))
    return os.WriteFile(path, data, 0o644)
}
```

One task = one JSON file. Filename format: `task_{id}.json`. `MarshalIndent` produces human-readable JSON.

---

## Lines 101-115: `Create()` — Create Task

```go
func (tm *TaskManager) Create(subject, description string) string {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    task := &Task{
        ID:        tm.nextID,
        Subject:   subject,
        Status:    "pending",
        BlockedBy: []int{},   // Empty slice, serializes as [] not null
        Blocks:    []int{},
    }
    tm.save(task)
    tm.nextID++
    data, _ := json.MarshalIndent(task, "", "  ")
    return string(data)
}
```

Note `[]int{}` instead of `nil`: ensures JSON output is `"blockedBy": []` rather than `"blockedBy": null`.

---

## Lines 123-155: `Update()` — Status + Dependency Updates

```go
func (tm *TaskManager) Update(taskID int, status string, addBlockedBy, addBlocks []int) string {
    task, err := tm.load(taskID)
    // ...

    if status != "" {
        switch status {
        case "pending", "in_progress", "completed":
            task.Status = status
            if status == "completed" {
                tm.clearDependency(taskID)      // <- Key: auto-unlock
            }
        default:
            return "Error: Invalid status"
        }
    }

    if len(addBlocks) > 0 {
        task.Blocks = uniqueAppend(task.Blocks, addBlocks)
        // Bidirectional maintenance: also update blocked task's blockedBy
        for _, blockedID := range addBlocks {
            blocked, _ := tm.load(blockedID)
            if !containsInt(blocked.BlockedBy, taskID) {
                blocked.BlockedBy = append(blocked.BlockedBy, taskID)
                tm.save(blocked)
            }
        }
    }

    tm.save(task)
    return string(json.MarshalIndent(task))
}
```

Three update dimensions:
1. **Status transition**: `pending` -> `in_progress` -> `completed`
2. **Add upstream deps**: `addBlockedBy` — "who blocks me"
3. **Add downstream deps**: `addBlocks` — "who I block" (bidirectional maintenance)

---

## Lines 157-172: `clearDependency()` — Dependency Removal

```go
func (tm *TaskManager) clearDependency(completedID int) {
    entries, _ := os.ReadDir(tm.dir)
    for _, e := range entries {
        // ... read each task file ...
        if containsInt(task.BlockedBy, completedID) {
            task.BlockedBy = removeInt(task.BlockedBy, completedID)
            tm.save(&task)
        }
    }
}
```

Iterates all task files, removing `completedID` from their `blockedBy`. Effect:

```
Before completion:               After completion:
task 2: blockedBy=[1]           task 2: blockedBy=[]     <- unblocked!
task 3: blockedBy=[1,4]         task 3: blockedBy=[4]    <- partially unblocked
```

---

## Lines 174-210: `ListAll()` — Task List

```go
func (tm *TaskManager) ListAll() string {
    // Scan .tasks/ directory, read all task files
    // Sort by ID
    sort.Slice(tasks, func(i, j int) bool { return tasks[i].ID < tasks[j].ID })

    for _, t := range tasks {
        marker := "[?]"
        switch t.Status {
        case "pending":     marker = "[ ]"
        case "in_progress": marker = "[>]"
        case "completed":   marker = "[x]"
        }
        line := fmt.Sprintf("%s #%d: %s", marker, t.ID, t.Subject)
        if len(t.BlockedBy) > 0 {
            line += fmt.Sprintf(" (blocked by: %v)", t.BlockedBy)
        }
    }
    return strings.Join(lines, "\n")
}
```

Example output:
```
[x] #1: Setup
[ ] #2: Code (blocked by: [1])
[ ] #3: Test (blocked by: [2])
```

---

## Lines 212-234: Integer Set Helper Functions

```go
func containsInt(slice []int, val int) bool { ... }
func removeInt(slice []int, val int) []int { ... }
func uniqueAppend(existing, additions []int) []int { ... }
```

Go lacks built-in set operations, requiring manual implementation. Python handles this with `if x in list` and `list.remove()` in one line.

---

## Lines 240-311: Global Initialization — 8 Tools

```go
func init() {
    tasks = NewTaskManager(filepath.Join(workdir, ".tasks"))
    system = "You are a coding agent at %s. Use task tools to plan and track work."

    tools = []anthropic.ToolUnionParam{
        // 4 base tools: bash, read_file, write_file, edit_file
        // 4 task tools: task_create, task_update, task_list, task_get
    }

    dispatch = map[string]toolHandler{
        "bash":        handleBash,
        // ...
        "task_create": handleTaskCreate,
        "task_update": handleTaskUpdate,
        "task_list":   handleTaskList,
        "task_get":    handleTaskGet,
    }
}
```

Grew from s05's 5 tools to 8. The dispatch map's extensibility is demonstrated again — just add entries, no agent loop changes.

---

## `task_update` Tool Schema — Notable Details

```go
"task_id": map[string]interface{}{"type": "integer"},
"status": map[string]interface{}{
    "type": "string",
    "enum": []string{"pending", "in_progress", "completed"},
},
"addBlockedBy": map[string]interface{}{
    "type":  "array",
    "items": map[string]interface{}{"type": "integer"},
},
```

- `enum` constrains status values, preventing the model from passing invalid statuses
- `array` + `items` declares an integer array type

---

## Agent Loop — Identical to s02

s07's agent loop has **no changes whatsoever**. All task operations dispatch normally through the dispatch map with no special logic. This is the sixth reuse of the s02 dispatch pattern.

---

## s06 -> s07 Change Comparison

| Dimension | s06 | s07 |
|-----------|-----|-----|
| New concept | Three-layer context compression | **Persistent task graph** |
| Tools | 5 (including compact) | **8** (including task_create/update/list/get) |
| State storage | In-memory messages | **Disk** .tasks/*.json |
| Dependencies | None | **blockedBy + blocks bidirectional edges** |
| New data structures | toolResultLoc | **Task, TaskManager** |
| Concurrency safety | None | **sync.Mutex** |
| Agent Loop | Pointer, special compact handling | **Returns to s02's pure dispatch** |

---

## Core Architecture Diagram

```
At startup:
  NewTaskManager(".tasks/")
    +-- os.MkdirAll ensures directory exists
    +-- maxID() scans existing files, determines nextID

At runtime:
  agentLoop()
    for {
        resp = LLM(system, tools)
        |
        tool_use: task_create("Setup")
        |   -> tasks.Create("Setup")
        |   -> Write .tasks/task_1.json
        |   -> Return JSON
        |
        tool_use: task_update(2, addBlockedBy=[1])
        |   -> tasks.Update(2, ...)
        |   -> task_2.blockedBy = [1]
        |   -> Bidirectional: task_1.blocks += [2]
        |
        tool_use: task_update(1, status="completed")
        |   -> tasks.Update(1, "completed")
        |   -> clearDependency(1):
        |       Iterate all tasks, remove 1 from blockedBy
        |       task_2.blockedBy: [1] -> []  <- unblocked!
        |
        tool_use: task_list
            -> tasks.ListAll()
            -> "[x] #1: Setup\n[ ] #2: Code"
    }
```

**s07's core insight**: An agent's planning state can't live solely in the conversation. Once the context is compressed or the agent restarts, in-memory state vanishes. Persisting the task graph to disk means state is never lost. This lays the groundwork for multi-agent collaboration (s09+) and worktree isolation (s12) with a shared state foundation.
