package telemetry

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// ArtifactStore manages artifacts linked to traces
type ArtifactStore struct {
	db *sql.DB
}

// ArtifactType represents the type of artifact
type ArtifactType string

const (
	ArtifactTypeDiff       ArtifactType = "diff"
	ArtifactTypeLog        ArtifactType = "log"
	ArtifactTypeTestResult ArtifactType = "test_result"
	ArtifactTypePatch      ArtifactType = "patch"
	ArtifactTypeOutput     ArtifactType = "output"
	ArtifactTypeError      ArtifactType = "error"
)

// Artifact represents a stored artifact
type Artifact struct {
	ID          string
	TraceID     string
	SpanID      string
	Type        ArtifactType
	Name        string
	Content     string
	Metadata    map[string]interface{}
	CreatedAt   time.Time
}

// NewArtifactStore creates a new artifact store
func NewArtifactStore(dbPath string) (*ArtifactStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &ArtifactStore{db: db}

	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

// initSchema creates the database schema
func (as *ArtifactStore) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS artifacts (
		id TEXT PRIMARY KEY,
		trace_id TEXT NOT NULL,
		span_id TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		content TEXT NOT NULL,
		metadata TEXT,
		created_at DATETIME NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_artifact_trace ON artifacts(trace_id);
	CREATE INDEX IF NOT EXISTS idx_artifact_span ON artifacts(span_id);
	CREATE INDEX IF NOT EXISTS idx_artifact_type ON artifacts(type);
	CREATE INDEX IF NOT EXISTS idx_artifact_created ON artifacts(created_at DESC);

	CREATE VIRTUAL TABLE IF NOT EXISTS artifacts_fts USING fts5(
		id UNINDEXED,
		name,
		content,
		content='artifacts',
		content_rowid='rowid'
	);
	`

	_, err := as.db.Exec(schema)
	return err
}

// Store stores an artifact
func (as *ArtifactStore) Store(artifact *Artifact) error {
	if artifact.ID == "" {
		artifact.ID = generateArtifactID()
	}
	if artifact.CreatedAt.IsZero() {
		artifact.CreatedAt = time.Now()
	}

	// Serialize metadata
	metadataJSON := "{}"
	if len(artifact.Metadata) > 0 {
		data, err := json.Marshal(artifact.Metadata)
		if err == nil {
			metadataJSON = string(data)
		}
	}

	_, err := as.db.Exec(`
		INSERT INTO artifacts (id, trace_id, span_id, type, name, content, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, artifact.ID, artifact.TraceID, artifact.SpanID, artifact.Type, artifact.Name, artifact.Content, metadataJSON, artifact.CreatedAt)

	return err
}

// Get retrieves an artifact by ID
func (as *ArtifactStore) Get(id string) (*Artifact, error) {
	var artifact Artifact
	var metadataJSON string

	err := as.db.QueryRow(`
		SELECT id, trace_id, span_id, type, name, content, metadata, created_at
		FROM artifacts WHERE id = ?
	`, id).Scan(&artifact.ID, &artifact.TraceID, &artifact.SpanID, &artifact.Type, &artifact.Name, &artifact.Content, &metadataJSON, &artifact.CreatedAt)

	if err != nil {
		return nil, err
	}

	if metadataJSON != "" && metadataJSON != "{}" {
		json.Unmarshal([]byte(metadataJSON), &artifact.Metadata)
	}

	return &artifact, nil
}

// GetByTrace retrieves all artifacts for a trace
func (as *ArtifactStore) GetByTrace(traceID string) ([]*Artifact, error) {
	rows, err := as.db.Query(`
		SELECT id, trace_id, span_id, type, name, content, metadata, created_at
		FROM artifacts
		WHERE trace_id = ?
		ORDER BY created_at
	`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return as.scanArtifacts(rows)
}

// GetBySpan retrieves all artifacts for a span
func (as *ArtifactStore) GetBySpan(spanID string) ([]*Artifact, error) {
	rows, err := as.db.Query(`
		SELECT id, trace_id, span_id, type, name, content, metadata, created_at
		FROM artifacts
		WHERE span_id = ?
		ORDER BY created_at
	`, spanID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return as.scanArtifacts(rows)
}

// Search searches artifacts by content
func (as *ArtifactStore) Search(query string, limit int) ([]*Artifact, error) {
	rows, err := as.db.Query(`
		SELECT a.id, a.trace_id, a.span_id, a.type, a.name, a.content, a.metadata, a.created_at
		FROM artifacts a
		INNER JOIN artifacts_fts fts ON a.id = fts.id
		WHERE artifacts_fts MATCH ?
		ORDER BY a.created_at DESC
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return as.scanArtifacts(rows)
}

// scanArtifacts scans artifacts from query results
func (as *ArtifactStore) scanArtifacts(rows *sql.Rows) ([]*Artifact, error) {
	artifacts := []*Artifact{}

	for rows.Next() {
		var artifact Artifact
		var metadataJSON string

		err := rows.Scan(&artifact.ID, &artifact.TraceID, &artifact.SpanID, &artifact.Type, &artifact.Name, &artifact.Content, &metadataJSON, &artifact.CreatedAt)
		if err != nil {
			return nil, err
		}

		if metadataJSON != "" && metadataJSON != "{}" {
			json.Unmarshal([]byte(metadataJSON), &artifact.Metadata)
		}

		artifacts = append(artifacts, &artifact)
	}

	return artifacts, rows.Err()
}

// Close closes the database connection
func (as *ArtifactStore) Close() error {
	return as.db.Close()
}

// Helper functions

func generateArtifactID() string {
	return fmt.Sprintf("art_%d", time.Now().UnixNano())
}

// StoreDiff stores a file diff as an artifact
func (as *ArtifactStore) StoreDiff(traceID, spanID, filePath, oldContent, newContent string) (string, error) {
	diff := generateDiff(oldContent, newContent)

	artifact := &Artifact{
		TraceID: traceID,
		SpanID:  spanID,
		Type:    ArtifactTypeDiff,
		Name:    filePath,
		Content: diff,
		Metadata: map[string]interface{}{
			"file_path":    filePath,
			"old_size":     len(oldContent),
			"new_size":     len(newContent),
		},
	}

	if err := as.Store(artifact); err != nil {
		return "", err
	}

	return artifact.ID, nil
}

// StoreCommandOutput stores command output as an artifact
func (as *ArtifactStore) StoreCommandOutput(traceID, spanID, command, stdout, stderr string, exitCode int) (string, error) {
	artifact := &Artifact{
		TraceID: traceID,
		SpanID:  spanID,
		Type:    ArtifactTypeOutput,
		Name:    command,
		Content: fmt.Sprintf("STDOUT:\n%s\n\nSTDERR:\n%s", stdout, stderr),
		Metadata: map[string]interface{}{
			"command":   command,
			"exit_code": exitCode,
		},
	}

	if err := as.Store(artifact); err != nil {
		return "", err
	}

	return artifact.ID, nil
}

// generateDiff generates a simple diff between two strings
func generateDiff(old, new string) string {
	// Simple diff - just show old and new
	// Could be enhanced with proper diff algorithm
	return fmt.Sprintf("--- OLD ---\n%s\n\n+++ NEW +++\n%s", old, new)
}
