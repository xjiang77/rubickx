# s12: Worktree + Task Isolation — Go 逐行解读

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > s10 > s11 > [ s12 ]`

> 源码: [`go/s12/main.go`](../../go/s12/main.go)

---

## s12 vs s11 的核心区别

s11 实现了自治智能体团队, 但所有任务共享一个目录。s12 回归单 agent 架构, 加入 **worktree 目录隔离**: 每个任务绑定独立的 git worktree, 互不干扰。

架构从"团队协作"转向"控制平面 + 执行平面":

```
Control plane (.tasks/)     Execution plane (.worktrees/)
  task_1.json  <------>     auth-refactor/ (branch: wt/auth-refactor)
  task_2.json  <------>     ui-login/      (branch: wt/ui-login)
                            index.json     (registry)
                            events.jsonl   (lifecycle log)
```

> **"Isolate by directory, coordinate by task ID."**

---

## EventBus 数据结构

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

EventBus 是 append-only 的 JSONL 事件日志。每个生命周期步骤 (create/remove/keep) 都会写入一条事件:

- `Emit()` 追加一行 JSON 到 `events.jsonl`
- `ListRecent(limit)` 读取最后 N 行, 上限 200

事件类型: `worktree.create.before/after/failed`, `worktree.remove.before/after/failed`, `worktree.keep`, `task.completed`。

`sync.Mutex` 保护写入, 读取也加锁防止并发写入期间读到半行数据。

---

## TaskManager: 带 worktree 绑定的任务板

```go
type taskData struct {
    ID          int       `json:"id"`
    Subject     string    `json:"subject"`
    Description string    `json:"description"`
    Status      string    `json:"status"`
    Owner       string    `json:"owner"`
    Worktree    string    `json:"worktree"`   // ← 新增: 绑定的 worktree 名
    BlockedBy   []int     `json:"blockedBy"`
    CreatedAt   float64   `json:"created_at"`
    UpdatedAt   float64   `json:"updated_at"`
}
```

相比 s07 的 TaskManager, s12 新增了 `Worktree` 字段和两个方法:

- **`BindWorktree(id, worktree, owner)`**: 设置 `worktree` 字段, 同时将 `pending` 推进到 `in_progress`
- **`UnbindWorktree(id)`**: 清空 `worktree` 字段

```go
func (tm *TaskManager) BindWorktree(id int, worktree, owner string) (string, error) {
    task, err := tm.load(id)
    task.Worktree = worktree
    if owner != "" { task.Owner = owner }
    if task.Status == "pending" {
        task.Status = "in_progress"      // ← 绑定自动推进状态
    }
    tm.save(task)
}
```

绑定自动推进状态是关键设计: 创建 worktree 的那一刻, 任务就"开始了"。

---

## WorktreeManager: git worktree 生命周期

```go
type worktreeEntry struct {
    Name      string  `json:"name"`
    Path      string  `json:"path"`
    Branch    string  `json:"branch"`       // wt/<name>
    TaskID    *int    `json:"task_id"`       // 可选绑定
    Status    string  `json:"status"`        // active | removed | kept
    CreatedAt float64 `json:"created_at,omitempty"`
    RemovedAt float64 `json:"removed_at,omitempty"`
    KeptAt    float64 `json:"kept_at,omitempty"`
}

type WorktreeManager struct {
    repoRoot     string
    dir          string           // .worktrees/
    indexPath    string           // .worktrees/index.json
    tasks        *TaskManager     // 控制平面引用
    events       *EventBus        // 事件日志引用
    gitAvailable bool
    mu           sync.Mutex
}
```

WorktreeManager 持有 TaskManager 和 EventBus 的引用, 形成三角依赖:

```
WorktreeManager
   ├── tasks  (TaskManager)   ← 绑定/解绑
   └── events (EventBus)      ← 生命周期日志
