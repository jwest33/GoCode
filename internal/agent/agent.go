package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/confirmation"
	ctxmgr "github.com/jake/gocode/internal/context"
	"github.com/jake/gocode/internal/initialization"
	"github.com/jake/gocode/internal/llm"
	"github.com/jake/gocode/internal/logging"
	"github.com/jake/gocode/internal/memory"
	"github.com/jake/gocode/internal/planning"
	"github.com/jake/gocode/internal/prompts"
	"github.com/jake/gocode/internal/theme"
	"github.com/jake/gocode/internal/tools"
)

type Agent struct {
	config        *config.Config
	llmClient     *llm.Client
	serverManager *llm.ServerManager
	toolRegistry  *tools.Registry
	confirmSys    *confirmation.System
	logger        *logging.Logger
	promptMgr     *prompts.PromptManager
	messages      []llm.Message
	rl            *readline.Instance
	historyFile   string
	ltm           *memory.LongTermMemory // Long-term memory for persistent planning
	contextMgr    *ctxmgr.Manager        // Context window manager
	planMgr       *planning.PlanManager  // Hierarchical plan manager
	tokenMonitor  *ctxmgr.TokenMonitor   // Async token usage monitor
}

func New(cfg *config.Config, projectAnalysis *initialization.ProjectAnalysis) (*Agent, error) {
	// Initialize logger
	logger, err := logging.New(&cfg.Logging, cfg.BaseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize and start llama-server if auto-management is enabled
	serverManager := llm.NewServerManager(&cfg.LLM)
	if err := serverManager.Start(); err != nil {
		logger.Close()
		return nil, fmt.Errorf("failed to start llama-server: %w", err)
	}

	// Initialize LLM client
	llmClient := llm.NewClient(&cfg.LLM)

	// Initialize tool registry
	registry := tools.NewRegistry()

	// Initialize long-term memory if enabled
	var ltm *memory.LongTermMemory
	var planMgr *planning.PlanManager
	if cfg.Memory.Enabled {
		memoryPath := filepath.Join(cfg.BaseDir, ".gocode", cfg.Memory.DBPath)
		ltm, err = memory.NewLongTermMemory(memoryPath)
		if err != nil {
			logger.Close()
			serverManager.Stop()
			return nil, fmt.Errorf("failed to initialize long-term memory: %w", err)
		}
	}

	// Initialize context manager
	contextConfig := ctxmgr.DefaultBudgetConfig()
	contextConfig.MaxTokens = cfg.LLM.ContextWindow
	contextMgr := ctxmgr.NewManager(contextConfig)

	// Initialize token monitor (checks every 5 seconds)
	tokenMonitor := ctxmgr.NewTokenMonitor(contextMgr, 5*time.Second)

	// Register enabled tools
	bashTool := tools.NewBashTool()
	todoPath := filepath.Join(cfg.WorkingDir, "TODO.md")
	todoTool := tools.NewTodoWriteTool(todoPath)

	// Initialize plan manager if memory is enabled
	if ltm != nil {
		planMgr = planning.NewPlanManager(ltm, todoTool)
	}

	for _, toolName := range cfg.Tools.Enabled {
		switch toolName {
		case "read":
			registry.Register(&tools.ReadTool{})
		case "write":
			registry.Register(&tools.WriteTool{})
		case "edit":
			registry.Register(&tools.EditTool{})
		case "glob":
			registry.Register(&tools.GlobTool{})
		case "grep":
			registry.Register(&tools.GrepTool{})
		case "bash":
			registry.Register(bashTool)
		case "bash_output":
			registry.Register(tools.NewBashOutputTool(bashTool))
		case "kill_shell":
			registry.Register(tools.NewKillShellTool(bashTool))
		case "todo_write":
			registry.Register(todoTool)
		case "web_fetch":
			registry.Register(tools.NewWebFetchTool())
		case "web_search":
			registry.Register(tools.NewWebSearchTool())
		case "store_memory":
			if ltm != nil {
				registry.Register(tools.NewStoreMemoryTool(ltm))
			}
		case "recall_memory":
			if ltm != nil {
				registry.Register(tools.NewRecallMemoryTool(ltm))
			}
		}
	}

	// Initialize confirmation system
	confirmSys := confirmation.New(&cfg.Confirmation)

	// Initialize prompt manager
	promptMgr, err := prompts.NewPromptManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create prompt manager: %w", err)
	}

	// Initialize readline
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          theme.GetPinkPrompt(),
		HistoryFile:     filepath.Join(cfg.BaseDir, ".gocode", "input_history"),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize readline: %w", err)
	}

	// Build tool info for system prompt
	toolInfos := buildToolInfos(registry)

	// Build project context if analysis is available
	var projectContext *prompts.ProjectContext
	if projectAnalysis != nil {
		projectContext = buildProjectContext(projectAnalysis)
	}

	// Render system message with project context
	systemPrompt, err := promptMgr.RenderSystemWithProject(cfg, toolInfos, projectContext)
	if err != nil {
		return nil, fmt.Errorf("failed to render system prompt: %w", err)
	}

	messages := []llm.Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}

	// Set up conversation history file
	historyFile := filepath.Join(cfg.BaseDir, ".gocode", "conversation_history")

	// Set messages in context manager
	contextMgr.SetMessages(messages)

	return &Agent{
		config:        cfg,
		llmClient:     llmClient,
		serverManager: serverManager,
		toolRegistry:  registry,
		confirmSys:    confirmSys,
		logger:        logger,
		promptMgr:     promptMgr,
		messages:      messages,
		rl:            rl,
		historyFile:   historyFile,
		ltm:           ltm,
		contextMgr:    contextMgr,
		planMgr:       planMgr,
		tokenMonitor:  tokenMonitor,
	}, nil
}

