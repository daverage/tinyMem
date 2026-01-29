package ralph

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/daverage/tinymem/internal/config"
	"github.com/daverage/tinymem/internal/evidence"
	"github.com/daverage/tinymem/internal/llm"
	"github.com/daverage/tinymem/internal/memory"
	"go.uber.org/zap"
)

// Engine orchestrates the Ralph Wiggum loop
type Engine struct {
	cfg       *config.Config
	mem       *memory.Service
	llmClient *llm.Client
	projectID string
	logger    *zap.Logger
}

// NewEngine creates a new Ralph engine
func NewEngine(cfg *config.Config, mem *memory.Service, projectID string, logger *zap.Logger) *Engine {
	return &Engine{
		cfg:       cfg,
		mem:       mem,
		llmClient: llm.NewClient(cfg),
		projectID: projectID,
		logger:    logger,
	}
}

// ExecuteLoop runs the autonomous repair loop
func (e *Engine) ExecuteLoop(ctx context.Context, opt Options) (*Result, error) {
	result := &Result{
		Status:   StatusAborted,
		Evidence: make(map[string]interface{}),
		Log:      []LogEntry{},
	}

	for i := 1; i <= opt.MaxIterations; i++ {
		e.logger.Info("Ralph loop iteration started", zap.Int("iteration", i), zap.String("task", opt.Task))
		result.Iterations = i
		startTime := time.Now()

		// 1. EXECUTE Phase
		state, err := e.execute(ctx, opt, i)
		duration := time.Since(startTime).Seconds() * 1000

		entry := LogEntry{
			Iteration: i,
			Duration:  duration,
		}

		if err != nil {
			e.logger.Error("Ralph loop execution failed", zap.Int("iteration", i), zap.Error(err))
			entry.Result = "error"
			entry.Error = err.Error()
			result.Log = append(result.Log, entry)
			return result, err
		}

		// 2. EVIDENCE Phase
		passed, evidenceResults := e.checkEvidence(opt.Evidence)
		result.Evidence = evidenceResults

		if passed {
			e.logger.Info("Ralph loop success: all evidence passed", zap.Int("iteration", i))
			entry.Result = "pass"
			result.Log = append(result.Log, entry)
			result.Status = StatusSuccess
			result.FinalDiff = state.Diff
			return result, nil
		}

		e.logger.Warn("Ralph loop evidence check failed", zap.Int("iteration", i), zap.Any("evidence", evidenceResults))
		entry.Result = "fail"
		result.Log = append(result.Log, entry)

		// Check human gate
		if opt.HumanGate.AfterIterations > 0 && i >= opt.HumanGate.AfterIterations {
			e.logger.Info("Ralph loop human gate triggered", zap.Int("iteration", i))
			result.Status = StatusAborted
			return result, fmt.Errorf("human gate triggered after %d iterations", i)
		}

		// 3. RECALL Phase
		e.logger.Info("Ralph loop performing recall", zap.Int("iteration", i))
		memories, err := e.recall(ctx, opt, state)
		if err != nil {
			return result, fmt.Errorf("recall failed: %w", err)
		}
		for _, m := range memories {
			result.MemoryUsed = append(result.MemoryUsed, m.Summary)
		}

		// 4. REPAIR Phase
		if i < opt.MaxIterations {
			e.logger.Info("Ralph loop attempting repair", zap.Int("iteration", i))
			err = e.repair(ctx, opt, state, memories)
			if err != nil {
				e.logger.Error("Ralph loop repair failed", zap.Int("iteration", i), zap.Error(err))
				return result, fmt.Errorf("repair failed: %w", err)
			}
		} else {
			e.logger.Warn("Ralph loop reached max iterations without success")
			result.Status = StatusFailed
		}
	}

	return result, nil
}

func (e *Engine) execute(ctx context.Context, opt Options, i int) (*IterationState, error) {
	// Safety check for forbidden commands
	for _, forbidden := range opt.Safety.ForbidCommands {
		if strings.Contains(opt.Command, forbidden) {
			return nil, fmt.Errorf("safety violation: command contains forbidden string '%s'", forbidden)
		}
	}

	startTime := time.Now()

	command := strings.TrimSpace(opt.Command)
	if command == "" {
		return nil, fmt.Errorf("command is empty")
	}

	// Prepare the command with safety gating.
	// Allow shell usage only when explicitly permitted.
	containsShellMeta := strings.ContainsAny(command, "|&;><`$()[]{}") || strings.Contains(command, "\n") || strings.Contains(command, "\r")
	var cmd *exec.Cmd
	if containsShellMeta {
		if !opt.Safety.AllowShell {
			return nil, fmt.Errorf("command contains shell metacharacters but allow_shell is false")
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	} else {
		parts := strings.Fields(command)
		if len(parts) == 0 {
			return nil, fmt.Errorf("command is empty")
		}
		cmd = exec.CommandContext(ctx, parts[0], parts[1:]...)
	}
	cmd.Dir = e.cfg.ProjectRoot

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(startTime)

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to run command: %w", err)
		}
	}

	diff, _ := e.getGitDiff()

	return &IterationState{
		Iteration: i, // Correctly set from argument
		ExitCode:  exitCode,
		Stdout:    stdout.String(),
		Stderr:    stderr.String(),
		Duration:  duration,
		Diff:      diff,
	}, nil
}

