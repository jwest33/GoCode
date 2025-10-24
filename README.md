# GoCode - AI-Powered Development Assistant

A local, fully configurable autonomous coding agent with human-in-the-loop confirmation and comprehensive logging for GRPO fine-tuning.

## Features

- **Auto-managed llama-server** - Automatically starts and stops llama-server
- **Async logging** - Non-blocking JSONL logging for GRPO fine-tuning
- **Full tool suite** - File ops, search, bash execution, web fetch, task management
- **Human-in-the-loop** - Configurable confirmation for tool execution
- **Smart reuse** - Detects existing llama-server and reuses it

## Quick Start

### 1. Prerequisites

- Go 1.23+ installed
- llama-server installed globally (add to PATH)
- Qwen model downloaded: `Qwen3-Yoyo-V3-42B-A3B-Thinking-TOTAL-RECALL-ST-TNG-III.i1-Q4_K_M.gguf`

### 2. Configure

Edit `config.yaml` and set the path to your model:

```yaml
llm:
  auto_manage: true  # Automatically start/stop llama-server
  server:
    model_path: "C:\\path\\to\\your\\model.gguf"
```

All other settings have sensible defaults optimized for RTX 4090/5090.

### 3. Build

```bash
go build -o gocode.exe cmd/gocode/main.go
```

### 4. Add to PATH

**Quick Setup:**
```bash
.\setup-path.bat
```
Then **restart PowerShell**.

See [INSTALL.md](INSTALL.md) for detailed installation options.

### 5. Run from Anywhere

```bash
# After adding to PATH and restarting PowerShell:
gocode --version
cd C:\Projects\MyApp
gocode
```

That's it! The agent will:
1. Check if llama-server is already running
2. If not, start it automatically with optimal settings
3. Wait for it to be ready
4. Start the interactive agent
5. Shutdown llama-server on exit (if it started it)

### Running from Any Directory

**Option 1: Add to PATH**
```bash
# Add C:\Users\Jake\gocode to PATH
gocode  # Run from anywhere!
```

**Option 2: Specify config path**
```bash
cd C:\anywhere
gocode --config C:\Users\Jake\gocode\config.yaml
```

**Option 3: Environment variable**
```bash
set GOCODE_CONFIG=C:\Users\Jake\gocode\config.yaml
gocode  # Automatically finds config
```

**Option 4: Copy to home directory**
```bash
mkdir %USERPROFILE%\.gocode
copy C:\Users\Jake\gocode\config.yaml %USERPROFILE%\.gocode\
copy C:\Users\Jake\gocode\gocode.exe %USERPROFILE%\.gocode\
gocode  # Searches ~/.gocode/config.yaml
```

The agent searches for `config.yaml` in this order:
1. `--config` flag
2. `GOCODE_CONFIG` environment variable
3. Current working directory
4. Executable's directory
5. `~/.gocode/config.yaml`

## Usage

Once running, the agent provides an interactive CLI:

```
❯ help me fix the bug in main.go
```

The agent will:
1. Use tools to read files, search code, run commands
2. Request confirmation for destructive operations (configurable)
3. Log all interactions to `logs/` directory (relative to config.yaml location)

**Additional commands:**
- `gocode --version` - Show version
- `gocode --help` - Show usage help

Type `exit` to quit.

### File Paths

File paths are intelligently managed:
- **Logs**: Always saved to `<config-dir>/logs/` (centralized across all projects)
- **TODO.md**: Saved to current working directory (project-specific task tracking)
- You can run the agent from anywhere!

**Example:**
```bash
cd C:\Projects\MyApp
gocode  # Config found at C:\Users\Jake\gocode\config.yaml
```
Results in:
- Logs → `C:\Users\Jake\gocode\logs\`
- TODO.md → `C:\Projects\MyApp\TODO.md`

## Configuration

### llama-server Management

```yaml
llm:
  auto_manage: true  # Enable auto-management
  startup_timeout: 60  # Seconds to wait for server startup

  server:
    model_path: "path/to/model.gguf"
    # All llama-server flags are configurable
    ctx_size: 102400
    flash_attn: true
    n_gpu_layers: 99
    # ... see config.yaml for full options
```

Set `auto_manage: false` to use an externally managed llama-server.

### Confirmation Modes

```yaml
confirmation:
  mode: "interactive"  # interactive, auto, destructive_only
  auto_approve_tools:
    - read
    - glob
    - grep
  always_confirm_tools:
    - write
    - edit
    - bash
```

### Available Tools

- `read`, `write`, `edit` - File operations
- `glob`, `grep` - Search operations
- `bash`, `bash_output`, `kill_shell` - Command execution with background process support
- `todo_write` - Task tracking (persists to TODO.md)
- `web_fetch`, `web_search` - Web capabilities

## Logging

All sessions are logged asynchronously to `logs/session_YYYY-MM-DD_HH-MM-SS.jsonl` with:
- User inputs
- LLM requests and responses (with prompts and tool definitions)
- Tool calls and results
- Token counts and metadata
- Timestamps for all events

Logs are buffered (1000 entries) and written in the background for optimal performance. Suitable for GRPO fine-tuning pipelines.

## Troubleshooting

### llama-server fails to start

- Verify model path in `config.yaml` is correct
- Check llama-server is in PATH: `llama-server --version`
- Ensure port 8080 is not already in use
- Check GPU has enough VRAM for the model

### Server already running

If llama-server is already running on the configured port, gocode will detect and reuse it. It won't be shutdown when gocode exits.
