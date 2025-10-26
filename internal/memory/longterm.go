package memory

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// LongTermMemory stores facts, learnings, and artifacts across sessions
type LongTermMemory struct {
	db *sql.DB
}

// MemoryType represents the type of memory
type MemoryType string

const (
	TypeFact      MemoryType = "fact"      // Learned facts about the codebase
	TypeArtifact  MemoryType = "artifact"  // Stored artifacts (patches, logs, etc.)
	TypeDecision  MemoryType = "decision"  // Important decisions made
	TypePattern   MemoryType = "pattern"   // Recognized patterns
	TypeError     MemoryType = "error"     // Errors and their solutions
)

// Memory represents a long-term memory entry
type Memory struct {
	ID          string                 `json:"id"`
	Type        MemoryType             `json:"type"`
	Content     string                 `json:"content"`
	Summary     string                 `json:"summary"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	AccessCount int                    `json:"access_count"`
	LastAccess  time.Time              `json:"last_access"`
	Importance  float32                `json:"importance"` // 0-1 score
	TraceID     string                 `json:"trace_id,omitempty"`     // Link to OTel trace
	ArtifactID  string                 `json:"artifact_id,omitempty"`  // Link to artifact
}

// NewLongTermMemory creates a new long-term memory store
func NewLongTermMemory(dbPath string) (*LongTermMemory, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	ltm := &LongTermMemory{db: db}

	if err := ltm.initSchema(); err != nil {
		return nil, err
	}

	return ltm, nil
}

// initSchema creates the database schema
func (ltm *LongTermMemory) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS memories (
		id TEXT PRIMARY KEY,
		type TEXT NOT NULL,
		content TEXT NOT NULL,
		summary TEXT NOT NULL,
		tags TEXT,
		metadata TEXT,
		created_at DATETIME NOT NULL,
		access_count INTEGER DEFAULT 0,
		last_access DATETIME,
		importance REAL DEFAULT 0.5,
		trace_id TEXT,
		artifact_id TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_memory_type ON memories(type);
	CREATE INDEX IF NOT EXISTS idx_memory_tags ON memories(tags);
	CREATE INDEX IF NOT EXISTS idx_memory_importance ON memories(importance DESC);
	CREATE INDEX IF NOT EXISTS idx_memory_created ON memories(created_at DESC);

	CREATE VIRTUAL TABLE IF NOT EXISTS memory_fts USING fts5(
		id UNINDEXED,
		content,
		summary,
		tags
	);
	`

	_, err := ltm.db.Exec(schema)
	return err
}

// Store stores a memory
func (ltm *LongTermMemory) Store(memory *Memory) error {
	if memory.ID == "" {
		memory.ID = generateMemoryID()
	}
	if memory.CreatedAt.IsZero() {
		memory.CreatedAt = time.Now()
	}

	tagsJSON, _ := json.Marshal(memory.Tags)
	metadataJSON, _ := json.Marshal(memory.Metadata)

	tx, err := ltm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert into main table
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO memories
		(id, type, content, summary, tags, metadata, created_at, access_count, last_access, importance, trace_id, artifact_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, memory.ID, memory.Type, memory.Content, memory.Summary, string(tagsJSON), string(metadataJSON),
		memory.CreatedAt, memory.AccessCount, memory.LastAccess, memory.Importance, memory.TraceID, memory.ArtifactID)

	if err != nil {
		return err
	}

	// Insert into FTS table
	_, err = tx.Exec(`
		INSERT OR REPLACE INTO memory_fts (id, content, summary, tags)
		VALUES (?, ?, ?, ?)
	`, memory.ID, memory.Content, memory.Summary, string(tagsJSON))

	if err != nil {
		return err
	}

	return tx.Commit()
}