func (a *Agent) Run() error {
	defer a.serverManager.Stop()
	defer a.logger.Close()
	defer a.rl.Close()
	if a.ltm != nil {
		defer a.ltm.Close()
	}

	// Start token monitor
	a.tokenMonitor.Start()
	defer a.tokenMonitor.Stop()

	// Start goroutine to handle token warnings
	go a.handleTokenWarnings()

	fmt.Print(theme.SynthwaveBanner("v1.0"))

	for {
		line, err := a.rl.Readline()
		if err != nil { // io.EOF, readline.ErrInterrupt
			fmt.Printf("\n%s\n", theme.User("Goodbye!"))
			return nil
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "quit" {
			fmt.Println(theme.User("Goodbye!"))
			return nil
		}

		// Process user input
		if err := a.processInput(line); err != nil {
			fmt.Printf("\n%s\n\n", theme.Error("Error: %v", err))
		}
	}
}

// handleTokenWarnings listens for token warnings and displays them
func (a *Agent) handleTokenWarnings() {
	for warning := range a.tokenMonitor.Warnings() {
		// Display warning to user
		msg := warning.FormatWarning()

		switch warning.Level {
		case ctxmgr.LevelInfo:
			fmt.Printf("\n%s\n", theme.Dim(msg))
		case ctxmgr.LevelWarning:
			fmt.Printf("\n%s\n", theme.Warning(msg))
			// Optionally trigger automatic memory offload here
			a.offloadContextToMemory(warning)
		case ctxmgr.LevelCritical:
			fmt.Printf("\n%s\n", theme.Error(msg))
			// Urgent: offload immediately
			a.offloadContextToMemory(warning)
		}
	}
}

func (a *Agent) processInput(input string) error {
	a.logger.LogUserInput(input)

	// Inject plan context if relevant (and plan manager is available)
	userInput := input
	if a.planMgr != nil && a.planMgr.ShouldInjectPlanContext(input) {
		planContext := a.planMgr.GetCurrentContext()
		if planContext != "" {
			userInput = fmt.Sprintf("%s\n\n%s", planContext, input)
		}
	}

	// Add user message
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: userInput,
	})

	// Update context manager
	a.contextMgr.SetMessages(a.messages)

	// Append to conversation history (original input, not with plan context)
	a.appendToConversationHistory("user", input)

	// Main conversation loop
	for {
		// Check if context needs pruning
		if a.contextMgr.NeedsPruning() {
			fmt.Println(theme.Dim("  [Context window getting full, pruning old messages...]"))
			a.messages = a.contextMgr.PruneMessages()
			// Reset warning level after pruning
			a.tokenMonitor.ResetWarningLevel()
		}

		// Prepare tools for LLM
		toolDefs := make([]llm.Tool, 0)
		for _, tool := range a.toolRegistry.All() {
			toolDefs = append(toolDefs, llm.Tool{
				Type: "function",
				Function: llm.Function{
					Name:        tool.Name(),
					Description: tool.Description(),
					Parameters:  tool.Parameters(),
				},
			})
		}

		// Prepare messages for LLM (with context management)
		messagesToSend := a.contextMgr.GetMessages()

		// Request completion from LLM
		req := llm.CompletionRequest{
			Messages:    messagesToSend,
			Tools:       toolDefs,
			Temperature: a.config.LLM.Temperature,
			MaxTokens:   a.config.LLM.MaxTokens,
		}

		a.logger.LogLLMRequest(a.convertMessagesToInterface(), a.config.LLM.Model, a.config.LLM.Temperature)

		resp, err := a.llmClient.Complete(context.Background(), req)
		if err != nil {
			return fmt.Errorf("LLM completion failed: %w", err)
		}

		a.logger.LogLLMResponse(resp.Content, a.convertToolCallsToInterface(resp.ToolCalls))

		// Update token monitor with actual token count if available
		if resp.TotalTokens > 0 {
			a.tokenMonitor.UpdateActualTokens(resp.TotalTokens)
		}

		// Display assistant response
		if resp.Content != "" {
			fmt.Printf("\n%s\n", theme.Agent(resp.Content))
			// Append assistant response to conversation history
			a.appendToConversationHistory("assistant", resp.Content)
		}

		// Handle tool calls
		if len(resp.ToolCalls) > 0 {
			// Add assistant message with tool calls
			assistantMsg := llm.Message{
				Role:    "assistant",
				Content: resp.Content,
			}
			a.messages = append(a.messages, assistantMsg)

			allApproved := true
			for _, toolCall := range resp.ToolCalls {
				a.logger.LogToolCall(toolCall.Function.Name, toolCall.Function.Arguments)

				// Check if confirmation needed
				if a.confirmSys.ShouldConfirm(toolCall.Function.Name, toolCall.Function.Arguments) {
					approved, err := a.confirmSys.RequestConfirmation(toolCall.Function.Name, toolCall.Function.Arguments)
					if err != nil {
						return err
					}
					if !approved {
						fmt.Println(theme.Error("❌ Tool execution rejected"))
						allApproved = false
						continue
					}
				}

				// Execute tool with enhanced output
				fmt.Printf("\n%s %s\n", theme.Tool("🔧 Using"), theme.ToolBold(toolCall.Function.Name))

				// Display formatted arguments
				formattedArgs := formatToolArgs(toolCall.Function.Name, toolCall.Function.Arguments)
				for _, arg := range formattedArgs {
					fmt.Printf("   %s %s\n", theme.Dim("└─"), theme.Dim(arg))
				}

				result, err := a.toolRegistry.Execute(context.Background(), toolCall.Function.Name, toolCall.Function.Arguments)

				a.logger.LogToolResult(toolCall.Function.Name, result, err)

				// Display result summary
				summary := summarizeToolResult(toolCall.Function.Name, result, err)
				if err != nil {
					fmt.Printf("%s %s\n", theme.Error("❌"), theme.Error(summary))
				} else {
					fmt.Printf("%s %s\n", theme.Success("✓"), theme.Success(summary))
				}

				// Add tool result message
				resultContent := result
				if err != nil {
					resultContent = fmt.Sprintf("Error: %v", err)
				}

				a.messages = append(a.messages, llm.Message{
					Role:    "tool",
					Content: resultContent,
					ToolID:  toolCall.ID,
				})
			}

			if !allApproved {
				break
			}

			// Continue loop to get LLM's next response
			continue
		}

		// No more tool calls, conversation turn complete
		if resp.FinishReason == "stop" {
			break
		}
	}

	fmt.Println()
	return nil
}

