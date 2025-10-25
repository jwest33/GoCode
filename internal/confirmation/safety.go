package confirmation

import (
	"encoding/json"
	"strings"
)

// SafetyChecker performs pattern-based safety checks on tool executions
type SafetyChecker struct {
	dangerousPatterns []string
}

// NewSafetyChecker creates a new safety checker with default dangerous patterns
func NewSafetyChecker(customPatterns []string) *SafetyChecker {
	// Default dangerous patterns
	defaultPatterns := []string{
		// Destructive file operations
		"rm -rf /",
		"rm -rf /*",
		"rm -rf ~",
		"rm -r /",
		"rmdir /",

		// Disk operations
		"dd if=/dev/zero",
		"dd if=/dev/random",
		"dd if=/dev/urandom",
		"mkfs",
		"fdisk",
		"parted",

		// Permission changes
		"chmod -R 777",
		"chmod 777 /",
		"chown -R",

		// Remote execution
		"curl | bash",
		"curl | sh",
		"wget | sh",
		"wget | bash",
		"curl.*|.*bash",
		"wget.*|.*sh",

		// System modification
		"format ",
		"> /dev/",
		">/dev/",
	}

	patterns := append(defaultPatterns, customPatterns...)

	return &SafetyChecker{
		dangerousPatterns: patterns,
	}
}

// IsDangerous checks if a tool execution is potentially dangerous
// Returns (isDangerous, reason)
func (sc *SafetyChecker) IsDangerous(toolName, args string) (bool, string) {
	switch toolName {
	case "bash":
		return sc.isDangerousBashCommand(args)
	case "write", "edit":
		return sc.isDangerousFileOperation(args)
	case "read", "glob", "grep", "bash_output", "kill_shell", "todo_write":
		// Read-only operations are generally safe
		return false, ""
	default:
		// Unknown tools - be conservative
		return false, ""
	}
}

// isDangerousBashCommand checks if a bash command is dangerous
func (sc *SafetyChecker) isDangerousBashCommand(args string) (bool, string) {
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
		// Can't parse - assume safe
		return false, ""
	}

	command, ok := argsMap["command"].(string)
	if !ok {
		return false, ""
	}

	commandLower := strings.ToLower(command)

	// Check against dangerous patterns
	for _, pattern := range sc.dangerousPatterns {
		if strings.Contains(commandLower, strings.ToLower(pattern)) {
			return true, "command matches dangerous pattern: " + pattern
		}
	}

	// Check for system directory access
	systemPaths := []string{
		"c:\\windows",
		"c:\\program files",
		"/usr/bin",
		"/usr/sbin",
		"/etc/",
		"/var/",
		"/system",
		"/boot",
	}

	for _, path := range systemPaths {
		// Check if command tries to modify system paths
		if (strings.Contains(commandLower, "rm ") ||
			strings.Contains(commandLower, "del ") ||
			strings.Contains(commandLower, "rmdir ") ||
			strings.Contains(commandLower, "format ")) &&
			strings.Contains(commandLower, path) {
			return true, "attempting to modify system path: " + path
		}
	}

	return false, ""
}

// isDangerousFileOperation checks if a file operation is dangerous
func (sc *SafetyChecker) isDangerousFileOperation(args string) (bool, string) {
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(args), &argsMap); err != nil {
		return false, ""
	}

	filePath, ok := argsMap["file_path"].(string)
	if !ok {
		return false, ""
	}

	// Check for system file modification
	dangerousLocations := []string{
		"c:\\windows\\system32",
		"c:\\windows\\syswow64",
		"c:\\program files",
		"/usr/bin",
		"/usr/sbin",
		"/etc/passwd",
		"/etc/shadow",
		"/etc/sudoers",
		"/system",
		"/boot",
		"/dev/",
	}

	filePathLower := strings.ToLower(filePath)

	for _, location := range dangerousLocations {
		if strings.Contains(filePathLower, location) {
			return true, "attempting to modify system file: " + location
		}
	}

	return false, ""
}

// IsReadOnlyTool checks if a tool only performs read operations
func IsReadOnlyTool(toolName string) bool {
	readOnlyTools := []string{
		"read",
		"glob",
		"grep",
		"bash_output",
	}

	for _, tool := range readOnlyTools {
		if tool == toolName {
			return true
		}
	}

	return false
}
