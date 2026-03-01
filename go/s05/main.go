// s05 - Skills
//
// Two-layer skill injection that avoids bloating the system prompt:
//
//	Layer 1 (cheap): skill names in system prompt (~100 tokens/skill)
//	Layer 2 (on demand): full skill body in tool_result
//
//	System prompt:
//	+--------------------------------------+
//	| You are a coding agent.              |
//	| Skills available:                    |
//	|   - git: Git workflow helpers        |  <-- Layer 1: metadata only
//	|   - test: Testing best practices     |
//	+--------------------------------------+
//
//	When model calls load_skill("git"):
//	+--------------------------------------+
//	| tool_result:                         |
//	| <skill>                              |
//	|   Full git workflow instructions...  |  <-- Layer 2: full body
//	|   Step 1: ...                        |
//	|   Step 2: ...                        |
//	| </skill>                             |
//	+--------------------------------------+
//
// Key insight: "Don't put everything in the system prompt. Load on demand."
package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// ========== SkillLoader ==========

// Skill holds parsed metadata and body from a SKILL.md file.
type Skill struct {
	Name string
	Meta map[string]string // frontmatter key-value pairs
	Body string            // content after frontmatter
	Path string            // file path for debugging
}

// SkillLoader discovers and parses skill files from a directory.
type SkillLoader struct {
	Skills map[string]*Skill
}

// NewSkillLoader scans the given directory for skills.
// Supports two layouts:
//   - flat:   .skills/git.md       (s05 Python convention)
//   - nested: skills/git/SKILL.md  (rubickx convention)
func NewSkillLoader(dirs ...string) *SkillLoader {
	sl := &SkillLoader{Skills: make(map[string]*Skill)}
	for _, dir := range dirs {
		sl.loadDir(dir)
	}
	return sl
}

func (sl *SkillLoader) loadDir(dir string) {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			// nested layout: skills/<name>/SKILL.md
			skillFile := filepath.Join(dir, e.Name(), "SKILL.md")
			if data, err := os.ReadFile(skillFile); err == nil {
				meta, body := parseFrontmatter(string(data))
				sl.Skills[e.Name()] = &Skill{
					Name: e.Name(), Meta: meta, Body: body, Path: skillFile,
				}
			}
		} else if strings.HasSuffix(e.Name(), ".md") {
			// flat layout: .skills/<name>.md
			name := strings.TrimSuffix(e.Name(), ".md")
			path := filepath.Join(dir, e.Name())
			if data, err := os.ReadFile(path); err == nil {
				meta, body := parseFrontmatter(string(data))
				sl.Skills[name] = &Skill{
					Name: name, Meta: meta, Body: body, Path: path,
				}
			}
		}
	}
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)`)

// parseFrontmatter extracts YAML-like key: value pairs between --- delimiters.
func parseFrontmatter(text string) (map[string]string, string) {
	match := frontmatterRe.FindStringSubmatch(text)
	if match == nil {
		return map[string]string{}, text
	}
	meta := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(match[1]), "\n") {
		if idx := strings.Index(line, ":"); idx >= 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			meta[key] = val
		}
	}
	return meta, strings.TrimSpace(match[2])
}

// GetDescriptions returns Layer 1: short descriptions for the system prompt.
func (sl *SkillLoader) GetDescriptions() string {
	if len(sl.Skills) == 0 {
		return "(no skills available)"
	}
	// Sort for deterministic output
	names := make([]string, 0, len(sl.Skills))
	for name := range sl.Skills {
		names = append(names, name)
	}
	sort.Strings(names)

	var lines []string
	for _, name := range names {
		skill := sl.Skills[name]
		desc := skill.Meta["description"]
		if desc == "" {
			desc = "No description"
		}
		line := fmt.Sprintf("  - %s: %s", name, desc)
		if tags, ok := skill.Meta["tags"]; ok && tags != "" {
			line += fmt.Sprintf(" [%s]", tags)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

// GetContent returns Layer 2: full skill body wrapped in <skill> tags.
func (sl *SkillLoader) GetContent(name string) string {
	skill, ok := sl.Skills[name]
	if !ok {
		available := make([]string, 0, len(sl.Skills))
		for k := range sl.Skills {
			available = append(available, k)
		}
		return fmt.Sprintf("Error: Unknown skill '%s'. Available: %s", name, strings.Join(available, ", "))
	}
	return fmt.Sprintf("<skill name=%q>\n%s\n</skill>", name, skill.Body)
}

// ========== Globals ==========

var (
	model       string
	system      string
	workdir     string
	skillLoader *SkillLoader
	tools       []anthropic.ToolUnionParam
)

type toolHandler func(input json.RawMessage) string

var dispatch map[string]toolHandler

func init() {
	model = os.Getenv("MODEL_ID")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	workdir, _ = os.Getwd()

	// Load skills from both possible directories
	skillLoader = NewSkillLoader(
		filepath.Join(workdir, ".skills"),
		filepath.Join(workdir, "skills"),
	)

	// Layer 1: skill metadata injected into system prompt
	system = fmt.Sprintf(`You are a coding agent at %s.
Use load_skill to access specialized knowledge before tackling unfamiliar topics.

Skills available:
%s`, workdir, skillLoader.GetDescriptions())

	// -- Tool definitions --
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
			Name:        "load_skill",
			Description: anthropic.String("Load specialized knowledge by name."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"name": map[string]interface{}{"type": "string", "description": "Skill name to load"},
				},
				Required: []string{"name"},
			},
		}},
	}

	// -- Dispatch map --
	dispatch = map[string]toolHandler{
		"bash":       handleBash,
		"read_file":  handleReadFile,
		"write_file": handleWriteFile,
		"edit_file":  handleEditFile,
		"load_skill": handleLoadSkill,
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

func handleLoadSkill(raw json.RawMessage) string {
	var input struct {
		Name string `json:"name"`
	}
	_ = json.Unmarshal(raw, &input)
	return skillLoader.GetContent(input.Name)
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

// ========== Agent Loop ==========

func agentLoop(client *anthropic.Client, messages []anthropic.MessageParam) []anthropic.MessageParam {
	for {
		resp, err := client.Messages.New(context.Background(), anthropic.MessageNewParams{
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
	client := anthropic.NewClient(opts...)

	var messages []anthropic.MessageParam
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\033[36ms05 >> \033[0m")
		if !scanner.Scan() {
			break
		}
		query := strings.TrimSpace(scanner.Text())
		if query == "" || query == "q" || query == "exit" {
			break
		}

		messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(query)))
		messages = agentLoop(&client, messages)

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
