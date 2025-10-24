package theme

import (
	"encoding/json"
	"strings"

	"github.com/fatih/color"
)

// Synthwave color palette
var (
	// Primary colors
	Cyan   = color.New(color.FgCyan)
	Pink   = color.New(color.FgMagenta)
	Purple = color.New(color.FgMagenta, color.FgCyan) // Blend effect
	Green  = color.New(color.FgGreen)
	Red    = color.New(color.FgRed)
	Yellow = color.New(color.FgYellow)
	Gray   = color.New(color.FgHiBlack)
	White  = color.New(color.FgWhite)

	// Bold variants for emphasis
	CyanBold   = color.New(color.FgCyan, color.Bold)
	PinkBold   = color.New(color.FgMagenta, color.Bold)
	PurpleBold = color.New(color.FgMagenta, color.Bold)
	GreenBold  = color.New(color.FgGreen, color.Bold)
	RedBold    = color.New(color.FgRed, color.Bold)
)

// Semantic color functions

// Agent returns cyan color for agent responses
func Agent(format string, a ...interface{}) string {
	return Cyan.Sprintf(format, a...)
}

// AgentBold returns bold cyan for emphasis in agent responses
func AgentBold(format string, a ...interface{}) string {
	return CyanBold.Sprintf(format, a...)
}

// User returns pink color for user-related text
func User(format string, a ...interface{}) string {
	return Pink.Sprintf(format, a...)
}

// UserBold returns bold pink for user prompts
func UserBold(format string, a ...interface{}) string {
	return PinkBold.Sprintf(format, a...)
}

// Success returns green color for success messages
func Success(format string, a ...interface{}) string {
	return Green.Sprintf(format, a...)
}

// SuccessBold returns bold green for emphasis
func SuccessBold(format string, a ...interface{}) string {
	return GreenBold.Sprintf(format, a...)
}

// Error returns red color for error messages
func Error(format string, a ...interface{}) string {
	return Red.Sprintf(format, a...)
}

// ErrorBold returns bold red for emphasis
func ErrorBold(format string, a ...interface{}) string {
	return RedBold.Sprintf(format, a...)
}

// Warning returns yellow color for warnings
func Warning(format string, a ...interface{}) string {
	return Yellow.Sprintf(format, a...)
}

// Tool returns cyan color for tool-related messages
func Tool(format string, a ...interface{}) string {
	return Cyan.Sprintf(format, a...)
}

// ToolBold returns bold cyan for tool names
func ToolBold(format string, a ...interface{}) string {
	return CyanBold.Sprintf(format, a...)
}

// Header returns purple (synthwave) color for headers
func Header(format string, a ...interface{}) string {
	return Purple.Sprintf(format, a...)
}

// HeaderBold returns bold purple for main headers
func HeaderBold(format string, a ...interface{}) string {
	return PurpleBold.Sprintf(format, a...)
}

// Dim returns gray color for secondary information
func Dim(format string, a ...interface{}) string {
	return Gray.Sprintf(format, a...)
}

// JSON syntax highlighter
type jsonHighlighter struct {
	key    *color.Color
	string *color.Color
	number *color.Color
	bool   *color.Color
	null   *color.Color
	punct  *color.Color
}

var jsonColors = jsonHighlighter{
	key:    color.New(color.FgMagenta), // Pink for keys (synthwave)
	string: color.New(color.FgGreen),   // Green for string values
	number: color.New(color.FgCyan),    // Cyan for numbers
	bool:   color.New(color.FgYellow),  // Yellow for booleans
	null:   color.New(color.FgHiBlack), // Gray for null
	punct:  color.New(color.FgWhite),   // White for punctuation
}