// formatToolArgs extracts and formats key arguments from tool call JSON
func formatToolArgs(toolName, argsJSON string) []string {
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil
	}

	var formatted []string

	// Extract relevant parameters based on tool type
	switch toolName {
	case "read":
		if path, ok := args["file_path"].(string); ok {
			formatted = append(formatted, fmt.Sprintf("file: %s", filepath.Base(path)))
		}
		if offset, ok := args["offset"].(float64); ok {
			formatted = append(formatted, fmt.Sprintf("offset: %d", int(offset)))
		}
		if limit, ok := args["limit"].(float64); ok {
			formatted = append(formatted, fmt.Sprintf("limit: %d", int(limit)))
		}

	case "write":
		if path, ok := args["file_path"].(string); ok {
			formatted = append(formatted, fmt.Sprintf("file: %s", filepath.Base(path)))
		}
		if content, ok := args["content"].(string); ok {
			lines := strings.Count(content, "\n") + 1
			formatted = append(formatted, fmt.Sprintf("lines: %d", lines))
		}

	case "edit":
		if path, ok := args["file_path"].(string); ok {
			formatted = append(formatted, fmt.Sprintf("file: %s", filepath.Base(path)))
		}

	case "glob":
		if pattern, ok := args["pattern"].(string); ok {
			formatted = append(formatted, fmt.Sprintf("pattern: %s", pattern))
		}

	case "grep":
		if pattern, ok := args["pattern"].(string); ok {
			formatted = append(formatted, fmt.Sprintf("pattern: %s", pattern))
		}
		if glob, ok := args["glob"].(string); ok {
			formatted = append(formatted, fmt.Sprintf("files: %s", glob))
		}

	case "bash":
		if cmd, ok := args["command"].(string); ok {
			// Truncate long commands
			if len(cmd) > 60 {
				cmd = cmd[:57] + "..."
			}
			formatted = append(formatted, fmt.Sprintf("cmd: %s", cmd))
		}

	case "todo_write":
		if todos, ok := args["todos"].([]interface{}); ok {
			formatted = append(formatted, fmt.Sprintf("tasks: %d", len(todos)))
		}
	}

	return formatted
}

