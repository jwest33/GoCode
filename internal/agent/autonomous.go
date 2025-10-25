package agent

import (
	"time"

	"github.com/jake/gocode/internal/llm"
)

// AutonomousOptions configures autonomous execution behavior
type AutonomousOptions struct {
	// MaxIterations limits the number of LLM calls (default: 50)
	MaxIterations int

	// Timeout sets the maximum duration for task execution
	Timeout time.Duration

	// StopOnError determines whether to stop on first tool error
	StopOnError bool

	// CompletionKeywords are phrases that indicate task completion
	CompletionKeywords []string

	// MinIterations prevents premature completion (default: 3)
	MinIterations int
}

// DefaultAutonomousOptions returns sensible defaults
func DefaultAutonomousOptions() *AutonomousOptions {
	return &AutonomousOptions{
		MaxIterations: 50,
		Timeout:       30 * time.Minute,
		StopOnError:   false,
		CompletionKeywords: []string{
			"task complete",
			"task is complete",
			"i've completed",
			"i have completed",
			"finished the task",
			"all done",
			"implementation is complete",
		},
		MinIterations: 3,
	}
}

// AutonomousResult contains the results of autonomous execution
type AutonomousResult struct {
	// Success indicates whether the task completed successfully
	Success bool

	// FinalMessage is the last message from the assistant
	FinalMessage string

	// TokensUsed is the total number of tokens consumed
	TokensUsed int

	// ToolCallCount is the total number of tool executions
	ToolCallCount int

	// Iterations is the number of LLM calls made
	Iterations int

	// CompletionReason explains why execution stopped
	CompletionReason string

	// Error contains any error that occurred
	Error error

	// Messages contains the full conversation history
	Messages []llm.Message

	// ExecutionTime is the total time taken
	ExecutionTime time.Duration
}

// CompletionReason constants
const (
	CompletionReasonSuccess        = "task_complete"
	CompletionReasonNaturalStop    = "natural_stop"
	CompletionReasonMaxIterations  = "max_iterations"
	CompletionReasonTimeout        = "timeout"
	CompletionReasonErrorThreshold = "error_threshold"
	CompletionReasonError          = "error"
)
