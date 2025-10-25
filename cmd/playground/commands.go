package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jake/gocode/internal/agent"
	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/initialization"
	"github.com/jake/gocode/internal/llm"
	"github.com/jake/gocode/internal/logging"
	"github.com/jake/gocode/internal/tools"
)

// handleConfig manages configuration commands
func handleConfig(ctx *PlaygroundContext, args []string) {
	if len(args) == 0 {
		failure("Usage: config <create|validate|show>")
		return
	}

	switch args[0] {
	case "create":
		info("Creating minimal config...")
		cfg := createMinimalConfig(ctx.TempDir)
		ctx.Config = cfg
		success("Config created")
		fmt.Printf("  BaseDir: %s\n", cfg.BaseDir)
		fmt.Printf("  WorkingDir: %s\n", cfg.WorkingDir)
		fmt.Printf("  LogDir: %s\n", filepath.Join(cfg.BaseDir, cfg.Logging.Directory))

	case "validate":
		if ctx.Config == nil {
			failure("No config loaded. Run 'config create' first")
			return
		}
		validateConfig(ctx.Config)

	case "show":
		if ctx.Config == nil {
			failure("No config loaded")
			return
		}
		showConfig(ctx.Config)

	default:
		failure(fmt.Sprintf("Unknown config command: %s", args[0]))
	}
}

// handleLogger manages logger commands
func handleLogger(ctx *PlaygroundContext, args []string) {
	if len(args) == 0 {
		failure("Usage: logger <init|test|show>")
		return
	}

	switch args[0] {
	case "init":
		if ctx.Config == nil {
			failure("No config loaded. Run 'config create' first")
			return
		}

		info("Initializing logger...")
		logger, err := logging.New(&ctx.Config.Logging, ctx.Config.BaseDir)
		if err != nil {
			failure(fmt.Sprintf("Failed to create logger: %v", err))
			return
		}

		ctx.Logger = logger
		success("Logger initialized")
		logDir := filepath.Join(ctx.Config.BaseDir, ctx.Config.Logging.Directory)
		fmt.Printf("  Log directory: %s\n", logDir)

	case "test":
		if ctx.Logger == nil {
			failure("Logger not initialized. Run 'logger init' first")
			return
		}

		info("Testing logger...")
		ctx.Logger.LogUserInput("test input")
		ctx.Logger.LogToolCall("read", `{"file_path": "test.txt"}`)
		ctx.Logger.LogToolResult("read", "file contents", nil)
		success("Test entries logged")

	case "show":
		if ctx.Config == nil {
			failure("No config loaded")
			return
		}
		logDir := filepath.Join(ctx.Config.BaseDir, ctx.Config.Logging.Directory)
		fmt.Printf("Log directory: %s\n", logDir)

	default:
		failure(fmt.Sprintf("Unknown logger command: %s", args[0]))
	}
}

// handleTool manages tool commands
func handleTool(ctx *PlaygroundContext, args []string) {
	if len(args) == 0 {
		failure("Usage: tool <list|name> [args...]")
		return
	}

	if args[0] == "list" {
		listTools(ctx)
		return
	}

	toolName := args[0]
	toolArgs := args[1:]

	// Quick shortcuts for common tools
	switch toolName {
	case "read":
		if len(toolArgs) == 0 {
			failure("Usage: tool read <file_path>")
			return
		}
		testReadTool(toolArgs[0])

	case "bash":
		if len(toolArgs) == 0 {
			failure("Usage: tool bash <command>")
			return
		}
		command := strings.Join(toolArgs, " ")
		testBashTool(command)

	default:
		info(fmt.Sprintf("Interactive testing for '%s' tool", toolName))
		warn("Interactive tool testing not yet implemented")
		info("Try: tool read <file> or tool bash <command>")
	}
}

