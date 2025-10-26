# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoCode is an autonomous AI development assistant that runs locally with llama.cpp. It features automatic llama-server management, intelligent project analysis, hybrid retrieval systems (BM25/semantic/trigram), LSP integration, and a full suite of development tools.

**Key Technologies:**
- Go 1.24.0
- llama.cpp for local LLM inference
- OpenTelemetry for tracing
- SQLite for persistence (checkpoints, memory, embeddings)
- Language Server Protocol for code navigation

## Build and Development Commands

### Building
```bash
# Build the main executable
go build -o gocode.exe cmd/gocode/main.go

# Build on Windows and add to PATH
.\setup-path.bat
```

### Running
```bash
# Run from any directory (auto-finds config.yaml)
gocode

# Run with specific config
gocode --config path/to/config.yaml

# Show version
gocode --version
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Test specific package
go test ./internal/agent
```

**Note:** As of current state, there are no test files (`*_test.go`) in the project. Tests should be added in the same directory as the package being tested.

## Configuration

**Config search order:**
1. `--config` flag
2. `GOCODE_CONFIG` environment variable
3. Current working directory (`./config.yaml`)
4. Executable's directory
5. `~/.gocode/config.yaml`

**Critical config.yaml settings:**
- `llm.server.model_path`: Path to GGUF model file
- `llm.auto_manage`: Set to `true` to auto-start llama-server
- `lsp.enabled`: Enable LSP-based code navigation
- `memory.enabled`: Enable long-term memory across sessions
- `checkpoint.enabled`: Enable session persistence

## Architecture Overview

### Main Entry Point
- `cmd/gocode/main.go`: Application entry, config loading, initialization flow, agent creation

### Core Agent Loop
- `internal/agent/agent.go`: Main conversation loop, tool coordination, message management
- `internal/agent/selfcheck.go`: Self-checking system for agent operations

### LLM Integration
- `internal/llm/`: OpenAI-compatible client and llama-server auto-management
- Auto-starts llama-server if `llm.auto_manage: true`
- Reuses existing server instances on the configured port
- Gracefully shuts down managed servers on exit

### Context Management
- `internal/context/manager.go`: Token budget allocation and message pruning
- Default budget: 100K context window with dynamic allocation
- Automatically prunes history at 80% capacity
- Budget breakdown: system (2K), user input (4K), context (30K), history (60K), response (4K)

### Tool System
- `internal/tools/`: Registry and implementations (read, write, edit, glob, grep, bash, LSP tools)
- `internal/tools/bash.go`: Command execution with timeout and background process support
- All tools implement a common interface with confirmation policies

### Code Navigation
- `internal/lsp/manager.go`: Multi-language LSP client management
- `internal/lsp/pathfinder.go`: Advanced path resolution for LSP operations
- `internal/codegraph/`: Symbol tracking with definition/reference graphs
- Supports: Go (gopls), Python (pylsp), TypeScript (typescript-language-server), Rust (rust-analyzer)

### Memory and Persistence
- `internal/memory/longterm.go`: Long-term memory store (facts, decisions, patterns, errors, artifacts)
- `internal/checkpoint/`: SQLite-backed session persistence with thread management
- Memory types: fact, artifact, decision, pattern, error
- Stored in `.gocode/memory.db` in working directory

### Retrieval System
- `internal/retrieval/`: Hybrid search combining BM25 (keyword), semantic (vector), and trigram (fuzzy)
- Default weights: 40% BM25, 50% semantic, 10% trigram
- `internal/embeddings/`: Vector store with chunking and similarity search
- Requires separate embedding server on port 8081

### Project Initialization
- `internal/initialization/analyzer.go`: First-run project analysis
- `internal/initialization/features.go`: Feature recommendations based on project characteristics
- `internal/initialization/summary.go`: Display project summary and recommendations
- Creates `.gocode/` directory with `state.json` and `analysis.json`
- Background indexing for search systems

### Other Components
- `internal/confirmation/`: User confirmation system with configurable policies (interactive, auto, destructive_only)
- `internal/logging/`: Async JSONL logging with buffered writes to `logs/` directory
- `internal/prompts/`: Template-based system prompt rendering with project context
- `internal/telemetry/`: OpenTelemetry tracing with GenAI semantic conventions
- `internal/theme/`: Synthwave color palette (cyan, pink, purple, green, red, yellow)
- `internal/parser/`: Simple language detection and code parsing
- `internal/config/`: YAML configuration with environment variable overrides

## Critical Implementation Details

### File Path Management
- **Logs**: Stored in `<config-dir>/logs/` (centralized, near config.yaml)
- **TODO.md**: Stored in current working directory (project-specific)
- **State files**: Stored in `.gocode/` in working directory (`state.json`, `analysis.json`, `memory.db`)
- Memory database path resolves to `.gocode/memory.db` if not absolute

### LSP Integration
- LSP servers must be in PATH or their paths configured in config.yaml
- Manager validates servers on startup and warns about missing ones
- File extension mapping determines which LSP client to use
- Each language gets its own LSP client instance

### llama-server Management
- Auto-starts if `llm.auto_manage: true` and no server on configured port
- Detects existing instances and reuses them
- Applies KV cache quantization (q8_0, q4_1, etc.) and Flash Attention
- Graceful shutdown on exit (only if GoCode started it)
- Startup timeout: 60 seconds (configurable via `llm.startup_timeout`)