// summarizeToolResult creates a human-readable summary of tool execution result
func summarizeToolResult(toolName, result string, err error) string {
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	// Generate summaries based on tool type and result content
	switch toolName {
	case "read":
		lines := strings.Count(result, "\n")
		if lines > 0 {
			return fmt.Sprintf("Read %d lines", lines)
		}
		return "File read successfully"

	case "write":
		return "File written successfully"

	case "edit":
		return "File edited successfully"

	case "glob":
		matches := strings.Split(strings.TrimSpace(result), "\n")
		if len(matches) == 1 && matches[0] == "" {
			return "No files found"
		}
		return fmt.Sprintf("Found %d file(s)", len(matches))

	case "grep":
		if strings.Contains(result, "No matches found") || result == "" {
			return "No matches found"
		}
		matches := strings.Count(result, "\n")
		if matches > 0 {
			return fmt.Sprintf("Found matches in %d location(s)", matches)
		}
		return "Found matches"

	case "bash":
		lines := strings.Count(result, "\n")
		if lines > 5 {
			return fmt.Sprintf("Command executed (%d lines output)", lines)
		} else if result == "" {
			return "Command executed successfully"
		}
		return "Command executed"

	case "todo_write":
		return "Task list updated"

	case "web_fetch":
		return "Content fetched successfully"

	case "web_search":
		return "Search completed"

	default:
		// Generic summary
		if len(result) > 100 {
			return fmt.Sprintf("Completed (%d chars)", len(result))
		}
		return "Completed"
	}
}

func (a *Agent) convertMessagesToInterface() []interface{} {
	result := make([]interface{}, len(a.messages))
	for i, msg := range a.messages {
		result[i] = msg
	}
	return result
}

func (a *Agent) convertToolCallsToInterface(toolCalls []llm.ToolCall) []interface{} {
	result := make([]interface{}, len(toolCalls))
	for i, tc := range toolCalls {
		data, _ := json.Marshal(tc)
		var m map[string]interface{}
		json.Unmarshal(data, &m)
		result[i] = m
	}
	return result
}