// handleLLM manages LLM commands
func handleLLM(ctx *PlaygroundContext, args []string) {
	if len(args) == 0 {
		failure("Usage: llm <ping|complete> [text...]")
		return
	}

	if ctx.Config == nil {
		failure("No config loaded. Run 'config create' first")
		return
	}

	switch args[0] {
	case "ping":
		info(fmt.Sprintf("Testing LLM endpoint: %s", ctx.Config.LLM.Endpoint))
		testLLMConnection(ctx.Config.LLM.Endpoint)

	case "complete":
		if len(args) < 2 {
			failure("Usage: llm complete <prompt text>")
			return
		}
		prompt := strings.Join(args[1:], " ")
		testLLMCompletion(ctx.Config, prompt)

	default:
		failure(fmt.Sprintf("Unknown llm command: %s", args[0]))
	}
}

// handleAgent manages agent commands
func handleAgent(ctx *PlaygroundContext, args []string) {
	if len(args) == 0 {
		failure("Usage: agent <create|prompt|status>")
		return
	}

	switch args[0] {
	case "create":
		if ctx.Config == nil {
			failure("No config loaded. Run 'config create' first")
			return
		}

		info("Creating agent...")
		detector, err := initialization.NewDetector(ctx.TempDir)
		if err != nil {
			failure(fmt.Sprintf("Failed to create detector: %v", err))
			return
		}

		analyzer := initialization.NewAnalyzer(ctx.TempDir, detector)
		analysis, err := analyzer.Analyze()
		if err != nil {
			failure(fmt.Sprintf("Failed to analyze project: %v", err))
			return
		}

		ag, err := agent.New(ctx.Config, analysis)
		if err != nil {
			failure(fmt.Sprintf("Failed to create agent: %v", err))
			return
		}

		ctx.Agent = ag
		success("Agent created")
		info(fmt.Sprintf("Tools registered: %d", len(ctx.Config.Tools.Enabled)))

	case "status":
		if ctx.Agent == nil {
			failure("No agent created. Run 'agent create' first")
			return
		}
		success("Agent is ready")

	case "prompt":
		failure("Agent prompting not yet implemented")
		info("Use 'autonomous simple' for testing")

	default:
		failure(fmt.Sprintf("Unknown agent command: %s", args[0]))
	}
}

// handleAutonomous manages autonomous execution commands
func handleAutonomous(ctx *PlaygroundContext, args []string) {
	if len(args) == 0 {
		failure("Usage: autonomous <simple|task>")
		return
	}

	if ctx.Agent == nil {
		failure("No agent created. Run 'agent create' first")
		return
	}

	switch args[0] {
	case "simple":
		testAutonomousSimple(ctx)

	case "task":
		if len(args) < 2 {
			failure("Usage: autonomous task <task description>")
			return
		}
		task := strings.Join(args[1:], " ")
		testAutonomousTask(ctx, task)

	default:
		failure(fmt.Sprintf("Unknown autonomous command: %s", args[0]))
	}
}

// Helper functions

func createMinimalConfig(baseDir string) *config.Config {
	return &config.Config{
		BaseDir:    baseDir,
		WorkingDir: baseDir,
		Logging: config.LoggingConfig{
			Directory:      "logs",
			Format:         "jsonl",
			Level:          "info",
			LogToolResults: true,
		},
		Tools: config.ToolsConfig{
			Enabled: []string{
				"read", "write", "edit", "bash",
				"glob", "grep", "todo_write",
			},
		},
		Confirmation: config.ConfirmationConfig{
			Mode: "never",
		},
		LLM: config.LLMConfig{
			Endpoint:      "http://localhost:8080/v1",
			Model:         "llama-3",
			Temperature:   0.7,
			MaxTokens:     2048,
			ContextWindow: 32768,
		},
		Checkpoint: config.CheckpointConfig{
			Enabled: false,
		},
		Memory: config.MemoryConfig{
			Enabled: false,
		},
		Telemetry: config.TelemetryConfig{
			Enabled: false,
		},
		Retrieval: config.RetrievalConfig{
			Enabled: false,
		},
		LSP: config.LSPConfig{
			Enabled: false,
		},
	}
}

