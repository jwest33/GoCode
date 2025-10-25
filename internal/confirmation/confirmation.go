package confirmation

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/theme"
)

type System struct {
	config     *config.ConfirmationConfig
	reader     *bufio.Reader
	workingDir string
}

func New(cfg *config.ConfirmationConfig) *System {
	workingDir, _ := os.Getwd()
	return &System{
		config:     cfg,
		reader:     bufio.NewReader(os.Stdin),
		workingDir: workingDir,
	}
}

func (s *System) ShouldConfirm(toolName string, args string) bool {
	if s.config.Mode == "auto" {
		return false
	}

	if s.config.Mode == "interactive" {
		// Check if tool is in auto-approve list
		for _, t := range s.config.AutoApproveTools {
			if t == toolName {
				return false
			}
		}

		// For write and edit tools, check if path is within working directory
		if toolName == "write" || toolName == "edit" {
			if s.isWithinWorkingDir(args) {
				return false // Auto-approve if within working dir
			}
		}

		// For bash tool, check if accessing files outside working directory
		if toolName == "bash" {
			if !s.accessesExternalPaths(args) {
				return false // Auto-approve if no external access
			}
		}

		return true
	}

	if s.config.Mode == "destructive_only" {
		// Only confirm tools in always_confirm list
		for _, t := range s.config.AlwaysConfirmTools {
			if t == toolName {
				return true
			}
		}
		return false
	}

	return false
}

// isWithinWorkingDir checks if the file_path in args is within the working directory
func (s *System) isWithinWorkingDir(args string) bool {
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
		return false
	}

	filePath, ok := argsMap["file_path"].(string)
	if !ok {
		return false
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	// Check if path is within working directory
	relPath, err := filepath.Rel(s.workingDir, absPath)
	if err != nil {
		return false
	}

	// If relative path starts with "..", it's outside working dir
	return !strings.HasPrefix(relPath, "..")
}

// accessesExternalPaths checks if a bash command tries to access paths outside working directory
func (s *System) accessesExternalPaths(args string) bool {
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
		return false
	}

	command, ok := argsMap["command"].(string)
	if !ok {
		return false
	}

	// Check for absolute paths that are outside working directory
	// This is a simple heuristic - if command contains absolute paths starting with drive letters
	// or paths like C:\ that are not within the working directory, flag for confirmation

	// For now, we'll be conservative and only auto-approve if the command doesn't look like
	// it's accessing absolute paths outside the working directory
	lowerCmd := strings.ToLower(command)

	// Check for common patterns that might access external paths
	externalPatterns := []string{
		"c:\\windows", "c:\\program", "/usr/", "/etc/", "/var/",
		"d:\\", "e:\\", "f:\\", // Other drives
	}

	for _, pattern := range externalPatterns {
		if strings.Contains(lowerCmd, pattern) {
			return true
		}
	}

	return false
}

func (s *System) RequestConfirmation(toolName string, args string) (bool, error) {
	fmt.Printf("\n%s\n", theme.UserBold("╭─────────────────────────────────────────╮"))
	fmt.Printf("%s\n", theme.UserBold("│ Tool Execution Request                 │"))
	fmt.Printf("%s\n", theme.UserBold("╰─────────────────────────────────────────╯"))
	fmt.Printf("\n%s %s\n", theme.User("Tool:"), theme.ToolBold(toolName))
	fmt.Printf("\n%s\n%s\n", theme.User("Arguments:"), theme.HighlightJSON(args))
	fmt.Printf("\n%s\n", theme.Dim("───────────────────────────────────────────"))
	fmt.Printf("%s", theme.UserBold("Approve execution? [y/n/m]: "))

	response, err := s.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read confirmation: %w", err)
	}

	response = strings.ToLower(strings.TrimSpace(response))

	switch response {
	case "y", "yes":
		return true, nil
	case "n", "no":
		return false, nil
	case "m", "modify":
		fmt.Println(theme.Warning("Modification not yet implemented - treating as reject"))
		return false, nil
	default:
		fmt.Println(theme.Warning("Invalid response - treating as reject"))
		return false, nil
	}
}
