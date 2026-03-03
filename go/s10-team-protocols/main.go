// s10 - Team Protocols
//
// Shutdown protocol and plan approval protocol, both using the same
// request_id correlation pattern. Builds on s09's team messaging.
//
//	Shutdown FSM: pending -> approved | rejected
//
//	Lead                              Teammate
//	+---------------------+          +---------------------+
//	| shutdown_request     |          |                     |
//	| {                    | -------> | receives request    |
//	|   request_id: abc    |          | decides: approve?   |
//	| }                    |          |                     |
//	+---------------------+          +---------------------+
//	                                         |
//	+---------------------+          +-------v-------------+
//	| shutdown_response    | <------- | shutdown_response   |
//	| {                    |          | {                   |
//	|   request_id: abc    |          |   request_id: abc   |
//	|   approve: true      |          |   approve: true     |
//	| }                    |          | }                   |
//	+---------------------+          +---------------------+
//	        |
//	        v
//	status -> "shutdown", goroutine stops
//
//	Plan approval FSM: pending -> approved | rejected
//
//	Teammate                          Lead
//	+---------------------+          +---------------------+
//	| plan_approval        |          |                     |
//	| submit: {plan:"..."}| -------> | reviews plan text   |
//	+---------------------+          | approve/reject?     |
//	                                 +---------------------+
//	                                         |
//	+---------------------+          +-------v-------------+
//	| plan_approval_resp   | <------- | plan_approval       |
//	| {approve: true}      |          | review: {req_id,    |
//	+---------------------+          |   approve: true}     |
//	                                 +---------------------+
//
//	Trackers: {request_id: {"target|from": name, "status": "pending|..."}}
//
// Key insight: "Same request_id correlation pattern, two domains."
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
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ========== Request trackers ==========

// shutdownReq tracks a pending shutdown handshake.
type shutdownReq struct {
	Target string `json:"target"`
	Status string `json:"status"` // pending | approved | rejected
}

// planReq tracks a pending plan approval.
type planReq struct {
	From   string `json:"from"`
	Plan   string `json:"plan"`
	Status string `json:"status"` // pending | approved | rejected
}

var (
	shutdownRequests = map[string]*shutdownReq{}
	planRequests     = map[string]*planReq{}
	trackerMu        sync.Mutex
)

// ========== MessageBus ==========

var validMsgTypes = map[string]bool{
	"message": true, "broadcast": true,
	"shutdown_request": true, "shutdown_response": true,
	"plan_approval_response": true,
}

type inboxMsg struct {
	Type      string      `json:"type"`
	From      string      `json:"from"`
	Content   string      `json:"content"`
	Timestamp float64     `json:"timestamp"`
	Extra     interface{} `json:"-"`
}

// rawInboxMsg is used for serialization with extra fields merged in.
type rawInboxMsg map[string]interface{}

type MessageBus struct {
	dir string
	mu  sync.Mutex
}

func NewMessageBus(dir string) *MessageBus {
	_ = os.MkdirAll(dir, 0o755)
	return &MessageBus{dir: dir}
}

// Send appends a message to a teammate's inbox. Extra fields are merged into the JSON.
func (bus *MessageBus) Send(sender, to, content, msgType string, extra map[string]interface{}) string {
	if msgType == "" {
		msgType = "message"
	}
	if !validMsgTypes[msgType] {
		return fmt.Sprintf("Error: Invalid type '%s'", msgType)
	}
	msg := rawInboxMsg{
		"type":      msgType,
		"from":      sender,
		"content":   content,
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

// ReadInbox reads all messages from a teammate's inbox and drains it.
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

// Broadcast sends a message to all teammates except the sender.
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
	tm := &TeammateManager{
		dir:        dir,
		configPath: filepath.Join(dir, "config.json"),
	}
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
		tm.config.Members = append(tm.config.Members, memberConfig{
			Name: name, Role: role, Status: "working",
		})
	}
	tm.saveConfig()

	go tm.teammateLoop(name, role, prompt)
	return fmt.Sprintf("Spawned '%s' (role: %s)", name, role)
}

