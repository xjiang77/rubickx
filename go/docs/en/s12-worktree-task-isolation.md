# s12: Worktree + Task Isolation — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > [ s12 ]`

> Source: [`go/s12-worktree-task-isolation/main.go`](../../s12-worktree-task-isolation/main.go)

---

## Core Difference: s12 vs s11

s11 implemented autonomous agent teams, but all tasks share a single directory. s12 returns to a single-agent architecture, adding **worktree directory isolation**: each task is bound to an independent git worktree, preventing interference.

The architecture shifts from "team collaboration" to "control plane + execution plane":

```
Control plane (.tasks/)     Execution plane (.worktrees/)
  task_1.json  <------>     auth-refactor/ (branch: wt/auth-refactor)
  task_2.json  <------>     ui-login/      (branch: wt/ui-login)
                            index.json     (registry)
                            events.jsonl   (lifecycle log)
```

> **"Isolate by directory, coordinate by task ID."**

---

## EventBus Data Structure

```go
type EventBus struct {
    path string
    mu   sync.Mutex
}

type eventPayload struct {
    Event    string      `json:"event"`
    Ts       float64     `json:"ts"`
    Task     interface{} `json:"task"`
    Worktree interface{} `json:"worktree"`
    Error    string      `json:"error,omitempty"`
}
```

EventBus is an append-only JSONL event log. Every lifecycle step (create/remove/keep) writes one event:

- `Emit()` appends a line of JSON to `events.jsonl`
- `ListRecent(limit)` reads the last N lines, capped at 200

Event types: `worktree.create.before/after/failed`, `worktree.remove.before/after/failed`, `worktree.keep`, `task.completed`.

`sync.Mutex` protects writes; reads are also locked to prevent reading half-written lines during concurrent writes.

---

## TaskManager: Task Board with Worktree Binding

```go
type taskData struct {
    ID          int       `json:"id"`
    Subject     string    `json:"subject"`
    Description string    `json:"description"`
    Status      string    `json:"status"`
    Owner       string    `json:"owner"`
    Worktree    string    `json:"worktree"`   // <- New: bound worktree name
    BlockedBy   []int     `json:"blockedBy"`
    CreatedAt   float64   `json:"created_at"`
    UpdatedAt   float64   `json:"updated_at"`
}
```

Compared to s07's TaskManager, s12 adds a `Worktree` field and two methods:

- **`BindWorktree(id, worktree, owner)`**: sets the `worktree` field and advances `pending` to `in_progress`
- **`UnbindWorktree(id)`**: clears the `worktree` field

```go
func (tm *TaskManager) BindWorktree(id int, worktree, owner string) (string, error) {
    task, err := tm.load(id)
    task.Worktree = worktree
    if owner != "" { task.Owner = owner }
    if task.Status == "pending" {
        task.Status = "in_progress"      // <- Binding auto-advances status
    }
    tm.save(task)
}
```

Auto-advancing status on binding is a key design: the moment a worktree is created, the task has "started."

---

## WorktreeManager: Git Worktree Lifecycle

```go
type worktreeEntry struct {
    Name      string  `json:"name"`
    Path      string  `json:"path"`
    Branch    string  `json:"branch"`       // wt/<name>
    TaskID    *int    `json:"task_id"`       // Optional binding
    Status    string  `json:"status"`        // active | removed | kept
    CreatedAt float64 `json:"created_at,omitempty"`
    RemovedAt float64 `json:"removed_at,omitempty"`
    KeptAt    float64 `json:"kept_at,omitempty"`
}

type WorktreeManager struct {
    repoRoot     string
    dir          string           // .worktrees/
    indexPath    string           // .worktrees/index.json
    tasks        *TaskManager     // Control plane reference
    events       *EventBus        // Event log reference
    gitAvailable bool
    mu           sync.Mutex
}
```

WorktreeManager holds references to TaskManager and EventBus, forming a triangular dependency:

```
WorktreeManager
   +-- tasks  (TaskManager)   <- bind/unbind
   +-- events (EventBus)      <- lifecycle logging
```

**Name validation**: `regexp.MustCompile("^[A-Za-z0-9._-]{1,40}$")` — prevents path injection.

**`TaskID *int`**: pointer used to distinguish "not provided" from "provided as 0". More precise than Python's `None` check.

---

## `Create()` — Create Worktree

```go
func (wm *WorktreeManager) Create(name string, taskID *int, baseRef string) (string, error) {
    wm.validateName(name)                         // Regex validation
    if wm.find(name) != nil { ... }               // Index dedup check
    if taskID != nil && !wm.tasks.Exists(*taskID)  // Task existence check

    wm.events.Emit("worktree.create.before", ...)

    // git worktree add -b wt/<name> .worktrees/<name> HEAD
    wm.runGit("worktree", "add", "-b", branch, path, baseRef)

    // Write to index.json
    idx.Worktrees = append(idx.Worktrees, entry)
    wm.saveIndex(idx)

    // Bind task
    if taskID != nil { wm.tasks.BindWorktree(*taskID, name, "") }

    wm.events.Emit("worktree.create.after", ...)
}
```

