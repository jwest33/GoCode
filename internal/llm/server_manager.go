package llm

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/jake/gocode/internal/config"
	"github.com/jake/gocode/internal/theme"
)

type ServerManager struct {
	config      *config.LLMConfig
	process     *exec.Cmd
	managedByUs bool
}

func NewServerManager(cfg *config.LLMConfig) *ServerManager {
	return &ServerManager{
		config:      cfg,
		managedByUs: false,
	}
}

// Start checks if server is running, and starts it if needed
func (sm *ServerManager) Start() error {
	if !sm.config.AutoManage {
		fmt.Println(theme.Dim("llama-server auto-management disabled, using external server"))
		return nil
	}

	fmt.Println(theme.Dim("Checking if llama-server is already running..."))

	// Check if server is already available
	if sm.isServerAvailable() {
		fmt.Println(theme.Success("✓ llama-server is already running and responding"))
		sm.managedByUs = false
		return nil
	}

	fmt.Println(theme.Agent("Starting llama-server..."))

	// Build command with all flags
	args := sm.buildCommandArgs()

	sm.process = exec.Command("llama-server", args...)
	// Discard llama-server logs to keep console clean
	sm.process.Stdout = io.Discard
	sm.process.Stderr = io.Discard

	if err := sm.process.Start(); err != nil {
		return fmt.Errorf("failed to start llama-server: %w", err)
	}

	sm.managedByUs = true
	fmt.Println(theme.Dim(fmt.Sprintf("Waiting for llama-server to become ready (timeout: %ds)...", sm.config.StartupTimeout)))

	// Wait for server to be ready
	timeout := time.Duration(sm.config.StartupTimeout) * time.Second
	if err := sm.waitForServer(timeout); err != nil {
		sm.Stop()
		return fmt.Errorf("llama-server failed to start: %w", err)
	}

	fmt.Println(theme.Success("✓ llama-server is ready!"))
	return nil
}

// Stop gracefully stops the server if we started it
func (sm *ServerManager) Stop() error {
	if !sm.managedByUs || sm.process == nil {
		return nil
	}

	fmt.Println(theme.Dim("🛑 Shutting down llama-server..."))

	if sm.process.Process != nil {
		if err := sm.process.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop llama-server: %w", err)
		}

		// Wait for process to exit
		sm.process.Wait()
	}

	fmt.Println(theme.Success("✓ llama-server stopped"))
	return nil
}

func (sm *ServerManager) buildCommandArgs() []string {
	cfg := sm.config.Server
	args := []string{
		"--model", cfg.ModelPath,
		"--host", cfg.Host,
		"--port", strconv.Itoa(cfg.Port),
		"--api-key", sm.config.APIKey,
		"--ctx-size", strconv.Itoa(cfg.CtxSize),
		"--batch-size", strconv.Itoa(cfg.BatchSize),
		"--ubatch-size", strconv.Itoa(cfg.UBatchSize),
		"--threads", strconv.Itoa(cfg.Threads),
		"--n-gpu-layers", strconv.Itoa(cfg.NGpuLayers),
		"--repeat-last-n", strconv.Itoa(cfg.RepeatLastN),
		"--repeat-penalty", fmt.Sprintf("%.2f", cfg.RepeatPenalty),
		"--cache-type-k", cfg.CacheTypeK,
		"--cache-type-v", cfg.CacheTypeV,
	}

	if cfg.FlashAttn {
		args = append(args, "--flash-attn", "on")
	}

	if cfg.Jinja {
		args = append(args, "--jinja")
	}

	if cfg.NCpuMoe > 0 {
		args = append(args, "--n-cpu-moe", strconv.Itoa(cfg.NCpuMoe))
	}

	return args
}

func (sm *ServerManager) isServerAvailable() bool {
	client := &http.Client{
		Timeout: 2 * time.Second,
	}

	// Try to hit the health endpoint
	healthURL := fmt.Sprintf("http://%s:%d/health", sm.config.Server.Host, sm.config.Server.Port)
	resp, err := client.Get(healthURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound
}

func (sm *ServerManager) waitForServer(timeout time.Duration) error {
	start := time.Now()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if sm.isServerAvailable() {
				return nil
			}

			// Check if process has exited with error
			if sm.process != nil && sm.process.ProcessState != nil && sm.process.ProcessState.Exited() {
				return fmt.Errorf("process exited prematurely with code: %d", sm.process.ProcessState.ExitCode())
			}

			if time.Since(start) > timeout {
				return fmt.Errorf("timeout waiting for server to become ready")
			}
		}
	}
}
