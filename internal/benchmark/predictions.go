package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// PatchGenerator handles extraction and formatting of patches
type PatchGenerator struct {
	repoDir string
}

// NewPatchGenerator creates a new patch generator for a repository
func NewPatchGenerator(repoDir string) *PatchGenerator {
	return &PatchGenerator{
		repoDir: repoDir,
	}
}

// ExtractPatch generates a unified diff patch from current git state
func (pg *PatchGenerator) ExtractPatch() (string, error) {
	// Generate git diff for all changes (staged and unstaged)
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = pg.repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w\n%s", err, output)
	}

	patch := string(output)

	// Validate patch is not empty
	if strings.TrimSpace(patch) == "" {
		return "", fmt.Errorf("no changes detected in repository")
	}

	return patch, nil
}

// ExtractPatchFromCommit generates a patch from the latest commit
func (pg *PatchGenerator) ExtractPatchFromCommit() (string, error) {
	// Get diff of the last commit
	cmd := exec.Command("git", "show", "HEAD", "--format=", "--patch")
	cmd.Dir = pg.repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git show failed: %w\n%s", err, output)
	}

	return string(output), nil
}

// ValidatePatch checks if a patch is well-formed
func ValidatePatch(patch string) error {
	if strings.TrimSpace(patch) == "" {
		return fmt.Errorf("patch is empty")
	}

	// Basic validation: should contain diff markers
	if !strings.Contains(patch, "diff --git") {
		return fmt.Errorf("patch does not appear to be a valid git diff")
	}

	if !strings.Contains(patch, "@@") {
		return fmt.Errorf("patch does not contain hunk markers")
	}

	return nil
}

// PredictionFile represents a single prediction in SWE-bench format
type PredictionFile struct {
	InstanceID  string `json:"instance_id"`
	ModelPatch  string `json:"model_patch"`
	ModelNameOrPath string `json:"model_name_or_path"`
}

// WritePredictions writes predictions in the format expected by SWE-bench harness
func WritePredictions(outputPath string, results []TaskResult, modelName string) error {
	predictions := make([]PredictionFile, 0, len(results))

	for _, result := range results {
		if !result.Success || result.Patch == "" {
			// Skip failed tasks or tasks without patches
			continue
		}

		predictions = append(predictions, PredictionFile{
			InstanceID:      result.InstanceID,
			ModelPatch:      result.Patch,
			ModelNameOrPath: modelName,
		})
	}

	// Write as JSONL (one prediction per line)
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create predictions file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	for _, pred := range predictions {
		if err := encoder.Encode(pred); err != nil {
			return fmt.Errorf("failed to write prediction: %w", err)
		}
	}

	return nil
}

// LoadPredictions reads predictions from a JSONL file
func LoadPredictions(path string) ([]PredictionFile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open predictions file: %w", err)
	}
	defer file.Close()

	var predictions []PredictionFile
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var pred PredictionFile
		if err := decoder.Decode(&pred); err != nil {
			return nil, fmt.Errorf("failed to decode prediction: %w", err)
		}
		predictions = append(predictions, pred)
	}

	return predictions, nil
}

// GeneratePredictionsFile creates a predictions file from task results
func GeneratePredictionsFile(dataDir, runID string, results []TaskResult) error {
	predictionsDir := filepath.Join(dataDir, "predictions")
	if err := os.MkdirAll(predictionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create predictions directory: %w", err)
	}

	outputPath := filepath.Join(predictionsDir, runID+".jsonl")

	modelName := "gocode-agent-" + runID

	return WritePredictions(outputPath, results, modelName)
}

// encodeJSON is a helper to marshal JSON with indentation
func encodeJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}
