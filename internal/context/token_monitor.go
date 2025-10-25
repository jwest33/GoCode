package context

import (
	"fmt"
	"sync"
	"time"
)

// TokenWarningLevel represents severity of token usage warning
type TokenWarningLevel string

const (
	LevelInfo     TokenWarningLevel = "info"     // 60% - heads up
	LevelWarning  TokenWarningLevel = "warning"  // 70% - start offloading
	LevelCritical TokenWarningLevel = "critical" // 80% - pruning imminent
)

// TokenWarning represents a warning about token usage
type TokenWarning struct {
	Level      TokenWarningLevel
	Percentage float64
	Current    int
	Max        int
	Message    string
	Timestamp  time.Time
}

// TokenMonitor watches token usage asynchronously and sends warnings
type TokenMonitor struct {
	contextMgr     *Manager
	warnings       chan TokenWarning
	actualTokens   chan int          // Receive actual token counts from LLM
	stop           chan struct{}
	checkInterval  time.Duration
	thresholds     map[TokenWarningLevel]float64
	lastWarning    TokenWarningLevel // Track last warning level to avoid spam
	actualUsage    int               // Actual tokens from LLM (when available)
	mu             sync.RWMutex
	running        bool
}

// NewTokenMonitor creates a new async token usage monitor
func NewTokenMonitor(contextMgr *Manager, checkInterval time.Duration) *TokenMonitor {
	return &TokenMonitor{
		contextMgr:    contextMgr,
		warnings:      make(chan TokenWarning, 10),
		actualTokens:  make(chan int, 10),
		stop:          make(chan struct{}),
		checkInterval: checkInterval,
		thresholds: map[TokenWarningLevel]float64{
			LevelInfo:     0.60, // 60%
			LevelWarning:  0.70, // 70%
			LevelCritical: 0.80, // 80%
		},
		lastWarning: "",
	}
}

// Start begins async monitoring in a background goroutine
func (tm *TokenMonitor) Start() {
	tm.mu.Lock()
	if tm.running {
		tm.mu.Unlock()
		return
	}
	tm.running = true
	tm.mu.Unlock()

	go tm.monitorLoop()
}

// Stop stops the monitoring goroutine
func (tm *TokenMonitor) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.running {
		return
	}

	close(tm.stop)
	tm.running = false
}

// Warnings returns the channel for receiving warnings
func (tm *TokenMonitor) Warnings() <-chan TokenWarning {
	return tm.warnings
}

// UpdateActualTokens updates with actual token count from LLM response
func (tm *TokenMonitor) UpdateActualTokens(tokens int) {
	select {
	case tm.actualTokens <- tokens:
	default:
		// Channel full, skip this update
	}
}

// monitorLoop runs in background, checking token usage periodically
func (tm *TokenMonitor) monitorLoop() {
	ticker := time.NewTicker(tm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-tm.stop:
			return

		case actualTokens := <-tm.actualTokens:
			// Update with actual token count from LLM
			tm.mu.Lock()
			tm.actualUsage = actualTokens
			tm.mu.Unlock()

		case <-ticker.C:
			tm.checkUsage()
		}
	}
}

// checkUsage checks current token usage and sends warnings if needed
func (tm *TokenMonitor) checkUsage() {
	// Calculate current usage
	usage := tm.contextMgr.CalculateCurrentUsage()
	currentTokens := usage.Total

	// Use actual tokens if available (more accurate than estimate)
	tm.mu.RLock()
	if tm.actualUsage > 0 && tm.actualUsage > currentTokens {
		currentTokens = tm.actualUsage
	}
	lastWarningLevel := tm.lastWarning
	tm.mu.RUnlock()

	maxTokens := tm.contextMgr.config.MaxTokens
	percentage := float64(currentTokens) / float64(maxTokens)

	// Determine warning level
	var warningLevel TokenWarningLevel
	var message string

	switch {
	case percentage >= tm.thresholds[LevelCritical]:
		warningLevel = LevelCritical
		message = "Context window at 80%+ - pruning will begin soon. Save important context to memory now!"

	case percentage >= tm.thresholds[LevelWarning]:
		warningLevel = LevelWarning
		message = "Context window at 70%+ - consider offloading context to long-term memory"

	case percentage >= tm.thresholds[LevelInfo]:
		warningLevel = LevelInfo
		message = "Context window at 60%+ - heads up, may need to prune soon"

	default:
		// No warning needed
		return
	}

	// Only send warning if level increased (avoid spam)
	if warningLevel <= lastWarningLevel {
		return
	}

	// Update last warning level
	tm.mu.Lock()
	tm.lastWarning = warningLevel
	tm.mu.Unlock()

	// Send warning
	warning := TokenWarning{
		Level:      warningLevel,
		Percentage: percentage * 100,
		Current:    currentTokens,
		Max:        maxTokens,
		Message:    message,
		Timestamp:  time.Now(),
	}

	select {
	case tm.warnings <- warning:
		// Warning sent successfully
	default:
		// Warning channel full, skip
	}
}

// GetCurrentUsage returns current token usage info
func (tm *TokenMonitor) GetCurrentUsage() (current int, max int, percentage float64) {
	usage := tm.contextMgr.CalculateCurrentUsage()
	current = usage.Total

	tm.mu.RLock()
	if tm.actualUsage > 0 && tm.actualUsage > current {
		current = tm.actualUsage
	}
	tm.mu.RUnlock()

	max = tm.contextMgr.config.MaxTokens
	percentage = float64(current) / float64(max) * 100

	return
}

// ResetWarningLevel resets the warning tracking (e.g., after pruning)
func (tm *TokenMonitor) ResetWarningLevel() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.lastWarning = ""
	tm.actualUsage = 0
}

// FormatWarning formats a warning for display
func (w TokenWarning) FormatWarning() string {
	icon := ""
	switch w.Level {
	case LevelInfo:
		icon = "ℹ️"
	case LevelWarning:
		icon = "⚠️"
	case LevelCritical:
		icon = "🔴"
	}

	return fmt.Sprintf("%s Token Usage: %.1f%% (%d/%d tokens) - %s",
		icon, w.Percentage, w.Current, w.Max, w.Message)
}
