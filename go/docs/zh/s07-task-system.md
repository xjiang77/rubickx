# s07: Task System — Go 逐行解读

`s01 > s02 > s03 > s04 > s05 > s06 | [ s07 ] s08 > s09 > s10 > s11 > s12`

> 源码: [`go/s07/main.go`](../../s07/main.go)

---

## s07 vs s06 的核心区别

s06 解决了"上下文太长怎么办"。s07 解决相反的问题: **压缩后状态丢了怎么办**。

> **"State that survives compression — because it's outside the conversation."**

核心问题: s03 的 TodoManager 只活在内存里 (`[]TodoItem`), 上下文压缩 (s06) 一跑, 清单就没了。而且扁平清单没有依赖关系, 分不清什么能做、什么被卡住。解决方案: **持久化的任务图**。

```
s03 TodoManager (内存)           s07 TaskManager (磁盘)
+-------------------+           .tasks/
| []TodoItem        |             task_1.json  {"status":"completed"}
| 重启/压缩后消失    |             task_2.json  {"blockedBy":[1]}
| 无依赖关系         |             task_3.json  {"blockedBy":[2]}
+-------------------+           永久存活, 有依赖图
```

---

## 第 42-54 行: Task 数据结构

```go
type Task struct {
    ID          int    `json:"id"`
    Subject     string `json:"subject"`
    Description string `json:"description"`
    Status      string `json:"status"`      // pending | in_progress | completed
    BlockedBy   []int  `json:"blockedBy"`   // 上游依赖 (谁挡着我)
    Blocks      []int  `json:"blocks"`      // 下游依赖 (我挡着谁)
    Owner       string `json:"owner"`       // 预留给多 agent
}
```

关键字段:
- `BlockedBy`: 上游依赖列表。`[1, 3]` 表示 task 1 和 task 3 完成前, 当前任务被阻塞
- `Blocks`: 下游依赖列表。双向维护, 方便查询"我完成后能解锁谁"

---

## 第 57-62 行: TaskManager 结构

```go
type TaskManager struct {
    dir    string      // .tasks/ 目录路径
    nextID int         // 下一个可用 ID
    mu     sync.Mutex  // 并发保护
}
```

比 Python 版多了 `sync.Mutex`, 为后续 s08 (多 goroutine 后台执行) 做准备。

---

## 第 64-72 行: `NewTaskManager()` — 初始化

```go
func NewTaskManager(dir string) *TaskManager {
    _ = os.MkdirAll(dir, 0o755)           // 确保目录存在
    tm := &TaskManager{dir: dir}
    tm.nextID = tm.maxID() + 1            // 扫描已有文件, 接续编号
    return tm
}
```

`maxID()` 扫描 `.tasks/task_*.json` 文件, 找出最大 ID。启动时调用一次, 确保 ID 不重复。

---

## 第 86-99 行: `load()` 和 `save()` — 文件 I/O

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

一个 task = 一个 JSON 文件。文件名格式: `task_{id}.json`。`MarshalIndent` 产生人类可读的 JSON。

---

## 第 101-115 行: `Create()` — 创建任务

```go
func (tm *TaskManager) Create(subject, description string) string {
    tm.mu.Lock()
    defer tm.mu.Unlock()

    task := &Task{
        ID:        tm.nextID,
        Subject:   subject,
        Status:    "pending",
        BlockedBy: []int{},   // 空切片, 序列化为 [] 而非 null
        Blocks:    []int{},
    }
    tm.save(task)
    tm.nextID++
    data, _ := json.MarshalIndent(task, "", "  ")
    return string(data)
}
```

注意 `[]int{}` 而非 `nil`: 确保 JSON 输出是 `"blockedBy": []` 而非 `"blockedBy": null`。

---

## 第 123-155 行: `Update()` — 状态 + 依赖更新

