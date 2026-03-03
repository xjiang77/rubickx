# s10: Team Protocols — Go 逐行解读

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > [ s10 ] s11 > s12`

> 源码: [`go/s10-team-protocols/main.go`](../../s10-team-protocols/main.go)

---

## s10 vs s09 的核心区别

s09 实现了 agent 团队, 但缺少**结构化协调**: 关机是直接退出, 任务分配没有审批环节。s10 加入两个**请求-响应协议**, 都用同一个 `request_id` 关联模式。

> **"Same request_id correlation pattern, two domains."**

```
Shutdown Protocol:           Plan Approval Protocol:
Lead → Teammate              Teammate → Lead
  shutdown_request{req_id}     plan_approval{req_id, plan}
  ←── shutdown_response        ←── plan_approval_response
       {req_id, approve}            {req_id, approve}

Shared FSM:
  [pending] ──approve──> [approved]
  [pending] ──reject───> [rejected]
```

---

## 第 60-73 行: Request Tracker 数据结构

```go
type shutdownReq struct {
    Target string `json:"target"`     // 谁被请求关机
    Status string `json:"status"`     // pending | approved | rejected
}

type planReq struct {
    From   string `json:"from"`       // 谁提交的计划
    Plan   string `json:"plan"`       // 计划内容
    Status string `json:"status"`     // pending | approved | rejected
}

var (
    shutdownRequests = map[string]*shutdownReq{}
    planRequests     = map[string]*planReq{}
    trackerMu        sync.Mutex
)
```

两个 tracker map, 同一种 FSM (`pending → approved | rejected`), 用 `request_id` 做 key。这是 s10 的核心抽象: **一个模式, 两个领域**。

---

## MessageBus — `Send()` 支持 extra 字段

```go
func (bus *MessageBus) Send(sender, to, content, msgType string, extra map[string]interface{}) string {
    msg := rawInboxMsg{
        "type":      msgType,
        "from":      sender,
        "content":   content,
        "timestamp": ...,
    }
    for k, v := range extra {
        msg[k] = v          // ← 关键: 合并协议特有字段
    }
}
```

s09 的 `Send()` 只传固定字段。s10 加了 `extra map[string]interface{}`, 用于携带协议特有字段:
- shutdown_request: `{"request_id": "abc123"}`
- plan_approval_response: `{"request_id": "xyz", "approve": true, "feedback": "..."}`

为此, inbox 消息类型也从 struct 升级为 `rawInboxMsg = map[string]interface{}`, 支持动态字段。

---

## `handleShutdownRequest()` — Lead 发起关机

```go
func handleShutdownRequest(teammate string) string {
    reqID := randomID()
    trackerMu.Lock()
    shutdownRequests[reqID] = &shutdownReq{Target: teammate, Status: "pending"}
    trackerMu.Unlock()
    bus.Send("lead", teammate, "Please shut down gracefully.",
        "shutdown_request", map[string]interface{}{"request_id": reqID})
    return fmt.Sprintf("Shutdown request %s sent to '%s' (status: pending)", reqID, teammate)
}
```

流程:
1. 生成唯一 `request_id`
2. 在 tracker 中注册为 `pending`
3. 通过收件箱发送给 teammate
4. 返回 request_id 供后续查询

---

## Teammate 端: `shutdown_response` 工具

```go
case "shutdown_response":
    trackerMu.Lock()
    if req, ok := shutdownRequests[input.RequestID]; ok {
        if input.Approve {
            req.Status = "approved"
        } else {
            req.Status = "rejected"
        }
    }
    trackerMu.Unlock()
    bus.Send(sender, "lead", input.Reason, "shutdown_response",
        map[string]interface{}{"request_id": input.RequestID, "approve": input.Approve})
```

Teammate 收到 shutdown_request 后, 用 `shutdown_response` 工具回复。`approve=true` 触发退出。

---

## Teammate 端: `plan_approval` 工具

```go
case "plan_approval":
    reqID := randomID()
    trackerMu.Lock()
    planRequests[reqID] = &planReq{From: sender, Plan: input.Plan, Status: "pending"}
    trackerMu.Unlock()
    bus.Send(sender, "lead", input.Plan, "plan_approval_response",
        map[string]interface{}{"request_id": reqID, "plan": input.Plan})
    return fmt.Sprintf("Plan submitted (request_id=%s). Waiting for lead approval.", reqID)
