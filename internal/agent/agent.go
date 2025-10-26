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
	"github.com/jake/gocode/internal/codegraph"
	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/confirmation"
	"github.com/jake/gocode/internal/initialization"
	"github.com/jake/gocode/internal/llm"
	"github.com/jake/gocode/internal/logging"
	"github.com/jake/gocode/internal/lsp"
	"github.com/jake/gocode/internal/memory"
	"github.com/jake/gocode/internal/prompts"
	"github.com/jake/gocode/internal/theme"
	"github.com/jake/gocode/internal/tools"
)

type Agent struct {
	config           *config.Config
	llmClient        *llm.Client
	serverManager    *llm.ServerManager
	toolRegistry     *tools.Registry
	confirmSys       *confirmation.System
	logger           *logging.Logger
	promptMgr        *prompts.PromptManager
	messages         []llm.Message
	rl               *readline.Instance
	historyFile      string
	todoTool         *tools.TodoWriteTool
	selfCheck        *SelfCheckSystem
	memory           *memory.LongTermMemory
	lastBashExitCode int      // Track last bash command exit code
	lastBashTool     string   // Track if last tool was bash
	toolsUsedInTurn  []string // Track tools used in current turn
}

func New(cfg *config.Config, projectAnalysis *initialization.ProjectAnalysis) (*Agent, error) {
	// Discover and add LSP binary paths to PATH
	lsp.DiscoverAndAddLSPPaths()

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

	// Initialize LSP manager and CodeGraph if LSP is enabled
	var lspMgr *lsp.Manager
	var codeGraph *codegraph.Graph
	if cfg.LSP.Enabled {
		// Build LSP server configs from config
		lspConfigs := make(map[string]lsp.LanguageServerConfig)
		for lang, serverCfg := range cfg.LSP.Servers {
			lspConfigs[lang] = lsp.LanguageServerConfig{
				Command: serverCfg.Command,
				Args:    serverCfg.Args,
			}
		}

		// Create LSP manager
		lspMgr = lsp.NewManager(cfg.WorkingDir, lspConfigs)

		// Validate LSP servers and warn about missing ones
		serverStatus := lspMgr.ValidateServers()
		for lang, available := range serverStatus {
			if !available {
				serverCfg := cfg.LSP.Servers[lang]
				fmt.Printf("%s LSP server for %s (%s) not found in PATH. LSP features will be unavailable for %s files.\n",
					theme.Warning("⚠️"),
					lang,
					serverCfg.Command,
					lang)
			}
		}

		// Create CodeGraph
		codeGraph = codegraph.NewGraph(cfg.WorkingDir, lspMgr)
	}

	// Register enabled tools
	bashTool := tools.NewBashTool()
	todoPath := filepath.Join(cfg.WorkingDir, "TODO.md")
	todoTool := tools.NewTodoWriteTool(todoPath)

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
		case "find_definition":
			if codeGraph != nil {
				registry.Register(tools.NewFindDefinitionTool(codeGraph))
			}
		case "find_references":
			if codeGraph != nil {
				registry.Register(tools.NewFindReferencesTool(codeGraph))
			}
		case "list_symbols":
			if codeGraph != nil {
				registry.Register(tools.NewListSymbolsTool(codeGraph))
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

	// Ensure .gocode directory exists for history files
	gocodeDir := filepath.Join(cfg.WorkingDir, ".gocode")
	if err := os.MkdirAll(gocodeDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .gocode directory: %w", err)
	}

	// Initialize readline
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          theme.GetPinkPrompt(),
		HistoryFile:     filepath.Join(gocodeDir, "history"),
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
	historyFile := filepath.Join(gocodeDir, "conversation_history")

	// Initialize self-check system
	selfCheck := NewSelfCheckSystem(registry)

	// Initialize long-term memory if enabled
	var ltm *memory.LongTermMemory
	if cfg.Memory.Enabled {
		ltm, err = memory.NewLongTermMemory(cfg.Memory.DBPath)
		if err != nil {
			logger.Close()
			serverManager.Stop()
			rl.Close()
			return nil, fmt.Errorf("failed to initialize long-term memory: %w", err)
		}
	}

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
		todoTool:      todoTool,
		selfCheck:     selfCheck,
		memory:        ltm,
	}, nil
}

