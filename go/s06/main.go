// s06 - Compact
//
// Three-layer compression pipeline so the agent can work forever:
//
//	Every turn:
//	+------------------+
//	| Tool call result |
//	+------------------+
//	        |
//	        v
//	[Layer 1: microCompact]        (silent, every turn)
//	  Replace tool_result content older than last 3
//	  with "[Previous: used {tool_name}]"
//	        |
//	        v
//	[Check: tokens > 50000?]
//	   |               |
//	   no              yes
//	   |               |
//	   v               v
//	continue    [Layer 2: autoCompact]
//	              Save full transcript to .transcripts/
//	              Ask LLM to summarize conversation.
//	              Replace all messages with [summary].
//	                    |
//	                    v
//	            [Layer 3: compact tool]
//	              Model calls compact -> immediate summarization.
//	              Same as auto, triggered manually.
//
// Key insight: "The agent can forget strategically and keep working forever."
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
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ========== Constants ==========

const (
	threshold  = 50000 // token estimate threshold for auto_compact
	keepRecent = 3     // number of recent tool results to preserve
)

// ========== Globals ==========

var (
	model         string
	system        string
	workdir       string
	transcriptDir string
	tools         []anthropic.ToolUnionParam
	globalClient  *anthropic.Client
)

type toolHandler func(input json.RawMessage) string

var dispatch map[string]toolHandler

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	workdir, _ = os.Getwd()
	transcriptDir = filepath.Join(workdir, ".transcripts")
	system = fmt.Sprintf("You are a coding agent at %s. Use tools to solve tasks.", workdir)

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
			Name:        "compact",
			Description: anthropic.String("Trigger manual conversation compression."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"focus": map[string]interface{}{
						"type":        "string",
						"description": "What to preserve in the summary",
					},
				},
			},
		}},
	}

	dispatch = map[string]toolHandler{
		"bash":      handleBash,
		"read_file": handleReadFile,
		"write_file": handleWriteFile,
		"edit_file":  handleEditFile,
	}
}

// ========== Token estimation ==========

// estimateTokens gives a rough token count: ~4 chars per token.
func estimateTokens(messages []anthropic.MessageParam) int {
	data, _ := json.Marshal(messages)
	return len(data) / 4
}

// ========== Layer 1: micro_compact ==========

// microCompact replaces old tool_result contents with short placeholders.
// Keeps only the last keepRecent tool results intact.
func microCompact(messages []anthropic.MessageParam) {
	// Collect all tool result locations
	type toolResultLoc struct {
		msgIdx  int
		partIdx int
	}
	var locs []toolResultLoc

	for i, msg := range messages {
		if msg.Role != "user" {
			continue
		}
		for j, part := range msg.Content {
			if part.OfToolResult != nil {
				locs = append(locs, toolResultLoc{i, j})
			}
		}
	}

	if len(locs) <= keepRecent {
		return
	}

	// Build tool_use_id → tool_name map from assistant messages
	toolNameMap := map[string]string{}
	for _, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for _, part := range msg.Content {
			if part.OfToolUse != nil {
				toolNameMap[part.OfToolUse.ID] = part.OfToolUse.Name
			}
		}
	}

	// Clear old results (keep last keepRecent)
	toClear := locs[:len(locs)-keepRecent]
	for _, loc := range toClear {
		tr := messages[loc.msgIdx].Content[loc.partIdx].OfToolResult
		if tr == nil {
			continue
		}
		// Check if content is long enough to compact
		contentLen := 0
		for _, c := range tr.Content {
			if c.OfText != nil {
				contentLen += len(c.OfText.Text)
			}
		}
		if contentLen > 100 {
			toolName := toolNameMap[tr.ToolUseID]
			if toolName == "" {
				toolName = "unknown"
			}
			tr.Content = []anthropic.ToolResultBlockParamContentUnion{
				{OfText: &anthropic.TextBlockParam{Text: fmt.Sprintf("[Previous: used %s]", toolName)}},
			}
		}
	}
}

// ========== Layer 2: auto_compact ==========

