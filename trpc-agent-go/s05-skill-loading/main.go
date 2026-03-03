// s05 - Skill Loading (trpc-agent-go)
//
// Two-layer skill injection so the system prompt stays lean:
//
//	Layer 1 (cheap): skill names+descriptions in system prompt every turn
//	Layer 2 (on demand): full skill body returned by load_skill FunctionTool
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
//	| </skill>                             |
//	+--------------------------------------+
//
// Key insight: "Don't put everything in the system prompt. Load on demand."
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"trpc.group/trpc-go/trpc-agent-go/agent/llmagent"
	"trpc.group/trpc-go/trpc-agent-go/event"
	"trpc.group/trpc-go/trpc-agent-go/model"
	"trpc.group/trpc-go/trpc-agent-go/model/openai"
	"trpc.group/trpc-go/trpc-agent-go/runner"
	"trpc.group/trpc-go/trpc-agent-go/session/inmemory"
	"trpc.group/trpc-go/trpc-agent-go/tool"
	"trpc.group/trpc-go/trpc-agent-go/tool/function"
)

// ========== SkillLoader ==========

// Skill holds parsed metadata and body from a skill markdown file.
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

// NewSkillLoader scans the given directories for skills.
// Supports two layouts:
//   - flat:   .skills/git.md
//   - nested: skills/git/SKILL.md
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
		sort.Strings(available)
		return fmt.Sprintf("Error: Unknown skill '%s'. Available: %s", name, strings.Join(available, ", "))
	}
	return fmt.Sprintf("<skill name=%q>\n%s\n</skill>", name, skill.Body)
}

// ========== Globals ==========

var (
	workdir     string
	skillLoader *SkillLoader
)

var guided = []struct {
	step   string
	prompt string
}{
	{"Discover available skills", `What skills are available?`},
	{"Load and use a skill", `Load the git skill and show me its instructions`},
	{"Contextual skill loading", `I need to do a code review -- load the relevant skill first`},
	{"Complex skill workflow", `Use the agent-builder skill to plan an MCP server`},
}

// ========== Input types ==========

type BashInput struct {
	Command string `json:"command" description:"Shell command to execute"`
}

type ReadFileInput struct {
	Path  string `json:"path" description:"File path to read"`
	Limit int    `json:"limit,omitempty" description:"Max lines to return"`
}

type WriteFileInput struct {
	Path    string `json:"path" description:"File path to write"`
	Content string `json:"content" description:"Content to write"`
}

type EditFileInput struct {
	Path    string `json:"path" description:"File path to edit"`
	OldText string `json:"old_text" description:"Exact text to find and replace"`
	NewText string `json:"new_text" description:"Text to replace with"`
}