func (a *Agent) Run() error {
	defer a.serverManager.Stop()
	defer a.logger.Close()
	defer a.rl.Close()
	if a.memory != nil {
		defer a.memory.Close()
	}

	fmt.Print(theme.SynthwaveBanner("v1.0"))

	// Display enabled features
	if a.config.Memory.Enabled || a.config.LSP.Enabled {
		fmt.Println()
		if a.config.LSP.Enabled {
			fmt.Printf("%s %s\n", theme.Success("✓"), theme.Dim("LSP code navigation enabled"))
		}
		if a.config.Memory.Enabled {
			fmt.Printf("%s %s\n", theme.Success("✓"), theme.Dim("Long-term memory enabled"))
		}
	}

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

func (a *Agent) processInput(input string) error {
	a.logger.LogUserInput(input)

	// Reset tools used in this turn
	a.toolsUsedInTurn = []string{}

	// Inject current TODO state before processing
	todos := a.todoTool.GetTodos()
	if len(todos) > 0 {
		todoContext := a.formatTodoContext(todos)
		a.messages = append(a.messages, llm.Message{
			Role:    "system",
			Content: todoContext,
		})
	}

	// Add user message
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: input,
	})

	// Append to conversation history
	a.appendToConversationHistory("user", input)

	// Retrieve relevant memories if memory is enabled
	if a.memory != nil {
		memories, err := a.memory.Search(input, 3) // Get top 3 relevant memories
		if err == nil && len(memories) > 0 {
			var memoryContext strings.Builder
			memoryContext.WriteString("📚 **Relevant memories from past sessions:**\n\n")
			for i, mem := range memories {
				memoryContext.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, mem.Type, mem.Summary))
				if len(mem.Content) > 200 {
					memoryContext.WriteString(fmt.Sprintf("   %s...\n", mem.Content[:200]))
				} else {
					memoryContext.WriteString(fmt.Sprintf("   %s\n", mem.Content))
				}
			}

			// Inject memories as system message
			a.messages = append(a.messages, llm.Message{
				Role:    "system",
				Content: memoryContext.String(),
			})
		}
	}

	// Main conversation loop
	for {
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

		// Request completion from LLM
		req := llm.CompletionRequest{
			Messages:    a.messages,
			Tools:       toolDefs,
			Temperature: a.config.LLM.Temperature,
			MaxTokens:   a.config.LLM.MaxTokens,
		}

		a.logger.LogLLMRequest(a.convertMessagesToInterface(), a.config.LLM.Model, a.config.LLM.Temperature)

		// Show thinking indicator
		fmt.Printf("\n%s", theme.Dim("🤔 Thinking...\r"))

		resp, err := a.llmClient.Complete(context.Background(), req)

		// Clear thinking indicator
		fmt.Printf("\r%s\r", strings.Repeat(" ", 20))

		if err != nil {
			return fmt.Errorf("LLM completion failed: %w", err)
		}

		a.logger.LogLLMResponse(resp.Content, a.convertToolCallsToInterface(resp.ToolCalls))

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

				// Execute tool - show command for bash
				if toolCall.Function.Name == "bash" {
					// Try to extract command from arguments
					var bashArgs map[string]interface{}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &bashArgs); err == nil {
						if cmd, ok := bashArgs["command"].(string); ok {
							// Truncate long commands
							displayCmd := cmd
							if len(displayCmd) > 60 {
								displayCmd = displayCmd[:57] + "..."
							}
							fmt.Printf("\n%s %s %s\n", theme.Tool("🔧 Executing:"), theme.ToolBold(toolCall.Function.Name), theme.Dim("(%s)", displayCmd))
						} else {
							fmt.Printf("\n%s %s\n", theme.Tool("🔧 Executing:"), theme.ToolBold(toolCall.Function.Name))
						}
					} else {
						fmt.Printf("\n%s %s\n", theme.Tool("🔧 Executing:"), theme.ToolBold(toolCall.Function.Name))
					}
				} else {
					fmt.Printf("\n%s %s\n", theme.Tool("🔧 Executing:"), theme.ToolBold(toolCall.Function.Name))
				}

				result, err := a.toolRegistry.Execute(context.Background(), toolCall.Function.Name, toolCall.Function.Arguments)

				// Track tool usage
				a.toolsUsedInTurn = append(a.toolsUsedInTurn, toolCall.Function.Name)

				a.logger.LogToolResult(toolCall.Function.Name, result, err)

				// Track bash tool execution for validation
				if toolCall.Function.Name == "bash" {
					a.lastBashTool = toolCall.Function.Name
					// Parse exit code from error if present
					if err != nil {
						// Error format: "command failed: exit status N"
						if strings.Contains(err.Error(), "exit status") {
							var exitCode int
							fmt.Sscanf(err.Error(), "command failed: exit status %d", &exitCode)
							a.lastBashExitCode = exitCode
						} else {
							a.lastBashExitCode = 1 // Generic error
						}
					} else {
						a.lastBashExitCode = 0 // Success
					}
				}

				// Add tool result message
				resultContent := result
				if err != nil {
					// Enhance error message for bash commands to prevent hallucination
					if toolCall.Function.Name == "bash" {
						resultContent = fmt.Sprintf("Command failed: %v\n\n⚠️  IMPORTANT: The command FAILED with exit code %d.\nDO NOT claim the command succeeded or that tests passed.\nYou must fix the actual problem before claiming success.", err, a.lastBashExitCode)
					} else {
						resultContent = fmt.Sprintf("Error: %v", err)
					}
					fmt.Printf("%s\n", theme.Error("❌ %s", resultContent))
				} else {
					fmt.Printf("%s\n", theme.Success("✓ Complete"))

					// Display TODO status after todo_write execution
					if toolCall.Function.Name == "todo_write" {
						summary := a.todoTool.GetProgressSummary()
						if summary != "" {
							fmt.Printf("%s\n", theme.Dim(summary))
						}
					}
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

		// No more tool calls, check if we need to verify claims
		if resp.FinishReason == "stop" {
			// Create message from response
			assistantMsg := llm.Message{
				Role:    "assistant",
				Content: resp.Content,
			}

			// Check if self-check should trigger
			if a.selfCheck.ShouldTriggerCheck(assistantMsg) {
				// Detect claims
				claims := a.selfCheck.DetectCompletionClaims(resp.Content)

				// Build project context for test detection
				projectContext := a.config.WorkingDir
				if len(a.config.LSP.Servers) > 0 {
					for lang := range a.config.LSP.Servers {
						projectContext += " " + lang
					}
				}

				// Verify claims
				verifiedClaims, err := a.selfCheck.VerifyClaims(context.Background(), claims, projectContext)
				if err != nil {
					fmt.Printf("\n%s\n", theme.Error("Self-check error: %v", err))
				}

				// Generate feedback
				feedback := a.selfCheck.GenerateFeedbackMessage(verifiedClaims)
				if feedback != "" {
					// Check if any claims failed verification
					anyFailed := false
					for _, claim := range verifiedClaims {
						if !claim.Verified {
							anyFailed = true
							break
						}
					}

					if anyFailed {
						// Inject feedback back into conversation
						fmt.Printf("\n%s\n", theme.Warning(feedback))

						a.messages = append(a.messages, llm.Message{
							Role:    "system",
							Content: feedback,
						})

						// Continue the loop to let the agent respond to the feedback
						continue
					}
				}
			}

			// Store important learnings to long-term memory
			if a.memory != nil {
				a.storeConversationMemories(input, resp.Content)
			}

			// Display turn summary
			a.displayTurnSummary()

			break
		}
	}

	fmt.Println()
	return nil
}