```

注意: 是 **teammate 生成 request_id** (不是 lead)。发送方向和 shutdown 相反:
- Shutdown: lead → teammate (lead 生成 ID)
- Plan: teammate → lead (teammate 生成 ID)

---

## Lead 端: `handlePlanReview()` — 审批计划

```go
func handlePlanReview(requestID string, approve bool, feedback string) string {
    trackerMu.Lock()
    req := planRequests[requestID]
    if approve {
        req.Status = "approved"
    } else {
        req.Status = "rejected"
    }
    from := req.From
    trackerMu.Unlock()

    bus.Send("lead", from, feedback, "plan_approval_response",
        map[string]interface{}{"request_id": requestID, "approve": approve, "feedback": feedback})
    return fmt.Sprintf("Plan %s for '%s'", req.Status, from)
}
```

Lead 引用同一个 `request_id` 做审批, 结果通过收件箱回传给 teammate。

---

## `teammateLoop()` 的 shouldExit 机制

```go
func (tm *TeammateManager) teammateLoop(name, role, prompt string) {
    shouldExit := false

    for i := 0; i < 50; i++ {
        inbox := bus.ReadInbox(name)
        // ... inject inbox messages ...

        if shouldExit {
            break           // ← 收到 approve 后, 下一轮退出
        }

        resp, _ := apiClient.Messages.New(...)

        for _, block := range resp.Content {
            if block.Name == "shutdown_response" {
                var input struct{ Approve bool `json:"approve"` }
                json.Unmarshal(block.Input, &input)
                if input.Approve {
                    shouldExit = true
                }
            }
        }
    }

    if shouldExit {
        member.Status = "shutdown"
    } else {
        member.Status = "idle"
    }
}
```

为什么不立即退出? 因为 tool_result 还需要返回给 LLM。设置 `shouldExit = true` 后, 当前轮完成, 下一轮开头检查 `if shouldExit { break }`。

---

## 工具集对比: Lead (12) vs Teammate (8)

```
Lead (12 tools):                    Teammate (8 tools):
├── bash                            ├── bash
├── read_file                       ├── read_file
├── write_file                      ├── write_file
├── edit_file                       ├── edit_file
├── spawn_teammate   ← lead only   ├── send_message
├── list_teammates   ← lead only   ├── read_inbox
├── send_message                    ├── shutdown_response   ← 新增
├── read_inbox                      └── plan_approval       ← 新增
├── broadcast        ← lead only
├── shutdown_request ← lead only
├── shutdown_response (= check status)
└── plan_approval    (= approve/reject)
```

同一工具名, 不同角色不同行为:
- Lead: shutdown_response = **查询**状态; plan_approval = **审批**计划
- Teammate: shutdown_response = **回复**请求; plan_approval = **提交**计划

---

## s09 → s10 变化对照表

| 维度 | s09 | s10 |
|------|-----|-----|
| 新增概念 | 团队 + 消息总线 | **请求-响应协议 (shutdown + plan)** |
| 工具 | Lead 9 / Teammate 6 | **Lead 12 / Teammate 8** |
| 关机 | 自然退出, 无协商 | **request_id 握手, approve/reject** |
| 计划审批 | 无 | **提交-审查-批准/驳回** |
| 状态机 | 无 | **pending → approved / rejected** |
| MessageBus | 固定字段 | **extra map 支持协议字段** |
| Tracker | 无 | **shutdownRequests + planRequests** |

---

## 核心架构图

```
Shutdown Protocol:
  Lead                                     Teammate (alice)
    │                                          │
    │  shutdown_request(teammate="alice")       │
    │  ┌─ reqID = randomID()                   │
    │  │  shutdownRequests[reqID] = pending     │
    │  │  bus.Send → alice.jsonl                │
    │  └───────────────────────────────────────>│
    │                                          │  收到 inbox msg
    │                                          │  LLM: approve
    │                                          │  shutdown_response(reqID, true)
    │  <───────────────────────────────────────│  shutdownRequests[reqID] = approved
    │  lead.jsonl << {shutdown_response}       │  shouldExit = true
    │                                          │  → status: "shutdown"

Plan Approval Protocol:
  Teammate (bob)                           Lead
    │                                          │
    │  plan_approval(plan="重构 auth")          │
    │  ┌─ reqID = randomID()                   │
    │  │  planRequests[reqID] = pending         │
    │  │  bus.Send → lead.jsonl                │
    │  └───────────────────────────────────────>│
    │                                          │  drain inbox: 看到 plan
    │                                          │  plan_approval(reqID, true, "lgtm")
    │  <───────────────────────────────────────│  planRequests[reqID] = approved
    │  bob.jsonl << {approve:true}             │
    │  继续执行                                 │
```

**s10 的核心洞察**: 两种看似不同的协议 (关机和计划审批) 共享同一个模式 — `request_id` 关联的请求-响应 FSM。这个模式可以扩展到任何需要协商的场景: 代码审查、资源申请、权限提升... 只需要新的 tracker map 和工具对。
