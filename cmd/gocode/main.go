package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jake/gocode/internal/agent"
	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/initialization"
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

	// Handle first-run initialization
	var projectAnalysis *initialization.ProjectAnalysis
	if shouldInit, analysis := handleInitialization(workingDir); shouldInit {
		projectAnalysis = analysis
	}

	// Create and run agent
	a, err := agent.New(cfg, projectAnalysis)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", theme.Error("Error creating agent: %v", err))
		os.Exit(1)
	}

	if err := a.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", theme.Error("Error running agent: %v", err))
		os.Exit(1)
	}
}

// handleInitialization checks if this is a first run and handles initialization
func handleInitialization(workingDir string) (bool, *initialization.ProjectAnalysis) {
	// Create detector
	detector, err := initialization.NewDetector(workingDir)
	if err != nil {
		// If we can't create detector, just continue without initialization
		return false, nil
	}

	// Check if this is first run
	if !detector.ShouldInitialize() {
		// Not first run, try to load cached analysis
		analyzer := initialization.NewAnalyzer(workingDir, detector)
		if analysis, err := analyzer.LoadCachedAnalysis(); err == nil {
			return true, analysis
		}
		return false, nil
	}

	// Prompt user for initialization
	projectName := filepath.Base(workingDir)
	if !initialization.DisplayInitPrompt(projectName) {
		// User declined, mark as skipped
		detector.MarkSkipped()
		initialization.DisplaySkipMessage()
		return false, nil
	}

	// User accepted, perform initialization
	initialization.DisplayInitProgress("Analyzing project structure...")

	analyzer := initialization.NewAnalyzer(workingDir, detector)
	analysis, err := analyzer.Analyze()
	if err != nil {
		initialization.DisplayInitError(err)
		detector.MarkSkipped()
		return false, nil
	}

	// Generate recommendations
	initialization.DisplayInitProgress("Generating recommendations...")
	featureDetector := initialization.NewFeatureDetector(analysis)
	recommendations := featureDetector.GenerateRecommendations()
	analysis.Recommendations = recommendations

	// Start background indexing (non-blocking)
	indexer := initialization.NewIndexer(workingDir, detector, analyzer)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		<-indexer.StartBackgroundIndexing(ctx)
	}()

	// Display summary
	initialization.DisplaySummary(analysis, recommendations)

	// Mark as initialized
	detector.MarkInitialized()

	// Create .gitignore recommendation
	gitignorePath := filepath.Join(workingDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); err == nil {
		// .gitignore exists, check if .gocode is already in it
		content, _ := os.ReadFile(gitignorePath)
		if !strings.Contains(string(content), ".gocode") {
			fmt.Println(theme.Dim("💡 Tip: Add .gocode/ to your .gitignore file"))
			fmt.Println()
		}
	}

	return true, analysis
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
