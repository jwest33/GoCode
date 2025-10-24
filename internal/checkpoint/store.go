package checkpoint

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/jake/gocode/internal/llm"
)

// Store manages persistent checkpoints using SQLite
type Store struct {
	db *sql.DB
}

// Checkpoint represents a saved conversation state
type Checkpoint struct {
	ID          string    `json:"id"`
	ThreadID    string    `json:"thread_id"`
	ParentID    string    `json:"parent_id,omitempty"` // For branching
	Timestamp   time.Time `json:"timestamp"`
	Messages    []llm.Message `json:"messages"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Description string    `json:"description,omitempty"`
}

// Thread represents a conversation thread
type Thread struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CurrentCheckpoint string            `json:"current_checkpoint"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewStore creates a new checkpoint store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}

	if err := store.initSchema(); err != nil {
		return nil, err
	}

	return store, nil
}

// initSchema creates the database schema
func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS threads (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		current_checkpoint TEXT,
		metadata TEXT,
		FOREIGN KEY (current_checkpoint) REFERENCES checkpoints(id)
	);

	CREATE TABLE IF NOT EXISTS checkpoints (
		id TEXT PRIMARY KEY,
		thread_id TEXT NOT NULL,
		parent_id TEXT,
		timestamp DATETIME NOT NULL,
		messages TEXT NOT NULL,
		metadata TEXT,
		description TEXT,
		FOREIGN KEY (thread_id) REFERENCES threads(id),
		FOREIGN KEY (parent_id) REFERENCES checkpoints(id)
	);

	CREATE INDEX IF NOT EXISTS idx_thread_checkpoints ON checkpoints(thread_id, timestamp);
	CREATE INDEX IF NOT EXISTS idx_checkpoint_parent ON checkpoints(parent_id);
	`

	_, err := s.db.Exec(schema)
	return err
}

// CreateThread creates a new conversation thread
func (s *Store) CreateThread(name string, metadata map[string]interface{}) (*Thread, error) {
	thread := &Thread{
		ID:        generateID(),
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  metadata,
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO threads (id, name, created_at, updated_at, metadata)
		VALUES (?, ?, ?, ?, ?)
	`, thread.ID, thread.Name, thread.CreatedAt, thread.UpdatedAt, string(metadataJSON))

	if err != nil {
		return nil, fmt.Errorf("failed to create thread: %w", err)
	}

	return thread, nil
}

// GetThread retrieves a thread by ID
func (s *Store) GetThread(id string) (*Thread, error) {
	var thread Thread
	var metadataJSON string

	err := s.db.QueryRow(`
		SELECT id, name, created_at, updated_at, current_checkpoint, metadata
		FROM threads WHERE id = ?
	`, id).Scan(&thread.ID, &thread.Name, &thread.CreatedAt, &thread.UpdatedAt, &thread.CurrentCheckpoint, &metadataJSON)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("thread not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get thread: %w", err)
	}

	if err := json.Unmarshal([]byte(metadataJSON), &thread.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &thread, nil
}

// ListThreads lists all threads
func (s *Store) ListThreads() ([]*Thread, error) {
	rows, err := s.db.Query(`
		SELECT id, name, created_at, updated_at, current_checkpoint, metadata
		FROM threads
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list threads: %w", err)
	}
	defer rows.Close()

	threads := []*Thread{}
	for rows.Next() {
		var thread Thread
		var metadataJSON sql.NullString

		if err := rows.Scan(&thread.ID, &thread.Name, &thread.CreatedAt, &thread.UpdatedAt, &thread.CurrentCheckpoint, &metadataJSON); err != nil {
			return nil, err
		}

		if metadataJSON.Valid {
			json.Unmarshal([]byte(metadataJSON.String), &thread.Metadata)
		}

		threads = append(threads, &thread)
	}

	return threads, rows.Err()
}

// SaveCheckpoint saves a checkpoint for a thread
func (s *Store) SaveCheckpoint(checkpoint *Checkpoint) error {
	messagesJSON, err := json.Marshal(checkpoint.Messages)
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	metadataJSON, err := json.Marshal(checkpoint.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = s.db.Exec(`
		INSERT INTO checkpoints (id, thread_id, parent_id, timestamp, messages, metadata, description)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, checkpoint.ID, checkpoint.ThreadID, checkpoint.ParentID, checkpoint.Timestamp, string(messagesJSON), string(metadataJSON), checkpoint.Description)

	if err != nil {
		return fmt.Errorf("failed to save checkpoint: %w", err)
	}

	// Update thread's current checkpoint
	_, err = s.db.Exec(`
		UPDATE threads SET current_checkpoint = ?, updated_at = ? WHERE id = ?
	`, checkpoint.ID, time.Now(), checkpoint.ThreadID)

	return err
}

