package confirmation

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/theme"
)

type System struct {
	config *config.ConfirmationConfig
	reader *bufio.Reader
}

func New(cfg *config.ConfirmationConfig) *System {
	return &System{
		config: cfg,
		reader: bufio.NewReader(os.Stdin),
	}
}

func (s *System) ShouldConfirm(toolName string) bool {
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