// autoCompact saves the full transcript to disk, asks the LLM to summarize,
// and returns a fresh two-message history with the summary.
func autoCompact(messages []anthropic.MessageParam) []anthropic.MessageParam {
	// Save transcript to .transcripts/
	os.MkdirAll(transcriptDir, 0o755)
	transcriptPath := filepath.Join(transcriptDir,
		fmt.Sprintf("transcript_%d.jsonl", time.Now().Unix()))

	f, err := os.Create(transcriptPath)
	if err == nil {
		encoder := json.NewEncoder(f)
		for _, msg := range messages {
			encoder.Encode(msg)
		}
		f.Close()
	}
	fmt.Printf("[transcript saved: %s]\n", transcriptPath)

	// Ask LLM to summarize
	conversationData, _ := json.Marshal(messages)
	conversationText := string(conversationData)
	if len(conversationText) > 80000 {
		conversationText = conversationText[:80000]
	}

	resp, err := globalClient.Messages.New(context.Background(), anthropic.MessageNewParams{
		Model: anthropic.Model(model),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(
				"Summarize this conversation for continuity. Include: " +
					"1) What was accomplished, 2) Current state, 3) Key decisions made. " +
					"Be concise but preserve critical details.\n\n" + conversationText)),
		},
		MaxTokens: 2000,
	})

	summary := "(summary failed)"
	if err == nil && len(resp.Content) > 0 {
		summary = resp.Content[0].Text
	}

	// Replace all messages with compressed summary
	return []anthropic.MessageParam{
		anthropic.NewUserMessage(anthropic.NewTextBlock(
			fmt.Sprintf("[Conversation compressed. Transcript: %s]\n\n%s", transcriptPath, summary))),
		anthropic.NewAssistantMessage(anthropic.NewTextBlock(
			"Understood. I have the context from the summary. Continuing.")),
	}
}

// ========== Path safety ==========

func safePath(p string) (string, error) {
	full := filepath.Join(workdir, p)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", p)
	}
	if !strings.HasPrefix(abs, workdir) {
		return "", fmt.Errorf("path escapes workspace: %s", p)
	}
	return abs, nil
}

// ========== Tool handlers ==========

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

// ========== Agent Loop (with three-layer compression) ==========

func agentLoop(messages *[]anthropic.MessageParam) {
	for {
		// Layer 1: micro_compact before each LLM call
		microCompact(*messages)

		// Layer 2: auto_compact if token estimate exceeds threshold
		if estimateTokens(*messages) > threshold {
			fmt.Println("[auto_compact triggered]")
			*messages = autoCompact(*messages)
		}

		resp, err := globalClient.Messages.New(context.Background(), anthropic.MessageNewParams{
			Model:     anthropic.Model(model),
			System:    []anthropic.TextBlockParam{{Text: system}},
			Messages:  *messages,
			Tools:     tools,
			MaxTokens: 8000,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "API error: %v\n", err)
			return
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
		*messages = append(*messages, anthropic.NewAssistantMessage(assistantBlocks...))

		if resp.StopReason != "tool_use" {
			return
		}

		// Execute tools
		var results []anthropic.ContentBlockParamUnion
		manualCompact := false
		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				var output string
				if block.Name == "compact" {
					manualCompact = true
					output = "Compressing..."
				} else {
					handler, ok := dispatch[block.Name]
					if ok {
						output = handler(block.Input)
					} else {
						output = fmt.Sprintf("Unknown tool: %s", block.Name)
					}
				}
				fmt.Printf("> %s: %s\n", block.Name, truncate(output, 200))
				results = append(results, anthropic.NewToolResultBlock(block.ID, output, false))
			}
		}
		*messages = append(*messages, anthropic.NewUserMessage(results...))

		// Layer 3: manual compact triggered by the compact tool
		if manualCompact {
			fmt.Println("[manual compact]")
			*messages = autoCompact(*messages)
		}
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
	client := anthropic.NewClient(opts...)
	globalClient = &client

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\033[36ms06 >> \033[0m")
		if !scanner.Scan() {
			break
		}
		query := strings.TrimSpace(scanner.Text())
		if query == "" || query == "q" || query == "exit" {
			break
		}

		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
		agentLoop(&messages)

		// Print the last assistant response text
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
