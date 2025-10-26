package lsp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// DiscoverAndAddLSPPaths finds common LSP binary locations and adds them to PATH
func DiscoverAndAddLSPPaths() {
	var pathsToAdd []string

	if runtime.GOOS == "windows" {
		pathsToAdd = discoverWindowsLSPPaths()
	} else {
		pathsToAdd = discoverUnixLSPPaths()
	}

	if len(pathsToAdd) == 0 {
		return
	}

	// Get current PATH
	currentPath := os.Getenv("PATH")

	// Add new paths
	var newPaths []string
	for _, p := range pathsToAdd {
		// Check if path already in PATH
		if !strings.Contains(currentPath, p) {
			newPaths = append(newPaths, p)
		}
	}

	if len(newPaths) == 0 {
		return
	}

	// Update PATH
	separator := string(os.PathListSeparator)
	updatedPath := strings.Join(append(newPaths, currentPath), separator)
	os.Setenv("PATH", updatedPath)
}

// discoverWindowsLSPPaths finds LSP binaries on Windows
func discoverWindowsLSPPaths() []string {
	var paths []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return paths
	}

	appData := os.Getenv("APPDATA")
	localAppData := os.Getenv("LOCALAPPDATA")

	// Python LSP paths (pylsp)
	pythonPaths := []string{
		// User installation
		filepath.Join(appData, "Python", "Python311", "Scripts"),
		filepath.Join(appData, "Python", "Python310", "Scripts"),
		filepath.Join(appData, "Python", "Python39", "Scripts"),
		filepath.Join(localAppData, "Programs", "Python", "Python311", "Scripts"),
		filepath.Join(localAppData, "Programs", "Python", "Python310", "Scripts"),
		filepath.Join(localAppData, "Programs", "Python", "Python39", "Scripts"),
		// Global installation
		"C:\\Python311\\Scripts",
		"C:\\Python310\\Scripts",
		"C:\\Python39\\Scripts",
	}

	for _, p := range pythonPaths {
		if dirExists(p) {
			paths = append(paths, p)
		}
	}

	// Go LSP path (gopls)
	goBin := filepath.Join(homeDir, "go", "bin")
	if dirExists(goBin) {
		paths = append(paths, goBin)
	}

	// Node.js LSP path (typescript-language-server, etc.)
	npmPath := filepath.Join(appData, "npm")
	if dirExists(npmPath) {
		paths = append(paths, npmPath)
	}

	// Rust LSP path (rust-analyzer via rustup)
	cargoBin := filepath.Join(homeDir, ".cargo", "bin")
	if dirExists(cargoBin) {
		paths = append(paths, cargoBin)
	}

	return paths
}

// discoverUnixLSPPaths finds LSP binaries on Unix-like systems
func discoverUnixLSPPaths() []string {
	var paths []string

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return paths
	}

	// Python LSP (pip install --user)
	pythonUserBin := filepath.Join(homeDir, ".local", "bin")
	if dirExists(pythonUserBin) {
		paths = append(paths, pythonUserBin)
	}

	// Go LSP
	goBin := filepath.Join(homeDir, "go", "bin")
	if dirExists(goBin) {
		paths = append(paths, goBin)
	}

	// Node.js global npm
	npmBin := filepath.Join(homeDir, ".npm-global", "bin")
	if dirExists(npmBin) {
		paths = append(paths, npmBin)
	}

	// Rust LSP
	cargoBin := filepath.Join(homeDir, ".cargo", "bin")
	if dirExists(cargoBin) {
		paths = append(paths, cargoBin)
	}

	return paths
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
