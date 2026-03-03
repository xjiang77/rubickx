// s11 - Autonomous Agents
//
// Idle cycle with task board polling, auto-claiming unclaimed tasks, and
// identity re-injection after context compression. Builds on s10's protocols.
//
//	Teammate lifecycle:
//	+-------+
//	| spawn |
//	+---+---+
//	    |
//	    v
//	+-------+  tool_use    +-------+
//	| WORK  | <----------- |  LLM  |
//	+---+---+              +-------+
//	    |
//	    | stop_reason != tool_use (or idle tool called)
//	    v
//	+--------+
//	| IDLE   | poll every 5s for up to 60s
//	+---+----+
//	    |
//	    +---> check inbox -> message? -> resume WORK
//	    |
//	    +---> scan .tasks/ -> unclaimed? -> claim -> resume WORK
//	    |
//	    +---> timeout (60s) -> shutdown
//
//	Identity re-injection after compression:
//	messages = [identity_block, ...remaining...]
//	"You are 'coder', role: backend, team: my-team"
//
// Key insight: "The agent finds work itself."
package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const (
	pollInterval = 5  // seconds between idle polls
	idleTimeout  = 60 // seconds before auto-shutdown
)

// ========== Request trackers ==========

type shutdownReq struct {
	Target string `json:"target"`
	Status string `json:"status"`
}
type planReq struct {
	From   string `json:"from"`
	Plan   string `json:"plan"`
	Status string `json:"status"`
}

var (
	shutdownRequests = map[string]*shutdownReq{}
	planRequests     = map[string]*planReq{}
	trackerMu        sync.Mutex
	claimMu          sync.Mutex
)

// ========== MessageBus ==========

var validMsgTypes = map[string]bool{
	"message": true, "broadcast": true,
	"shutdown_request": true, "shutdown_response": true,
	"plan_approval_response": true,
}

type rawInboxMsg = map[string]interface{}

type MessageBus struct {
	dir string
	mu  sync.Mutex
}

func NewMessageBus(dir string) *MessageBus {
	_ = os.MkdirAll(dir, 0o755)
	return &MessageBus{dir: dir}
}

func (bus *MessageBus) Send(sender, to, content, msgType string, extra map[string]interface{}) string {
	if msgType == "" {
		msgType = "message"
	}
	if !validMsgTypes[msgType] {
		return fmt.Sprintf("Error: Invalid type '%s'", msgType)
	}
	msg := rawInboxMsg{
		"type": msgType, "from": sender, "content": content,
		"timestamp": float64(time.Now().UnixMilli()) / 1000.0,
	}
	for k, v := range extra {
		msg[k] = v
	}
	data, _ := json.Marshal(msg)
	bus.mu.Lock()
	defer bus.mu.Unlock()
	path := filepath.Join(bus.dir, to+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
	return fmt.Sprintf("Sent %s to %s", msgType, to)
}

func (bus *MessageBus) ReadInbox(name string) []rawInboxMsg {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	path := filepath.Join(bus.dir, name+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var messages []rawInboxMsg
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var msg rawInboxMsg
		if json.Unmarshal([]byte(line), &msg) == nil {
			messages = append(messages, msg)
		}
	}
	_ = os.WriteFile(path, []byte(""), 0o644)
	return messages
}

func (bus *MessageBus) Broadcast(sender, content string, teammates []string) string {
	count := 0
	for _, name := range teammates {
		if name != sender {
			bus.Send(sender, name, content, "broadcast", nil)
			count++
		}
	}
	return fmt.Sprintf("Broadcast to %d teammates", count)
}

// ========== Task board scanning ==========

func scanUnclaimedTasks() []map[string]interface{} {
	_ = os.MkdirAll(tasksDir, 0o755)
	entries, _ := os.ReadDir(tasksDir)
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "task_") && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	var unclaimed []map[string]interface{}
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(tasksDir, name))
		if err != nil {
			continue
		}
		var task map[string]interface{}
		if json.Unmarshal(data, &task) != nil {
			continue
		}
		status, _ := task["status"].(string)
		owner, _ := task["owner"].(string)
		blockedBy, _ := task["blockedBy"].([]interface{})
		if status == "pending" && owner == "" && len(blockedBy) == 0 {
			unclaimed = append(unclaimed, task)
		}
	}
	return unclaimed
}