func validateConfig(cfg *config.Config) {
	info("Validating config...")

	checks := 0
	passed := 0

	// Check BaseDir
	checks++
	if cfg.BaseDir == "" {
		failure("BaseDir is empty")
	} else {
		success(fmt.Sprintf("BaseDir: %s", cfg.BaseDir))
		passed++
	}

	// Check log directory
	checks++
	logDir := filepath.Join(cfg.BaseDir, cfg.Logging.Directory)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		failure(fmt.Sprintf("Cannot create log directory: %v", err))
	} else {
		success(fmt.Sprintf("Log directory: %s", logDir))
		passed++
	}

	// Check LLM endpoint
	checks++
	if cfg.LLM.Endpoint == "" {
		warn("LLM endpoint not set")
	} else {
		success(fmt.Sprintf("LLM endpoint: %s", cfg.LLM.Endpoint))
		passed++
	}

	fmt.Printf("\nValidation: %d/%d checks passed\n", passed, checks)
}

func showConfig(cfg *config.Config) {
	fmt.Println("\n\033[1;33m═══ Current Configuration ═══\033[0m")
	fmt.Printf("BaseDir:      %s\n", cfg.BaseDir)
	fmt.Printf("WorkingDir:   %s\n", cfg.WorkingDir)
	fmt.Printf("Log Dir:      %s\n", cfg.Logging.Directory)
	fmt.Printf("LLM Endpoint: %s\n", cfg.LLM.Endpoint)
	fmt.Printf("LLM Model:    %s\n", cfg.LLM.Model)
	fmt.Printf("Tools:        %d enabled\n", len(cfg.Tools.Enabled))
	fmt.Printf("Confirmation: %s\n", cfg.Confirmation.Mode)
	fmt.Println()
}

func listTools(ctx *PlaygroundContext) {
	fmt.Println("\n\033[1;33m═══ Available Tools ═══\033[0m")

	// Define all possible tools with categories and requirements
	type toolInfo struct {
		name        string
		description string
		requirement string // empty if no special requirement
	}

	categories := map[string][]toolInfo{
		"File Operations": {
			{"read", "Read file contents", ""},
			{"write", "Write file contents", ""},
			{"edit", "Edit existing files", ""},
			{"glob", "Find files by pattern", ""},
			{"grep", "Search file contents", ""},
		},
		"Shell": {
			{"bash", "Execute shell commands", ""},
			{"bash_output", "Monitor background shells", "bash"},
			{"kill_shell", "Kill background shells", "bash"},
		},
		"Planning": {
			{"todo_write", "Manage TODO list", ""},
		},
		"Web": {
			{"web_fetch", "Fetch web content", ""},
			{"web_search", "Search the web", ""},
		},
		"Memory": {
			{"store_memory", "Store to long-term memory", "Memory"},
			{"recall_memory", "Recall from long-term memory", "Memory"},
		},
		"LSP": {
			{"find_definition", "Find symbol definitions", "LSP"},
			{"find_references", "Find symbol references", "LSP"},
			{"list_symbols", "List code symbols", "LSP"},
		},
	}

	// Check if agent exists to query registry
	var enabledTools map[string]bool
	if ctx.Agent != nil {
		// Access agent's tool registry through reflection or by getting all tools
		// Since Agent.toolRegistry is private, we'll check against config
		enabledTools = make(map[string]bool)
		if ctx.Config != nil {
			for _, toolName := range ctx.Config.Tools.Enabled {
				enabledTools[toolName] = true
			}
		}
	}

	// Display each category
	categoryOrder := []string{"File Operations", "Shell", "Planning", "Web", "Memory", "LSP"}
	for _, category := range categoryOrder {
		tools := categories[category]
		fmt.Printf("\n\033[1;36m%s:\033[0m\n", category)

		for _, tool := range tools {
			var status, color string
			var note string

			if enabledTools != nil {
				if enabledTools[tool.name] {
					status = "✓"
					color = "\033[1;32m" // Green
				} else {
					status = "✗"
					color = "\033[1;30m" // Gray
					if tool.requirement != "" {
						note = fmt.Sprintf(" (requires %s)", tool.requirement)
					} else {
						note = " (not enabled)"
					}
				}
			} else {
				status = "•"
				color = "\033[1;33m" // Yellow
			}

			fmt.Printf("  %s%s\033[0m %-16s - %s%s\n",
				color, status, tool.name, tool.description, note)
		}
	}

	fmt.Println()
	if enabledTools == nil {
		info("Run 'agent create' to see which tools are currently enabled")
	} else {
		info(fmt.Sprintf("%d tools enabled", len(enabledTools)))
	}
	fmt.Println()
}

