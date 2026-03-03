# s10: Team Protocols — Go Line-by-Line Walkthrough

`s01 > s02 > s03 > s04 > s05 > s06 | s07 > s08 > s09 > [ s10 ] s11 > s12`

> Source: [`go/s10-team-protocols/main.go`](../../s10-team-protocols/main.go)

---

## Core Difference: s10 vs s09

s09 implemented agent teams but lacked **structured coordination**: shutdown was just exiting, and task assignment had no approval step. s10 adds two **request-response protocols**, both using the same `request_id` correlation pattern.

> **"Same request_id correlation pattern, two domains."**

```
Shutdown Protocol:           Plan Approval Protocol:
Lead -> Teammate              Teammate -> Lead
  shutdown_request{req_id}     plan_approval{req_id, plan}
  <-- shutdown_response        <-- plan_approval_response
       {req_id, approve}            {req_id, approve}

Shared FSM:
  [pending] --approve--> [approved]
  [pending] --reject---> [rejected]
```

---

## Lines 60-73: Request Tracker Data Structures

```go
type shutdownReq struct {
    Target string `json:"target"`     // Who is being asked to shut down
    Status string `json:"status"`     // pending | approved | rejected
}

type planReq struct {
    From   string `json:"from"`       // Who submitted the plan
    Plan   string `json:"plan"`       // Plan content
    Status string `json:"status"`     // pending | approved | rejected
}

var (
    shutdownRequests = map[string]*shutdownReq{}
    planRequests     = map[string]*planReq{}
    trackerMu        sync.Mutex
)
```

Two tracker maps, same FSM (`pending -> approved | rejected`), keyed by `request_id`. This is s10's core abstraction: **one pattern, two domains**.

---

## MessageBus — `Send()` Supports Extra Fields

```go
func (bus *MessageBus) Send(sender, to, content, msgType string, extra map[string]interface{}) string {
    msg := rawInboxMsg{
        "type":      msgType,
        "from":      sender,
        "content":   content,
        "timestamp": ...,
    }
    for k, v := range extra {
        msg[k] = v          // <- Key: merge protocol-specific fields
    }
}
```

s09's `Send()` only passed fixed fields. s10 adds `extra map[string]interface{}` for carrying protocol-specific fields:
- shutdown_request: `{"request_id": "abc123"}`
- plan_approval_response: `{"request_id": "xyz", "approve": true, "feedback": "..."}`

For this, the inbox message type was also upgraded from struct to `rawInboxMsg = map[string]interface{}`, supporting dynamic fields.

---

## `handleShutdownRequest()` — Lead Initiates Shutdown

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

Flow:
1. Generate unique `request_id`
2. Register as `pending` in the tracker
3. Send to teammate via inbox
4. Return request_id for later querying

---

## Teammate Side: `shutdown_response` Tool

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

After the teammate receives a shutdown_request, it replies with the `shutdown_response` tool. `approve=true` triggers exit.

---

## Teammate Side: `plan_approval` Tool

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

Note: it's the **teammate that generates the request_id** (not the lead). The direction is reversed compared to shutdown:
- Shutdown: lead -> teammate (lead generates ID)
- Plan: teammate -> lead (teammate generates ID)

---

## Lead Side: `handlePlanReview()` — Approve Plans

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

The lead references the same `request_id` for approval, and the result is sent back to the teammate via inbox.

---

## `teammateLoop()` shouldExit Mechanism

```go
func (tm *TeammateManager) teammateLoop(name, role, prompt string) {
    shouldExit := false

    for i := 0; i < 50; i++ {
        inbox := bus.ReadInbox(name)
        // ... inject inbox messages ...

        if shouldExit {
            break           // <- After receiving approve, exit on next iteration
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

Why not exit immediately? Because the tool_result still needs to be returned to the LLM. After setting `shouldExit = true`, the current round completes, and the next iteration checks `if shouldExit { break }`.

---

## Tool Set Comparison: Lead (12) vs Teammate (8)

```
Lead (12 tools):                    Teammate (8 tools):
+-- bash                            +-- bash
+-- read_file                       +-- read_file
+-- write_file                      +-- write_file
+-- edit_file                       +-- edit_file
+-- spawn_teammate   <- lead only   +-- send_message
+-- list_teammates   <- lead only   +-- read_inbox
+-- send_message                    +-- shutdown_response   <- new
+-- read_inbox                      +-- plan_approval       <- new
+-- broadcast        <- lead only
+-- shutdown_request <- lead only
+-- shutdown_response (= check status)
+-- plan_approval    (= approve/reject)
```

Same tool name, different behavior per role:
- Lead: shutdown_response = **query** status; plan_approval = **approve** plans
- Teammate: shutdown_response = **reply** to request; plan_approval = **submit** plans

---

## s09 -> s10 Change Comparison

| Dimension | s09 | s10 |
|-----------|-----|-----|
| New concept | Team + message bus | **Request-response protocols (shutdown + plan)** |
| Tools | Lead 9 / Teammate 6 | **Lead 12 / Teammate 8** |
| Shutdown | Natural exit, no negotiation | **request_id handshake, approve/reject** |
| Plan approval | None | **Submit-review-approve/reject** |
| State machine | None | **pending -> approved / rejected** |
| MessageBus | Fixed fields | **extra map supports protocol fields** |
| Tracker | None | **shutdownRequests + planRequests** |

---

## Core Architecture Diagram

```
Shutdown Protocol:
  Lead                                     Teammate (alice)
    |                                          |
    |  shutdown_request(teammate="alice")       |
    |  +- reqID = randomID()                   |
    |  |  shutdownRequests[reqID] = pending     |
    |  |  bus.Send -> alice.jsonl                |
    |  +---------------------------------------->|
    |                                          |  Receives inbox msg
    |                                          |  LLM: approve
    |                                          |  shutdown_response(reqID, true)
    |  <----------------------------------------|  shutdownRequests[reqID] = approved
    |  lead.jsonl << {shutdown_response}       |  shouldExit = true
    |                                          |  -> status: "shutdown"

Plan Approval Protocol:
  Teammate (bob)                           Lead
    |                                          |
    |  plan_approval(plan="refactor auth")     |
    |  +- reqID = randomID()                   |
    |  |  planRequests[reqID] = pending         |
    |  |  bus.Send -> lead.jsonl                |
    |  +---------------------------------------->|
    |                                          |  drain inbox: sees plan
    |                                          |  plan_approval(reqID, true, "lgtm")
    |  <----------------------------------------|  planRequests[reqID] = approved
    |  bob.jsonl << {approve:true}             |
    |  Continues execution                     |
```

**s10's core insight**: Two seemingly different protocols (shutdown and plan approval) share the same pattern — a `request_id`-correlated request-response FSM. This pattern can be extended to any scenario requiring negotiation: code review, resource requests, permission escalation... just add a new tracker map and tool pair.