func claimTask(taskID int, owner string) string {
	claimMu.Lock()
	defer claimMu.Unlock()
	path := filepath.Join(tasksDir, fmt.Sprintf("task_%d.json", taskID))
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("Error: Task %d not found", taskID)
	}
	var task map[string]interface{}
	if json.Unmarshal(data, &task) != nil {
		return fmt.Sprintf("Error: corrupt task %d", taskID)
	}
	task["owner"] = owner
	task["status"] = "in_progress"
	out, _ := json.MarshalIndent(task, "", "  ")
	_ = os.WriteFile(path, out, 0o644)
	return fmt.Sprintf("Claimed task #%d for %s", taskID, owner)
}

// ========== Identity re-injection ==========

func makeIdentityMessages(name, role, teamName string) (anthropic.MessageParam, anthropic.MessageParam) {
	return anthropic.NewUserMessage(
			anthropic.NewTextBlock(
				fmt.Sprintf("<identity>You are '%s', role: %s, team: %s. Continue your work.</identity>", name, role, teamName),
			),
		),
		anthropic.NewAssistantMessage(
			anthropic.NewTextBlock(fmt.Sprintf("I am %s. Continuing.", name)),
		)
}

// ========== TeammateManager ==========

type memberConfig struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"`
}
type teamConfig struct {
	TeamName string         `json:"team_name"`
	Members  []memberConfig `json:"members"`
}

type TeammateManager struct {
	dir        string
	configPath string
	config     teamConfig
	mu         sync.Mutex
}

func NewTeammateManager(dir string) *TeammateManager {
	_ = os.MkdirAll(dir, 0o755)
	tm := &TeammateManager{dir: dir, configPath: filepath.Join(dir, "config.json")}
	tm.config = tm.loadConfig()
	return tm
}

func (tm *TeammateManager) loadConfig() teamConfig {
	data, err := os.ReadFile(tm.configPath)
	if err != nil {
		return teamConfig{TeamName: "default", Members: []memberConfig{}}
	}
	var cfg teamConfig
	if json.Unmarshal(data, &cfg) != nil {
		return teamConfig{TeamName: "default", Members: []memberConfig{}}
	}
	return cfg
}
func (tm *TeammateManager) saveConfig() {
	data, _ := json.MarshalIndent(tm.config, "", "  ")
	_ = os.WriteFile(tm.configPath, data, 0o644)
}
func (tm *TeammateManager) findMember(name string) *memberConfig {
	for i := range tm.config.Members {
		if tm.config.Members[i].Name == name {
			return &tm.config.Members[i]
		}
	}
	return nil
}
func (tm *TeammateManager) setStatus(name, status string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	m := tm.findMember(name)
	if m != nil {
		m.Status = status
		tm.saveConfig()
	}
}

func (tm *TeammateManager) Spawn(name, role, prompt string) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	member := tm.findMember(name)
	if member != nil {
		if member.Status != "idle" && member.Status != "shutdown" {
			return fmt.Sprintf("Error: '%s' is currently %s", name, member.Status)
		}
		member.Status = "working"
		member.Role = role
	} else {
		tm.config.Members = append(tm.config.Members, memberConfig{Name: name, Role: role, Status: "working"})
	}
	tm.saveConfig()
	go tm.loop(name, role, prompt)
	return fmt.Sprintf("Spawned '%s' (role: %s)", name, role)
}