func testReadTool(filePath string) {
	info(fmt.Sprintf("Testing read tool with: %s", filePath))

	// Create tool
	readTool := &tools.ReadTool{}
	args := fmt.Sprintf(`{"file_path": "%s"}`, filePath)

	result, err := readTool.Execute(context.Background(), args)
	if err != nil {
		failure(fmt.Sprintf("Error: %v", err))
		return
	}

	// Show first 200 chars
	preview := result
	if len(preview) > 200 {
		preview = preview[:200] + "..."
	}

	success("File read successfully")
	fmt.Printf("Preview:\n%s\n", preview)
}

func testBashTool(command string) {
	info(fmt.Sprintf("Testing bash tool with: %s", command))

	bashTool := &tools.BashTool{}
	args := fmt.Sprintf(`{"command": "%s"}`, command)

	result, err := bashTool.Execute(context.Background(), args)
	if err != nil {
		failure(fmt.Sprintf("Error: %v", err))
		return
	}

	success("Command executed")
	fmt.Printf("Output:\n%s\n", result)
}

func testLLMConnection(endpoint string) {
	client := http.Client{Timeout: 5 * time.Second}

	// Try /models endpoint
	modelsURL := endpoint + "/models"
	resp, err := client.Get(modelsURL)
	if err != nil {
		failure(fmt.Sprintf("Connection failed: %v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		success("LLM endpoint reachable")
	} else {
		warn(fmt.Sprintf("Unexpected status: %d", resp.StatusCode))
	}
}

func testLLMCompletion(cfg *config.Config, prompt string) {
	info("Sending completion request...")

	client := llm.NewClient(&cfg.LLM)

	req := llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		Temperature: cfg.LLM.Temperature,
		MaxTokens:   cfg.LLM.MaxTokens,
	}

	resp, err := client.Complete(context.Background(), req)
	if err != nil {
		failure(fmt.Sprintf("Completion failed: %v", err))
		return
	}

	success("Completion received")
	fmt.Printf("Response: %s\n", resp.Content)
	fmt.Printf("Tokens: %d\n", resp.TotalTokens)
}

func testAutonomousSimple(ctx *PlaygroundContext) {
	task := "List the files in the current directory"
	info(fmt.Sprintf("Running simple autonomous task: %s", task))

	options := agent.DefaultAutonomousOptions()
	options.MaxIterations = 10
	options.Timeout = 30 * time.Second

	result, err := ctx.Agent.RunAutonomous(task, options)
	if err != nil {
		failure(fmt.Sprintf("Execution failed: %v", err))
		return
	}

	if result.Success {
		success("Task completed successfully")
	} else {
		warn(fmt.Sprintf("Task incomplete: %s", result.CompletionReason))
	}

	fmt.Printf("  Iterations: %d\n", result.Iterations)
	fmt.Printf("  Tool calls: %d\n", result.ToolCallCount)
	fmt.Printf("  Tokens: %d\n", result.TokensUsed)
	fmt.Printf("  Duration: %s\n", result.ExecutionTime)
	fmt.Printf("  Final message: %s\n", result.FinalMessage)
}

func testAutonomousTask(ctx *PlaygroundContext, task string) {
	info(fmt.Sprintf("Running custom task: %s", task))

	options := agent.DefaultAutonomousOptions()
	result, err := ctx.Agent.RunAutonomous(task, options)
	if err != nil {
		failure(fmt.Sprintf("Execution failed: %v", err))
		return
	}

	if result.Success {
		success("Task completed")
	} else {
		warn(fmt.Sprintf("Task incomplete: %s", result.CompletionReason))
	}

	fmt.Printf("  Result: %s\n", result.FinalMessage)
}
