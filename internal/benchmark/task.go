package benchmark

import (
	"encoding/json"
	"fmt"
	"time"
)

// Task represents a single SWE-bench task instance
type Task struct {
	// Core identification
	InstanceID string `json:"instance_id"` // Unique identifier (e.g., "django__django-12345")
	Repo       string `json:"repo"`        // Repository name (e.g., "django/django")

	// Git information
	BaseCommit    string `json:"base_commit"`    // Commit hash to checkout
	PatchCommit   string `json:"patch"`          // Optional: reference patch
	TestPatch     string `json:"test_patch"`     // Test suite additions

	// Problem description
	ProblemStatement string   `json:"problem_statement"` // Issue description
	Hints            string   `json:"hints_text"`        // Optional hints
	CreatedAt        string   `json:"created_at"`        // Issue creation date
	Version          string   `json:"version"`           // Python/framework version

	// Test information (stored as JSON-encoded strings in dataset)
	FailToPassRaw string `json:"FAIL_TO_PASS"` // JSON array of tests that should pass after fix
	PassToPassRaw string `json:"PASS_TO_PASS"` // JSON array of tests that should remain passing

	// Environment
	EnvironmentCommit string `json:"environment_setup_commit"` // Setup commit hash
}

// GetFailToPass parses and returns the FAIL_TO_PASS tests as a string slice
func (t *Task) GetFailToPass() ([]string, error) {
	var tests []string
	if err := json.Unmarshal([]byte(t.FailToPassRaw), &tests); err != nil {
		return nil, fmt.Errorf("failed to parse FAIL_TO_PASS: %w", err)
	}
	return tests, nil
}

// GetPassToPass parses and returns the PASS_TO_PASS tests as a string slice
func (t *Task) GetPassToPass() ([]string, error) {
	var tests []string
	if err := json.Unmarshal([]byte(t.PassToPassRaw), &tests); err != nil {
		return nil, fmt.Errorf("failed to parse PASS_TO_PASS: %w", err)
	}
	return tests, nil
}

// TaskResult represents the outcome of running the agent on a task
type TaskResult struct {
	InstanceID    string        `json:"instance_id"`
	Success       bool          `json:"success"`
	Patch         string        `json:"patch"`          // Generated patch
	ExecutionTime time.Duration `json:"execution_time"` // Time taken
	TokensUsed    int           `json:"tokens_used"`    // LLM tokens consumed
	Error         string        `json:"error"`          // Error message if failed
	AgentLog      string        `json:"agent_log"`      // Full agent interaction log
	ToolCalls     int           `json:"tool_calls"`     // Number of tool invocations
}

// EvaluationResult represents the outcome from the evaluation harness
type EvaluationResult struct {
	InstanceID      string  `json:"instance_id"`
	Resolved        bool    `json:"resolved"`         // Whether the issue was resolved
	TestsRun        int     `json:"tests_run"`        // Number of tests executed
	TestsPassed     int     `json:"tests_passed"`     // Number of tests passed
	TestsFailed     int     `json:"tests_failed"`     // Number of tests failed
	ResolveRate     float64 `json:"resolve_rate"`     // Percentage resolved (0-100)
	EvaluationError string  `json:"evaluation_error"` // Error from harness
}

// BenchmarkReport aggregates results across all tasks
type BenchmarkReport struct {
	RunID         string             `json:"run_id"`
	Timestamp     time.Time          `json:"timestamp"`
	TotalTasks    int                `json:"total_tasks"`
	Completed     int                `json:"completed"`
	Resolved      int                `json:"resolved"`
	Failed        int                `json:"failed"`
	ResolveRate   float64            `json:"resolve_rate"` // Percentage (0-100)
	AvgTime       time.Duration      `json:"avg_execution_time"`
	AvgTokens     int                `json:"avg_tokens"`
	TaskResults   []TaskResult       `json:"task_results"`
	EvalResults   []EvaluationResult `json:"evaluation_results"`
	Leaderboard   LeaderboardComparison `json:"leaderboard_comparison"`
}

// LeaderboardComparison shows how this run compares to known baselines
type LeaderboardComparison struct {
	GoCodeScore   float64            `json:"gocode_score"`
	Baselines     []BaselineScore    `json:"baselines"`
	Rank          int                `json:"estimated_rank"`
}

// BaselineScore represents a known model's performance
type BaselineScore struct {
	Model       string  `json:"model"`
	Score       float64 `json:"score"` // Percentage (0-100)
	Date        string  `json:"date"`
	Source      string  `json:"source"` // e.g., "Official SWE-bench leaderboard"
}

// GetBaselines returns known SWE-bench Verified baselines as of October 2025
func GetBaselines() []BaselineScore {
	return []BaselineScore{
		{
			Model:  "Claude Sonnet 4.5",
			Score:  77.2,
			Date:   "2025-10",
			Source: "Official SWE-bench leaderboard",
		},
		{
			Model:  "GPT-5",
			Score:  74.9,
			Date:   "2025-10",
			Source: "Official SWE-bench leaderboard",
		},
		{
			Model:  "Claude Opus 4.1",
			Score:  74.5,
			Date:   "2025-10",
			Source: "Official SWE-bench leaderboard",
		},
		{
			Model:  "o3",
			Score:  69.1,
			Date:   "2025-10",
			Source: "Official SWE-bench leaderboard",
		},
		{
			Model:  "Gemini 2.5 Pro",
			Score:  63.8,
			Date:   "2025-10",
			Source: "Official SWE-bench leaderboard",
		},
	}
}