// loop is the autonomous teammate lifecycle: WORK → IDLE → WORK → ... → SHUTDOWN
func (tm *TeammateManager) loop(name, role, prompt string) {
	teamName := tm.config.TeamName
	sysPrompt := fmt.Sprintf(
		"You are '%s', role: %s, team: %s, at %s. "+
			"Use idle tool when you have no more work. You will auto-claim new tasks.",
		name, role, teamName, workdir,
	)
	tools := teammateTools()
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	for { // outer loop: WORK → IDLE → repeat
		// ======== WORK PHASE ========
		idleRequested := false
		for round := 0; round < 50; round++ {
			// Check inbox
			inbox := bus.ReadInbox(name)
			for _, msg := range inbox {
				msgType, _ := msg["type"].(string)
				if msgType == "shutdown_request" {
					tm.setStatus(name, "shutdown")
					return
				}
				msgJSON, _ := json.Marshal(msg)
				messages = append(messages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(string(msgJSON))),
				)
			}

			resp, err := apiClient.Messages.New(context.Background(), anthropic.MessageNewParams{
				Model:     anthropic.Model(model),
				System:    []anthropic.TextBlockParam{{Text: sysPrompt}},
				Messages:  messages,
				Tools:     tools,
				MaxTokens: 8000,
			})
			if err != nil {
				tm.setStatus(name, "idle")
				return
			}

			assistantBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(resp.Content))
			for _, block := range resp.Content {
				switch block.Type {
				case "text":
					assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
				case "tool_use":
					assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))
				}
			}
			messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))

			if resp.StopReason != "tool_use" {
				break
			}

			var results []anthropic.ContentBlockParamUnion
			for _, block := range resp.Content {
				if block.Type == "tool_use" {
					var output string
					if block.Name == "idle" {
						idleRequested = true
						output = "Entering idle phase. Will poll for new tasks."
					} else {
						output = tm.execTool(name, block.Name, block.Input)
					}
					fmt.Printf("  [%s] %s: %s\n", name, block.Name, truncate(output, 120))
					results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
				}
			}
			messages = append(messages, anthropic.NewUserMessage(results...))
			if idleRequested {
				break
			}
		}

		// ======== IDLE PHASE ========
		tm.setStatus(name, "idle")
		resume := false
		polls := idleTimeout / pollInterval
		for p := 0; p < polls; p++ {
			time.Sleep(time.Duration(pollInterval) * time.Second)

			// Check inbox
			inbox := bus.ReadInbox(name)
			if len(inbox) > 0 {
				for _, msg := range inbox {
					msgType, _ := msg["type"].(string)
					if msgType == "shutdown_request" {
						tm.setStatus(name, "shutdown")
						return
					}
					msgJSON, _ := json.Marshal(msg)
					messages = append(messages,
						anthropic.NewUserMessage(anthropic.NewTextBlock(string(msgJSON))),
					)
				}
				resume = true
				break
			}

			// Scan task board
			unclaimed := scanUnclaimedTasks()
			if len(unclaimed) > 0 {
				task := unclaimed[0]
				taskIDf, _ := task["id"].(float64)
				taskID := int(taskIDf)
				subject, _ := task["subject"].(string)
				desc, _ := task["description"].(string)
				claimTask(taskID, name)

				taskPrompt := fmt.Sprintf("<auto-claimed>Task #%d: %s\n%s</auto-claimed>", taskID, subject, desc)

				// Identity re-injection if messages are short (compression happened)
				if len(messages) <= 3 {
					idUser, idAssist := makeIdentityMessages(name, role, teamName)
					messages = append([]anthropic.MessageParam{idUser, idAssist}, messages...)
				}
				messages = append(messages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(taskPrompt)),
					anthropic.NewAssistantMessage(anthropic.NewTextBlock(fmt.Sprintf("Claimed task #%d. Working on it.", taskID))),
				)
				resume = true
				break
			}
		}

		if !resume {
			tm.setStatus(name, "shutdown")
			return
		}
		tm.setStatus(name, "working")
	}
}