// GetCheckpoint retrieves a checkpoint by ID
func (s *Store) GetCheckpoint(id string) (*Checkpoint, error) {
	var checkpoint Checkpoint
	var messagesJSON, metadataJSON string
	var parentID sql.NullString

	err := s.db.QueryRow(`
		SELECT id, thread_id, parent_id, timestamp, messages, metadata, description
		FROM checkpoints WHERE id = ?
	`, id).Scan(&checkpoint.ID, &checkpoint.ThreadID, &parentID, &checkpoint.Timestamp, &messagesJSON, &metadataJSON, &checkpoint.Description)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("checkpoint not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}

	if parentID.Valid {
		checkpoint.ParentID = parentID.String
	}

	if err := json.Unmarshal([]byte(messagesJSON), &checkpoint.Messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	if err := json.Unmarshal([]byte(metadataJSON), &checkpoint.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &checkpoint, nil
}

// GetThreadCheckpoints retrieves all checkpoints for a thread
func (s *Store) GetThreadCheckpoints(threadID string) ([]*Checkpoint, error) {
	rows, err := s.db.Query(`
		SELECT id, thread_id, parent_id, timestamp, messages, metadata, description
		FROM checkpoints WHERE thread_id = ?
		ORDER BY timestamp DESC
	`, threadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get thread checkpoints: %w", err)
	}
	defer rows.Close()

	checkpoints := []*Checkpoint{}
	for rows.Next() {
		var checkpoint Checkpoint
		var messagesJSON, metadataJSON string
		var parentID sql.NullString

		if err := rows.Scan(&checkpoint.ID, &checkpoint.ThreadID, &parentID, &checkpoint.Timestamp, &messagesJSON, &metadataJSON, &checkpoint.Description); err != nil {
			return nil, err
		}

		if parentID.Valid {
			checkpoint.ParentID = parentID.String
		}

		json.Unmarshal([]byte(messagesJSON), &checkpoint.Messages)
		json.Unmarshal([]byte(metadataJSON), &checkpoint.Metadata)

		checkpoints = append(checkpoints, &checkpoint)
	}

	return checkpoints, rows.Err()
}

// BranchFromCheckpoint creates a new thread branching from a checkpoint
func (s *Store) BranchFromCheckpoint(checkpointID string, newThreadName string) (*Thread, error) {
	// Get the source checkpoint
	sourceCheckpoint, err := s.GetCheckpoint(checkpointID)
	if err != nil {
		return nil, err
	}

	// Create new thread
	newThread, err := s.CreateThread(newThreadName, map[string]interface{}{
		"branched_from": checkpointID,
		"parent_thread": sourceCheckpoint.ThreadID,
	})
	if err != nil {
		return nil, err
	}

	// Create initial checkpoint in new thread with same messages
	branchCheckpoint := &Checkpoint{
		ID:          generateID(),
		ThreadID:    newThread.ID,
		ParentID:    checkpointID, // Link to source checkpoint
		Timestamp:   time.Now(),
		Messages:    sourceCheckpoint.Messages,
		Description: fmt.Sprintf("Branched from checkpoint %s", checkpointID),
		Metadata: map[string]interface{}{
			"branch_source": checkpointID,
		},
	}

	if err := s.SaveCheckpoint(branchCheckpoint); err != nil {
		return nil, err
	}

	return newThread, nil
}

// DeleteThread deletes a thread and all its checkpoints
func (s *Store) DeleteThread(threadID string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete checkpoints
	_, err = tx.Exec("DELETE FROM checkpoints WHERE thread_id = ?", threadID)
	if err != nil {
		return err
	}

	// Delete thread
	_, err = tx.Exec("DELETE FROM threads WHERE id = ?", threadID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// generateID generates a unique ID (simple timestamp-based for now)
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
