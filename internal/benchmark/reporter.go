package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Reporter generates benchmark reports
type Reporter struct {
	dataDir string
}

// NewReporter creates a new reporter instance
func NewReporter(dataDir string) *Reporter {
	return &Reporter{
		dataDir: dataDir,
	}
}

// GenerateReport creates a comprehensive report for a benchmark run
func (r *Reporter) GenerateReport(runID string, format string) (*BenchmarkReport, error) {
	// Load task results
	resultsFile := filepath.Join(r.dataDir, "results", runID+"_results.json")
	var taskResults []TaskResult

	if _, err := os.Stat(resultsFile); err == nil {
		// Results file exists, load it
		data, err := os.ReadFile(resultsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read results: %w", err)
		}

		if err := decodeJSON(data, &taskResults); err != nil {
			return nil, fmt.Errorf("failed to parse results: %w", err)
		}
	} else {
		return nil, fmt.Errorf("results file not found: %s", resultsFile)
	}

	// Load evaluation results if available
	evalFile := filepath.Join(r.dataDir, "evaluation_results", runID, "results.json")
	var evalResults []EvaluationResult

	if _, err := os.Stat(evalFile); err == nil {
		evaluator, _ := NewEvaluator(r.dataDir)
		evalResults, _ = evaluator.ParseResults(runID)
	}

	// Build report
	report := r.buildReport(runID, taskResults, evalResults)

	// Output in requested format
	switch format {
	case "json":
		return report, r.outputJSON(report)
	case "markdown":
		return report, r.outputMarkdown(report)
	default:
		return report, r.outputText(report)
	}
}

// buildReport aggregates statistics and creates the report structure
func (r *Reporter) buildReport(runID string, taskResults []TaskResult, evalResults []EvaluationResult) *BenchmarkReport {
	report := &BenchmarkReport{
		RunID:       runID,
		Timestamp:   time.Now(),
		TaskResults: taskResults,
		EvalResults: evalResults,
	}

	// Calculate statistics from task results
	report.TotalTasks = len(taskResults)

	var totalTime time.Duration
	var totalTokens int
	completed := 0
	failed := 0

	for _, result := range taskResults {
		if result.Success {
			completed++
		} else {
			failed++
		}
		totalTime += result.ExecutionTime
		totalTokens += result.TokensUsed
	}

	report.Completed = completed
	report.Failed = failed

	if completed > 0 {
		report.AvgTime = totalTime / time.Duration(completed)
		report.AvgTokens = totalTokens / completed
	}

	// Calculate resolve rate from evaluation results
	if len(evalResults) > 0 {
		resolved := 0
		for _, eval := range evalResults {
			if eval.Resolved {
				resolved++
			}
		}
		report.Resolved = resolved
		report.ResolveRate = float64(resolved) / float64(len(evalResults)) * 100.0
	}

	// Build leaderboard comparison
	report.Leaderboard = r.buildLeaderboardComparison(report.ResolveRate)

	return report
}

// buildLeaderboardComparison compares results to known baselines
func (r *Reporter) buildLeaderboardComparison(score float64) LeaderboardComparison {
	baselines := GetBaselines()

	// Sort baselines by score
	sort.Slice(baselines, func(i, j int) bool {
		return baselines[i].Score > baselines[j].Score
	})

	// Determine rank
	rank := 1
	for _, baseline := range baselines {
		if score < baseline.Score {
			rank++
		}
	}

	return LeaderboardComparison{
		GoCodeScore: score,
		Baselines:   baselines,
		Rank:        rank,
	}
}

