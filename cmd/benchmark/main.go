package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jake/gocode/internal/benchmark"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "setup":
		setupCmd()
	case "run":
		runCmd()
	case "evaluate":
		evaluateCmd()
	case "report":
		reportCmd()
	case "version", "--version", "-v":
		fmt.Printf("gocode-benchmark v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("gocode-benchmark - Evaluate GoCode agent against SWE-bench Verified")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  benchmark <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  setup      Download SWE-bench Verified dataset")
	fmt.Println("  run        Execute agent against benchmark tasks")
	fmt.Println("  evaluate   Run official evaluation harness on predictions")
	fmt.Println("  report     Generate results report with leaderboard comparison")
	fmt.Println("  version    Print version information")
	fmt.Println("  help       Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  benchmark setup")
	fmt.Println("  benchmark run --limit 10 --run-id test-run")
	fmt.Println("  benchmark evaluate --run-id test-run")
	fmt.Println("  benchmark report --run-id test-run")
}

func setupCmd() {
	fs := flag.NewFlagSet("setup", flag.ExitOnError)
	dataDir := fs.String("data-dir", "benchmarks/swebench-verified", "Directory to store dataset")
	force := fs.Bool("force", false, "Force re-download if dataset exists")

	fs.Parse(os.Args[2:])

	fmt.Printf("Setting up SWE-bench Verified dataset in %s...\n", *dataDir)

	if err := runSetup(*dataDir, *force); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Setup complete!")
}

func runCmd() {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	dataDir := fs.String("data-dir", "benchmarks/swebench-verified", "Directory containing dataset")
	runID := fs.String("run-id", "", "Unique identifier for this run (required)")
	limit := fs.Int("limit", 0, "Limit number of tasks to run (0 = all)")
	filter := fs.String("filter", "", "Filter tasks by instance ID pattern")
	timeout := fs.Int("timeout", 600, "Timeout per task in seconds")
	tokenBudget := fs.Int("token-budget", 100000, "Token budget per task")
	workers := fs.Int("workers", 1, "Number of parallel workers")

	fs.Parse(os.Args[2:])

	if *runID == "" {
		fmt.Fprintf(os.Stderr, "Error: --run-id is required\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("Running benchmark (run-id: %s)...\n", *runID)

	if err := runBenchmark(*dataDir, *runID, *limit, *filter, *timeout, *tokenBudget, *workers); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Benchmark run complete!")
}

func evaluateCmd() {
	fs := flag.NewFlagSet("evaluate", flag.ExitOnError)
	dataDir := fs.String("data-dir", "benchmarks/swebench-verified", "Directory containing dataset")
	runID := fs.String("run-id", "", "Run ID to evaluate (required)")
	maxWorkers := fs.Int("max-workers", 4, "Maximum workers for evaluation harness")

	fs.Parse(os.Args[2:])

	if *runID == "" {
		fmt.Fprintf(os.Stderr, "Error: --run-id is required\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("Evaluating predictions for run-id: %s...\n", *runID)

	if err := runEvaluation(*dataDir, *runID, *maxWorkers); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Evaluation complete!")
}

func reportCmd() {
	fs := flag.NewFlagSet("report", flag.ExitOnError)
	dataDir := fs.String("data-dir", "benchmarks/swebench-verified", "Directory containing dataset")
	runID := fs.String("run-id", "", "Run ID to report on (required)")
	format := fs.String("format", "text", "Output format: text, json, markdown")

	fs.Parse(os.Args[2:])

	if *runID == "" {
		fmt.Fprintf(os.Stderr, "Error: --run-id is required\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	fmt.Printf("Generating report for run-id: %s...\n", *runID)

	if err := generateReport(*dataDir, *runID, *format); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Implementation functions

func runSetup(dataDir string, force bool) error {
	return benchmark.DownloadDataset(dataDir, force)
}

func runBenchmark(dataDir, runID string, limit int, filter string, timeout, tokenBudget, workers int) error {
	// Load dataset
	dataset, err := benchmark.LoadDataset(dataDir)
	if err != nil {
		return fmt.Errorf("failed to load dataset: %w", err)
	}

	// Filter tasks
	tasks := dataset.Filter(filter)

	// Apply limit
	if limit > 0 && limit < len(tasks) {
		tasks = tasks[:limit]
	}

	fmt.Printf("Running %d tasks...\n", len(tasks))

	// Create runner
	runner, err := benchmark.NewRunner(dataDir, runID, timeout, tokenBudget)
	if err != nil {
		return fmt.Errorf("failed to create runner: %w", err)
	}

	// Execute tasks
	// Note: workers > 1 would require parallel execution (future enhancement)
	if workers > 1 {
		fmt.Printf("Note: Parallel execution not yet implemented, running sequentially...\n")
	}

	results := make([]benchmark.TaskResult, 0, len(tasks))

	for i, task := range tasks {
		fmt.Printf("[%d/%d] Running task: %s\n", i+1, len(tasks), task.InstanceID)

		result, err := runner.RunTask(task)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Task execution failed: %v\n", err)
			continue
		}

		results = append(results, *result)

		if result.Success {
			fmt.Printf("  ✓ Success (%s, %d tokens)\n", result.ExecutionTime, result.TokensUsed)
		} else {
			fmt.Printf("  ✗ Failed: %s\n", result.Error)
		}
	}

	// Save results
	if err := runner.SaveResults(results); err != nil {
		return fmt.Errorf("failed to save results: %w", err)
	}

	// Generate predictions file for evaluation
	if err := benchmark.GeneratePredictionsFile(dataDir, runID, results); err != nil {
		return fmt.Errorf("failed to generate predictions: %w", err)
	}

	fmt.Printf("\nCompleted %d/%d tasks successfully\n", len(results), len(tasks))
	fmt.Printf("Results saved to: %s/results/%s_results.json\n", dataDir, runID)
	fmt.Printf("Predictions saved to: %s/predictions/%s.jsonl\n", dataDir, runID)

	return nil
}

func runEvaluation(dataDir, runID string, maxWorkers int) error {
	evaluator, err := benchmark.NewEvaluator(dataDir)
	if err != nil {
		return fmt.Errorf("failed to create evaluator: %w", err)
	}

	// Check if harness is installed
	if err := evaluator.CheckHarnessInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		fmt.Println("\nGenerating evaluation script instead...")
		return evaluator.GenerateEvaluationScript(runID, maxWorkers)
	}

	// Run evaluation
	_, err = evaluator.RunEvaluation(runID, maxWorkers)
	return err
}

func generateReport(dataDir, runID, format string) error {
	reporter := benchmark.NewReporter(dataDir)
	_, err := reporter.GenerateReport(runID, format)
	return err
}