type LoadSkillInput struct {
	Name string `json:"name" description:"Skill name to load (e.g. 'git', 'test')"`
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

// ========== Tool implementations ==========

func handleBash(_ context.Context, input BashInput) (string, error) {
	dangerous := []string{"rm -rf /", "sudo", "shutdown", "reboot", "> /dev/"}
	for _, d := range dangerous {
		if strings.Contains(input.Command, d) {
			return "Error: Dangerous command blocked", nil
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", input.Command)
	cmd.Dir = workdir
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "Error: Timeout (120s)", nil
	}
	out := strings.TrimSpace(string(output))
	if err != nil && out == "" {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if out == "" {
		return "(no output)", nil
	}
	if len(out) > 50000 {
		return out[:50000], nil
	}
	return out, nil
}

func handleReadFile(_ context.Context, input ReadFileInput) (string, error) {
	fp, err := safePath(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	lines := strings.Split(string(data), "\n")
	if input.Limit > 0 && input.Limit < len(lines) {
		lines = append(lines[:input.Limit], fmt.Sprintf("... (%d more lines)", len(lines)-input.Limit))
	}
	out := strings.Join(lines, "\n")
	if len(out) > 50000 {
		return out[:50000], nil
	}
	return out, nil
}

func handleWriteFile(_ context.Context, input WriteFileInput) (string, error) {
	fp, err := safePath(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if err := os.MkdirAll(filepath.Dir(fp), 0o755); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	if err := os.WriteFile(fp, []byte(input.Content), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("Wrote %d bytes to %s", len(input.Content), input.Path), nil
}

func handleEditFile(_ context.Context, input EditFileInput) (string, error) {
	fp, err := safePath(input.Path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	data, err := os.ReadFile(fp)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	content := string(data)
	if !strings.Contains(content, input.OldText) {
		return fmt.Sprintf("Error: Text not found in %s", input.Path), nil
	}
	newContent := strings.Replace(content, input.OldText, input.NewText, 1)
	if err := os.WriteFile(fp, []byte(newContent), 0o644); err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}
	return fmt.Sprintf("Edited %s", input.Path), nil
}

// handleLoadSkill is Layer 2: returns full skill body on demand.
func handleLoadSkill(_ context.Context, input LoadSkillInput) (string, error) {
	return skillLoader.GetContent(input.Name), nil
}

// ========== Event consumer ==========

func consumeEvents(events <-chan *event.Event) string {
	var lastText string
	for ev := range events {
		if ev.IsRunnerCompletion() {
			break
		}
		if ev.Response == nil {
			continue
		}
		for _, choice := range ev.Response.Choices {
			if choice.Delta.Content != "" {
				fmt.Print(choice.Delta.Content)
			}
			if choice.Message.Content != "" {
				lastText = choice.Message.Content
			}
			for _, tc := range choice.Delta.ToolCalls {
				if tc.Function.Name != "" {
					fmt.Printf("\033[33m> %s\033[0m\n", tc.Function.Name)
				}
			}
		}
	}
	return lastText
}

// ========== Main ==========

func main() {
	workdir, _ = os.Getwd()

	// Load skills from both possible directories
	skillLoader = NewSkillLoader(
		filepath.Join(workdir, ".skills"),
		filepath.Join(workdir, "skills"),
	)

	// -- Model --
	modelID := os.Getenv("MODEL_ID")
	if modelID == "" {
		modelID = "gpt-4o"
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	var modelOpts []openai.Option
	modelOpts = append(modelOpts, openai.WithAPIKey(apiKey))
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		modelOpts = append(modelOpts, openai.WithBaseURL(baseURL))
	}
	m := openai.New(modelID, modelOpts...)

	// -- Tools --
	tools := []tool.Tool{
		function.NewFunctionTool(handleBash,
			function.WithName("bash"),
			function.WithDescription("Run a shell command."),
		),
		function.NewFunctionTool(handleReadFile,
			function.WithName("read_file"),
			function.WithDescription("Read file contents."),
		),
		function.NewFunctionTool(handleWriteFile,
			function.WithName("write_file"),
			function.WithDescription("Write content to file."),
		),
		function.NewFunctionTool(handleEditFile,
			function.WithName("edit_file"),
			function.WithDescription("Replace exact text in file."),
		),
		function.NewFunctionTool(handleLoadSkill,
			function.WithName("load_skill"),
			function.WithDescription("Load specialized knowledge by name. Call this before tackling unfamiliar topics."),
		),
	}

	// Layer 1: skill metadata in system prompt (cheap, always present)
	instruction := fmt.Sprintf(
		`You are a coding agent at %s.
Use load_skill to access specialized knowledge before tackling unfamiliar topics.

Skills available:
%s`, workdir, skillLoader.GetDescriptions())

	// -- Agent --
	ag := llmagent.New("coder",
		llmagent.WithModel(m),
		llmagent.WithInstruction(instruction),
		llmagent.WithTools(tools),
		llmagent.WithGenerationConfig(model.GenerationConfig{
			MaxTokens: model.IntPtr(8000),
			Stream:    true,
		}),
		llmagent.WithMaxToolIterations(20),
	)

	// -- Runner --
	sess := inmemory.NewSessionService()
	r := runner.NewRunner("rubickx", ag, runner.WithSessionService(sess))
	defer r.Close()

	// -- REPL --
	fmt.Println("\n  s05: Skill Loading (trpc-agent-go)")
	fmt.Print("  \"Don't put everything in the system prompt. Load on demand.\"\n\n")

	scanner := bufio.NewScanner(os.Stdin)
	guidedIdx := 0
	freeModePrinted := false
	sessionID := "session-1"

	for {
		if guidedIdx < len(guided) {
			fmt.Printf("  Step %d/%d: %s\n", guidedIdx+1, len(guided), guided[guidedIdx].step)
			fmt.Printf("  -> %s\n", guided[guidedIdx].prompt)
			fmt.Printf("\033[36ms05 [%d/%d] >> \033[0m", guidedIdx+1, len(guided))
		} else {
			if !freeModePrinted {
				fmt.Print("\n  Guided tour complete. Free mode -- type anything, or q to quit.\n\n")
				freeModePrinted = true
			}
			fmt.Print("\033[36ms05 >> \033[0m")
		}

		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "q" || input == "exit" {
			break
		}

		if guidedIdx < len(guided) {
			query := input
			if query == "" {
				query = guided[guidedIdx].prompt
			}
			guidedIdx++
			events, err := r.Run(context.Background(), "user", sessionID, model.NewUserMessage(query))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(events)
			if text != "" {
				fmt.Println(text)
			}
		} else {
			if input == "" {
				continue
			}
			events, err := r.Run(context.Background(), "user", sessionID, model.NewUserMessage(input))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				continue
			}
			text := consumeEvents(events)
			if text != "" {
				fmt.Println(text)
			}
		}
		fmt.Println()
	}
}