// teammateLoop is the goroutine body: a complete agent loop for one teammate.
func (tm *TeammateManager) teammateLoop(name, role, prompt string) {
	sysPrompt := fmt.Sprintf(
		"You are '%s', role: %s, at %s. "+
			"Submit plans via plan_approval before major work. "+
			"Respond to shutdown_request with shutdown_response.",
		name, role, workdir,
	)
	tools := teammateTools()
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	shouldExit := false

	for i := 0; i < 50; i++ {
		// Check inbox
		inbox := bus.ReadInbox(name)
		if len(inbox) > 0 {
			for _, msg := range inbox {
				msgJSON, _ := json.Marshal(msg)
				messages = append(messages,
					anthropic.NewUserMessage(anthropic.NewTextBlock(string(msgJSON))),
				)
			}
		}
		if shouldExit {
			break
		}

		resp, err := apiClient.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			System:    []anthropic.TextBlockParam{{Text: sysPrompt}},
			Messages:  messages,
			Tools:     tools,
			MaxTokens: 8000,
		})
		if err != nil {
			break
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
				output := tm.execTool(name, block.Name, block.Input)
				fmt.Printf("  [%s] %s: %s\n", name, block.Name, truncate(output, 120))
				results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))

				// Check if teammate approved shutdown
				if block.Name == "shutdown_response" {
					var input struct {
						Approve bool `json:"approve"`
					}
					_ = json.Unmarshal(block.Input, &input)
					if input.Approve {
						shouldExit = true
					}
				}
			}
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}

	// Mark final status
	tm.mu.Lock()
	member := tm.findMember(name)
	if member != nil {
		if shouldExit {
			member.Status = "shutdown"
		} else {
			member.Status = "idle"
		}
		tm.saveConfig()
	}
	tm.mu.Unlock()
}

// execTool dispatches a tool call for a teammate.
func (tm *TeammateManager) execTool(sender, toolName string, rawInput json.RawMessage) string {
	switch toolName {
	case "bash":
		var input struct {
			Command string `json:"command"`
		}
		_ = json.Unmarshal(rawInput, &input)
		return runBash(input.Command)
	case "read_file":
		var input struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(rawInput, &input)
		return runRead(input.Path, 0)
	case "write_file":
		var input struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		_ = json.Unmarshal(rawInput, &input)
		return runWrite(input.Path, input.Content)
	case "edit_file":
		var input struct {
			Path    string `json:"path"`
			OldText string `json:"old_text"`
			NewText string `json:"new_text"`
		}
		_ = json.Unmarshal(rawInput, &input)
		return runEdit(input.Path, input.OldText, input.NewText)
	case "send_message":
		var input struct {
			To      string `json:"to"`
			Content string `json:"content"`
			MsgType string `json:"msg_type"`
		}
		_ = json.Unmarshal(rawInput, &input)
		return bus.Send(sender, input.To, input.Content, input.MsgType, nil)
	case "read_inbox":
		msgs := bus.ReadInbox(sender)
		data, _ := json.MarshalIndent(msgs, "", "  ")
		return string(data)
	case "shutdown_response":
		var input struct {
			RequestID string `json:"request_id"`
			Approve   bool   `json:"approve"`
			Reason    string `json:"reason"`
		}
		_ = json.Unmarshal(rawInput, &input)
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
		if input.Approve {
			return "Shutdown approved"
		}
		return "Shutdown rejected"
	case "plan_approval":
		var input struct {
			Plan string `json:"plan"`
		}
		_ = json.Unmarshal(rawInput, &input)
		reqID := randomID()
		trackerMu.Lock()
		planRequests[reqID] = &planReq{From: sender, Plan: input.Plan, Status: "pending"}
		trackerMu.Unlock()
		bus.Send(sender, "lead", input.Plan, "plan_approval_response",
			map[string]interface{}{"request_id": reqID, "plan": input.Plan})
		return fmt.Sprintf("Plan submitted (request_id=%s). Waiting for lead approval.", reqID)
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}

// teammateTools returns the 8 tools available to teammates.
func teammateTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name: "bash", Description: anthropic.String("Run a shell command."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"command": map[string]interface{}{"type": "string"}},
				Required:   []string{"command"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "read_file", Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}},
				Required:   []string{"path"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "write_file", Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "edit_file", Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path": map[string]interface{}{"type": "string"}, "old_text": map[string]interface{}{"type": "string"}, "new_text": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "old_text", "new_text"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "send_message", Description: anthropic.String("Send message to a teammate."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"to": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"},
					"msg_type": map[string]interface{}{"type": "string", "enum": []string{"message", "broadcast", "shutdown_request", "shutdown_response", "plan_approval_response"}},
				},
				Required: []string{"to", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "read_inbox", Description: anthropic.String("Read and drain your inbox."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "shutdown_response", Description: anthropic.String("Respond to a shutdown request. Approve to shut down, reject to keep working."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"request_id": map[string]interface{}{"type": "string"},
					"approve":    map[string]interface{}{"type": "boolean"},
					"reason":     map[string]interface{}{"type": "string"},
				},
				Required: []string{"request_id", "approve"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "plan_approval", Description: anthropic.String("Submit a plan for lead approval."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"plan": map[string]interface{}{"type": "string"}},
				Required:   []string{"plan"},
			},
		}},
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

