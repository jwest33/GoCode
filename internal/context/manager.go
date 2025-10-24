package context

import (
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/llm"
	"github.com/jake/gocode/internal/prompts"
)

// BudgetConfig defines context window budget allocation
type BudgetConfig struct {
	MaxTokens         int     // Total context window size
	SystemTokens      int     // Reserved for system prompt
	UserInputTokens   int     // Reserved for latest user input
	ContextTokens     int     // Reserved for retrieved context
	HistoryTokens     int     // Reserved for conversation history
	ResponseTokens    int     // Reserved for model response
	PruneThreshold    float64 // Prune when this % of budget is used (0.8 = 80%)
}

// DefaultBudgetConfig returns sensible defaults for 100K context window
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		MaxTokens:       102400,  // 100K context
		SystemTokens:    2000,    // System prompt
		UserInputTokens: 4000,    // Latest user message
		ContextTokens:   30000,   // Retrieved context (largest portion)
		HistoryTokens:   60000,   // Conversation history
		ResponseTokens:  4096,    // Model response
		PruneThreshold:  0.8,     // Prune at 80% capacity
	}
}

// Manager handles context window budget and message pruning
type Manager struct {
	config    BudgetConfig
	messages  []llm.Message
	promptMgr *prompts.PromptManager
}

// NewManager creates a new context manager
func NewManager(config BudgetConfig) *Manager {
	// Try to create prompt manager, but don't fail if it errors
	// (for backward compatibility with code that doesn't need templates)
	promptMgr, _ := prompts.NewPromptManager()

	return &Manager{
		config:    config,
		messages:  []llm.Message{},
		promptMgr: promptMgr,
	}
}

// AddMessage adds a message to the context
func (m *Manager) AddMessage(msg llm.Message) {
	m.messages = append(m.messages, msg)
}

// SetMessages replaces all messages
func (m *Manager) SetMessages(messages []llm.Message) {
	m.messages = messages
}

// GetMessages returns current messages
func (m *Manager) GetMessages() []llm.Message {
	return m.messages
}

// EstimateTokens estimates token count for text (rough approximation)
// Real implementation would use tiktoken or similar
func (m *Manager) EstimateTokens(text string) int {
	// Rough estimate: 1 token ≈ 4 characters for English text
	// For code, it's closer to 1 token ≈ 3 characters
	// We'll use 3.5 as a middle ground
	return len(text) * 10 / 35
}

// CalculateCurrentUsage calculates current token usage
func (m *Manager) CalculateCurrentUsage() TokenUsage {
	usage := TokenUsage{}

	for _, msg := range m.messages {
		tokens := m.EstimateTokens(msg.Content)

		switch msg.Role {
		case "system":
			usage.System += tokens
		case "user":
			usage.User += tokens
		case "assistant":
			usage.Assistant += tokens
		case "tool":
			usage.Context += tokens
		}
	}

	usage.Total = usage.System + usage.User + usage.Assistant + usage.Context

	return usage
}

// TokenUsage tracks token usage by role
type TokenUsage struct {
	System    int
	User      int
	Assistant int
	Context   int
	Total     int
}

// NeedsPruning checks if context needs pruning
func (m *Manager) NeedsPruning() bool {
	usage := m.CalculateCurrentUsage()
	threshold := int(float64(m.config.MaxTokens) * m.config.PruneThreshold)
	return usage.Total > threshold
}

// PruneMessages removes less important messages to fit budget
func (m *Manager) PruneMessages() []llm.Message {
	if !m.NeedsPruning() {
		return m.messages
	}

	pruned := []llm.Message{}

	// Always keep system message (should be first)
	if len(m.messages) > 0 && m.messages[0].Role == "system" {
		pruned = append(pruned, m.messages[0])
	}

	// Keep the most recent messages (sliding window)
	// Start from the end and add messages until we hit the budget
	budget := m.config.HistoryTokens
	consumed := 0

	for i := len(m.messages) - 1; i >= 0; i-- {
		msg := m.messages[i]

		// Skip system message (already added)
		if msg.Role == "system" {
			continue
		}

		msgTokens := m.EstimateTokens(msg.Content)

		// Check if adding this message would exceed budget
		if consumed+msgTokens > budget {
			// Try to add a summary of older messages instead
			break
		}

		pruned = append([]llm.Message{msg}, pruned...)
		consumed += msgTokens
	}

	m.messages = pruned
	return pruned
}

