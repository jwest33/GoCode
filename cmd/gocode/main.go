package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jake/gocode/internal/agent"
	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/theme"
)

const version = "1.0.0"

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to config.yaml (default: auto-search)")
	showVersion := flag.Bool("version", false, "Show version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println(theme.HeaderBold("GoCode v%s", version))
		os.Exit(0)
	}

	// Find configuration file
	configFile := *configPath
	if configFile == "" {
		var err error
		configFile, err = findConfig()
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", theme.Error("Error finding config: %v", err))
			fmt.Fprintf(os.Stderr, "%s\n", theme.Dim("Searched in:"))
			fmt.Fprintf(os.Stderr, "  %s\n", theme.Dim("- Current directory: ./config.yaml"))
			fmt.Fprintf(os.Stderr, "  %s\n", theme.Dim("- Executable directory: <exe-dir>/config.yaml"))
			fmt.Fprintf(os.Stderr, "  %s\n", theme.Dim("- Home directory: ~/.gocode/config.yaml"))
			fmt.Fprintf(os.Stderr, "\n%s\n", theme.Warning("Use --config flag to specify a custom path"))
			os.Exit(1)
		}
	}

	fmt.Printf("%s %s\n", theme.Dim("Using config:"), theme.Agent(configFile))

	// Load configuration
	cfg, err := config.Load(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", theme.Error("Error loading config: %v", err))
		os.Exit(1)
	}

	// Set base directory for logs (directory containing config)
	cfg.BaseDir = filepath.Dir(configFile)

	// Set working directory for TODO.md (current directory)
	workingDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", theme.Error("Error getting working directory: %v", err))
		os.Exit(1)
	}
	cfg.WorkingDir = workingDir

	// Create and run agent
	a, err := agent.New(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", theme.Error("Error creating agent: %v", err))
		os.Exit(1)
	}

	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", theme.Error("Error running agent: %v", err))
		os.Exit(1)
	}
}

// findConfig searches for config.yaml in multiple locations
func findConfig() (string, error) {
	// 1. Check environment variable
	if envPath := os.Getenv("GOCODE_CONFIG"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// 2. Check current working directory
	if _, err := os.Stat("config.yaml"); err == nil {
		absPath, _ := filepath.Abs("config.yaml")
		return absPath, nil
	}

	// 3. Check executable's directory
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		configPath := filepath.Join(exeDir, "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	// 4. Check home directory
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".gocode", "config.yaml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
	}

	return "", fmt.Errorf("config.yaml not found in any search location")
}