// HighlightJSON applies syntax highlighting to JSON strings
func HighlightJSON(jsonStr string) string {
	// Pretty print if it's valid JSON
	var obj interface{}
	if err := json.Unmarshal([]byte(jsonStr), &obj); err == nil {
		pretty, err := json.MarshalIndent(obj, "", "  ")
		if err == nil {
			jsonStr = string(pretty)
		}
	}

	var result strings.Builder
	inString := false
	inKey := false
	escapeNext := false
	depth := 0

	for i := 0; i < len(jsonStr); i++ {
		ch := jsonStr[i]

		// Handle escape sequences
		if escapeNext {
			result.WriteByte(ch)
			escapeNext = false
			continue
		}

		if ch == '\\' && inString {
			result.WriteByte(ch)
			escapeNext = true
			continue
		}

		// Handle strings
		if ch == '"' {
			if !inString {
				inString = true
				// Determine if this is a key or value
				// Keys come after { or ,
				lookback := strings.TrimSpace(jsonStr[:i])
				if strings.HasSuffix(lookback, "{") || strings.HasSuffix(lookback, ",") {
					inKey = true
					result.WriteString(jsonColors.key.Sprint("\""))
				} else {
					result.WriteString(jsonColors.string.Sprint("\""))
				}
			} else {
				if inKey {
					result.WriteString(jsonColors.key.Sprint("\""))
					inKey = false
				} else {
					result.WriteString(jsonColors.string.Sprint("\""))
				}
				inString = false
			}
			continue
		}

		// Inside string, use appropriate color
		if inString {
			if inKey {
				result.WriteString(jsonColors.key.Sprint(string(ch)))
			} else {
				result.WriteString(jsonColors.string.Sprint(string(ch)))
			}
			continue
		}

		// Handle numbers
		if ch >= '0' && ch <= '9' || ch == '-' || ch == '.' {
			// Collect the full number
			numStart := i
			for i < len(jsonStr) && (jsonStr[i] >= '0' && jsonStr[i] <= '9' || jsonStr[i] == '-' || jsonStr[i] == '.' || jsonStr[i] == 'e' || jsonStr[i] == 'E' || jsonStr[i] == '+') {
				i++
			}
			i-- // Step back one
			result.WriteString(jsonColors.number.Sprint(jsonStr[numStart : i+1]))
			continue
		}

		// Handle booleans and null
		if ch == 't' && i+3 < len(jsonStr) && jsonStr[i:i+4] == "true" {
			result.WriteString(jsonColors.bool.Sprint("true"))
			i += 3
			continue
		}
		if ch == 'f' && i+4 < len(jsonStr) && jsonStr[i:i+5] == "false" {
			result.WriteString(jsonColors.bool.Sprint("false"))
			i += 4
			continue
		}
		if ch == 'n' && i+3 < len(jsonStr) && jsonStr[i:i+4] == "null" {
			result.WriteString(jsonColors.null.Sprint("null"))
			i += 3
			continue
		}

		// Handle structural characters
		if ch == '{' || ch == '}' || ch == '[' || ch == ']' || ch == ':' || ch == ',' {
			if ch == '{' || ch == '[' {
				depth++
			} else if ch == '}' || ch == ']' {
				depth--
			}
			result.WriteString(jsonColors.punct.Sprint(string(ch)))
			continue
		}

		// Everything else (whitespace, etc.)
		result.WriteByte(ch)
	}

	return result.String()
}

// SynthwaveBanner returns a synthwave-themed ASCII art banner
func SynthwaveBanner(version string) string {
	lines := []string{
		"",
		"  " + Cyan.Sprint("╔═══════════════════════════════════════════════════════╗"),
		"  " + Cyan.Sprint("║") + "                                                       " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "            " + CyanBold.Sprint("░█▀▀█ █▀▀█ ░█▀▀█ █▀▀█ █▀▀▄ █▀▀") + "             " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "            " + PinkBold.Sprint("░█░▄▄░█░░█░░█░░░░█░░█░█░░█░█▀▀") + "             " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "            " + CyanBold.Sprint("░█▄▄█ ▀▀▀▀ ░█▄▄█ ▀▀▀▀ ▀▀▀  ▀▀▀") + "             " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "                                                       " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "               " + Pink.Sprint("AI-Powered Development Assistant") + "        " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "                      " + Dim("version "+version) + "                     " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "                                                       " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "     " + CyanBold.Sprint("►") + " " + Pink.Sprint("Autonomous code analysis and generation") + "         " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "     " + CyanBold.Sprint("►") + " " + Pink.Sprint("Multi-tool workflow orchestration") + "               " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "     " + CyanBold.Sprint("►") + " " + Pink.Sprint("Real-time task tracking") + "                         " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("║") + "                                                       " + Cyan.Sprint("║"),
		"  " + Cyan.Sprint("╚═══════════════════════════════════════════════════════╝"),
		"",
		"  " + Dim("Type 'exit' or press Ctrl+D to quit"),
		"",
	}

	return strings.Join(lines, "\n")
}

// GetPinkPrompt returns the pink-colored readline prompt
func GetPinkPrompt() string {
	return Pink.Sprint(">") + " "
}
