package benchmark

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/jake/gocode/internal/agent"
	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/initialization"
)

// Runner executes the agent against benchmark tasks
type Runner struct {
	dataDir      string
	runID        string
	timeout      time.Duration
	tokenBudget  int
	workspaceDir string
}

// NewRunner creates a new benchmark runner
func NewRunner(dataDir, runID string, timeoutSeconds, tokenBudget int) (*Runner, error) {
	workspaceDir := filepath.Join(dataDir, "workspaces", runID)
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return &Runner{
		dataDir:      dataDir,
		runID:        runID,
		timeout:      time.Duration(timeoutSeconds) * time.Second,
		tokenBudget:  tokenBudget,
		workspaceDir: workspaceDir,
	}, nil
}

// RunTask executes the agent on a single task
func (r *Runner) RunTask(task Task) (*TaskResult, error) {
	startTime := time.Now()

	result := &TaskResult{
		InstanceID: task.InstanceID,
		Success:    false,
	}

	// Create task-specific workspace
	taskWorkspace := filepath.Join(r.workspaceDir, sanitizeInstanceID(task.InstanceID))
	if err := os.MkdirAll(taskWorkspace, 0755); err != nil {
		result.Error = fmt.Sprintf("failed to create task workspace: %v", err)
		return result, nil
	}

	// Clone repository at base commit
	repoDir := filepath.Join(taskWorkspace, "repo")
	if err := r.cloneRepo(task.Repo, task.BaseCommit, repoDir); err != nil {
		result.Error = fmt.Sprintf("failed to clone repository: %v", err)
		return result, nil
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
	defer cancel()

	// Run agent
	patch, agentLog, tokensUsed, toolCalls, err := r.runAgent(ctx, task, repoDir)
	if err != nil {
		result.Error = err.Error()
		result.AgentLog = agentLog
		result.TokensUsed = tokensUsed
		result.ToolCalls = toolCalls
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	// Save results
	result.Success = true
	result.Patch = patch
	result.AgentLog = agentLog
	result.TokensUsed = tokensUsed
	result.ToolCalls = toolCalls
	result.ExecutionTime = time.Since(startTime)

	// Write patch to predictions directory
	if err := r.savePatch(task.InstanceID, patch); err != nil {
		result.Error = fmt.Sprintf("failed to save patch: %v", err)
		result.Success = false
	}

	return result, nil
}

// cloneRepo clones a repository at a specific commit
func (r *Runner) cloneRepo(repo, commit, targetDir string) error {
	// Extract owner/repo from format like "django/django"
	repoURL := fmt.Sprintf("https://github.com/%s.git", repo)

	// Clone with shallow history
	cloneCmd := exec.Command("git", "clone", "--depth=1", "--no-single-branch", repoURL, targetDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git clone failed: %w\n%s", err, output)
	}

	// Fetch the specific commit (in case it's not in shallow clone)
	fetchCmd := exec.Command("git", "fetch", "origin", commit)
	fetchCmd.Dir = targetDir
	_ = fetchCmd.Run() // Best effort, might already have it

	// Checkout the base commit
	checkoutCmd := exec.Command("git", "checkout", commit)
	checkoutCmd.Dir = targetDir
	if output, err := checkoutCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout failed: %w\n%s", err, output)
	}

	return nil
}

// runAgent executes the agent in the repository
func (r *Runner) runAgent(ctx context.Context, task Task, repoDir string) (string, string, int, int, error) {
	// 1. Create benchmark-optimized config
	cfg := r.createBenchmarkConfig(repoDir)

	// 2. Run project initialization/analysis
	detector, err := initialization.NewDetector(repoDir)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("failed to create detector: %w", err)
	}

	analyzer := initialization.NewAnalyzer(repoDir, detector)
	analysis, err := analyzer.Analyze()
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("project analysis failed: %w", err)
	}

	// 3. Create agent
	ag, err := agent.New(cfg, analysis)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("failed to create agent: %w", err)
	}

	// 4. Build prompt from task
	prompt := r.buildPrompt(task)

	// 5. Run agent autonomously
	options := &agent.AutonomousOptions{
		MaxIterations: 50,
		Timeout:       r.timeout,
		StopOnError:   false,
		MinIterations: 3,
	}

	result, err := ag.RunAutonomous(prompt, options)
	if err != nil {
		return "", "", 0, 0, fmt.Errorf("agent execution failed: %w", err)
	}

	// 6. Extract git patch
	patchGen := NewPatchGenerator(repoDir)
	patch, err := patchGen.ExtractPatch()
	if err != nil {
		// No changes made - not necessarily an error for benchmark
		patch = ""
	}

	// 7. Format agent log from messages
	agentLog := r.formatAgentLog(result)

	return patch, agentLog, result.TokensUsed, result.ToolCallCount, result.Error
}