```go
func (tm *TaskManager) Update(taskID int, status string, addBlockedBy, addBlocks []int) string {
    task, err := tm.load(taskID)
    // ...

    if status != "" {
        switch status {
        case "pending", "in_progress", "completed":
            task.Status = status
            if status == "completed" {
                tm.clearDependency(taskID)      // ← 关键: 自动解锁
            }
        default:
            return "Error: Invalid status"
        }
    }

    if len(addBlocks) > 0 {
        task.Blocks = uniqueAppend(task.Blocks, addBlocks)
        // 双向维护: 也更新被阻塞任务的 blockedBy
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

三个更新维度:
1. **状态转换**: `pending` → `in_progress` → `completed`
2. **添加上游依赖**: `addBlockedBy` — "谁挡着我"
3. **添加下游依赖**: `addBlocks` — "我挡着谁" (双向维护)

---

## 第 157-172 行: `clearDependency()` — 依赖解除

```go
func (tm *TaskManager) clearDependency(completedID int) {
    entries, _ := os.ReadDir(tm.dir)
    for _, e := range entries {
        // ... 读取每个 task 文件 ...
        if containsInt(task.BlockedBy, completedID) {
            task.BlockedBy = removeInt(task.BlockedBy, completedID)
            tm.save(&task)
        }
    }
}
```

遍历所有任务文件, 把 `completedID` 从它们的 `blockedBy` 中移除。效果:

```
完成前:                      完成后:
task 2: blockedBy=[1]       task 2: blockedBy=[]     ← 解锁!
task 3: blockedBy=[1,4]     task 3: blockedBy=[4]    ← 部分解锁
```

---

## 第 174-210 行: `ListAll()` — 任务列表

```go
func (tm *TaskManager) ListAll() string {
    // 扫描 .tasks/ 目录, 读取所有 task 文件
    // 按 ID 排序
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

输出示例:
```
[x] #1: Setup
[ ] #2: Code (blocked by: [1])
[ ] #3: Test (blocked by: [2])
```

---

## 第 212-234 行: 整数集合辅助函数

```go
func containsInt(slice []int, val int) bool { ... }
func removeInt(slice []int, val int) []int { ... }
func uniqueAppend(existing, additions []int) []int { ... }
```

Go 没有内置的集合操作, 需要手写。Python 用 `if x in list` 和 `list.remove()` 一行搞定。

---

## 第 240-311 行: 全局初始化 — 8 个工具

```go
func init() {
    tasks = NewTaskManager(filepath.Join(workdir, ".tasks"))
    system = "You are a coding agent at %s. Use task tools to plan and track work."

    tools = []anthropic.ToolUnionParam{
        // 4 个基础工具: bash, read_file, write_file, edit_file
        // 4 个任务工具: task_create, task_update, task_list, task_get
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

从 s05 的 5 个工具增长到 8 个。dispatch map 的扩展性在这里再次体现 — 只需添加条目, 不改 agent loop。

---

## `task_update` 工具 schema — 值得注意的细节

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

- `enum` 约束状态值, 防止模型传入无效状态
- `array` + `items` 声明整数数组类型

---

## Agent Loop — 和 s02 完全一样

s07 的 agent loop **没有任何变化**。所有任务操作通过 dispatch map 正常分发, 不需要特殊逻辑。这是 s02 dispatch 模式的第六次复用。

---

## s06 → s07 变化对照表

| 维度 | s06 | s07 |
|------|-----|-----|
| 新增概念 | 三层上下文压缩 | **持久化任务图** |
| 工具 | 5 个 (含 compact) | **8 个** (含 task_create/update/list/get) |
| 状态存储 | 内存中 messages | **磁盘** .tasks/*.json |
| 依赖关系 | 无 | **blockedBy + blocks 双向边** |
| 新数据结构 | toolResultLoc | **Task, TaskManager** |
| 并发安全 | 无 | **sync.Mutex** |
| Agent Loop | 指针, 特殊 compact 处理 | **回归 s02 的纯 dispatch** |

---

## 核心架构图

```
启动时:
  NewTaskManager(".tasks/")
    ├── os.MkdirAll 确保目录存在
    └── maxID() 扫描已有文件, 确定 nextID

运行时:
  agentLoop()
    for {
        resp = LLM(system, tools)
        │
        tool_use: task_create("Setup")
        │   → tasks.Create("Setup")
        │   → 写入 .tasks/task_1.json
        │   → 返回 JSON
        │
        tool_use: task_update(2, addBlockedBy=[1])
        │   → tasks.Update(2, ...)
        │   → task_2.blockedBy = [1]
        │   → 双向: task_1.blocks += [2]
        │
        tool_use: task_update(1, status="completed")
        │   → tasks.Update(1, "completed")
        │   → clearDependency(1):
        │       遍历所有 task, 从 blockedBy 中移除 1
        │       task_2.blockedBy: [1] → []  ← 解锁!
        │
        tool_use: task_list
            → tasks.ListAll()
            → "[x] #1: Setup\n[ ] #2: Code"
    }
```

**s07 的核心洞察**: Agent 的规划状态不能只活在对话里。一旦上下文被压缩或 agent 重启, 内存中的状态就消失了。把任务图持久化到磁盘, 状态就永远不会丢失。这为后续的多 agent 协作 (s09+) 和 worktree 隔离 (s12) 奠定了共享状态的基础。