// execTool dispatches a tool call for a teammate.
func (tm *TeammateManager) execTool(sender, toolName string, rawInput json.RawMessage) string {
	switch toolName {
	case "bash":
		var in struct {
			Command string `json:"command"`
		}
		_ = json.Unmarshal(rawInput, &in)
		return runBash(in.Command)
	case "read_file":
		var in struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(rawInput, &in)
		return runRead(in.Path, 0)
	case "write_file":
		var in struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		_ = json.Unmarshal(rawInput, &in)
		return runWrite(in.Path, in.Content)
	case "edit_file":
		var in struct {
			Path    string `json:"path"`
			OldText string `json:"old_text"`
			NewText string `json:"new_text"`
		}
		_ = json.Unmarshal(rawInput, &in)
		return runEdit(in.Path, in.OldText, in.NewText)
	case "send_message":
		var in struct {
			To      string `json:"to"`
			Content string `json:"content"`
			MsgType string `json:"msg_type"`
		}
		_ = json.Unmarshal(rawInput, &in)
		return bus.Send(sender, in.To, in.Content, in.MsgType, nil)
	case "read_inbox":
		msgs := bus.ReadInbox(sender)
		data, _ := json.MarshalIndent(msgs, "", "  ")
		return string(data)
	case "shutdown_response":
		var in struct {
			RequestID string `json:"request_id"`
			Approve   bool   `json:"approve"`
			Reason    string `json:"reason"`
		}
		_ = json.Unmarshal(rawInput, &in)
		trackerMu.Lock()
		if req, ok := shutdownRequests[in.RequestID]; ok {
			if in.Approve {
				req.Status = "approved"
			} else {
				req.Status = "rejected"
			}
		}
		trackerMu.Unlock()
		bus.Send(sender, "lead", in.Reason, "shutdown_response",
			map[string]interface{}{"request_id": in.RequestID, "approve": in.Approve})
		if in.Approve {
			return "Shutdown approved"
		}
		return "Shutdown rejected"
	case "plan_approval":
		var in struct {
			Plan string `json:"plan"`
		}
		_ = json.Unmarshal(rawInput, &in)
		reqID := randomID()
		trackerMu.Lock()
		planRequests[reqID] = &planReq{From: sender, Plan: in.Plan, Status: "pending"}
		trackerMu.Unlock()
		bus.Send(sender, "lead", in.Plan, "plan_approval_response",
			map[string]interface{}{"request_id": reqID, "plan": in.Plan})
		return fmt.Sprintf("Plan submitted (request_id=%s). Waiting for approval.", reqID)
	case "claim_task":
		var in struct {
			TaskID int `json:"task_id"`
		}
		_ = json.Unmarshal(rawInput, &in)
		return claimTask(in.TaskID, sender)
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}

func teammateTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{Name: "bash", Description: anthropic.String("Run a shell command."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"command": map[string]interface{}{"type": "string"}}, Required: []string{"command"}}}},
		{OfTool: &anthropic.ToolParam{Name: "read_file", Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}}, Required: []string{"path"}}}},
		{OfTool: &anthropic.ToolParam{Name: "write_file", Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}}, Required: []string{"path", "content"}}}},
		{OfTool: &anthropic.ToolParam{Name: "edit_file", Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "old_text": map[string]interface{}{"type": "string"}, "new_text": map[string]interface{}{"type": "string"}}, Required: []string{"path", "old_text", "new_text"}}}},
		{OfTool: &anthropic.ToolParam{Name: "send_message", Description: anthropic.String("Send message to a teammate."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"to": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"},
					"msg_type": map[string]interface{}{"type": "string", "enum": []string{"message", "broadcast", "shutdown_request", "shutdown_response", "plan_approval_response"}}},
				Required: []string{"to", "content"}}}},
		{OfTool: &anthropic.ToolParam{Name: "read_inbox", Description: anthropic.String("Read and drain your inbox."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}}}},
		{OfTool: &anthropic.ToolParam{Name: "shutdown_response", Description: anthropic.String("Respond to a shutdown request."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"request_id": map[string]interface{}{"type": "string"}, "approve": map[string]interface{}{"type": "boolean"}, "reason": map[string]interface{}{"type": "string"}},
				Required:   []string{"request_id", "approve"}}}},
		{OfTool: &anthropic.ToolParam{Name: "plan_approval", Description: anthropic.String("Submit a plan for lead approval."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"plan": map[string]interface{}{"type": "string"}}, Required: []string{"plan"}}}},
		{OfTool: &anthropic.ToolParam{Name: "idle", Description: anthropic.String("Signal that you have no more work. Enters idle polling phase."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}}}},
		{OfTool: &anthropic.ToolParam{Name: "claim_task", Description: anthropic.String("Claim a task from the task board by ID."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"task_id": map[string]interface{}{"type": "integer"}}, Required: []string{"task_id"}}}},
	}
}

