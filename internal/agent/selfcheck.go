package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/llm"
)

// SelfCheckSystem validates agent's claims before task completion
type SelfCheckSystem struct {
	bashTool ToolExecutor
}

// ToolExecutor interface for executing bash commands
type ToolExecutor interface {
	Execute(ctx context.Context, toolName string, args string) (string, error)
}

// NewSelfCheckSystem creates a new self-check system
func NewSelfCheckSystem(bashTool ToolExecutor) *SelfCheckSystem {
	return &SelfCheckSystem{
		bashTool: bashTool,
	}
}

// CompletionClaim represents a claim made by the agent
type CompletionClaim struct {
	ClaimType string // "tests_passed", "build_success", "task_complete"
	Content   string // The actual claim text
	Verified  bool
	Evidence  string // Evidence from actual execution
}

// DetectCompletionClaims analyzes agent response for completion claims
func (s *SelfCheckSystem) DetectCompletionClaims(response string) []CompletionClaim {
	var claims []CompletionClaim

	lowerResponse := strings.ToLower(response)

	// Pattern 1: Claims about tests passing
	testPassPatterns := []string{
		"all tests pass",
		"tests pass",
		"tests are passing",
		"tests passed",
		"all tests passed",
		"test suite passed",
		"✓ all tests passed",
	}

	for _, pattern := range testPassPatterns {
		if strings.Contains(lowerResponse, pattern) {
			claims = append(claims, CompletionClaim{
				ClaimType: "tests_passed",
				Content:   pattern,
				Verified:  false,
			})
			break // Only add one test claim
		}
	}

	// Pattern 2: Claims about build success
	buildPassPatterns := []string{
		"build successful",
		"build passed",
		"compilation successful",
		"build complete",
	}

	for _, pattern := range buildPassPatterns {
		if strings.Contains(lowerResponse, pattern) {
			claims = append(claims, CompletionClaim{
				ClaimType: "build_success",
				Content:   pattern,
				Verified:  false,
			})
			break
		}
	}

	// Pattern 3: Verification claims with test output quoted
	if strings.Contains(lowerResponse, "as verified by") ||
		strings.Contains(lowerResponse, "verified by the test output") {
		claims = append(claims, CompletionClaim{
			ClaimType: "verification_claim",
			Content:   "claims verification",
			Verified:  false,
		})
	}

	return claims
}

// VerifyClaims attempts to verify agent's claims by running actual tests
func (s *SelfCheckSystem) VerifyClaims(ctx context.Context, claims []CompletionClaim, projectContext string) ([]CompletionClaim, error) {
	if len(claims) == 0 {
		return claims, nil
	}

	var verified []CompletionClaim

	for _, claim := range claims {
		switch claim.ClaimType {
		case "tests_passed", "verification_claim":
			// Try to detect and run test command
			testCmd := s.detectTestCommand(projectContext)
			if testCmd != "" {
				// Execute test command
				result, err := s.bashTool.Execute(ctx, "bash", fmt.Sprintf(`{"command":"%s"}`, testCmd))

				claim.Evidence = result
				if err != nil {
					claim.Evidence += fmt.Sprintf("\nError: %v", err)
				}

				// Check if tests actually passed
				claim.Verified = s.checkTestSuccess(result, err)
			}
		case "build_success":
			// Try to detect and run build command
			buildCmd := s.detectBuildCommand(projectContext)
			if buildCmd != "" {
				result, err := s.bashTool.Execute(ctx, "bash", fmt.Sprintf(`{"command":"%s"}`, buildCmd))

				claim.Evidence = result
				if err != nil {
					claim.Evidence += fmt.Sprintf("\nError: %v", err)
				}

				claim.Verified = s.checkBuildSuccess(result, err)
			}
		}

		verified = append(verified, claim)
	}

	return verified, nil
}