// displayTurnSummary shows a summary of what happened in this turn
func (a *Agent) displayTurnSummary() {
	// Don't show summary if no tools were used
	if len(a.toolsUsedInTurn) == 0 {
		return
	}

	var summaryLines []string

	// Tools used
	toolList := strings.Join(a.uniqueTools(a.toolsUsedInTurn), ", ")
	summaryLines = append(summaryLines, fmt.Sprintf("Tools used: %s", toolList))

	// TODO status
	todos := a.todoTool.GetTodos()
	if len(todos) > 0 {
		pending := 0
		inProgress := 0
		completed := 0
		var currentTask string

		for _, todo := range todos {
			switch todo.Status {
			case "pending":
				pending++
			case "in_progress":
				inProgress++
				if currentTask == "" {
					currentTask = todo.Content
				}
			case "completed":
				completed++
			}
		}

		if currentTask != "" {
			// Truncate if too long
			if len(currentTask) > 50 {
				currentTask = currentTask[:47] + "..."
			}
			summaryLines = append(summaryLines, fmt.Sprintf("Current task: %s", currentTask))
		}

		if pending > 0 {
			summaryLines = append(summaryLines, fmt.Sprintf("Next tasks: %d pending", pending))
		}

		total := len(todos)
		percentComplete := (completed * 100) / total
		summaryLines = append(summaryLines, fmt.Sprintf("Progress: %d/%d (%d%%)", completed, total, percentComplete))
	}

	summaryLines = append(summaryLines, "Status: Waiting for your input")

	fmt.Printf("\n%s\n", theme.SummaryBox("🎯 Turn Summary", summaryLines))
}

