# Project Architecture

## Directory Structure

```
gocode/
├── cmd/
│   └── gocode/
│       └── main.go              # Entry point
├── internal/
│   ├── agent/
│   │   └── agent.go             # Main conversation loop
│   ├── config/
│   │   └── config.go            # Configuration loading
│   ├── llm/
│   │   └── client.go            # OpenAI-compatible client
│   ├── tools/
│   │   ├── tools.go             # Tool registry
│   │   ├── read.go              # File reading
│   │   ├── write.go             # File writing
│   │   ├── edit.go              # File editing
│   │   ├── glob.go              # File pattern matching
│   │   ├── grep.go              # Content search
│   │   ├── bash.go              # Command execution
│   │   ├── bash_output.go       # Background process output
│   │   ├── kill_shell.go        # Process termination
│   │   ├── todo_write.go        # Task management
│   │   ├── web_fetch.go         # HTTP fetching
│   │   └── web_search.go        # Web search (placeholder)
│   ├── confirmation/
│   │   └── confirmation.go      # Human-in-the-loop system
│   └── logging/
│       └── logging.go           # JSONL logging
├── docs/
│   ├── architecture.md          # This file
│   └── qwen_template.md         # Chat template info
├── logs/                        # Session logs (JSONL)
├── config.yaml                  # Configuration
├── build.bat                    # Build script
├── CLAUDE.md                    # Claude Code guidance
├── README.md                    # User documentation
└── go.mod                       # Go dependencies

```

## Component Overview

### 1. Main Entry (`cmd/gocode/main.go`)
- Loads configuration from `config.yaml`
- Creates agent instance
- Starts interactive REPL

### 2. Agent (`internal/agent/agent.go`)
- Core conversation loop
- Manages message history
- Coordinates LLM, tools, and confirmations
- Handles tool execution and results

### 3. LLM Client (`internal/llm/client.go`)
- OpenAI-compatible API client
- Communicates with llama-server
- Handles function calling format
- Supports streaming (future enhancement)

### 4. Tool System (`internal/tools/`)
- **Registry**: Central tool registration and execution
- **File Tools**: Read, Write, Edit for file operations
- **Search Tools**: Glob (pattern), Grep (content)
- **Execution Tools**: Bash with background process support
- **Management Tools**: TodoWrite for task tracking
- **Web Tools**: WebFetch for HTTP, WebSearch (TBD)

Each tool implements the `Tool` interface:
```go
type Tool interface {
    Name() string
    Description() string
    Parameters() map[string]interface{}
    Execute(ctx context.Context, args string) (string, error)
}
```

### 5. Confirmation System (`internal/confirmation/`)
- Interactive CLI prompts for tool execution
- Configurable policies (auto, interactive, destructive_only)
- Per-tool approval/rejection

### 6. Logging (`internal/logging/`)
- JSONL format for GRPO fine-tuning
- Captures:
  - User inputs
  - LLM requests/responses
  - Tool calls/results
  - Metadata (timestamps, token counts)

## Data Flow

```
User Input
    ↓
[Agent] → Add to message history
    ↓
[LLM Client] → Send to llama-server with tool definitions
    ↓
[llama-server] → Generate response with tool calls
    ↓
[Agent] → Parse response
    ↓
    ├─→ [Confirmation] → Prompt user (if configured)
    ↓
    ├─→ [Tool Registry] → Execute tool
    ↓
    └─→ [Logger] → Log everything
    ↓
Loop until finish_reason = "stop"
    ↓
Display final response
```

## Configuration Flow

```
config.yaml
    ↓
[Config Loader]
    ↓
    ├─→ LLM Config → Client creation
    ├─→ Tools Config → Tool registration
    ├─→ Confirmation Config → Confirmation system
    └─→ Logging Config → Logger initialization
```

## Logging Format

Each log entry in `logs/session_*.jsonl`:

```json
{
  "timestamp": "2025-10-23T10:30:00Z",
  "type": "user_input|llm_request|llm_response|tool_call|tool_result",
  "role": "user|assistant|tool",
  "content": "...",
  "tool_name": "read",
  "tool_args": "{...}",
  "tool_result": "...",
  "metadata": {...}
}
```

## Extension Points

### Adding New Tools
1. Create file in `internal/tools/`
2. Implement `Tool` interface
3. Register in `agent.New()`
4. Add to `config.yaml` enabled list

### Adding LLM Providers
1. Implement client in `internal/llm/`
2. Support OpenAI-compatible API format
3. Update config to select provider

### Custom Confirmation Policies
1. Extend `internal/confirmation/`
2. Add policy to config
3. Implement policy logic

## Performance Considerations

- **Context Window**: 100K+ tokens tracked via message history
- **Background Processes**: Managed via `bash` tool for long-running commands
- **Logging**: Asynchronous writes (future enhancement)
- **Memory**: Message history grows indefinitely (add pruning if needed)

## Security Considerations

- API keys stored in config (use environment variables for production)
- File operations unrestricted (add path validation for sandboxing)
- Command execution via `bash` tool (use confirmation for safety)
- No input sanitization on tool args (validate in production)