// Search searches memories using full-text search
func (ltm *LongTermMemory) Search(query string, limit int) ([]*Memory, error) {
	rows, err := ltm.db.Query(`
		SELECT m.id, m.type, m.content, m.summary, m.tags, m.metadata,
		       m.created_at, m.access_count, m.last_access, m.importance, m.trace_id, m.artifact_id
		FROM memories m
		INNER JOIN memory_fts fts ON m.id = fts.id
		WHERE memory_fts MATCH ?
		ORDER BY m.importance DESC, m.access_count DESC
		LIMIT ?
	`, query, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ltm.scanMemories(rows)
}

// Get retrieves a memory by ID
func (ltm *LongTermMemory) Get(id string) (*Memory, error) {
	var memory Memory
	var tagsJSON, metadataJSON string
	var lastAccess sql.NullTime

	err := ltm.db.QueryRow(`
		SELECT id, type, content, summary, tags, metadata, created_at, access_count, last_access, importance, trace_id, artifact_id
		FROM memories WHERE id = ?
	`, id).Scan(&memory.ID, &memory.Type, &memory.Content, &memory.Summary, &tagsJSON, &metadataJSON,
		&memory.CreatedAt, &memory.AccessCount, &lastAccess, &memory.Importance, &memory.TraceID, &memory.ArtifactID)

	if err != nil {
		return nil, err
	}

	json.Unmarshal([]byte(tagsJSON), &memory.Tags)
	json.Unmarshal([]byte(metadataJSON), &memory.Metadata)

	if lastAccess.Valid {
		memory.LastAccess = lastAccess.Time
	}

	// Update access count
	go ltm.recordAccess(id)

	return &memory, nil
}

// GetByType retrieves memories by type
func (ltm *LongTermMemory) GetByType(memType MemoryType, limit int) ([]*Memory, error) {
	rows, err := ltm.db.Query(`
		SELECT id, type, content, summary, tags, metadata, created_at, access_count, last_access, importance, trace_id, artifact_id
		FROM memories
		WHERE type = ?
		ORDER BY importance DESC, created_at DESC
		LIMIT ?
	`, memType, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ltm.scanMemories(rows)
}

// GetByTags retrieves memories with specific tags
func (ltm *LongTermMemory) GetByTags(tags []string, limit int) ([]*Memory, error) {
	// Simple implementation - can be optimized with JSON functions
	allMemories, err := ltm.GetRecent(1000) // Get recent memories
	if err != nil {
		return nil, err
	}

	matched := []*Memory{}
	for _, mem := range allMemories {
		if hasAllTags(mem.Tags, tags) {
			matched = append(matched, mem)
			if len(matched) >= limit {
				break
			}
		}
	}

	return matched, nil
}

// GetRecent retrieves recent memories
func (ltm *LongTermMemory) GetRecent(limit int) ([]*Memory, error) {
	rows, err := ltm.db.Query(`
		SELECT id, type, content, summary, tags, metadata, created_at, access_count, last_access, importance, trace_id, artifact_id
		FROM memories
		ORDER BY created_at DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ltm.scanMemories(rows)
}

// GetMostImportant retrieves most important memories
func (ltm *LongTermMemory) GetMostImportant(limit int) ([]*Memory, error) {
	rows, err := ltm.db.Query(`
		SELECT id, type, content, summary, tags, metadata, created_at, access_count, last_access, importance, trace_id, artifact_id
		FROM memories
		ORDER BY importance DESC, access_count DESC
		LIMIT ?
	`, limit)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return ltm.scanMemories(rows)
}

// Delete deletes a memory
func (ltm *LongTermMemory) Delete(id string) error {
	tx, err := ltm.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("DELETE FROM memories WHERE id = ?", id)
	if err != nil {
		return err
	}

	_, err = tx.Exec("DELETE FROM memory_fts WHERE id = ?", id)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// Prune removes old, low-importance memories (memory management)
func (ltm *LongTermMemory) Prune(olderThanDays int, minImportance float32, keepCount int) error {
	cutoffDate := time.Now().AddDate(0, 0, -olderThanDays)

	_, err := ltm.db.Exec(`
		DELETE FROM memories
		WHERE id IN (
			SELECT id FROM memories
			WHERE created_at < ? AND importance < ?
			ORDER BY importance ASC, access_count ASC
			LIMIT (SELECT MAX(0, COUNT(*) - ?) FROM memories)
		)
	`, cutoffDate, minImportance, keepCount)

	return err
}

// Helper functions

func (ltm *LongTermMemory) scanMemories(rows *sql.Rows) ([]*Memory, error) {
	memories := []*Memory{}

	for rows.Next() {
		var memory Memory
		var tagsJSON, metadataJSON string
		var lastAccess sql.NullTime
		var traceID, artifactID sql.NullString

		err := rows.Scan(&memory.ID, &memory.Type, &memory.Content, &memory.Summary, &tagsJSON, &metadataJSON,
			&memory.CreatedAt, &memory.AccessCount, &lastAccess, &memory.Importance, &traceID, &artifactID)

		if err != nil {
			return nil, err
		}

		json.Unmarshal([]byte(tagsJSON), &memory.Tags)
		json.Unmarshal([]byte(metadataJSON), &memory.Metadata)

		if lastAccess.Valid {
			memory.LastAccess = lastAccess.Time
		}
		if traceID.Valid {
			memory.TraceID = traceID.String
		}
		if artifactID.Valid {
			memory.ArtifactID = artifactID.String
		}

		memories = append(memories, &memory)
	}

	return memories, rows.Err()
}

func (ltm *LongTermMemory) recordAccess(id string) {
	ltm.db.Exec(`
		UPDATE memories
		SET access_count = access_count + 1, last_access = ?
		WHERE id = ?
	`, time.Now(), id)
}

func hasAllTags(memoryTags []string, requiredTags []string) bool {
	tagSet := make(map[string]bool)
	for _, tag := range memoryTags {
		tagSet[tag] = true
	}

	for _, required := range requiredTags {
		if !tagSet[required] {
			return false
		}
	}

	return true
}

func generateMemoryID() string {
	return fmt.Sprintf("mem_%d", time.Now().UnixNano())
}

// Close closes the database connection
func (ltm *LongTermMemory) Close() error {
	return ltm.db.Close()
}