// buildToolInfos creates tool information for the system prompt
func buildToolInfos(registry *tools.Registry) []prompts.ToolInfo {
	toolCategories := map[string]string{
		"read":            "file",
		"write":           "file",
		"edit":            "file",
		"glob":            "search",
		"grep":            "search",
		"bash":            "bash",
		"bash_output":     "bash",
		"kill_shell":      "bash",
		"web_fetch":       "web",
		"web_search":      "web",
		"todo_write":      "task",
		"store_memory":    "memory",
		"recall_memory":   "memory",
		"find_definition": "lsp",
		"find_references": "lsp",
		"list_symbols":    "lsp",
	}

	var toolInfos []prompts.ToolInfo
	for _, tool := range registry.All() {
		category := toolCategories[tool.Name()]
		if category == "" {
			category = "other"
		}

		toolInfos = append(toolInfos, prompts.ToolInfo{
			Name:        tool.Name(),
			Description: tool.Description(),
			Category:    category,
		})
	}

	return toolInfos
}

// buildProjectContext converts ProjectAnalysis to ProjectContext for prompt rendering
func buildProjectContext(analysis *initialization.ProjectAnalysis) *prompts.ProjectContext {
	// Build primary languages string
	primaryLangs := []string{}
	for _, lang := range analysis.Languages {
		if lang.Primary {
			primaryLangs = append(primaryLangs, lang.Name)
		}
	}
	primaryLanguages := strings.Join(primaryLangs, " + ")
	if primaryLanguages == "" && len(analysis.Languages) > 0 {
		primaryLanguages = analysis.Languages[0].Name
	}

	// Build frameworks string
	frameworkNames := []string{}
	for _, fw := range analysis.Frameworks {
		frameworkNames = append(frameworkNames, fw.Name)
	}
	frameworks := strings.Join(frameworkNames, ", ")

	// Build tech stack description
	techStack := buildTechStackDescription(analysis)

	// Build structure description
	structure := buildStructureDescription(analysis)

	// Get git branch
	gitBranch := ""
	if analysis.GitInfo != nil && analysis.GitInfo.CurrentBranch != "" {
		gitBranch = analysis.GitInfo.CurrentBranch
	}

	return &prompts.ProjectContext{
		ProjectName:      analysis.ProjectName,
		PrimaryLanguages: primaryLanguages,
		TotalFiles:       analysis.Statistics.TotalFiles,
		CodeFiles:        analysis.Statistics.CodeFiles,
		TotalLines:       analysis.Statistics.TotalLines,
		Frameworks:       frameworks,
		GitBranch:        gitBranch,
		TechStack:        techStack,
		Structure:        structure,
	}
}

// buildTechStackDescription creates a description of the tech stack
func buildTechStackDescription(analysis *initialization.ProjectAnalysis) string {
	var parts []string

	// Languages
	for _, lang := range analysis.Languages {
		parts = append(parts, fmt.Sprintf("- **%s**: %d files", lang.Name, lang.FileCount))
	}

	return strings.Join(parts, "\n")
}

// buildStructureDescription creates a description of the project structure
func buildStructureDescription(analysis *initialization.ProjectAnalysis) string {
	var parts []string

	if analysis.Structure.HasSrcDir {
		parts = append(parts, "- Source code organized in `src/` directory")
	}
	if analysis.Structure.HasTestsDir {
		parts = append(parts, "- Tests located in dedicated test directory")
	}
	if len(analysis.Structure.EntryPoints) > 0 {
		parts = append(parts, fmt.Sprintf("- Entry points: %s", strings.Join(analysis.Structure.EntryPoints, ", ")))
	}

	if len(parts) == 0 {
		return "- Standard project layout"
	}

	return strings.Join(parts, "\n")
}

