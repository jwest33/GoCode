package initialization

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	gocodeDir  = ".gocode"
	stateFile  = "state.json"
	indexFile  = "index.db"
	analysisFile = "analysis.json"
)

// State represents the initialization state for a project
type State struct {
	Initialized     bool      `json:"initialized"`
	SkipInit        bool      `json:"skip_init"`
	FirstRunTime    time.Time `json:"first_run_time"`
	LastAnalyzed    time.Time `json:"last_analyzed"`
	ProjectHash     string    `json:"project_hash"`
	AnalysisVersion int       `json:"analysis_version"`
}

// Detector handles first-run detection and state management
type Detector struct {
	workingDir string
	stateDir   string
	statePath  string
	state      *State
}

// NewDetector creates a new detector for the given working directory
func NewDetector(workingDir string) (*Detector, error) {
	stateDir := filepath.Join(workingDir, gocodeDir)
	statePath := filepath.Join(stateDir, stateFile)

	d := &Detector{
		workingDir: workingDir,
		stateDir:   stateDir,
		statePath:  statePath,
	}

	// Load existing state or create new
	if err := d.loadState(); err != nil {
		d.state = &State{
			Initialized:     false,
			SkipInit:        false,
			AnalysisVersion: 1,
		}
	}

	return d, nil
}

// IsFirstRun returns true if this is the first time gocode is run in this project
func (d *Detector) IsFirstRun() bool {
	return !d.state.Initialized && !d.state.SkipInit
}

// ShouldInitialize returns true if initialization should be offered
func (d *Detector) ShouldInitialize() bool {
	return d.IsFirstRun()
}

// MarkInitialized marks the project as initialized
func (d *Detector) MarkInitialized() error {
	d.state.Initialized = true
	d.state.FirstRunTime = time.Now()
	d.state.LastAnalyzed = time.Now()
	return d.saveState()
}

// MarkSkipped marks initialization as skipped by user
func (d *Detector) MarkSkipped() error {
	d.state.SkipInit = true
	d.state.FirstRunTime = time.Now()
	return d.saveState()
}

// UpdateLastAnalyzed updates the last analysis timestamp
func (d *Detector) UpdateLastAnalyzed() error {
	d.state.LastAnalyzed = time.Now()
	return d.saveState()
}

// GetStateDir returns the .gocode directory path
func (d *Detector) GetStateDir() string {
	return d.stateDir
}

// GetIndexPath returns the path to the index database
func (d *Detector) GetIndexPath() string {
	return filepath.Join(d.stateDir, indexFile)
}

// GetAnalysisPath returns the path to the cached analysis file
func (d *Detector) GetAnalysisPath() string {
	return filepath.Join(d.stateDir, analysisFile)
}

// EnsureStateDir creates the .gocode directory if it doesn't exist
func (d *Detector) EnsureStateDir() error {
	if err := os.MkdirAll(d.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	return nil
}

// loadState loads the state from disk
func (d *Detector) loadState() error {
	data, err := os.ReadFile(d.statePath)
	if err != nil {
		return err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	d.state = &state
	return nil
}

// saveState saves the current state to disk
func (d *Detector) saveState() error {
	// Ensure directory exists
	if err := d.EnsureStateDir(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(d.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(d.statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// GetState returns the current state
func (d *Detector) GetState() *State {
	return d.state
}
