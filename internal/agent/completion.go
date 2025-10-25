package agent

import (
	"strings"
	"time"

	"github.com/jake/gocode/internal/llm"
)

// CompletionDetector determines when an autonomous task is complete
type CompletionDetector struct {
	options           *AutonomousOptions
	startTime         time.Time
	currentIteration  int
	consecutiveErrors int
	totalToolCalls    int
}

// NewCompletionDetector creates a new completion detector
func NewCompletionDetector(options *AutonomousOptions) *CompletionDetector {
	return &CompletionDetector{
		options:           options,
		startTime:         time.Now(),
		currentIteration:  0,
		consecutiveErrors: 0,
		totalToolCalls:    0,
	}
}

// IncrementIteration increments the iteration counter
func (cd *CompletionDetector) IncrementIteration() {
	cd.currentIteration++
}

// RecordToolSuccess records a successful tool execution
func (cd *CompletionDetector) RecordToolSuccess() {
	cd.consecutiveErrors = 0
	cd.totalToolCalls++
}

// RecordToolError records a tool execution failure
func (cd *CompletionDetector) RecordToolError() {
	cd.consecutiveErrors++
	cd.totalToolCalls++
}

// GetToolCallCount returns the total number of tool calls
func (cd *CompletionDetector) GetToolCallCount() int {
	return cd.totalToolCalls
}

// IsComplete checks if the task should be considered complete
// Returns (isComplete, reason)
func (cd *CompletionDetector) IsComplete(resp *llm.CompletionResponse) (bool, string) {
	// Check timeout first
	if cd.isTimedOut() {
		return true, CompletionReasonTimeout
	}

	// Check max iterations
	if cd.hasReachedMaxIterations() {
		return true, CompletionReasonMaxIterations
	}

	// Check error threshold (5+ consecutive errors)
	if cd.hasExceededErrorThreshold() {
		return true, CompletionReasonErrorThreshold
	}

	// Must have minimum iterations before checking completion signals
	if cd.currentIteration < cd.options.MinIterations {
		return false, ""
	}

	// Check for explicit completion signal
	if cd.hasCompletionSignal(resp) {
		return true, CompletionReasonSuccess
	}

	// Check for natural stop (no tool calls + stop reason)
	if cd.isNaturalStop(resp) {
		return true, CompletionReasonNaturalStop
	}

	return false, ""
}

// isTimedOut checks if the timeout has been exceeded
func (cd *CompletionDetector) isTimedOut() bool {
	if cd.options.Timeout == 0 {
		return false
	}
	return time.Since(cd.startTime) > cd.options.Timeout
}

// hasReachedMaxIterations checks if the iteration limit is reached
func (cd *CompletionDetector) hasReachedMaxIterations() bool {
	if cd.options.MaxIterations == 0 {
		return false
	}
	return cd.currentIteration >= cd.options.MaxIterations
}

// hasExceededErrorThreshold checks if too many consecutive errors occurred
func (cd *CompletionDetector) hasExceededErrorThreshold() bool {
	return cd.consecutiveErrors >= 5
}

// hasCompletionSignal checks if the response contains explicit completion indicators
func (cd *CompletionDetector) hasCompletionSignal(resp *llm.CompletionResponse) bool {
	// Must have stop reason and no tool calls
	if resp.FinishReason != "stop" || len(resp.ToolCalls) > 0 {
		return false
	}

	// Check for completion keywords in content
	contentLower := strings.ToLower(resp.Content)

	for _, keyword := range cd.options.CompletionKeywords {
		if strings.Contains(contentLower, keyword) {
			return true
		}
	}

	return false
}

// isNaturalStop checks if the agent naturally stopped (no more tool calls)
func (cd *CompletionDetector) isNaturalStop(resp *llm.CompletionResponse) bool {
	// Agent stopped without tool calls
	return resp.FinishReason == "stop" && len(resp.ToolCalls) == 0
}

// GetElapsedTime returns the time elapsed since start
func (cd *CompletionDetector) GetElapsedTime() time.Duration {
	return time.Since(cd.startTime)
}

// GetCurrentIteration returns the current iteration count
func (cd *CompletionDetector) GetCurrentIteration() int {
	return cd.currentIteration
}