// uniqueTools returns unique tool names from a list
func (a *Agent) uniqueTools(tools []string) []string {
	seen := make(map[string]bool)
	unique := []string{}

	for _, tool := range tools {
		if !seen[tool] {
			seen[tool] = true
			unique = append(unique, tool)
		}
	}

	return unique
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

// appendToConversationHistory appends a message to the conversation history file
func (a *Agent) appendToConversationHistory(role, content string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	separator := strings.Repeat("=", 80)

	entry := fmt.Sprintf("\n%s\n[%s] %s:\n%s\n%s\n",
		separator, timestamp, strings.ToUpper(role), separator, content)

	f, err := os.OpenFile(a.historyFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Silently fail - history is not critical
		return
	}
	defer f.Close()

	f.WriteString(entry)
}

// formatTodoContext formats the current TODO list for injection into conversation
func (a *Agent) formatTodoContext(todos []tools.TodoItem) string {
	var parts []string
	parts = append(parts, "📋 **Current TODO List:**")
	parts = append(parts, "")

	for i, todo := range todos {
		var icon string
		switch todo.Status {
		case "pending":
			icon = "[ ]"
		case "in_progress":
			icon = "[→]"
		case "completed":
			icon = "[✓]"
		default:
			icon = "[ ]"
		}
		parts = append(parts, fmt.Sprintf("%d. %s %s", i+1, icon, todo.Content))
	}

	parts = append(parts, "")
	parts = append(parts, "_Remember to update this TODO list with `todo_write` as you make progress!_")

	return strings.Join(parts, "\n")
}

// storeConversationMemories extracts and stores important learnings from the conversation
func (a *Agent) storeConversationMemories(userInput, assistantResponse string) {
	// Extract key patterns to store

	// 1. Store architectural decisions
	if strings.Contains(strings.ToLower(userInput), "architecture") ||
		strings.Contains(strings.ToLower(userInput), "design") ||
		strings.Contains(strings.ToLower(userInput), "pattern") {
		mem := &memory.Memory{
			Type:       memory.TypeDecision,
			Content:    fmt.Sprintf("User: %s\nAssistant: %s", userInput, assistantResponse),
			Summary:    userInput,
			Tags:       []string{"architecture", "design"},
			Importance: 0.8,
		}
		a.memory.Store(mem)
	}

	// 2. Store error resolutions
	if strings.Contains(strings.ToLower(userInput), "error") ||
		strings.Contains(strings.ToLower(userInput), "bug") ||
		strings.Contains(strings.ToLower(userInput), "fix") ||
		strings.Contains(strings.ToLower(userInput), "issue") {
		mem := &memory.Memory{
			Type:       memory.TypeError,
			Content:    fmt.Sprintf("Problem: %s\nSolution: %s", userInput, assistantResponse),
			Summary:    userInput,
			Tags:       []string{"error", "troubleshooting"},
			Importance: 0.7,
		}
		a.memory.Store(mem)
	}

	// 3. Store project structure learnings (from read/glob/grep results)
	hasExploredStructure := false
	for _, msg := range a.messages {
		if msg.Role == "tool" && (strings.Contains(msg.Content, "files") || strings.Contains(msg.Content, "directory")) {
			hasExploredStructure = true
			break
		}
	}

	if hasExploredStructure {
		mem := &memory.Memory{
			Type:       memory.TypeFact,
			Content:    fmt.Sprintf("Project exploration - User query: %s\nFindings: %s", userInput, assistantResponse),
			Summary:    "Project structure and organization",
			Tags:       []string{"structure", "files"},
			Importance: 0.6,
		}
		a.memory.Store(mem)
	}

	// 4. Store code patterns and best practices
	if strings.Contains(strings.ToLower(assistantResponse), "pattern") ||
		strings.Contains(strings.ToLower(assistantResponse), "best practice") ||
		strings.Contains(strings.ToLower(assistantResponse), "recommendation") {
		mem := &memory.Memory{
			Type:       memory.TypePattern,
			Content:    assistantResponse,
			Summary:    userInput,
			Tags:       []string{"pattern", "best-practice"},
			Importance: 0.7,
		}
		a.memory.Store(mem)
	}
}