### Context Window Strategy
- Uses token budget system to prevent overflow
- System prompt includes project context (languages, frameworks, structure)
- Retrieval system injects relevant context within budget
- Automatic pruning when reaching 80% capacity
- Rough token estimation: 1 token ≈ 3.5 characters for code

### Tool Confirmation Policies
- `interactive` mode: Auto-approves safe tools (read, glob, grep), confirms destructive tools (write, edit, bash)
- `auto` mode: Never prompts (use with caution)
- `destructive_only` mode: Only confirms write, edit, bash
- Per-tool overrides via `auto_approve_tools` and `always_confirm_tools`

### Project Initialization Flow
1. Check for `.gocode/state.json` existence
2. If first run, prompt user for initialization
3. Analyze project: detect languages, frameworks, structure
4. Generate feature recommendations (LSP, semantic search, checkpointing)
5. Start background indexing (BM25, embeddings if enabled)
6. Display summary and save analysis to `.gocode/analysis.json`
7. Mark initialized in `state.json`

## Common Patterns

### Adding a New Tool
1. Create tool struct implementing the Tool interface in `internal/tools/`
2. Add to registry in `internal/agent/agent.go` (check enabled tools in config)
3. Update `config.yaml` to include tool in `tools.enabled`
4. Add confirmation policy if needed
5. Update system prompt template if tool needs special instructions

### Adding LSP Support for New Language
1. Add language server config to `internal/lsp/manager.go` DefaultLanguageServers()
2. Add file extensions for the language
3. Ensure LSP server command is in PATH or update config.yaml
4. Manager will auto-detect and initialize on first use

### Extending Memory Types
1. Add new MemoryType constant in `internal/memory/longterm.go`
2. Use Store() method with new type when saving memories
3. Query with RetrieveByType() or SearchByTags()
4. Memory automatically tracks access count and importance

### Working with Templates
- System prompts are in `internal/prompts/templates/`
- Use `system_with_project.tmpl` for project-aware prompts
- Template receives: ProjectContext, EnabledTools, Features
- Prompt manager handles template loading and rendering

## Development Guidelines

### Code Organization
- All internal packages in `internal/` (not importable externally)
- Each package should be focused on a single responsibility
- Use interfaces for dependency injection (e.g., Config interface in initialization)
- Manager pattern for stateful components (LSP, context, embeddings)

### Error Handling
- Return errors, don't panic (except for unrecoverable situations)
- Wrap errors with context: `fmt.Errorf("operation failed: %w", err)`
- Log errors before returning them at application boundaries
- Graceful degradation: continue if optional features fail (e.g., LSP, retrieval)

### State Management
- Use `.gocode/` directory for all project-specific state
- State files are JSON for easy inspection and debugging
- SQLite for structured data (checkpoints, memory, embeddings, traces)
- Clean up background processes and resources on shutdown

### Logging
- Use structured JSONL format for all logs
- Log tool calls, LLM requests/responses, token counts
- Logs are buffered and written asynchronously
- Close logger on shutdown to flush buffers

### Performance Considerations
- Background indexing runs asynchronously with 30s timeout
- Context pruning happens proactively at 80% capacity
- LSP operations may be slow for large files (timeout appropriately)
- Embedding generation is optional and requires external server

## File Structure Highlights

```
internal/
├── agent/           # Main agent loop, self-checking
├── checkpoint/      # Session persistence (SQLite)
├── codegraph/       # Symbol relationship tracking
├── config/          # YAML config loading
├── confirmation/    # User confirmation system
├── context/         # Token budget management
├── embeddings/      # Vector store for semantic search
├── initialization/  # First-run project analysis
├── llm/             # LLM client + server management
├── logging/         # Async JSONL logging
├── lsp/             # LSP client (manager, pathfinder)
├── memory/          # Long-term memory (facts, decisions)
├── parser/          # Language detection
├── prompts/         # System prompt templates
├── retrieval/       # Hybrid search (BM25/semantic/trigram)
├── telemetry/       # OpenTelemetry tracing
├── theme/           # Synthwave color formatting
└── tools/           # Tool registry and implementations
```

## Troubleshooting Tips

### llama-server issues
- Verify model path in config.yaml exists
- Check `llama-server --version` to ensure it's in PATH
- Ensure port 8080 is not in use: `netstat -ano | findstr :8080`
- Check GPU VRAM availability for large models
- Review startup timeout setting if slow to initialize

### LSP not working
- Verify LSP server is installed: `gopls version`, `pylsp --version`
- Check command is correct in config.yaml `lsp.servers.<language>.command`
- Ensure workspace has project files (go.mod, package.json, etc.)
- LSP logs written to `.gocode/lsp-<language>.log`

### Context overflow
- Increase `llm.server.ctx_size` in config.yaml
- Use more aggressive KV cache quantization (q4_1 instead of q8_0)
- Lower `llm.max_tokens` to reserve more space for history
- Reduce pruning threshold in context manager

### Memory/performance issues
- Reduce `ctx_size` to use less RAM
- Lower `n_gpu_layers` to offload to CPU
- Disable optional features (retrieval, embeddings, telemetry)
- Use smaller quantized models

## Windows-Specific Notes

- Uses Windows-style paths (`C:\Users\...`)
- readline history stored in `.gocode_history` (deleted in current state)
- Bash commands execute via Git Bash or WSL if available
- `setup-path.bat` adds executable to Windows PATH