func (e *Engine) getGitDiff() (string, error) {
	cmd := exec.Command("git", "diff")
	cmd.Dir = e.cfg.ProjectRoot
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (e *Engine) checkEvidence(predicates []string) (bool, map[string]interface{}) {
	results := make(map[string]interface{})
	allPass := true

	for _, p := range predicates {
		// Parse predicate (e.g., "test_pass::./...")
		parts := strings.SplitN(p, "::", 2)
		eType := parts[0]
		eContent := ""
		if len(parts) > 1 {
			eContent = parts[1]
		}

		passed, err := evidence.VerifyEvidence(eType, eContent, e.cfg)
		if err != nil {
			results[p] = fmt.Sprintf("error: %v", err)
			allPass = false
		} else {
			results[p] = passed
			if !passed {
				allPass = false
			}
		}
	}

	return allPass, results
}

func (e *Engine) recall(ctx context.Context, opt Options, state *IterationState) ([]*memory.Memory, error) {
	// Construct query from state and options
	query := strings.Join(opt.Recall.QueryTerms, " ")
	if query == "" {
		// Simple heuristic: use the last line of stderr as query if available
		lines := strings.Split(strings.TrimSpace(state.Stderr), "\n")
		if len(lines) > 0 {
			query = lines[len(lines)-1]
		}
	}

	// Use existing memory service to query
	return e.mem.SearchMemories(e.projectID, query, opt.Recall.Limit)
}

func (e *Engine) repair(ctx context.Context, opt Options, state *IterationState, memories []*memory.Memory) error {
	// 1. Construct the prompt
	var sb strings.Builder
	sb.WriteString("You are the repair engine for a 'Ralph Wiggum' autonomous loop.\n")
	sb.WriteString(fmt.Sprintf("Your task is: %s\n\n", opt.Task))
	sb.WriteString(fmt.Sprintf("The command '%s' failed with exit code %d.\n", opt.Command, state.ExitCode))

	if state.Stdout != "" {
		sb.WriteString("STDOUT:\n---\n" + state.Stdout + "\n---\n")
	}
	if state.Stderr != "" {
		sb.WriteString("STDERR:\n---\n" + state.Stderr + "\n---\n")
	}

	if len(memories) > 0 {
		sb.WriteString("\nRELEVANT PROJECT MEMORIES:\n")
		for _, m := range memories {
			sb.WriteString(fmt.Sprintf("- %s: %s\n", m.Type, m.Summary))
			if m.Detail != "" {
				sb.WriteString("  Detail: " + m.Detail + "\n")
			}
		}
	}

	if state.Diff != "" {
		sb.WriteString("\nCURRENT CHANGES (git diff):\n---\n" + state.Diff + "\n---\n")
	}

	sb.WriteString("\nPROPOSE A FIX. Return your response as a series of file updates in this EXACT format:\n")
	sb.WriteString("@@@ FILE: path/to/file @@@\n[FULL FILE CONTENT HERE]\n@@@ END_FILE @@@\n")
	sb.WriteString("\nOnly return the file content blocks. Do not include commentary.")

	// 2. Call LLM
	model := e.cfg.CoVeModel
	if model == "" {
		model = os.Getenv("TINYMEM_RALPH_MODEL")
	}
	if model == "" {
		model = "gpt-4o" // Default fallback
	}

	req := llm.ChatCompletionRequest{
		Model:    model,
		Messages: []llm.Message{
			{Role: "system", Content: "You are an expert software engineer fixing a codebase."},
			{Role: "user", Content: sb.String()},
		},
	}

	resp, err := e.llmClient.ChatCompletions(ctx, req)
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return fmt.Errorf("LLM returned no choices")
	}

	content := resp.Choices[0].Message.Content

	// 3. Parse and Apply changes
	return e.applyChanges(content, opt.Safety)
}

func (e *Engine) applyChanges(content string, safety SafetyOptions) error {
	filePattern := regexp.MustCompile(`(?s)@@@ FILE: (.*?) @@@\n(.*?)\n@@@ END_FILE @@@`)
	matches := filePattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return fmt.Errorf("no valid file changes found in LLM response")
	}

	for _, match := range matches {
		relPath := strings.TrimSpace(match[1])
		newContent := match[2]

		// Safety check for forbidden paths
		absPath := filepath.Join(e.cfg.ProjectRoot, relPath)

		for _, forbidden := range safety.ForbidPaths {
			if strings.Contains(relPath, forbidden) || strings.Contains(absPath, forbidden) {
				return fmt.Errorf("safety violation: attempt to modify forbidden path '%s'", relPath)
			}
		}

		// Ensure directory exists
		err := os.MkdirAll(filepath.Dir(absPath), 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory for %s: %w", relPath, err)
		}

		// Write file
		err = os.WriteFile(absPath, []byte(newContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write file %s: %w", relPath, err)
		}
		fmt.Printf("🔧 Applied repair to %s\n", relPath)
	}

	return nil
}