// outputText generates a human-readable text report
func (r *Reporter) outputText(report *BenchmarkReport) error {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  SWE-bench Verified Results Report")
	fmt.Println("========================================")
	fmt.Println()

	fmt.Printf("Run ID:        %s\n", report.RunID)
	fmt.Printf("Timestamp:     %s\n", report.Timestamp.Format(time.RFC3339))
	fmt.Println()

	fmt.Println("--- Execution Statistics ---")
	fmt.Printf("Total Tasks:   %d\n", report.TotalTasks)
	fmt.Printf("Completed:     %d\n", report.Completed)
	fmt.Printf("Failed:        %d\n", report.Failed)
	fmt.Printf("Avg Time:      %s\n", report.AvgTime.Round(time.Second))
	fmt.Printf("Avg Tokens:    %d\n", report.AvgTokens)
	fmt.Println()

	if len(report.EvalResults) > 0 {
		fmt.Println("--- Evaluation Results ---")
		fmt.Printf("Resolved:      %d / %d\n", report.Resolved, len(report.EvalResults))
		fmt.Printf("Resolve Rate:  %.2f%%\n", report.ResolveRate)
		fmt.Println()

		fmt.Println("--- Leaderboard Comparison ---")
		fmt.Printf("GoCode Score:  %.2f%%\n", report.Leaderboard.GoCodeScore)
		fmt.Printf("Est. Rank:     #%d\n", report.Leaderboard.Rank)
		fmt.Println()

		fmt.Println("Baseline Comparison:")
		for i, baseline := range report.Leaderboard.Baselines {
			marker := "  "
			if report.Leaderboard.GoCodeScore > baseline.Score {
				marker = "✓ "
			}
			fmt.Printf("%s%d. %s: %.2f%%\n", marker, i+1, baseline.Model, baseline.Score)
		}
		fmt.Println()
	}

	// Summary of failures
	if report.Failed > 0 {
		fmt.Println("--- Failed Tasks ---")
		failCount := 0
		for _, result := range report.TaskResults {
			if !result.Success && failCount < 10 {
				fmt.Printf("- %s: %s\n", result.InstanceID, truncate(result.Error, 60))
				failCount++
			}
		}
		if report.Failed > 10 {
			fmt.Printf("... and %d more\n", report.Failed-10)
		}
		fmt.Println()
	}

	return nil
}

// outputJSON outputs the report in JSON format
func (r *Reporter) outputJSON(report *BenchmarkReport) error {
	data, err := encodeJSON(report)
	if err != nil {
		return fmt.Errorf("failed to encode json: %w", err)
	}

	fmt.Println(string(data))
	return nil
}

// outputMarkdown generates a markdown report
func (r *Reporter) outputMarkdown(report *BenchmarkReport) error {
	var sb strings.Builder

	sb.WriteString("# SWE-bench Verified Results Report\n\n")
	sb.WriteString(fmt.Sprintf("**Run ID:** %s  \n", report.RunID))
	sb.WriteString(fmt.Sprintf("**Timestamp:** %s  \n\n", report.Timestamp.Format(time.RFC3339)))

	sb.WriteString("## Execution Statistics\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|--------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Total Tasks | %d |\n", report.TotalTasks))
	sb.WriteString(fmt.Sprintf("| Completed | %d |\n", report.Completed))
	sb.WriteString(fmt.Sprintf("| Failed | %d |\n", report.Failed))
	sb.WriteString(fmt.Sprintf("| Avg Time | %s |\n", report.AvgTime.Round(time.Second)))
	sb.WriteString(fmt.Sprintf("| Avg Tokens | %d |\n\n", report.AvgTokens))

	if len(report.EvalResults) > 0 {
		sb.WriteString("## Evaluation Results\n\n")
		sb.WriteString(fmt.Sprintf("**Resolved:** %d / %d  \n", report.Resolved, len(report.EvalResults)))
		sb.WriteString(fmt.Sprintf("**Resolve Rate:** %.2f%%  \n\n", report.ResolveRate))

		sb.WriteString("## Leaderboard Comparison\n\n")
		sb.WriteString("| Rank | Model | Score |\n")
		sb.WriteString("|------|-------|-------|\n")

		// Insert GoCode in the right position
		rank := 1
		inserted := false
		for _, baseline := range report.Leaderboard.Baselines {
			if !inserted && report.Leaderboard.GoCodeScore >= baseline.Score {
				sb.WriteString(fmt.Sprintf("| %d | **GoCode** | **%.2f%%** |\n", rank, report.Leaderboard.GoCodeScore))
				inserted = true
				rank++
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | %.2f%% |\n", rank, baseline.Model, baseline.Score))
			rank++
		}
		if !inserted {
			sb.WriteString(fmt.Sprintf("| %d | **GoCode** | **%.2f%%** |\n", rank, report.Leaderboard.GoCodeScore))
		}
		sb.WriteString("\n")
	}

	fmt.Println(sb.String())
	return nil
}

// Helper functions
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func decodeJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
