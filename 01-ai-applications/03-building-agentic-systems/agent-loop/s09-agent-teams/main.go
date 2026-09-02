// s09 - Agent Teams
//
// Persistent named agents with file-based JSONL inboxes. Each teammate runs
// its own agent loop in a separate goroutine. Communication via append-only inboxes.
//
//	Subagent (s04):  spawn -> execute -> return summary -> destroyed
//	Teammate (s09):  spawn -> work -> idle -> work -> ... -> shutdown
//
//	.team/config.json                   .team/inbox/
//	+----------------------------+      +------------------+
//	| {"team_name": "default",   |      | alice.jsonl      |
//	|  "members": [              |      | bob.jsonl        |
//	|    {"name":"alice",        |      | lead.jsonl       |
//	|     "role":"coder",        |      +------------------+
//	|     "status":"idle"}       |
//	|  ]}                        |      send_message("alice", "fix bug"):
//	+----------------------------+        open("alice.jsonl", "a").write(msg)
//
//	                                    read_inbox("alice"):
//	spawn_teammate("alice","coder",...)   msgs = [json.loads(l) for l in ...]
//	     |                                open("alice.jsonl", "w").close()
//	     v                                return msgs  # drain
//	Goroutine: alice           Goroutine: bob
//	+------------------+      +------------------+
//	| agent_loop       |      | agent_loop       |
//	| status: working  |      | status: idle     |
//	| ... runs tools   |      | ... waits ...    |
//	| status -> idle   |      |                  |
//	+------------------+      +------------------+
//
// Key insight: "Teammates that can talk to each other."
package main

import (
	"bufio"
	"context"
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

// ========== MessageBus: JSONL inbox per teammate ==========

var validMsgTypes = map[string]bool{
	"message":                true,
	"broadcast":              true,
	"shutdown_request":       true,
	"shutdown_response":      true,
	"plan_approval_response": true,
}

// MessageBus manages file-based JSONL inboxes for inter-agent communication.
type MessageBus struct {
	dir string
	mu  sync.Mutex
}

// NewMessageBus creates a bus with the given inbox directory.
func NewMessageBus(dir string) *MessageBus {
	_ = os.MkdirAll(dir, 0o755)
	return &MessageBus{dir: dir}
}

// inboxMsg represents a single message in a JSONL inbox.
type inboxMsg struct {
	Type      string  `json:"type"`
	From      string  `json:"from"`
	Content   string  `json:"content"`
	Timestamp float64 `json:"timestamp"`
}

// Send appends a message to a teammate's inbox JSONL file.
func (bus *MessageBus) Send(sender, to, content, msgType string) string {
	if msgType == "" {
		msgType = "message"
	}
	if !validMsgTypes[msgType] {
		return fmt.Sprintf("Error: Invalid type '%s'", msgType)
	}
	msg := inboxMsg{
		Type:      msgType,
		From:      sender,
		Content:   content,
		Timestamp: float64(time.Now().UnixMilli()) / 1000.0,
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

// ReadInbox reads all messages from a teammate's inbox and drains it (clears the file).
func (bus *MessageBus) ReadInbox(name string) []inboxMsg {
	bus.mu.Lock()
	defer bus.mu.Unlock()

	path := filepath.Join(bus.dir, name+".jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var messages []inboxMsg
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var msg inboxMsg
		if json.Unmarshal([]byte(line), &msg) == nil {
			messages = append(messages, msg)
		}
	}

	// Drain: clear the file
	_ = os.WriteFile(path, []byte(""), 0o644)
	return messages
}

// Broadcast sends a message to all teammates except the sender.
func (bus *MessageBus) Broadcast(sender, content string, teammates []string) string {
	count := 0
	for _, name := range teammates {
		if name != sender {
			bus.Send(sender, name, content, "broadcast")
			count++
		}
	}
	return fmt.Sprintf("Broadcast to %d teammates", count)
}

// ========== TeammateManager: persistent named agents ==========

// memberConfig is one entry in the team config.
type memberConfig struct {
	Name   string `json:"name"`
	Role   string `json:"role"`
	Status string `json:"status"` // idle | working | shutdown
}

// teamConfig is the team roster stored in config.json.
type teamConfig struct {
	TeamName string         `json:"team_name"`
	Members  []memberConfig `json:"members"`
}

// TeammateManager manages persistent named agents with config.json.
type TeammateManager struct {
	dir        string
	configPath string
	config     teamConfig
	mu         sync.Mutex
}

// NewTeammateManager loads or creates a team roster.
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

// Spawn creates or reactivates a teammate, running its agent loop in a goroutine.
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
		"You are '%s', role: %s, at %s. Use send_message to communicate. Complete your task.",
		name, role, workdir,
	)
	tools := teammateTools()
	messages := []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
	}

	for i := 0; i < 50; i++ {
		// Check inbox before each LLM call
		inbox := bus.ReadInbox(name)
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
			System:    []anthropic.TextBlockParam{{Text: sysPrompt}},
			Messages:  messages,
			Tools:     tools,
			MaxTokens: 8000,
		})
		if err != nil {
			break
		}

		// Append assistant turn
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

		// Execute tools
		var results []anthropic.ContentBlockParamUnion
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				output := tm.execTool(name, block.Name, block.Input)
				fmt.Printf("  [%s] %s: %s\n", name, block.Name, truncate(output, 120))
				results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
			}
		}
		messages = append(messages, anthropic.NewUserMessage(results...))
	}

	// Mark idle when done
	tm.mu.Lock()
	member := tm.findMember(name)
	if member != nil && member.Status != "shutdown" {
		member.Status = "idle"
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
		return bus.Send(sender, input.To, input.Content, input.MsgType)
	case "read_inbox":
		msgs := bus.ReadInbox(sender)
		data, _ := json.MarshalIndent(msgs, "", "  ")
		return string(data)
	default:
		return fmt.Sprintf("Unknown tool: %s", toolName)
	}
}

