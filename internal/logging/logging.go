package logging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jake/coder/internal/config"
)

type Logger struct {
	config      *config.LoggingConfig
	file        *os.File
	encoder     *json.Encoder
	logChan     chan LogEntry
	done        chan struct{}
	droppedLogs int
}

type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Type        string                 `json:"type"` // user_input, llm_request, llm_response, tool_call, tool_result
	Role        string                 `json:"role,omitempty"`
	Content     string                 `json:"content,omitempty"`
	ToolName    string                 `json:"tool_name,omitempty"`
	ToolArgs    string                 `json:"tool_args,omitempty"`
	ToolResult  string                 `json:"tool_result,omitempty"`
	ToolError   string                 `json:"tool_error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	TokenCount  int                    `json:"token_count,omitempty"`
	Model       string                 `json:"model,omitempty"`
	Temperature float32                `json:"temperature,omitempty"`
}

func New(cfg *config.LoggingConfig, baseDir string) (*Logger, error) {
	// Make log directory absolute if relative
	logDir := cfg.Directory
	if !filepath.IsAbs(logDir) {
		logDir = filepath.Join(baseDir, logDir)
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Create log file with timestamp
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := filepath.Join(logDir, fmt.Sprintf("session_%s.jsonl", timestamp))

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := &Logger{
		config:      cfg,
		file:        file,
		encoder:     json.NewEncoder(file),
		logChan:     make(chan LogEntry, 1000), // Buffer up to 1000 log entries
		done:        make(chan struct{}),
		droppedLogs: 0,
	}

	// Start background logging goroutine
	go logger.processLogs()

	// Log session start
	logger.Log(LogEntry{
		Type:    "session_start",
		Content: fmt.Sprintf("Session started at %s", time.Now().Format(time.RFC3339)),
	})

	return logger, nil
}

// processLogs runs in background goroutine to write logs asynchronously
func (l *Logger) processLogs() {
	for entry := range l.logChan {
		entry.Timestamp = time.Now()
		if err := l.encoder.Encode(entry); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing log: %v\n", err)
		}
	}
	close(l.done)
}

func (l *Logger) Log(entry LogEntry) error {
	// Try to send to channel, drop if buffer is full
	select {
	case l.logChan <- entry:
		return nil
	default:
		l.droppedLogs++
		fmt.Fprintf(os.Stderr, "Warning: Log buffer full, dropped log entry (total dropped: %d)\n", l.droppedLogs)
		return fmt.Errorf("log buffer full")
	}
}

func (l *Logger) LogUserInput(content string) {
	l.Log(LogEntry{
		Type:    "user_input",
		Role:    "user",
		Content: content,
	})
}

func (l *Logger) LogLLMRequest(messages []interface{}, model string, temp float32) {
	data, _ := json.Marshal(messages)
	l.Log(LogEntry{
		Type:        "llm_request",
		Content:     string(data),
		Model:       model,
		Temperature: temp,
	})
}

func (l *Logger) LogLLMResponse(content string, toolCalls []interface{}) {
	data, _ := json.Marshal(map[string]interface{}{
		"content":    content,
		"tool_calls": toolCalls,
	})
	l.Log(LogEntry{
		Type:    "llm_response",
		Role:    "assistant",
		Content: string(data),
	})
}

func (l *Logger) LogToolCall(name string, args string) {
	l.Log(LogEntry{
		Type:     "tool_call",
		ToolName: name,
		ToolArgs: args,
	})
}

func (l *Logger) LogToolResult(name string, result string, err error) {
	entry := LogEntry{
		Type:       "tool_result",
		ToolName:   name,
		ToolResult: result,
	}
	if err != nil {
		entry.ToolError = err.Error()
	}
	l.Log(entry)
}

func (l *Logger) Close() error {
	// Send session end log
	l.Log(LogEntry{
		Type:    "session_end",
		Content: fmt.Sprintf("Session ended at %s", time.Now().Format(time.RFC3339)),
	})

	// Close the channel to signal the goroutine to finish
	close(l.logChan)

	// Wait for all logs to be written
	<-l.done

	// Report dropped logs if any
	if l.droppedLogs > 0 {
		fmt.Fprintf(os.Stderr, "Warning: %d log entries were dropped during this session\n", l.droppedLogs)
	}

	return l.file.Close()
}