```

**名称验证**: `regexp.MustCompile("^[A-Za-z0-9._-]{1,40}$")` — 防止路径注入。

**`TaskID *int`**: 用指针区分"未传"和"传了 0"。这比 Python 的 `None` 检查更精确。

---

## `Create()` — 创建 worktree

```go
func (wm *WorktreeManager) Create(name string, taskID *int, baseRef string) (string, error) {
    wm.validateName(name)                         // 正则校验
    if wm.find(name) != nil { ... }               // index 查重
    if taskID != nil && !wm.tasks.Exists(*taskID)  // 任务存在检查

    wm.events.Emit("worktree.create.before", ...)

    // git worktree add -b wt/<name> .worktrees/<name> HEAD
    wm.runGit("worktree", "add", "-b", branch, path, baseRef)

    // 写入 index.json
    idx.Worktrees = append(idx.Worktrees, entry)
    wm.saveIndex(idx)

    // 绑定任务
    if taskID != nil { wm.tasks.BindWorktree(*taskID, name, "") }

    wm.events.Emit("worktree.create.after", ...)
}
```

branch 命名规则: `wt/<name>` — 简单前缀隔离, 容易识别哪些 branch 是 worktree 产生的。

失败时 emit `worktree.create.failed` 事件, 带 error 信息, 便于事后审计。

---

## `Remove()` — 拆除 + 可选任务完成

```go
func (wm *WorktreeManager) Remove(name string, force, completeTask bool) (string, error) {
    wm.events.Emit("worktree.remove.before", ...)

    // git worktree remove [--force] <path>
    wm.runGit(args...)

    // 一个调用搞定三件事
    if completeTask && wt.TaskID != nil {
        wm.tasks.Update(taskID, "completed", "")
        wm.tasks.UnbindWorktree(taskID)
        wm.events.Emit("task.completed", ...)
    }

    // 更新 index: status = "removed"
    idx.Worktrees[i].Status = "removed"
    idx.Worktrees[i].RemovedAt = now

    wm.events.Emit("worktree.remove.after", ...)
}
```

`completeTask=true` 是关键参数 — 不传则只删目录, 任务状态不变。一个调用搞定: 删目录 + 完成任务 + 发事件。

---

## `Keep()` — 保留 worktree

```go
idx.Worktrees[i].Status = "kept"
idx.Worktrees[i].KeptAt = now
```

`keep` 只改 index 状态, 不删目录。用于"做完了但想保留代码供参考"的场景。

---

## `Run()` — 在 worktree 中执行命令

```go
func (wm *WorktreeManager) Run(name, command string) string {
    cmd := exec.CommandContext(ctx, "sh", "-c", command)
    cmd.Dir = wt.Path       // ← cwd 指向 worktree 目录
}
```

与 `runBash()` 的唯一区别是 `cmd.Dir` 指向 worktree 路径而非 workDir。这实现了目录级别的隔离。

注意 `Run()` 不持锁 (`wm.mu`) — 只在 `find()` 时短暂持锁读取 index, 执行命令时释放锁, 允许多个 worktree 并发执行。

---

## `detectRepoRoot()` — 找到 git 仓库根

```go
func detectRepoRoot(cwd string) string {
    cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
    cmd.Dir = cwd
    // ...
}
```

worktree 必须创建在仓库根下的 `.worktrees/` 中, 不能在子目录下创建。

---

## 16 个工具: 4 + 5 + 7

```
Base (4):       bash, read_file, write_file, edit_file
Task (5):       task_create, task_list, task_get, task_update, task_bind_worktree
Worktree (7):   worktree_create, worktree_list, worktree_status,
                worktree_run, worktree_remove, worktree_keep, worktree_events
```

Dispatch map 同样采用 `map[string]func(json.RawMessage) string` 模式, 16 个 handler 一一对应。

---

## 初始化顺序: TaskManager → EventBus → WorktreeManager

```go
tasks     = NewTaskManager(filepath.Join(repoRoot, ".tasks"))
events    = NewEventBus(filepath.Join(repoRoot, ".worktrees", "events.jsonl"))
worktrees = NewWorktreeManager(repoRoot, tasks, events)
```

顺序很重要: WorktreeManager 依赖 tasks 和 events, 所以必须最后创建。

---

## 快捷命令

```go
if query == "/tasks"     { fmt.Println(tasks.ListAll()) }
if query == "/worktrees" { fmt.Println(worktrees.ListAll()) }
if query == "/events"    { fmt.Println(events.ListRecent(20)) }
```

三个 slash 命令直接查看控制平面 + 执行平面 + 事件流。

---

## s11 → s12 变化对照表

| 维度 | s11 | s12 |
|------|-----|-----|
| 架构 | 团队 (lead + teammates) | **单 agent** |
| 新增概念 | 自治 + idle cycle | **worktree 隔离 + EventBus** |
| 协调 | 任务板 (owner/status) | **任务板 + worktree 显式绑定** |
| 执行范围 | 共享目录 | **每个任务独立 git worktree** |
| 收尾 | 任务完成 | **任务完成 + keep/remove** |
| 可见性 | 隐式日志 | **EventBus JSONL 事件流** |
| 工具数 | 14 | **16** |
| 组件 | MessageBus + TeammateManager | **EventBus + WorktreeManager** |
| 状态机 | teammate: working/idle/shutdown | **worktree: active/removed/kept** |

---

## 核心架构图

```
                 Agent Loop (single agent)
                        │
            ┌───────────┼───────────────┐
            │           │               │
    Control Plane   Execution Plane   Observability
    (.tasks/)       (.worktrees/)     (events.jsonl)
            │           │               │
    task_create     worktree_create     │
    task_update     worktree_run     Emit()
    task_bind_wt    worktree_remove     │
    task_get        worktree_keep    ListRecent()
    task_list       worktree_list
                    worktree_status
                    worktree_events ──────┘

Binding:
  task_1.json { worktree: "auth-work" }
       ↕
  index.json { name: "auth-work", task_id: 1 }
```

**s12 的核心洞察**: 控制平面 (任务) 和执行平面 (worktree) 的分离 + 通过 task ID 双向绑定。崩溃后从 `.tasks/` + `.worktrees/index.json` 重建现场。会话记忆是易失的; 磁盘状态是持久的。
