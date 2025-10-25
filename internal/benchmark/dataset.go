package benchmark

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	// HuggingFace dataset URL for SWE-bench Verified
	swebenchVerifiedURL = "https://huggingface.co/datasets/princeton-nlp/SWE-bench_Verified/resolve/main/data/test-00000-of-00001.parquet"

	// Fallback: JSON version (if available)
	swebenchVerifiedJSONURL = "https://huggingface.co/datasets/princeton-nlp/SWE-bench_Verified/raw/main/data.json"

	// Alternative: Use the datasets API endpoint
	swebenchAPIURL = "https://datasets-server.huggingface.co/rows?dataset=princeton-nlp/SWE-bench_Verified&config=default&split=test"
)

// Dataset manages SWE-bench task data
type Dataset struct {
	Tasks   []Task
	dataDir string
}

// LoadDataset loads the SWE-bench Verified dataset from local cache or downloads it
func LoadDataset(dataDir string) (*Dataset, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	datasetPath := filepath.Join(dataDir, "dataset.json")

	// Check if dataset exists locally
	if _, err := os.Stat(datasetPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("dataset not found at %s, run 'benchmark setup' first", datasetPath)
	}

	// Load from local file
	data, err := os.ReadFile(datasetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dataset: %w", err)
	}

	var tasks []Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse dataset: %w", err)
	}

	return &Dataset{
		Tasks:   tasks,
		dataDir: dataDir,
	}, nil
}

// DownloadDataset downloads the SWE-bench Verified dataset from HuggingFace
func DownloadDataset(dataDir string, force bool) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	datasetPath := filepath.Join(dataDir, "dataset.json")

	// Check if already exists
	if !force {
		if _, err := os.Stat(datasetPath); err == nil {
			return fmt.Errorf("dataset already exists at %s (use --force to re-download)", datasetPath)
		}
	}

	fmt.Println("Downloading SWE-bench Verified dataset from HuggingFace...")
	fmt.Println("This may take a few minutes...")

	// Try the HuggingFace datasets API first (returns JSON directly)
	tasks, err := downloadFromAPI()
	if err != nil {
		fmt.Printf("API download failed (%v), trying alternative method...\n", err)

		// Fallback: try direct JSON URL
		tasks, err = downloadFromJSON()
		if err != nil {
			return fmt.Errorf("all download methods failed: %w", err)
		}
	}

	// Save to local file
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal dataset: %w", err)
	}

	if err := os.WriteFile(datasetPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write dataset: %w", err)
	}

	fmt.Printf("Successfully downloaded %d tasks to %s\n", len(tasks), datasetPath)
	return nil
}

// downloadFromAPI fetches the dataset using HuggingFace datasets API
func downloadFromAPI() ([]Task, error) {
	const (
		baseURL   = "https://datasets-server.huggingface.co/rows"
		dataset   = "princeton-nlp/SWE-bench_Verified"
		config    = "default"
		split     = "test"
		batchSize = 100
		totalRows = 500 // SWE-bench Verified has 500 tasks
	)

	var allTasks []Task

	// Fetch in batches to get all 500 rows
	for offset := 0; offset < totalRows; offset += batchSize {
		url := fmt.Sprintf("%s?dataset=%s&config=%s&split=%s&offset=%d&length=%d",
			baseURL, dataset, config, split, offset, batchSize)

		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("http request failed at offset %d: %w", offset, err)
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("http error at offset %d: %s", offset, resp.Status)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read response at offset %d: %w", offset, err)
		}

		// Parse the API response (format: {"rows": [{"row": {...}}, ...]})
		var apiResp struct {
			Rows []struct {
				Row Task `json:"row"`
			} `json:"rows"`
		}

		if err := json.Unmarshal(body, &apiResp); err != nil {
			return nil, fmt.Errorf("failed to parse api response at offset %d: %w", offset, err)
		}

		// Append tasks from this batch
		for _, row := range apiResp.Rows {
			allTasks = append(allTasks, row.Row)
		}

		fmt.Printf("Downloaded %d/%d tasks...\n", len(allTasks), totalRows)
	}

	if len(allTasks) == 0 {
		return nil, fmt.Errorf("no tasks found in api response")
	}

	return allTasks, nil
}

// downloadFromJSON tries to download from a direct JSON URL
func downloadFromJSON() ([]Task, error) {
	resp, err := http.Get(swebenchVerifiedJSONURL)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var tasks []Task
	if err := json.Unmarshal(body, &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse json: %w", err)
	}

	return tasks, nil
}

// Filter returns tasks matching the given criteria
func (d *Dataset) Filter(instanceIDPattern string) []Task {
	if instanceIDPattern == "" {
		return d.Tasks
	}

	var filtered []Task
	pattern := strings.ToLower(instanceIDPattern)

	for _, task := range d.Tasks {
		if strings.Contains(strings.ToLower(task.InstanceID), pattern) {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// GetTask retrieves a specific task by instance ID
func (d *Dataset) GetTask(instanceID string) (*Task, error) {
	for i := range d.Tasks {
		if d.Tasks[i].InstanceID == instanceID {
			return &d.Tasks[i], nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", instanceID)
}

// Stats returns dataset statistics
func (d *Dataset) Stats() map[string]interface{} {
	repos := make(map[string]int)
	for _, task := range d.Tasks {
		repos[task.Repo]++
	}

	return map[string]interface{}{
		"total_tasks":      len(d.Tasks),
		"unique_repos":     len(repos),
		"repos":            repos,
	}
}