// teammateTools returns the tool set available to teammates (no spawn, no broadcast).
func teammateTools() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "bash",
			Description: anthropic.String("Run a shell command."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"command": map[string]interface{}{"type": "string"},
				},
				Required: []string{"command"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "read_file",
			Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "write_file",
			Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":    map[string]interface{}{"type": "string"},
					"content": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "edit_file",
			Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":     map[string]interface{}{"type": "string"},
					"old_text": map[string]interface{}{"type": "string"},
					"new_text": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "old_text", "new_text"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "send_message",
			Description: anthropic.String("Send message to a teammate."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"to":       map[string]interface{}{"type": "string"},
					"content":  map[string]interface{}{"type": "string"},
					"msg_type": map[string]interface{}{"type": "string", "enum": []string{"message", "broadcast", "shutdown_request", "shutdown_response", "plan_approval_response"}},
				},
				Required: []string{"to", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "read_inbox",
			Description: anthropic.String("Read and drain your inbox."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{},
			},
		}},
	}
}

// ListAll returns a summary of all teammates.
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

// MemberNames returns a slice of all teammate names.
func (tm *TeammateManager) MemberNames() []string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	names := make([]string, len(tm.config.Members))
	for i, m := range tm.config.Members {
		names[i] = m.Name
	}
	return names
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
	{"Spawn teammates", `Spawn alice (coder) and bob (tester). Have alice send bob a message.`},
	{"Broadcast message", `Broadcast "status update: phase 1 complete" to all teammates`},
	{"Check lead inbox", `Check the lead inbox for any messages`},
	{"See team roster", `/team`},
	{"Drain inbox manually", `/inbox`},
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

	system = fmt.Sprintf("You are a team lead at %s. Spawn teammates and communicate via inboxes.", workdir)

	// -- Lead tool definitions (9 tools) --
	tools = []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "bash",
			Description: anthropic.String("Run a shell command."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"command": map[string]interface{}{"type": "string"},
				},
				Required: []string{"command"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "read_file",
			Description: anthropic.String("Read file contents."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":  map[string]interface{}{"type": "string"},
					"limit": map[string]interface{}{"type": "integer"},
				},
				Required: []string{"path"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "write_file",
			Description: anthropic.String("Write content to file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":    map[string]interface{}{"type": "string"},
					"content": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "edit_file",
			Description: anthropic.String("Replace exact text in file."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"path":     map[string]interface{}{"type": "string"},
					"old_text": map[string]interface{}{"type": "string"},
					"new_text": map[string]interface{}{"type": "string"},
				},
				Required: []string{"path", "old_text", "new_text"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "spawn_teammate",
			Description: anthropic.String("Spawn a persistent teammate that runs in its own goroutine."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"name":   map[string]interface{}{"type": "string"},
					"role":   map[string]interface{}{"type": "string"},
					"prompt": map[string]interface{}{"type": "string"},
				},
				Required: []string{"name", "role", "prompt"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "list_teammates",
			Description: anthropic.String("List all teammates with name, role, status."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "send_message",
			Description: anthropic.String("Send a message to a teammate's inbox."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"to":       map[string]interface{}{"type": "string"},
					"content":  map[string]interface{}{"type": "string"},
					"msg_type": map[string]interface{}{"type": "string", "enum": []string{"message", "broadcast", "shutdown_request", "shutdown_response", "plan_approval_response"}},
				},
				Required: []string{"to", "content"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "read_inbox",
			Description: anthropic.String("Read and drain the lead's inbox."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "broadcast",
			Description: anthropic.String("Send a message to all teammates."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"content": map[string]interface{}{"type": "string"},
				},
				Required: []string{"content"},
			},
		}},
	}

	// -- Lead dispatch map --
	dispatch = map[string]toolHandler{
		"bash":           handleBash,
		"read_file":      handleReadFile,
		"write_file":     handleWriteFile,
		"edit_file":      handleEditFile,
		"spawn_teammate": handleSpawnTeammate,
		"list_teammates": handleListTeammates,
		"send_message":   handleSendMessage,
		"read_inbox":     handleReadInbox,
		"broadcast":      handleBroadcast,
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
	var input struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(raw, &input)
	return runBash(input.Command)
}

func handleReadFile(raw json.RawMessage) string {
	var input struct {
		Path  string `json:"path"`
		Limit int    `json:"limit"`
	}
	_ = json.Unmarshal(raw, &input)
	return runRead(input.Path, input.Limit)
}

func handleWriteFile(raw json.RawMessage) string {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	_ = json.Unmarshal(raw, &input)
	return runWrite(input.Path, input.Content)
}

func handleEditFile(raw json.RawMessage) string {
	var input struct {
		Path    string `json:"path"`
		OldText string `json:"old_text"`
		NewText string `json:"new_text"`
	}
	_ = json.Unmarshal(raw, &input)
	return runEdit(input.Path, input.OldText, input.NewText)
}

func handleSpawnTeammate(raw json.RawMessage) string {
	var input struct {
		Name   string `json:"name"`
		Role   string `json:"role"`
		Prompt string `json:"prompt"`
	}
	_ = json.Unmarshal(raw, &input)
	return team.Spawn(input.Name, input.Role, input.Prompt)
}

func handleListTeammates(raw json.RawMessage) string {
	return team.ListAll()
}

func handleSendMessage(raw json.RawMessage) string {
	var input struct {
		To      string `json:"to"`
		Content string `json:"content"`
		MsgType string `json:"msg_type"`
	}
	_ = json.Unmarshal(raw, &input)
	return bus.Send("lead", input.To, input.Content, input.MsgType)
}

func handleReadInbox(raw json.RawMessage) string {
	msgs := bus.ReadInbox("lead")
	data, _ := json.MarshalIndent(msgs, "", "  ")
	return string(data)
}

func handleBroadcast(raw json.RawMessage) string {
	var input struct {
		Content string `json:"content"`
	}
	_ = json.Unmarshal(raw, &input)
	return bus.Broadcast("lead", input.Content, team.MemberNames())
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

// ========== Agent Loop (lead) ==========

func agentLoop(messages []anthropic.MessageParam) []anthropic.MessageParam {
	for {
		// Drain lead's inbox before each LLM call
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

		// Append assistant turn
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

		// Execute tools via dispatch
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

	fmt.Println("\n  s09: Agent Teams")
	fmt.Print("  \"Multi-agent coordination with message passing\"\n\n")

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false

	for {
		// Local commands check helper (used below)
		var query string

		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  → %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms09 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
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
			fmt.Printf("\033[36ms09 >> \033[0m")
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

		// Print last assistant text
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