// stripThinking removes LLM internal reasoning/thinking from content
// Many models output extended thinking or chain-of-thought reasoning that
// should not be shown to users or logged in conversation history
func stripThinking(content string) string {
	// If content starts with excessive internal reasoning patterns,
	// try to extract just the final user-facing response

	// Pattern 1: Look for lines that seem like actual responses vs thinking
	// Thinking often has phrases like "Wait,", "Let me", "Actually,", "Hmm,"
	// and tends to be very verbose stream-of-consciousness

	lines := strings.Split(content, "\n")
	if len(lines) < 10 {
		// Short responses are likely not thinking-heavy
		return content
	}

	// Check if the content starts with extended reasoning
	// Indicators: many short lines, lots of "I should", "Let me", "Wait"
	thinkingIndicators := []string{
		"Wait,", "Hmm,", "Actually,", "Let me think", "So,", "But wait",
		"I should", "I need to", "First,", "Then,", "Next,", "Alright,",
	}

	thinkingCount := 0
	for i := 0; i < len(lines) && i < 20; i++ {
		line := strings.TrimSpace(lines[i])
		for _, indicator := range thinkingIndicators {
			if strings.Contains(line, indicator) {
				thinkingCount++
				break
			}
		}
	}

	// If more than 30% of first 20 lines contain thinking indicators,
	// this is likely a thinking-heavy response
	if thinkingCount > 6 {
		// Try to find where the actual response starts
		// Usually after the thinking, there's a clear statement or action
		for i := thinkingCount; i < len(lines); i++ {
			line := strings.TrimSpace(lines[i])
			// Look for lines that start with clear action or response patterns
			if len(line) > 0 && !strings.HasPrefix(line, "Wait") &&
			   !strings.HasPrefix(line, "Hmm") && !strings.HasPrefix(line, "So ") {
				// Found potential start of real response
				// Return from here
				return strings.TrimSpace(strings.Join(lines[i:], "\n"))
			}
		}
	}

	return content
}

// offloadContextToMemory saves important context to long-term memory before pruning
func (a *Agent) offloadContextToMemory(warning ctxmgr.TokenWarning) {
	// Only offload if we have long-term memory enabled
	if a.ltm == nil {
		return
	}

	// For warning level, save a snapshot of recent context
	if warning.Level == ctxmgr.LevelWarning || warning.Level == ctxmgr.LevelCritical {
		// Get recent messages (last 5-10)
		recentCount := 10
		if len(a.messages) < recentCount {
			recentCount = len(a.messages)
		}

		var contextSummary strings.Builder
		contextSummary.WriteString("Recent conversation context (auto-saved before pruning):\n\n")

		// Capture recent exchanges
		for i := len(a.messages) - recentCount; i < len(a.messages); i++ {
			msg := a.messages[i]
			if msg.Role == "user" || msg.Role == "assistant" {
				// Truncate long messages
				content := msg.Content
				if len(content) > 500 {
					content = content[:497] + "..."
				}
				contextSummary.WriteString(fmt.Sprintf("[%s]: %s\n\n", msg.Role, content))
			}
		}

		// Store as a fact memory
		mem := &memory.Memory{
			Type:       memory.TypeFact,
			Summary:    fmt.Sprintf("Context snapshot at %.0f%% token usage", warning.Percentage),
			Content:    contextSummary.String(),
			Tags:       []string{"context-snapshot", "auto-saved"},
			Importance: 0.6, // Medium importance
		}

		if err := a.ltm.Store(mem); err != nil {
			// Log error but don't fail
			fmt.Printf("%s\n", theme.Dim(fmt.Sprintf("  [Failed to save context snapshot: %v]", err)))
		} else {
			fmt.Printf("%s\n", theme.Dim("  [Context snapshot saved to long-term memory]"))
		}
	}
}

// appendToConversationHistory appends a message to the conversation history file
func (a *Agent) appendToConversationHistory(role, content string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	separator := strings.Repeat("=", 80)

	// Strip thinking/reasoning from assistant responses before logging
	cleanContent := content
	if role == "assistant" {
		cleanContent = stripThinking(content)
	}

	entry := fmt.Sprintf("\n%s\n[%s] %s:\n%s\n%s\n",
		separator, timestamp, strings.ToUpper(role), separator, cleanContent)

	f, err := os.OpenFile(a.historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Silently fail - history is not critical
		return
	}
	defer f.Close()

	f.WriteString(entry)
}