Branch naming convention: `wt/<name>` — simple prefix isolation, easy to identify which branches were created by worktrees.

On failure, emits a `worktree.create.failed` event with error info for post-mortem auditing.

---

## `Remove()` — Teardown + Optional Task Completion

```go
func (wm *WorktreeManager) Remove(name string, force, completeTask bool) (string, error) {
    wm.events.Emit("worktree.remove.before", ...)

    // git worktree remove [--force] <path>
    wm.runGit(args...)

    // One call handles three things
    if completeTask && wt.TaskID != nil {
        wm.tasks.Update(taskID, "completed", "")
        wm.tasks.UnbindWorktree(taskID)
        wm.events.Emit("task.completed", ...)
    }

    // Update index: status = "removed"
    idx.Worktrees[i].Status = "removed"
    idx.Worktrees[i].RemovedAt = now

    wm.events.Emit("worktree.remove.after", ...)
}
```

`completeTask=true` is the key parameter — without it, only the directory is deleted and the task status remains unchanged. One call handles: delete directory + complete task + emit event.

---

## `Keep()` — Preserve Worktree

```go
idx.Worktrees[i].Status = "kept"
idx.Worktrees[i].KeptAt = now
```

`keep` only changes the index status; the directory is not deleted. Used for "done but want to keep the code for reference" scenarios.

---

## `Run()` — Execute Commands in Worktree

```go
func (wm *WorktreeManager) Run(name, command string) string {
    cmd := exec.CommandContext(ctx, "sh", "-c", command)
    cmd.Dir = wt.Path       // <- cwd points to worktree directory
}
```

The only difference from `runBash()` is `cmd.Dir` points to the worktree path instead of workDir. This achieves directory-level isolation.

Note that `Run()` doesn't hold the lock (`wm.mu`) — it only briefly holds the lock during `find()` to read the index, releasing the lock during command execution to allow multiple worktrees to execute concurrently.

---

## `detectRepoRoot()` — Find Git Repository Root

```go
func detectRepoRoot(cwd string) string {
    cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
    cmd.Dir = cwd
    // ...
}
```

Worktrees must be created under `.worktrees/` at the repository root, not in subdirectories.

---

## 16 Tools: 4 + 5 + 7

```
Base (4):       bash, read_file, write_file, edit_file
Task (5):       task_create, task_list, task_get, task_update, task_bind_worktree
Worktree (7):   worktree_create, worktree_list, worktree_status,
                worktree_run, worktree_remove, worktree_keep, worktree_events
```

The dispatch map uses the same `map[string]func(json.RawMessage) string` pattern, with 16 handlers in one-to-one correspondence.

---

## Initialization Order: TaskManager -> EventBus -> WorktreeManager

```go
tasks     = NewTaskManager(filepath.Join(repoRoot, ".tasks"))
events    = NewEventBus(filepath.Join(repoRoot, ".worktrees", "events.jsonl"))
worktrees = NewWorktreeManager(repoRoot, tasks, events)
```

Order matters: WorktreeManager depends on tasks and events, so it must be created last.

---

## Shortcut Commands

```go
if query == "/tasks"     { fmt.Println(tasks.ListAll()) }
if query == "/worktrees" { fmt.Println(worktrees.ListAll()) }
if query == "/events"    { fmt.Println(events.ListRecent(20)) }
```

Three slash commands for directly viewing the control plane + execution plane + event stream.

---

## s11 -> s12 Change Comparison

| Dimension | s11 | s12 |
|-----------|-----|-----|
| Architecture | Team (lead + teammates) | **Single agent** |
| New concept | Autonomy + idle cycle | **Worktree isolation + EventBus** |
| Coordination | Task board (owner/status) | **Task board + explicit worktree binding** |
| Execution scope | Shared directory | **Each task gets an independent git worktree** |
| Completion | Task done | **Task done + keep/remove** |
| Observability | Implicit logs | **EventBus JSONL event stream** |
| Tool count | 14 | **16** |
| Components | MessageBus + TeammateManager | **EventBus + WorktreeManager** |
| State machine | teammate: working/idle/shutdown | **worktree: active/removed/kept** |

---

## Core Architecture Diagram

```
                 Agent Loop (single agent)
                        |
            +-----------+---------------+
            |           |               |
    Control Plane   Execution Plane   Observability
    (.tasks/)       (.worktrees/)     (events.jsonl)
            |           |               |
    task_create     worktree_create     |
    task_update     worktree_run     Emit()
    task_bind_wt    worktree_remove     |
    task_get        worktree_keep    ListRecent()
    task_list       worktree_list
                    worktree_status
                    worktree_events --------+

Binding:
  task_1.json { worktree: "auth-work" }
       <=>
  index.json { name: "auth-work", task_id: 1 }
```

**s12's core insight**: Separation of control plane (tasks) and execution plane (worktrees) + bidirectional binding via task ID. After a crash, the scene can be reconstructed from `.tasks/` + `.worktrees/index.json`. Session memory is volatile; disk state is persistent.