func (tm *TeammateManager) ListAll() string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	if len(tm.config.Members) == 0 {
		return "No teammates."
	}
	lines := []string{fmt.Sprintf("Team: %s", tm.config.TeamName)}
	for _, m := range tm.config.Members {
		lines = append(lines, fmt.Sprintf("  %s (%s): %s", m.Name, m.Role, m.Status))
	}
	return strings.Join(lines, "\n")
}
func (tm *TeammateManager) MemberNames() []string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	names := make([]string, len(tm.config.Members))
	for i, m := range tm.config.Members {
		names[i] = m.Name
	}
	return names
}

// ========== Lead protocol handlers ==========

func handleShutdownRequest(teammate string) string {
	reqID := randomID()
	trackerMu.Lock()
	shutdownRequests[reqID] = &shutdownReq{Target: teammate, Status: "pending"}
	trackerMu.Unlock()
	bus.Send("lead", teammate, "Please shut down gracefully.", "shutdown_request",
		map[string]interface{}{"request_id": reqID})
	return fmt.Sprintf("Shutdown request %s sent to '%s'", reqID, teammate)
}
func checkShutdownStatus(requestID string) string {
	trackerMu.Lock()
	defer trackerMu.Unlock()
	req, ok := shutdownRequests[requestID]
	if !ok {
		return `{"error": "not found"}`
	}
	data, _ := json.Marshal(req)
	return string(data)
}
func handlePlanReview(requestID string, approve bool, feedback string) string {
	trackerMu.Lock()
	req, ok := planRequests[requestID]
	if !ok {
		trackerMu.Unlock()
		return fmt.Sprintf("Error: Unknown plan request_id '%s'", requestID)
	}
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

// ========== Globals ==========

var (
	model     string
	system    string
	workdir   string
	tasksDir  string
	bus       *MessageBus
	team      *TeammateManager
	apiClient anthropic.Client
	tools     []anthropic.ToolUnionParam
)

type toolHandler func(input json.RawMessage) string

var dispatch map[string]toolHandler

var guided = []struct {
	step   string
	prompt string
}{
	{"Create tasks for auto-claim", `Create 3 tasks on the board, then spawn alice and bob. Watch them auto-claim.`},
	{"Observe autonomous behavior", `Spawn a coder teammate and let it find work from the task board itself`},
	{"Task dependencies respected", `Create tasks with dependencies. Watch teammates respect the blocked order.`},
	{"Check task board", `/tasks`},
	{"Check team status", `/team`},
}

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	workdir, _ = os.Getwd()
	tasksDir = filepath.Join(workdir, ".tasks")

	teamDir := filepath.Join(workdir, ".team")
	bus = NewMessageBus(filepath.Join(teamDir, "inbox"))
	team = NewTeammateManager(teamDir)

	system = fmt.Sprintf("You are a team lead at %s. Teammates are autonomous -- they find work themselves.", workdir)

	// Lead tool definitions (14 tools)
	tools = []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{Name: "bash", Description: anthropic.String("Run a shell command."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"command": map[string]interface{}{"type": "string"}}, Required: []string{"command"}}}},
		{OfTool: &anthropic.ToolParam{Name: "read_file", Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "integer"}}, Required: []string{"path"}}}},
		{OfTool: &anthropic.ToolParam{Name: "write_file", Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}}, Required: []string{"path", "content"}}}},
		{OfTool: &anthropic.ToolParam{Name: "edit_file", Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "old_text": map[string]interface{}{"type": "string"}, "new_text": map[string]interface{}{"type": "string"}}, Required: []string{"path", "old_text", "new_text"}}}},
		{OfTool: &anthropic.ToolParam{Name: "spawn_teammate", Description: anthropic.String("Spawn an autonomous teammate."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "role": map[string]interface{}{"type": "string"}, "prompt": map[string]interface{}{"type": "string"}}, Required: []string{"name", "role", "prompt"}}}},
		{OfTool: &anthropic.ToolParam{Name: "list_teammates", Description: anthropic.String("List all teammates."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}}}},
		{OfTool: &anthropic.ToolParam{Name: "send_message", Description: anthropic.String("Send a message to a teammate."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"to": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"},
				"msg_type": map[string]interface{}{"type": "string", "enum": []string{"message", "broadcast", "shutdown_request", "shutdown_response", "plan_approval_response"}}}, Required: []string{"to", "content"}}}},
		{OfTool: &anthropic.ToolParam{Name: "read_inbox", Description: anthropic.String("Read and drain the lead's inbox."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}}}},
		{OfTool: &anthropic.ToolParam{Name: "broadcast", Description: anthropic.String("Send a message to all teammates."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"content": map[string]interface{}{"type": "string"}}, Required: []string{"content"}}}},
		{OfTool: &anthropic.ToolParam{Name: "shutdown_request", Description: anthropic.String("Request a teammate to shut down."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"teammate": map[string]interface{}{"type": "string"}}, Required: []string{"teammate"}}}},
		{OfTool: &anthropic.ToolParam{Name: "shutdown_response", Description: anthropic.String("Check shutdown request status."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"request_id": map[string]interface{}{"type": "string"}}, Required: []string{"request_id"}}}},
		{OfTool: &anthropic.ToolParam{Name: "plan_approval", Description: anthropic.String("Approve or reject a teammate's plan."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"request_id": map[string]interface{}{"type": "string"}, "approve": map[string]interface{}{"type": "boolean"}, "feedback": map[string]interface{}{"type": "string"}}, Required: []string{"request_id", "approve"}}}},
		{OfTool: &anthropic.ToolParam{Name: "idle", Description: anthropic.String("Enter idle state (for lead -- rarely used)."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}}}},
		{OfTool: &anthropic.ToolParam{Name: "claim_task", Description: anthropic.String("Claim a task from the board by ID."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{"task_id": map[string]interface{}{"type": "integer"}}, Required: []string{"task_id"}}}},
	}

	dispatch = map[string]toolHandler{
		"bash": func(raw json.RawMessage) string {
			var in struct {
				Command string `json:"command"`
			}
			_ = json.Unmarshal(raw, &in)
			return runBash(in.Command)
		},
		"read_file": func(raw json.RawMessage) string {
			var in struct {
				Path  string `json:"path"`
				Limit int    `json:"limit"`
			}
			_ = json.Unmarshal(raw, &in)
			return runRead(in.Path, in.Limit)
		},
		"write_file": func(raw json.RawMessage) string {
			var in struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			_ = json.Unmarshal(raw, &in)
			return runWrite(in.Path, in.Content)
		},
		"edit_file": func(raw json.RawMessage) string {
			var in struct {
				Path    string `json:"path"`
				OldText string `json:"old_text"`
				NewText string `json:"new_text"`
			}
			_ = json.Unmarshal(raw, &in)
			return runEdit(in.Path, in.OldText, in.NewText)
		},
		"spawn_teammate": func(raw json.RawMessage) string {
			var in struct {
				Name   string `json:"name"`
				Role   string `json:"role"`
				Prompt string `json:"prompt"`
			}
			_ = json.Unmarshal(raw, &in)
			return team.Spawn(in.Name, in.Role, in.Prompt)
		},
		"list_teammates": func(raw json.RawMessage) string { return team.ListAll() },
		"send_message": func(raw json.RawMessage) string {
			var in struct {
				To      string `json:"to"`
				Content string `json:"content"`
				MsgType string `json:"msg_type"`
			}
			_ = json.Unmarshal(raw, &in)
			return bus.Send("lead", in.To, in.Content, in.MsgType, nil)
		},
		"read_inbox": func(raw json.RawMessage) string {
			msgs := bus.ReadInbox("lead")
			data, _ := json.MarshalIndent(msgs, "", "  ")
			return string(data)
		},
		"broadcast": func(raw json.RawMessage) string {
			var in struct {
				Content string `json:"content"`
			}
			_ = json.Unmarshal(raw, &in)
			return bus.Broadcast("lead", in.Content, team.MemberNames())
		},
		"shutdown_request": func(raw json.RawMessage) string {
			var in struct {
				Teammate string `json:"teammate"`
			}
			_ = json.Unmarshal(raw, &in)
			return handleShutdownRequest(in.Teammate)
		},
		"shutdown_response": func(raw json.RawMessage) string {
			var in struct {
				RequestID string `json:"request_id"`
			}
			_ = json.Unmarshal(raw, &in)
			return checkShutdownStatus(in.RequestID)
		},
		"plan_approval": func(raw json.RawMessage) string {
			var in struct {
				RequestID string `json:"request_id"`
				Approve   bool   `json:"approve"`
				Feedback  string `json:"feedback"`
			}
			_ = json.Unmarshal(raw, &in)
			return handlePlanReview(in.RequestID, in.Approve, in.Feedback)
		},
		"idle": func(raw json.RawMessage) string { return "Lead does not idle." },
		"claim_task": func(raw json.RawMessage) string {
			var in struct {
				TaskID int `json:"task_id"`
			}
			_ = json.Unmarshal(raw, &in)
			return claimTask(in.TaskID, "lead")
		},
	}
}

