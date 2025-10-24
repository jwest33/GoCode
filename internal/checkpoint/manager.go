package checkpoint

import (
	"fmt"
	"time"

	"github.com/jake/gocode/internal/llm"
)

// Manager provides high-level thread and checkpoint management
type Manager struct {
	store         *Store
	currentThread *Thread
	autoSave      bool
	saveInterval  int // Save every N messages
	messageCount  int
}

// Config holds configuration for the checkpoint manager
type Config struct {
	DBPath       string
	AutoSave     bool
	SaveInterval int // Auto-save every N messages (0 = manual only)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		DBPath:       "checkpoints.db",
		AutoSave:     true,
		SaveInterval: 5, // Save every 5 messages
	}
}

// NewManager creates a new checkpoint manager
func NewManager(config Config) (*Manager, error) {
	store, err := NewStore(config.DBPath)
	if err != nil {
		return nil, err
	}

	return &Manager{
		store:        store,
		autoSave:     config.AutoSave,
		saveInterval: config.SaveInterval,
		messageCount: 0,
	}, nil
}

// StartNewThread starts a new conversation thread
func (m *Manager) StartNewThread(name string) (*Thread, error) {
	thread, err := m.store.CreateThread(name, map[string]interface{}{
		"started_at": time.Now(),
	})
	if err != nil {
		return nil, err
	}

	m.currentThread = thread
	m.messageCount = 0

	return thread, nil
}

// ResumeThread resumes an existing thread
func (m *Manager) ResumeThread(threadID string) ([]*llm.Message, error) {
	thread, err := m.store.GetThread(threadID)
	if err != nil {
		return nil, err
	}

	m.currentThread = thread

	// Get the current checkpoint
	if thread.CurrentCheckpoint != "" {
		checkpoint, err := m.store.GetCheckpoint(thread.CurrentCheckpoint)
		if err != nil {
			return nil, err
		}

		// Convert to pointer slice
		messages := make([]*llm.Message, len(checkpoint.Messages))
		for i := range checkpoint.Messages {
			messages[i] = &checkpoint.Messages[i]
		}

		return messages, nil
	}

	return []*llm.Message{}, nil
}

// SaveCheckpoint manually saves a checkpoint
func (m *Manager) SaveCheckpoint(messages []llm.Message, description string) (*Checkpoint, error) {
	if m.currentThread == nil {
		return nil, fmt.Errorf("no active thread")
	}

	checkpoint := &Checkpoint{
		ID:          generateID(),
		ThreadID:    m.currentThread.ID,
		Timestamp:   time.Now(),
		Messages:    messages,
		Description: description,
		Metadata: map[string]interface{}{
			"message_count": len(messages),
		},
	}

	if err := m.store.SaveCheckpoint(checkpoint); err != nil {
		return nil, err
	}

	m.currentThread.CurrentCheckpoint = checkpoint.ID
	m.messageCount = 0 // Reset counter after save

	return checkpoint, nil
}

// OnMessage should be called after each message exchange
func (m *Manager) OnMessage(messages []llm.Message) error {
	if !m.autoSave || m.currentThread == nil {
		return nil
	}

	m.messageCount++

	// Auto-save if interval reached
	if m.saveInterval > 0 && m.messageCount >= m.saveInterval {
		_, err := m.SaveCheckpoint(messages, fmt.Sprintf("Auto-save after %d messages", m.messageCount))
		return err
	}

	return nil
}

// BranchThread creates a new thread branching from a checkpoint
func (m *Manager) BranchThread(checkpointID string, newName string) (*Thread, error) {
	newThread, err := m.store.BranchFromCheckpoint(checkpointID, newName)
	if err != nil {
		return nil, err
	}

	m.currentThread = newThread
	m.messageCount = 0

	return newThread, nil
}

// GetCurrentThread returns the current active thread
func (m *Manager) GetCurrentThread() *Thread {
	return m.currentThread
}

// ListThreads lists all threads
func (m *Manager) ListThreads() ([]*Thread, error) {
	return m.store.ListThreads()
}

// GetThreadHistory returns all checkpoints for the current thread
func (m *Manager) GetThreadHistory() ([]*Checkpoint, error) {
	if m.currentThread == nil {
		return nil, fmt.Errorf("no active thread")
	}

	return m.store.GetThreadCheckpoints(m.currentThread.ID)
}

// RestoreCheckpoint restores messages from a specific checkpoint
func (m *Manager) RestoreCheckpoint(checkpointID string) ([]llm.Message, error) {
	checkpoint, err := m.store.GetCheckpoint(checkpointID)
	if err != nil {
		return nil, err
	}

	// Update current thread's checkpoint
	if m.currentThread != nil && m.currentThread.ID == checkpoint.ThreadID {
		m.currentThread.CurrentCheckpoint = checkpointID
	}

	return checkpoint.Messages, nil
}

// DeleteThread deletes a thread
func (m *Manager) DeleteThread(threadID string) error {
	// If deleting current thread, clear it
	if m.currentThread != nil && m.currentThread.ID == threadID {
		m.currentThread = nil
	}

	return m.store.DeleteThread(threadID)
}

// Close closes the checkpoint manager
func (m *Manager) Close() error {
	return m.store.Close()
}

// GetCheckpointTree returns a tree structure of checkpoints (for branching visualization)
func (m *Manager) GetCheckpointTree(threadID string) (*CheckpointTree, error) {
	checkpoints, err := m.store.GetThreadCheckpoints(threadID)
	if err != nil {
		return nil, err
	}

	// Build tree
	root := &CheckpointTree{
		Children: make([]*CheckpointTree, 0),
	}

	nodeMap := make(map[string]*CheckpointTree)

	// Create nodes
	for _, cp := range checkpoints {
		node := &CheckpointTree{
			Checkpoint: cp,
			Children:   make([]*CheckpointTree, 0),
		}
		nodeMap[cp.ID] = node
	}

	// Link children to parents
	for _, cp := range checkpoints {
		node := nodeMap[cp.ID]
		if cp.ParentID != "" {
			if parent, ok := nodeMap[cp.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			}
		} else {
			// Root checkpoint
			root.Children = append(root.Children, node)
		}
	}

	return root, nil
}

// CheckpointTree represents a tree of checkpoints for visualization
type CheckpointTree struct {
	Checkpoint *Checkpoint
	Children   []*CheckpointTree
}