// detectTestCommand tries to determine the appropriate test command
func (s *SelfCheckSystem) detectTestCommand(projectContext string) string {
	lowerContext := strings.ToLower(projectContext)

	// Python project
	if strings.Contains(lowerContext, "python") {
		if strings.Contains(lowerContext, "pytest") {
			return "pytest"
		}
		// Look for tests.py file
		if strings.Contains(lowerContext, "tests.py") {
			return "python tests.py"
		}
		return "python -m pytest"
	}

	// Node.js project
	if strings.Contains(lowerContext, "javascript") || strings.Contains(lowerContext, "typescript") ||
		strings.Contains(lowerContext, "node") {
		return "npm test"
	}

	// Go project
	if strings.Contains(lowerContext, "go") {
		return "go test ./..."
	}

	// Rust project
	if strings.Contains(lowerContext, "rust") {
		return "cargo test"
	}

	return ""
}

// detectBuildCommand tries to determine the appropriate build command
func (s *SelfCheckSystem) detectBuildCommand(projectContext string) string {
	lowerContext := strings.ToLower(projectContext)

	if strings.Contains(lowerContext, "go") {
		return "go build ./..."
	}

	if strings.Contains(lowerContext, "rust") {
		return "cargo build"
	}

	if strings.Contains(lowerContext, "node") || strings.Contains(lowerContext, "typescript") {
		return "npm run build"
	}

	return ""
}

// checkTestSuccess analyzes test output to determine if tests passed
func (s *SelfCheckSystem) checkTestSuccess(output string, err error) bool {
	if err != nil {
		return false
	}

	lowerOutput := strings.ToLower(output)

	// Success indicators
	successPatterns := []string{
		"all tests passed",
		"✓ all tests passed",
		"ok",
		"passed",
		"test result: ok",
	}

	// Failure indicators (these override success)
	failurePatterns := []string{
		"failed",
		"error",
		"assertion",
		"traceback",
		"panic",
		"fatal",
	}

	// Check for failures first
	for _, pattern := range failurePatterns {
		if strings.Contains(lowerOutput, pattern) {
			return false
		}
	}

	// Then check for success
	for _, pattern := range successPatterns {
		if strings.Contains(lowerOutput, pattern) {
			return true
		}
	}

	// If no clear indicators, assume failure
	return false
}

// checkBuildSuccess analyzes build output to determine if build succeeded
func (s *SelfCheckSystem) checkBuildSuccess(output string, err error) bool {
	if err != nil {
		return false
	}

	lowerOutput := strings.ToLower(output)

	failurePatterns := []string{
		"error",
		"failed",
		"fatal",
		"panic",
	}

	for _, pattern := range failurePatterns {
		if strings.Contains(lowerOutput, pattern) {
			return false
		}
	}

	return true
}

// GenerateFeedbackMessage creates a message to inject back into conversation
func (s *SelfCheckSystem) GenerateFeedbackMessage(claims []CompletionClaim) string {
	if len(claims) == 0 {
		return ""
	}

	var parts []string
	parts = append(parts, "**🔍 Self-Check Results:**\n")

	hasFailures := false

	for _, claim := range claims {
		if claim.Verified {
			parts = append(parts, fmt.Sprintf("✅ %s: **VERIFIED**", claim.ClaimType))
		} else {
			hasFailures = true
			parts = append(parts, fmt.Sprintf("❌ %s: **NOT VERIFIED**", claim.ClaimType))
			if claim.Evidence != "" {
				parts = append(parts, "\nActual output:")
				parts = append(parts, "```")
				parts = append(parts, s.truncateOutput(claim.Evidence, 500))
				parts = append(parts, "```")
			}
		}
		parts = append(parts, "")
	}

	if hasFailures {
		parts = append(parts, "**⚠️ WARNING: Your claims could not be verified. Please review the actual output above.**")
	}

	return strings.Join(parts, "\n")
}

// truncateOutput limits output length for readability
func (s *SelfCheckSystem) truncateOutput(output string, maxLen int) string {
	if len(output) <= maxLen {
		return output
	}
	return output[:maxLen] + "\n... (output truncated)"
}

// ShouldTriggerCheck determines if self-check should run based on message content
func (s *SelfCheckSystem) ShouldTriggerCheck(message llm.Message) bool {
	if message.Role != "assistant" {
		return false
	}

	// Check if this looks like a final response (no tool calls)
	// and contains completion claims
	claims := s.DetectCompletionClaims(message.Content)
	return len(claims) > 0
}