// SummarizeMessages creates a summary of old messages
// Simple implementation - can be enhanced with LLM-based summarization
func (m *Manager) SummarizeMessages(messages []llm.Message) string {
	var summary strings.Builder

	summary.WriteString("[Previous conversation summary]\n")

	userMessages := 0
	assistantMessages := 0
	toolCalls := 0

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		case "tool":
			toolCalls++
		}
	}

	summary.WriteString(fmt.Sprintf("- %d user messages\n", userMessages))
	summary.WriteString(fmt.Sprintf("- %d assistant responses\n", assistantMessages))
	summary.WriteString(fmt.Sprintf("- %d tool executions\n", toolCalls))

	return summary.String()
}

// PrepareMessagesForLLM prepares messages with context injection and pruning
func (m *Manager) PrepareMessagesForLLM(retrievedContext []string) []llm.Message {
	// Prune if necessary
	m.PruneMessages()

	// If we have retrieved context, inject it before the last user message
	if len(retrievedContext) > 0 {
		contextMsg := m.buildContextMessage(retrievedContext)

		// Insert context before the last user message
		if len(m.messages) > 0 {
			// Find last user message
			lastUserIdx := -1
			for i := len(m.messages) - 1; i >= 0; i-- {
				if m.messages[i].Role == "user" {
					lastUserIdx = i
					break
				}
			}

			if lastUserIdx != -1 {
				// Insert context message before last user message
				newMessages := make([]llm.Message, 0, len(m.messages)+1)
				newMessages = append(newMessages, m.messages[:lastUserIdx]...)
				newMessages = append(newMessages, contextMsg)
				newMessages = append(newMessages, m.messages[lastUserIdx:]...)
				return newMessages
			}
		}
	}

	return m.messages
}

// buildContextMessage creates a message with retrieved context
func (m *Manager) buildContextMessage(contexts []string) llm.Message {
	var content string

	// Try to use template if prompt manager is available
	if m.promptMgr != nil {
		rendered, err := m.promptMgr.RenderContextInjection(contexts, "")
		if err == nil {
			content = rendered
		} else {
			// Fallback to simple formatting
			content = m.buildContextMessageSimple(contexts)
		}
	} else {
		content = m.buildContextMessageSimple(contexts)
	}

	return llm.Message{
		Role:    "user",
		Content: content,
	}
}

// buildContextMessageSimple creates a simple context message without templates
func (m *Manager) buildContextMessageSimple(contexts []string) string {
	var content strings.Builder

	content.WriteString("# Retrieved Context\n\n")
	content.WriteString("The following code snippets may be relevant to your query:\n\n")

	for i, ctx := range contexts {
		content.WriteString(fmt.Sprintf("## Context %d\n", i+1))
		content.WriteString("```\n")
		content.WriteString(ctx)
		content.WriteString("\n```\n\n")
	}

	return content.String()
}

// GetAvailableContextBudget returns how many tokens we can use for retrieved context
func (m *Manager) GetAvailableContextBudget() int {
	usage := m.CalculateCurrentUsage()
	used := usage.System + usage.User + usage.Assistant

	// Calculate how much room we have left
	available := m.config.MaxTokens - used - m.config.ResponseTokens

	// Cap at configured context budget
	if available > m.config.ContextTokens {
		available = m.config.ContextTokens
	}

	if available < 0 {
		available = 0
	}

	return available
}

// FilterContextByBudget filters retrieved context to fit within budget
func (m *Manager) FilterContextByBudget(contexts []string) []string {
	budget := m.GetAvailableContextBudget()
	if budget <= 0 {
		return []string{}
	}

	filtered := []string{}
	consumed := 0

	for _, ctx := range contexts {
		tokens := m.EstimateTokens(ctx)

		if consumed+tokens > budget {
			break
		}

		filtered = append(filtered, ctx)
		consumed += tokens
	}

	return filtered
}
