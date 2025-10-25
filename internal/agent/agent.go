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
	"github.com/jake/gocode/internal/initialization"
	"github.com/jake/gocode/internal/llm"
	"github.com/jake/gocode/internal/logging"
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
		HistoryFile:     ".gocode_history",
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
	historyFile := filepath.Join(cfg.WorkingDir, ".gocode_conversation_history")

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
	}, nil
}

func (a *Agent) Run() error {
	defer a.serverManager.Stop()
	defer a.logger.Close()
	defer a.rl.Close()

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

func (a *Agent) processInput(input string) error {
	a.logger.LogUserInput(input)

	// Add user message
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: input,
	})

	// Append to conversation history
	a.appendToConversationHistory("user", input)

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

		resp, err := a.llmClient.Complete(context.Background(), req)
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

				// Execute tool
				fmt.Printf("\n%s %s\n", theme.Tool("🔧 Executing:"), theme.ToolBold(toolCall.Function.Name))
				result, err := a.toolRegistry.Execute(context.Background(), toolCall.Function.Name, toolCall.Function.Arguments)

				a.logger.LogToolResult(toolCall.Function.Name, result, err)

				// Add tool result message
				resultContent := result
				if err != nil {
					resultContent = fmt.Sprintf("Error: %v", err)
					fmt.Printf("%s\n", theme.Error("❌ %s", resultContent))
				} else {
					fmt.Printf("%s\n", theme.Success("✓ Complete"))
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
