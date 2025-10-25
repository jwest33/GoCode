package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Evaluator manages integration with the SWE-bench evaluation harness
type Evaluator struct {
	dataDir    string
	harnessDir string
}

// NewEvaluator creates a new evaluator instance
func NewEvaluator(dataDir string) (*Evaluator, error) {
	harnessDir := filepath.Join(dataDir, "swebench-harness")

	return &Evaluator{
		dataDir:    dataDir,
		harnessDir: harnessDir,
	}, nil
}

// CheckHarnessInstalled verifies that the SWE-bench harness is set up
func (e *Evaluator) CheckHarnessInstalled() error {
	// Check if harness directory exists
	if _, err := os.Stat(e.harnessDir); os.IsNotExist(err) {
		return fmt.Errorf("swe-bench harness not found, run setup script first")
	}

	// Check if Python is available
	if _, err := exec.LookPath("python"); err != nil {
		if _, err := exec.LookPath("python3"); err != nil {
			return fmt.Errorf("python not found, please install python 3.8+")
		}
	}

	return nil
}

// RunEvaluation executes the SWE-bench evaluation harness on predictions
func (e *Evaluator) RunEvaluation(runID string, maxWorkers int) ([]EvaluationResult, error) {
	if err := e.CheckHarnessInstalled(); err != nil {
		return nil, err
	}

	predictionsFile := filepath.Join(e.dataDir, "predictions", runID+".jsonl")
	if _, err := os.Stat(predictionsFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("predictions file not found: %s", predictionsFile)
	}

	// Prepare output directory
	outputDir := filepath.Join(e.dataDir, "evaluation_results", runID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build evaluation command
	pythonCmd := "python"
	if _, err := exec.LookPath("python3"); err == nil {
		pythonCmd = "python3"
	}

	args := []string{
		"-m", "swebench.harness.run_evaluation",
		"--dataset_name", "princeton-nlp/SWE-bench_Verified",
		"--predictions_path", predictionsFile,
		"--max_workers", fmt.Sprintf("%d", maxWorkers),
		"--run_id", runID,
		"--output_dir", outputDir,
	}

	cmd := exec.Command(pythonCmd, args...)
	cmd.Dir = e.harnessDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Running SWE-bench evaluation harness...")
	fmt.Printf("Command: %s %s\n", pythonCmd, strings.Join(args, " "))
	fmt.Println("This may take several hours depending on the number of predictions...")

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("evaluation harness failed: %w", err)
	}

	// Parse results
	return e.ParseResults(runID)
}

// ParseResults reads and parses evaluation results
func (e *Evaluator) ParseResults(runID string) ([]EvaluationResult, error) {
	resultsFile := filepath.Join(e.dataDir, "evaluation_results", runID, "results.json")

	data, err := os.ReadFile(resultsFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read results file: %w", err)
	}

	// The SWE-bench harness outputs results in various formats
	// This is a simplified parser - adjust based on actual format
	var rawResults map[string]interface{}
	if err := json.Unmarshal(data, &rawResults); err != nil {
		return nil, fmt.Errorf("failed to parse results: %w", err)
	}

	// Convert to EvaluationResult format
	var results []EvaluationResult

	// Parse based on the actual SWE-bench output format
	// This is a placeholder - you'll need to adjust based on the real schema
	if instances, ok := rawResults["results"].([]interface{}); ok {
		for _, inst := range instances {
			if instMap, ok := inst.(map[string]interface{}); ok {
				result := EvaluationResult{
					InstanceID: getString(instMap, "instance_id"),
					Resolved:   getBool(instMap, "resolved"),
				}
				results = append(results, result)
			}
		}
	}

	return results, nil
}

// GenerateEvaluationScript creates a shell script to run evaluation
func (e *Evaluator) GenerateEvaluationScript(runID string, maxWorkers int) error {
	scriptPath := filepath.Join(e.dataDir, fmt.Sprintf("run_evaluation_%s.sh", runID))

	predictionsFile := filepath.Join(e.dataDir, "predictions", runID+".jsonl")
	outputDir := filepath.Join(e.dataDir, "evaluation_results", runID)

	script := fmt.Sprintf(`#!/bin/bash
# SWE-bench Evaluation Script for run: %s
# Generated automatically

set -e

echo "Starting SWE-bench evaluation..."
echo "Run ID: %s"
echo "Predictions: %s"
echo "Output: %s"
echo ""

cd %s

python -m swebench.harness.run_evaluation \
  --dataset_name princeton-nlp/SWE-bench_Verified \
  --predictions_path %s \
  --max_workers %d \
  --run_id %s \
  --output_dir %s

echo ""
echo "Evaluation complete! Results saved to: %s"
`, runID, runID, predictionsFile, outputDir, e.harnessDir,
		predictionsFile, maxWorkers, runID, outputDir, outputDir)

	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write evaluation script: %w", err)
	}

	fmt.Printf("Evaluation script written to: %s\n", scriptPath)
	return nil
}

// Helper functions for parsing JSON
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0.0
}