// createBenchmarkConfig creates a config optimized for benchmarking
func (r *Runner) createBenchmarkConfig(workingDir string) *config.Config {
	// Create a basic config structure
	// In a real implementation, you would either:
	// 1. Load the user's existing config.yaml and modify it
	// 2. Create a template config and save it to the repo dir
	// 3. Use the global config with benchmark-specific overrides

	cfg := &config.Config{
		BaseDir:    workingDir, // Critical: set BaseDir for logger to work
		WorkingDir: workingDir,
		LLM: config.LLMConfig{
			ContextWindow: r.tokenBudget,
			Endpoint:      "http://localhost:8080/v1", // Default local endpoint
			Model:         "llama-3",
			Temperature:   0.7,
			MaxTokens:     2048,
		},
		Logging: config.LoggingConfig{
			Directory:      "logs",
			Format:         "jsonl",
			Level:          "info",
			LogToolResults: true,
		},
		Tools: config.ToolsConfig{
			Enabled: []string{
				"read", "write", "edit", "bash", "glob", "grep",
				"bash_output", "kill_shell", "todo_write",
			},
		},
		Confirmation: config.ConfirmationConfig{
			Mode: "never", // Auto-approve all tool executions for benchmark
		},
		Checkpoint: config.CheckpointConfig{
			Enabled: false, // Disable for benchmarking
		},
		Memory: config.MemoryConfig{
			Enabled: false, // Disable for benchmarking
		},
		Telemetry: config.TelemetryConfig{
			Enabled: false, // Disable for benchmarking
		},
		Retrieval: config.RetrievalConfig{
			Enabled: false, // Disable for benchmarking
		},
		LSP: config.LSPConfig{
			Enabled: false, // Disable for benchmarking
		},
	}

	return cfg
}

// buildPrompt constructs the prompt for the agent
func (r *Runner) buildPrompt(task Task) string {
	var sb strings.Builder

	sb.WriteString("You are a software engineer tasked with fixing a bug in this repository.\n\n")
	sb.WriteString("## Problem Statement\n\n")
	sb.WriteString(task.ProblemStatement)
	sb.WriteString("\n\n")

	if task.Hints != "" {
		sb.WriteString("## Hints\n\n")
		sb.WriteString(task.Hints)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## Instructions\n\n")
	sb.WriteString("1. Analyze the problem and explore the repository\n")
	sb.WriteString("2. Identify the root cause of the issue\n")
	sb.WriteString("3. Implement a fix that resolves the problem\n")
	sb.WriteString("4. Ensure your fix doesn't break existing functionality\n\n")

	sb.WriteString("When you're done, the repository should contain your fix committed to git.\n")
	sb.WriteString("The evaluation will run tests to verify your solution.\n")

	return sb.String()
}

// savePatch writes the patch to the predictions directory
func (r *Runner) savePatch(instanceID, patch string) error {
	predictionsDir := filepath.Join(r.dataDir, "predictions", r.runID)
	if err := os.MkdirAll(predictionsDir, 0755); err != nil {
		return err
	}

	patchFile := filepath.Join(predictionsDir, sanitizeInstanceID(instanceID)+".patch")
	return os.WriteFile(patchFile, []byte(patch), 0644)
}

// sanitizeInstanceID converts instance ID to valid filename
func sanitizeInstanceID(instanceID string) string {
	// Replace problematic characters
	s := strings.ReplaceAll(instanceID, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ":", "_")
	return s
}

// SaveResults writes task results to disk
func (r *Runner) SaveResults(results []TaskResult) error {
	resultsDir := filepath.Join(r.dataDir, "results")
	if err := os.MkdirAll(resultsDir, 0755); err != nil {
		return err
	}

	// Save as JSON
	resultsFile := filepath.Join(resultsDir, r.runID+"_results.json")

	// Marshal with indentation for readability
	data, err := encodeJSON(results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	return os.WriteFile(resultsFile, data, 0644)
}

// formatAgentLog converts agent result messages into a readable log
func (r *Runner) formatAgentLog(result *agent.AutonomousResult) string {
	var sb strings.Builder

	sb.WriteString("=== Agent Execution Log ===\n")
	sb.WriteString(fmt.Sprintf("Success: %v\n", result.Success))
	sb.WriteString(fmt.Sprintf("Completion Reason: %s\n", result.CompletionReason))
	sb.WriteString(fmt.Sprintf("Iterations: %d\n", result.Iterations))
	sb.WriteString(fmt.Sprintf("Tool Calls: %d\n", result.ToolCallCount))
	sb.WriteString(fmt.Sprintf("Tokens Used: %d\n", result.TokensUsed))
	sb.WriteString(fmt.Sprintf("Execution Time: %s\n\n", result.ExecutionTime))

	sb.WriteString("=== Conversation Messages ===\n\n")
	for i, msg := range result.Messages {
		sb.WriteString(fmt.Sprintf("[%d] Role: %s\n", i+1, msg.Role))
		if msg.Content != "" {
			// Truncate very long messages
			content := msg.Content
			if len(content) > 500 {
				content = content[:500] + "... (truncated)"
			}
			sb.WriteString(fmt.Sprintf("Content: %s\n", content))
		}
		if msg.ToolID != "" {
			sb.WriteString(fmt.Sprintf("Tool ID: %s\n", msg.ToolID))
		}
		sb.WriteString("\n")
	}

	if result.FinalMessage != "" {
		sb.WriteString("=== Final Message ===\n")
		sb.WriteString(result.FinalMessage)
		sb.WriteString("\n")
	}

	if result.Error != nil {
		sb.WriteString("\n=== Error ===\n")
		sb.WriteString(result.Error.Error())
		sb.WriteString("\n")
	}

	return sb.String()
}