// ========== Path safety + tool implementations ==========

func safePath(p string) (string, error) {
	full := filepath.Join(workdir, p)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", p)
	}
	if abs != workdir && !strings.HasPrefix(abs, workdir+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return abs, nil
}

func runBash(command string) string {
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(command, d) {
			return "Error: Dangerous command blocked"
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "Error: Timeout (120s)"
	}
	out := strings.TrimSpace(string(output))
	if err != nil && out == "" {
		return fmt.Sprintf("Error: %v", err)
	}
	if out == "" {
		return "(no output)"
	}
	if len(out) > 50000 {
		return out[:50000]
	}
	return out
}
func runRead(path string, limit int) string {
	fp, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	if limit > 0 && limit < len(lines) {
		lines = append(lines[:limit], fmt.Sprintf("... (%d more lines)", len(lines)-limit))
	}
	out := strings.Join(lines, "\n")
	if len(out) > 50000 {
		return out[:50000]
	}
	return out
}
func runWrite(path string, content string) string {
	fp, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	_ = os.MkdirAll(filepath.Dir(fp), 0o755)
	if err := os.WriteFile(fp, []byte(content), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Wrote %d bytes", len(content))
}
func runEdit(path string, oldText string, newText string) string {
	fp, err := safePath(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, oldText) {
		return fmt.Sprintf("Error: Text not found in %s", path)
	}
	return func() string {
		if err := os.WriteFile(fp, []byte(strings.Replace(content, oldText, newText, 1)), 0o644); err != nil {
			return fmt.Sprintf("Error: %v", err)
		}
		return fmt.Sprintf("Edited %s", path)
	}()
}

func randomID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

// ========== Agent Loop (lead) ==========

func agentLoop(messages []anthropic.MessageParam) []anthropic.MessageParam {
	for {
		inbox := bus.ReadInbox("lead")
		if len(inbox) > 0 {
			inboxJSON, _ := json.MarshalIndent(inbox, "", "  ")
			messages = append(messages,
				anthropic.NewUserMessage(anthropic.NewTextBlock(fmt.Sprintf("<inbox>%s</inbox>", string(inboxJSON)))),
				anthropic.NewAssistantMessage(anthropic.NewTextBlock("Noted inbox messages.")),
			)
		}
		resp, err := apiClient.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model: anthropic.Model(model), System: []anthropic.TextBlockParam{{Text: system}},
			Messages: messages, Tools: tools, MaxTokens: 8000,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "API error: %v\n", err)
			return messages
		}
		assistantBlocks := make([]anthropic.ContentBlockParamUnion, 0, len(resp.Content))
		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				assistantBlocks = append(assistantBlocks, anthropic.NewTextBlock(block.Text))
			case "tool_use":
				assistantBlocks = append(assistantBlocks, anthropic.NewToolUseBlock(block.ID, block.Input, block.Name))
			}
		}
		messages = append(messages, anthropic.NewAssistantMessage(assistantBlocks...))
		if resp.StopReason != "tool_use" {
			return messages
		}
		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				handler, ok := dispatch[block.Name]
				var output string
				if ok {
					output = handler(block.Input)
				} else {
					output = fmt.Sprintf("Unknown tool: %s", block.Name)
				}
				fmt.Printf("> %s: %s\n", block.Name, truncate(output, 200))
				results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
			}
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}

