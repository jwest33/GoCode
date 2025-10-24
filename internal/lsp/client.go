package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// Client represents an LSP client connection
type Client struct {
	cmd           *exec.Cmd
	stdin         io.WriteCloser
	stdout        io.ReadCloser
	stderr        io.ReadCloser
	reader        *bufio.Reader
	nextID        atomic.Int64
	pendingCalls  map[int64]chan *Response
	mu            sync.Mutex
	notifications chan *Notification
	shutdown      chan struct{}
}

// Message types
const (
	RequestMessage      = "request"
	ResponseMessage     = "response"
	NotificationMessage = "notification"
)

// Request represents an LSP request
type Request struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// Response represents an LSP response
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError represents an LSP error
type ResponseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification represents an LSP notification
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// NewClient creates a new LSP client for the given language server command
func NewClient(serverCmd string, args ...string) (*Client, error) {
	cmd := exec.Command(serverCmd, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start LSP server: %w", err)
	}

	client := &Client{
		cmd:           cmd,
		stdin:         stdin,
		stdout:        stdout,
		stderr:        stderr,
		reader:        bufio.NewReader(stdout),
		pendingCalls:  make(map[int64]chan *Response),
		notifications: make(chan *Notification, 100),
		shutdown:      make(chan struct{}),
	}

	// Start message reader
	go client.readMessages()

	return client, nil
}

// Initialize initializes the LSP session
func (c *Client) Initialize(ctx context.Context, rootURI string, capabilities ClientCapabilities) (*InitializeResult, error) {
	params := InitializeParams{
		ProcessID: nil, // Use nil for unknown process
		RootURI:   rootURI,
		Capabilities: capabilities,
	}

	var result InitializeResult
	if err := c.Call(ctx, "initialize", params, &result); err != nil {
		return nil, err
	}

	// Send initialized notification
	c.Notify("initialized", struct{}{})

	return &result, nil
}

// Call sends an LSP request and waits for response
func (c *Client) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	id := c.nextID.Add(1)

	req := Request{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Create response channel
	respChan := make(chan *Response, 1)
	c.mu.Lock()
	c.pendingCalls[id] = respChan
	c.mu.Unlock()

	// Send request
	if err := c.sendMessage(req); err != nil {
		c.mu.Lock()
		delete(c.pendingCalls, id)
		c.mu.Unlock()
		return err
	}

	// Wait for response
	select {
	case resp := <-respChan:
		if resp.Error != nil {
			return fmt.Errorf("LSP error %d: %s", resp.Error.Code, resp.Error.Message)
		}

		if result != nil && resp.Result != nil {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return fmt.Errorf("failed to unmarshal result: %w", err)
			}
		}

		return nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.pendingCalls, id)
		c.mu.Unlock()
		return ctx.Err()
	}
}

// Notify sends an LSP notification (no response expected)
func (c *Client) Notify(method string, params interface{}) error {
	notif := Notification{
		JSONRPC: "2.0",
		Method:  method,
	}

	if params != nil {
		paramsJSON, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		notif.Params = paramsJSON
	}

	return c.sendMessage(notif)
}

// sendMessage sends a message over the LSP protocol
func (c *Client) sendMessage(msg interface{}) error {
	jsonData, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// LSP uses Content-Length header
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(jsonData))

	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	if _, err := c.stdin.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// readMessages reads messages from the LSP server
func (c *Client) readMessages() {
	defer close(c.shutdown)

	for {
		// Read headers
		headers := make(map[string]string)
		for {
			line, err := c.reader.ReadString('\n')
			if err != nil {
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				break // End of headers
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		// Get content length
		contentLengthStr, ok := headers["Content-Length"]
		if !ok {
			continue
		}

		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			continue
		}

		// Read content
		content := make([]byte, contentLength)
		if _, err := io.ReadFull(c.reader, content); err != nil {
			return
		}

		// Parse message
		c.handleMessage(content)
	}
}

// handleMessage handles a received message
func (c *Client) handleMessage(data []byte) {
	// Try to parse as response first
	var resp Response
	if err := json.Unmarshal(data, &resp); err == nil && resp.ID != 0 {
		c.mu.Lock()
		if respChan, ok := c.pendingCalls[resp.ID]; ok {
			delete(c.pendingCalls, resp.ID)
			c.mu.Unlock()
			respChan <- &resp
			return
		}
		c.mu.Unlock()
	}

	// Try to parse as notification
	var notif Notification
	if err := json.Unmarshal(data, &notif); err == nil && notif.Method != "" {
		select {
		case c.notifications <- &notif:
		default:
			// Drop notification if channel is full
		}
	}
}

// Shutdown gracefully shuts down the LSP client
func (c *Client) Shutdown(ctx context.Context) error {
	// Send shutdown request
	if err := c.Call(ctx, "shutdown", nil, nil); err != nil {
		return err
	}

	// Send exit notification
	c.Notify("exit", nil)

	// Wait for process to exit
	select {
	case <-c.shutdown:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Kill process if still running
	if c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}

	return nil
}

// Close closes the LSP client connection
func (c *Client) Close() error {
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Process.Kill()
	}
	return nil
}
