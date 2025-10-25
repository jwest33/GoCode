package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jake/gocode/internal/agent"
	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/logging"
	"github.com/jake/gocode/internal/tools"
)

const version = "0.1.0"

// PlaygroundContext holds the state for the interactive session
type PlaygroundContext struct {
	WorkingDir string
	TempDir    string
	Config     *config.Config
	Logger     *logging.Logger
	Registry   *tools.Registry
	Agent      *agent.Agent
}

func main() {
	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║   GoCode Dev Playground v" + version + "       ║")
	fmt.Println("║   Interactive Component Testing        ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Type 'help' for available commands")
	fmt.Println("Type 'exit' to quit")
	fmt.Println()

	ctx := NewPlaygroundContext()
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n\033[1;36m>\033[0m ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		parts := strings.Fields(input)
		command := parts[0]
		args := parts[1:]

		switch command {
		case "help", "h", "?":
			printHelp()

		case "config":
			handleConfig(ctx, args)

		case "logger":
			handleLogger(ctx, args)

		case "tool":
			handleTool(ctx, args)

		case "llm":
			handleLLM(ctx, args)

		case "agent":
			handleAgent(ctx, args)

		case "autonomous":
			handleAutonomous(ctx, args)

		case "clear", "cls":
			clearScreen()

		case "exit", "quit", "q":
			fmt.Println("\033[1;32m✓\033[0m Goodbye!")
			return

		default:
			fmt.Printf("\033[1;31m✗\033[0m Unknown command: %s\n", command)
			fmt.Println("Type 'help' for available commands")
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	}
}

// NewPlaygroundContext initializes the playground environment
func NewPlaygroundContext() *PlaygroundContext {
	wd, _ := os.Getwd()
	tempDir := filepath.Join(wd, "playground")

	// Create playground directory
	os.MkdirAll(tempDir, 0755)

	return &PlaygroundContext{
		WorkingDir: wd,
		TempDir:    tempDir,
	}
}

func printHelp() {
	fmt.Println()
	fmt.Println("\033[1;33m═══ Available Commands ═══\033[0m")
	fmt.Println()
	fmt.Println("\033[1;36mConfiguration:\033[0m")
	fmt.Println("  config create         Create minimal config")
	fmt.Println("  config validate       Validate current config")
	fmt.Println("  config show           Show current config")
	fmt.Println()
	fmt.Println("\033[1;36mLogging:\033[0m")
	fmt.Println("  logger init           Initialize logger")
	fmt.Println("  logger test           Test logging functionality")
	fmt.Println("  logger show           Show log file location")
	fmt.Println()
	fmt.Println("\033[1;36mTools:\033[0m")
	fmt.Println("  tool list             List all available tools")
	fmt.Println("  tool <name>           Test a specific tool interactively")
	fmt.Println("  tool read <file>      Quick test: read a file")
	fmt.Println("  tool bash <cmd>       Quick test: run a command")
	fmt.Println()
	fmt.Println("\033[1;36mLLM:\033[0m")
	fmt.Println("  llm ping              Test LLM connection")
	fmt.Println("  llm complete <text>   Test completion")
	fmt.Println()
	fmt.Println("\033[1;36mAgent:\033[0m")
	fmt.Println("  agent create          Create minimal agent")
	fmt.Println("  agent prompt <text>   Send prompt to agent")
	fmt.Println("  agent status          Show agent status")
	fmt.Println()
	fmt.Println("\033[1;36mAutonomous:\033[0m")
	fmt.Println("  autonomous simple     Test simple autonomous task")
	fmt.Println("  autonomous task       Custom autonomous task")
	fmt.Println()
	fmt.Println("\033[1;36mUtility:\033[0m")
	fmt.Println("  clear                 Clear screen")
	fmt.Println("  help                  Show this help")
	fmt.Println("  exit                  Exit playground")
	fmt.Println()
}

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

// Utility functions for colored output
func success(msg string) {
	fmt.Printf("\033[1;32m✓\033[0m %s\n", msg)
}

func failure(msg string) {
	fmt.Printf("\033[1;31m✗\033[0m %s\n", msg)
}

func info(msg string) {
	fmt.Printf("\033[1;34mℹ\033[0m %s\n", msg)
}

func warn(msg string) {
	fmt.Printf("\033[1;33m⚠\033[0m %s\n", msg)
}