// ========== Lead-specific protocol handlers ==========

func handleShutdownRequest(teammate string) string {
	reqID := randomID()
	trackerMu.Lock()
	shutdownRequests[reqID] = &shutdownReq{Target: teammate, Status: "pending"}
	trackerMu.Unlock()
	bus.Send("lead", teammate, "Please shut down gracefully.",
		"shutdown_request", map[string]interface{}{"request_id": reqID})
	return fmt.Sprintf("Shutdown request %s sent to '%s' (status: pending)", reqID, teammate)
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
	{"Spawn and shutdown", `Spawn alice as a coder. Then request her shutdown.`},
	{"Check status after shutdown", `/team`},
	{"Plan approval flow", `Spawn bob with a risky refactoring task. Review and reject his plan.`},
	{"Approve a plan", `Spawn charlie, have him submit a plan, then approve it.`},
}

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	workdir, _ = os.Getwd()

	teamDir := filepath.Join(workdir, ".team")
	bus = NewMessageBus(filepath.Join(teamDir, "inbox"))
	team = NewTeammateManager(teamDir)

	system = fmt.Sprintf("You are a team lead at %s. Manage teammates with shutdown and plan approval protocols.", workdir)

	// -- Lead tool definitions (12 tools) --
	tools = []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name: "bash", Description: anthropic.String("Run a shell command."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"command": map[string]interface{}{"type": "string"}},
				Required:   []string{"command"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "read_file", Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "limit": map[string]interface{}{"type": "integer"}},
				Required:   []string{"path"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "write_file", Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"}},
				Required:   []string{"path", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "edit_file", Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"path": map[string]interface{}{"type": "string"}, "old_text": map[string]interface{}{"type": "string"}, "new_text": map[string]interface{}{"type": "string"}},
				Required:   []string{"path", "old_text", "new_text"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "spawn_teammate", Description: anthropic.String("Spawn a persistent teammate."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"name": map[string]interface{}{"type": "string"}, "role": map[string]interface{}{"type": "string"}, "prompt": map[string]interface{}{"type": "string"}},
				Required:   []string{"name", "role", "prompt"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "list_teammates", Description: anthropic.String("List all teammates."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "send_message", Description: anthropic.String("Send a message to a teammate."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"to": map[string]interface{}{"type": "string"}, "content": map[string]interface{}{"type": "string"},
					"msg_type": map[string]interface{}{"type": "string", "enum": []string{"message", "broadcast", "shutdown_request", "shutdown_response", "plan_approval_response"}},
				},
				Required: []string{"to", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "read_inbox", Description: anthropic.String("Read and drain the lead's inbox."),
			InputSchema: anthropic.ToolInputSchemaParam{Properties: map[string]interface{}{}},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "broadcast", Description: anthropic.String("Send a message to all teammates."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"content": map[string]interface{}{"type": "string"}},
				Required:   []string{"content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "shutdown_request", Description: anthropic.String("Request a teammate to shut down gracefully. Returns request_id for tracking."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"teammate": map[string]interface{}{"type": "string"}},
				Required:   []string{"teammate"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "shutdown_response", Description: anthropic.String("Check the status of a shutdown request by request_id."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{"request_id": map[string]interface{}{"type": "string"}},
				Required:   []string{"request_id"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name: "plan_approval", Description: anthropic.String("Approve or reject a teammate's plan."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"request_id": map[string]interface{}{"type": "string"},
					"approve":    map[string]interface{}{"type": "boolean"},
					"feedback":   map[string]interface{}{"type": "string"},
				},
				Required: []string{"request_id", "approve"},
			},
		}},
	}

	dispatch = map[string]toolHandler{
		"bash":              handleBash,
		"read_file":         handleReadFile,
		"write_file":        handleWriteFile,
		"edit_file":         handleEditFile,
		"spawn_teammate":    handleSpawnTeammate,
		"list_teammates":    handleListTeammates,
		"send_message":      handleSendMessage,
		"read_inbox":        handleReadInboxLead,
		"broadcast":         handleBroadcast,
		"shutdown_request":  handleShutdownReq,
		"shutdown_response": handleShutdownResp,
		"plan_approval":     handlePlanApproval,
	}
}

// ========== Path safety ==========

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

// ========== Lead tool handlers ==========

func handleBash(raw json.RawMessage) string {
	var in struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(raw, &in)
	return runBash(in.Command)
}
func handleReadFile(raw json.RawMessage) string {
	var in struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}
	_ = json.Unmarshal(raw, &in)
	return runRead(in.Path, in.Limit)
}
func handleWriteFile(raw json.RawMessage) string {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	_ = json.Unmarshal(raw, &in)
	return runWrite(in.Path, in.Content)
}
func handleEditFile(raw json.RawMessage) string {
	var in struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	_ = json.Unmarshal(raw, &in)
	return runEdit(in.Path, in.OldText, in.NewText)
}
func handleSpawnTeammate(raw json.RawMessage) string {
	var in struct {
		Name   string `json:"name"`
		Role   string `json:"role"`
		Prompt string `json:"prompt"`
	}
	_ = json.Unmarshal(raw, &in)
	return team.Spawn(in.Name, in.Role, in.Prompt)
}
func handleListTeammates(raw json.RawMessage) string { return team.ListAll() }
func handleSendMessage(raw json.RawMessage) string {
	var in struct {
		To      string `json:"to"`
		Content string `json:"content"`
		MsgType string `json:"msg_type"`
	}
	_ = json.Unmarshal(raw, &in)
	return bus.Send("lead", in.To, in.Content, in.MsgType, nil)
}
func handleReadInboxLead(raw json.RawMessage) string {
	msgs := bus.ReadInbox("lead")
	data, _ := json.MarshalIndent(msgs, "", "  ")
	return string(data)
}
func handleBroadcast(raw json.RawMessage) string {
	var in struct {
		Content string `json:"content"`
	}
	_ = json.Unmarshal(raw, &in)
	return bus.Broadcast("lead", in.Content, team.MemberNames())
}
func handleShutdownReq(raw json.RawMessage) string {
	var in struct {
		Teammate string `json:"teammate"`
	}
	_ = json.Unmarshal(raw, &in)
	return handleShutdownRequest(in.Teammate)
}
func handleShutdownResp(raw json.RawMessage) string {
	var in struct {
		RequestID string `json:"request_id"`
	}
	_ = json.Unmarshal(raw, &in)
	return checkShutdownStatus(in.RequestID)
}
func handlePlanApproval(raw json.RawMessage) string {
	var in struct {
		RequestID string `json:"request_id"`
		Approve   bool   `json:"approve"`
		Feedback  string `json:"feedback"`
	}
	_ = json.Unmarshal(raw, &in)
	return handlePlanReview(in.RequestID, in.Approve, in.Feedback)
}

// ========== Tool implementations ==========

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
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
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
	newContent := strings.Replace(content, oldText, newText, 1)
	if err := os.WriteFile(fp, []byte(newContent), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return fmt.Sprintf("Edited %s", path)
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
				anthropic.NewUserMessage(
					anthropic.NewTextBlock(fmt.Sprintf("<inbox>%s</inbox>", string(inboxJSON))),
				),
				anthropic.NewAssistantMessage(
					anthropic.NewTextBlock("Noted inbox messages."),
				),
			)
		}

		resp, err := apiClient.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			System:    []anthropic.TextBlockParam{{Text: system}},
			Messages:  messages,
			Tools:     tools,
			MaxTokens: 8000,
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

	fmt.Println("\n  s10: Team Protocols")
	fmt.Print("  \"Graceful shutdown & plan approval\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	for {
		var query string

		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms10 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
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
			fmt.Printf("\033[36ms10 >> \033[0m")
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