// ========== Main ==========

func main() {
	opts := []option.RequestOption{}
	if baseURL := os.Getenv("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	apiClient = anthropic.NewClient(opts...)

	fmt.Println("\n  s11: Autonomous Agents")
	fmt.Print("  \"Idle-cycle polling & self-directed work\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	// tasksLocalCmd handles the /tasks local command inline.
	tasksLocalCmd := func() {
		_ = os.MkdirAll(tasksDir, 0o755)
		entries, _ := os.ReadDir(tasksDir)
		var names []string
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), "task_") && strings.HasSuffix(e.Name(), ".json") {
				names = append(names, e.Name())
			}
		}
		sort.Strings(names)
		for _, n := range names {
			data, _ := os.ReadFile(filepath.Join(tasksDir, n))
			var t map[string]interface{}
			_ = json.Unmarshal(data, &t)
			status, _ := t["status"].(string)
			subject, _ := t["subject"].(string)
			owner, _ := t["owner"].(string)
			marker := map[string]string{"pending": "[ ]", "in_progress": "[>]", "completed": "[x]"}[status]
			if marker == "" {
				marker = "[?]"
			}
			ownerStr := ""
			if owner != "" {
				ownerStr = " @" + owner
			}
			id, _ := t["id"].(float64)
			fmt.Printf("  %s #%d: %s%s\n", marker, int(id), subject, ownerStr)
		}
	}

	for {
		var query string

		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms11 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
			if !scanner.Scan() {
				break
			}
			query = strings.TrimSpace(scanner.Text())
			if query == "q" || query == "exit" {
				break
			}
			// Local commands: handle without advancing guidedIdx
			if strings.HasPrefix(query, "/") {
				if query == "/team" {
					fmt.Println(team.ListAll())
				} else if query == "/inbox" {
					msgs := bus.ReadInbox("lead")
					data, _ := json.MarshalIndent(msgs, "", "  ")
					fmt.Println(string(data))
				} else if query == "/tasks" {
					tasksLocalCmd()
				}
				continue
			}
			// Empty input: use guided prompt
			if query == "" {
				query = guided[guidedIdx].prompt
			}
			guidedIdx++
		} else {
			if !freeModePrinted {
				fmt.Print("\n  ✓ Guided tour complete. Free mode — type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Printf("\033[36ms11 >> \033[0m")
			if !scanner.Scan() {
				break
			}
			query = strings.TrimSpace(scanner.Text())
			if query == "q" || query == "exit" {
				break
			}
			if query == "" {
				continue
			}
			// Local commands
			if query == "/team" {
				fmt.Println(team.ListAll())
				continue
			}
			if query == "/inbox" {
				msgs := bus.ReadInbox("lead")
				data, _ := json.MarshalIndent(msgs, "", "  ")
				fmt.Println(string(data))
				continue
			}
			if query == "/tasks" {
				tasksLocalCmd()
				continue
			}
		}

		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
		messages = agentLoop(messages)
		if len(messages) > 0 {
			last := messages[len(messages)-1]
			for _, part := range last.Content {
				if part.OfText != nil {
					fmt.Println(part.OfText.Text)
				}
			}
		}
		fmt.Println()
	}
}
