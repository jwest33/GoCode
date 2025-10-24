<div align="center">

<pre style="background: transparent; border: none; padding: 0; margin: 0;">
  <span style="color: #00d9ff;">╔═══════════════════════════════════════════════════════╗</span>
  <span style="color: #00d9ff;">║</span>                                                       <span style="color: #00d9ff;">║</span>
  <span style="color: #00d9ff;">║</span>            <span style="color: #00d9ff; font-weight: bold;">░█▀▀█ █▀▀█ ░█▀▀█ █▀▀█ █▀▀▄ █▀▀</span>             <span style="color: #00d9ff;">║</span>
  <span style="color: #00d9ff;">║</span>            <span style="color: #ff00ff; font-weight: bold;">░█░▄▄░█░░█░░█░░░░█░░█░█░░█░█▀▀</span>             <span style="color: #00d9ff;">║</span>
  <span style="color: #00d9ff;">║</span>            <span style="color: #00d9ff; font-weight: bold;">░█▄▄█ ▀▀▀▀ ░█▄▄█ ▀▀▀▀ ▀▀▀  ▀▀▀</span>             <span style="color: #00d9ff;">║</span>
  <span style="color: #00d9ff;">║</span>                                                       <span style="color: #00d9ff;">║</span>
  <span style="color: #00d9ff;">║</span>               <span style="color: #ff00ff;">AI-Powered Development Assistant</span>        <span style="color: #00d9ff;">║</span>
  <span style="color: #00d9ff;">║</span>                                                       <span style="color: #00d9ff;">║</span>
  <span style="color: #00d9ff;">╚═══════════════════════════════════════════════════════╝</span>
</pre>

<h3>
  <span style="color: #00d9ff;">Local AI Coding Agent</span> •
  <span style="color: #ff00ff;">llama.cpp Integration</span> •
  <span style="color: #00ff9f;">Production Ready</span>
</h3>

[![License](https://img.shields.io/badge/license-MIT-ff00ff.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23+-00d9ff.svg)](https://go.dev)
[![llama.cpp](https://img.shields.io/badge/llama.cpp-compatible-00ff9f.svg)](https://github.com/ggerganov/llama.cpp)

</div>

---

## Overview

GoCode is an autonomous AI development assistant that runs entirely on your machine. It features automatic llama-server management, intelligent project analysis, hybrid retrieval systems, and a full suite of development tools with configurable human-in-the-loop confirmation.

<table>
<tr>
<td width="50%">

**Core Capabilities**
- Zero external API dependencies
- Automatic project analysis
- Context window management
- Session persistence
- Structured JSONL logging

</td>
<td width="50%">

**Infrastructure**
- llama.cpp server management
- LSP integration (6+ languages)
- Hybrid retrieval (BM25 + semantic + trigram)
- OpenTelemetry tracing
- Long-term memory system

</td>
</tr>
</table>

---

## <span style="color: #ff00ff;">▸</span> Features

### <span style="color: #00d9ff;">Project Initialization</span>

<table>
<tr><td width="30%"><b>First-run detection</b></td><td>Automatically analyzes projects on first launch</td></tr>
<tr><td><b>Language detection</b></td><td>Identifies primary languages and frameworks (Go, Python, TypeScript, Rust, Java, C/C++, C#)</td></tr>
<tr><td><b>Framework recognition</b></td><td>Detects package.json, go.mod, requirements.txt, Cargo.toml, pom.xml, etc.</td></tr>
<tr><td><b>Smart recommendations</b></td><td>Suggests LSP servers, semantic search, checkpointing based on project characteristics</td></tr>
<tr><td><b>Cached analysis</b></td><td>Stores results in <code>.gocode/analysis.json</code> for fast subsequent runs</td></tr>
<tr><td><b>Background indexing</b></td><td>Builds search indices asynchronously</td></tr>
</table>

### <span style="color: #00d9ff;">Context & Memory Management</span>

<table>
<tr><td width="30%"><b>Budget-based allocation</b></td><td>Intelligent token distribution across system, user, context, history, and response</td></tr>
<tr><td><b>Automatic pruning</b></td><td>Removes old messages at 80% context capacity</td></tr>
<tr><td><b>Long-term memory</b></td><td>Persistent SQLite store for facts, artifacts, decisions, patterns, and errors</td></tr>
<tr><td><b>Session checkpointing</b></td><td>Resume conversations across restarts with thread management</td></tr>
<tr><td><b>Context window support</b></td><td>Handles up to 1M+ token contexts with proper KV cache quantization</td></tr>
</table>

### <span style="color: #00d9ff;">Code Navigation</span>

<table>
<tr><td width="30%"><b>LSP integration</b></td><td>Find definitions, references, and symbols using Language Server Protocol</td></tr>
<tr><td><b>CodeGraph tracking</b></td><td>Symbol relationship graphs with call hierarchies</td></tr>
<tr><td><b>Semantic search</b></td><td>Vector embeddings for conceptual code search (optional, requires embedding server)</td></tr>
<tr><td><b>Hybrid retrieval</b></td><td>Combines BM25 keyword search, trigram fuzzy matching, and semantic vectors</td></tr>
<tr><td><b>Configurable weights</b></td><td>Tune retrieval fusion (default: 40% BM25, 50% semantic, 10% trigram)</td></tr>
</table>

### <span style="color: #00d9ff;">llama.cpp Integration</span>

<table>
<tr><td width="30%"><b>Auto-managed server</b></td><td>Automatically starts/stops llama-server with optimal settings</td></tr>
<tr><td><b>Smart reuse</b></td><td>Detects existing llama-server instances and reuses them</td></tr>
<tr><td><b>KV cache quantization</b></td><td>Supports f32, f16, bf16, q8_0, q4_0, q4_1, iq4_nl, q5_0, q5_1 for both K and V caches</td></tr>
<tr><td><b>Flash Attention</b></td><td>Enabled for quantized V cache with extended context support</td></tr>
<tr><td><b>GPU offloading</b></td><td>Configurable layer distribution (n_gpu_layers)</td></tr>
<tr><td><b>Custom parameters</b></td><td>Full control over ctx_size, batch_size, threads, repeat penalty, etc.</td></tr>
</table>

### <span style="color: #00d9ff;">Tool Suite</span>

<details>
<summary><b>File Operations</b></summary>

| Tool | Description |
|------|-------------|
| `read` | Read files with line numbers and multimodal support (images, PDFs, notebooks) |
| `write` | Create new files |
| `edit` | Precise string replacement edits |
| `glob` | Pattern-based file discovery |
| `grep` | Regex search with ripgrep backend |

</details>

<details>
<summary><b>Command Execution</b></summary>

| Tool | Description |
|------|-------------|
| `bash` | Execute shell commands with timeout support |
| `bash_output` | Stream output from background processes |
| `kill_shell` | Terminate background jobs |

</details>

<details>
<summary><b>Web & Productivity</b></summary>

| Tool | Description |
|------|-------------|
| `web_fetch` | Fetch and analyze web content |
| `web_search` | Search the web for information |
| `todo_write` | Task management persisted to TODO.md |

</details>

<details>
<summary><b>Code Navigation (LSP)</b></summary>

| Tool | Requires |
|------|----------|
| `lsp_find_definition` | LSP server for language |
| `lsp_find_references` | LSP server for language |
| `lsp_list_symbols` | LSP server for language |

</details>

### <span style="color: #00d9ff;">Observability & Persistence</span>

<table>
<tr><td width="30%"><b>Structured logging</b></td><td>Async JSONL logs with user inputs, LLM requests/responses, tool calls, and token counts</td></tr>
<tr><td><b>OpenTelemetry tracing</b></td><td>GenAI semantic conventions for LLM operations</td></tr>
<tr><td><b>Session checkpointing</b></td><td>SQLite-backed conversation persistence with thread management</td></tr>
<tr><td><b>Telemetry export</b></td><td>Trace data stored locally for analysis</td></tr>
<tr><td><b>Performance tracking</b></td><td>Token usage, timing, and operation metadata</td></tr>
</table>

### <span style="color: #00d9ff;">Developer Experience</span>

<table>
<tr><td width="30%"><b>Synthwave theme</b></td><td>Cyan, pink, purple, green, red, yellow color palette</td></tr>
<tr><td><b>Human-in-the-loop</b></td><td>Configurable confirmation modes (interactive, auto, destructive_only)</td></tr>
<tr><td><b>Readline support</b></td><td>Command history with <code>.gocode_history</code></td></tr>
<tr><td><b>Clean shutdown</b></td><td>Graceful cleanup with deferred resource management</td></tr>
<tr><td><b>Flexible deployment</b></td><td>Run from any directory with automatic config discovery</td></tr>
</table>

---

## <span style="color: #ff00ff;">▸</span> Quick Start

### Prerequisites

<table>
<tr>
<td width="20%"><b>Go 1.23+</b></td>
<td><a href="https://go.dev/dl/">Download</a></td>
</tr>
<tr>
<td><b>llama-server</b></td>
<td>

Install llama.cpp and add `llama-server` to PATH

```bash
git clone https://github.com/ggerganov/llama.cpp
cd llama.cpp
cmake -B build -DGGML_CUDA=ON -DGGML_CUDA_FA_ALL_QUANTS=ON
cmake --build build --config Release
# Add build/bin to PATH
```

</td>
</tr>
<tr>
<td><b>GGUF model</b></td>
<td>Download a quantized model (e.g., Qwen, Llama, Mistral)</td>
</tr>
</table>

### Installation

```bash
# Clone repository
git clone https://github.com/jwest33/gocode
cd gocode

# Configure model path in config.yaml
# Edit: llm.server.model_path

# Build executable
go build -o gocode.exe cmd/gocode/main.go

# Add to PATH (Windows)
.\setup-path.bat
# Then restart PowerShell

# Verify installation
gocode --version
```

### First Run

```bash
cd C:\Projects\YourProject
gocode
```

**On first run, GoCode will:**
1. Detect this is a new project
2. Prompt for project initialization
3. Analyze languages, frameworks, and structure
4. Display project summary with feature recommendations
5. Start background indexing
6. Launch the interactive agent

<details>
<summary><b>Example initialization output</b></summary>

```
┌─────────────────────────────────────────┐
│  Project Analysis: YourProject          │
└─────────────────────────────────────────┘

Languages:
  • Go (primary) - 45 files
  • JavaScript - 12 files

Frameworks:
  • Go modules (go.mod)
  • Node.js (package.json)

Structure:
  • 127 total files
  • 89 code files
  • 3,450 lines of code

Recommendations:
  ✓ LSP server detected: gopls (installed)
  ⚠ Install typescript-language-server for enhanced TypeScript support
  💡 Enable semantic search for large codebase (89+ files)
  💡 Enable session checkpointing for complex projects
```

</details>

---

## <span style="color: #ff00ff;">▸</span> Configuration

**Config search order:**
1. `--config` flag
2. `GOCODE_CONFIG` environment variable
3. Current working directory
4. Executable's directory
5. `~/.gocode/config.yaml`

### Complete Reference

<details open>
<summary><b>LLM Configuration</b></summary>

```yaml
llm:
  endpoint: "http://localhost:8080/v1"
  api_key: "secret"
  model: "qwen"
  temperature: 0.7
  max_tokens: 4096
  context_window: 102400

  # llama-server Auto-Management
  auto_manage: true  # Set false to use external server
  startup_timeout: 60  # Seconds to wait for server startup

  # llama-server Configuration
  server:
    model_path: "C:\\models\\your-model.gguf"
    host: "0.0.0.0"
    port: 8080
    ctx_size: 102400
    flash_attn: true  # Enable Flash Attention
    jinja: true  # Use Jinja templates

    # KV Cache Quantization (f32, f16, bf16, q8_0, q4_0, q4_1, iq4_nl, q5_0, q5_1)
    cache_type_k: "q8_0"  # Use q4_1/q5_1 for more memory savings
    cache_type_v: "q8_0"  # Requires flash_attn for quantized V cache

    batch_size: 1024
    ubatch_size: 512
    n_cpu_moe: 28  # For MoE models
    n_gpu_layers: 99  # GPU layer offloading
    repeat_last_n: 192
    repeat_penalty: 1.05
    threads: 16
```

</details>

<details>
<summary><b>Tool Configuration</b></summary>

```yaml
tools:
  enabled:
    - read
    - write
    - edit
    - glob
    - grep
    - bash
    - bash_output
    - kill_shell
    - todo_write
    - web_fetch
    - web_search
```

</details>

<details>
<summary><b>Confirmation Policies</b></summary>

```yaml
confirmation:
  mode: "interactive"  # Options: interactive, auto, destructive_only
  auto_approve_tools:
    - read
    - glob
    - grep
  always_confirm_tools:
    - write
    - edit
    - bash
```

</details>

<details>
<summary><b>Logging Configuration</b></summary>

```yaml
logging:
  format: "jsonl"
  directory: "logs"  # Relative to config.yaml location
  level: "info"  # debug, info, warn, error
  log_tool_results: true
  log_reasoning: true
```

</details>

<details>
<summary><b>Embeddings Configuration</b></summary>

```yaml
embeddings:
  enabled: false  # Requires embedding server
  endpoint: "http://localhost:8081"
  dimension: 384  # For nomic-embed-text or bge-small-en-v1.5
  db_path: "embeddings.db"
```

</details>

<details>
<summary><b>Retrieval Configuration</b></summary>

```yaml
retrieval:
  enabled: false  # Enable for large codebases
  weights:
    bm25: 0.4      # Keyword search weight
    semantic: 0.5  # Semantic search weight (requires embeddings)
    trigram: 0.1   # Fuzzy matching weight
```

</details>

<details>
<summary><b>LSP Configuration</b></summary>

```yaml
lsp:
  enabled: false  # Enable for LSP-based navigation
  servers:
    go:
      command: "gopls"
      args: []
    python:
      command: "pylsp"
      args: []
    typescript:
      command: "typescript-language-server"
      args: ["--stdio"]
    rust:
      command: "rust-analyzer"
      args: []
    java:
      command: "jdtls"
      args: []
    cpp:
      command: "clangd"
      args: []
```

</details>

<details>
<summary><b>Checkpoint Configuration</b></summary>

```yaml
checkpoint:
  enabled: false
  db_path: "checkpoints.db"
  auto_save: true
  save_interval: 5  # Auto-save every N messages
```

</details>

<details>
<summary><b>Memory Configuration</b></summary>

```yaml
memory:
  enabled: false
  db_path: "memory.db"
```

</details>

<details>
<summary><b>Telemetry Configuration</b></summary>

```yaml
telemetry:
  enabled: false
  service_name: "gocode-agent"
  db_path: "traces.db"
```

</details>

<details>
<summary><b>Evaluation Configuration</b></summary>

```yaml
evaluation:
  enabled: false
  track_metrics: true
```

</details>

---

## <span style="color: #ff00ff;">▸</span> Usage

### Basic Interaction

```bash
gocode
```

**Example prompts:**
```
❯ help me refactor the authentication logic in auth.go

❯ find all TODO comments in the codebase

❯ run the tests and fix any failures

❯ exit
```

### Commands

| Command | Description |
|---------|-------------|
| `exit` / `quit` | Exit the agent |
| `gocode --version` | Show version |
| `gocode --config <path>` | Use custom config |

### File Path Management

| File Type | Location |
|-----------|----------|
| **Logs** | `<config-dir>/logs/` (centralized) |
| **TODO.md** | Current working directory (project-specific) |
| **State files** | `.gocode/` in working directory |

**Example:**
```bash
cd C:\Projects\MyApp
gocode  # Config at C:\Users\Jake\gocode\config.yaml
```

Results:
- Logs → `C:\Users\Jake\gocode\logs/session_2025-01-15_14-30-22.jsonl`
- TODO → `C:\Projects\MyApp\TODO.md`
- State → `C:\Projects\MyApp\.gocode/state.json`

### Confirmation Modes

| Mode | Behavior |
|------|----------|
| **Interactive** (default) | Auto-approves: read, glob, grep<br>Always confirms: write, edit, bash |
| **Auto** | Never prompts (use with caution) |
| **Destructive-only** | Only confirms write, edit, bash |

---

## <span style="color: #ff00ff;">▸</span> Extended Features

### Session Checkpointing

Resume conversations across restarts:

```yaml
checkpoint:
  enabled: true
  db_path: "checkpoints.db"
  auto_save: true
  save_interval: 5  # Messages between auto-saves
```

**Data stored:**
- Complete message history
- Tool call sequences
- Thread metadata
- Timestamps

### Long-term Memory

Persistent knowledge across sessions:

```yaml
memory:
  enabled: true
  db_path: "memory.db"
```

**Memory types:**
- **Facts** - Learned information about the codebase
- **Artifacts** - Important files and configurations
- **Decisions** - Architectural choices and rationale
- **Patterns** - Common code patterns and conventions
- **Errors** - Past bugs and solutions

### Hybrid Retrieval

Intelligent context retrieval for large codebases:

```yaml
retrieval:
  enabled: true
  weights:
    bm25: 0.4      # Keyword matching
    semantic: 0.5  # Conceptual similarity
    trigram: 0.1   # Fuzzy string matching
```

**Components:**
- **BM25** - Traditional keyword search (no dependencies)
- **Semantic** - Vector embeddings (requires embedding server on port 8081)
- **Trigram** - Fuzzy matching for typos and variations

**Start embedding server:**
```bash
llama-embedding --model nomic-embed-text-v1.5.gguf --port 8081
```

### LSP Integration

Code-aware navigation with Language Server Protocol:

```yaml
lsp:
  enabled: true
  servers:
    go:
      command: "gopls"
      args: []
```

**Supported languages:** Go, Python, TypeScript, Rust, Java, C/C++, C#

**Install LSP servers:**
```bash
# Go
go install golang.org/x/tools/gopls@latest

# Python
pip install python-lsp-server

# TypeScript
npm install -g typescript-language-server

# Rust
rustup component add rust-analyzer
```

### OpenTelemetry Tracing

Monitor LLM operations with distributed tracing:

```yaml
telemetry:
  enabled: true
  service_name: "gocode-agent"
  db_path: "traces.db"
```

**Captures:**
- Request/response timing
- Token counts (prompt, completion, total)
- Model parameters
- Tool execution spans
- GenAI semantic attributes

---

## <span style="color: #ff00ff;">▸</span> Architecture

### Internal Packages

```
internal/
├── agent/          # Main agent loop, tool coordination, message management
├── checkpoint/     # SQLite-backed session persistence
├── codegraph/      # Symbol tracking with definition/reference graphs
├── config/         # YAML configuration with env var overrides
├── confirmation/   # User confirmation system for tool execution
├── context/        # Context window budget management and pruning
├── embeddings/     # Vector store with chunking and similarity search
├── initialization/ # Project analysis, language/framework detection
├── llm/            # OpenAI-compatible client, llama-server management
├── logging/        # Async JSONL logging with buffered writes
├── lsp/            # Language Server Protocol client
├── memory/         # Long-term memory store (facts, decisions, patterns)
├── parser/         # Simple language detection and code parsing
├── prompts/        # Template-based system prompt rendering
├── retrieval/      # Hybrid search (BM25, semantic, trigram fusion)
├── telemetry/      # OpenTelemetry tracing with GenAI conventions
├── theme/          # Synthwave color palette and formatting
└── tools/          # Tool registry and implementations
```

### Data Flow

```
User Input
  ↓
Agent (conversation loop)
  ↓
Context Manager (budget allocation, pruning)
  ↓
Retrieval (optional context injection)
  ↓
LLM Client (OpenAI-compatible API)
  ↓
llama-server (auto-managed or external)
  ↓
Response + Tool Calls
  ↓
Confirmation System (policy-based approval)
  ↓
Tool Execution (parallel or sequential)
  ↓
Results → Agent → Next iteration
  ↓
Logging (async JSONL)
Telemetry (OpenTelemetry spans)
Checkpoint (session persistence)
Memory (long-term storage)
```

---

## <span style="color: #ff00ff;">▸</span> Development

### Building from Source

```bash
git clone https://github.com/jwest33/gocode
cd gocode
go build -o gocode.exe cmd/gocode/main.go
```

### Project Structure

```
gocode/
├── cmd/gocode/           # Main entry point
├── internal/             # Internal packages (see Architecture)
├── config.yaml           # Default configuration
├── setup-path.bat        # Windows PATH setup script
└── README.md             # This file
```

### Testing

```bash
go test ./...
```

### Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

---

## <span style="color: #ff00ff;">▸</span> Troubleshooting

<details>
<summary><b>llama-server fails to start</b></summary>

**Symptoms:** Agent exits with "llama-server failed to start"

**Solutions:**
1. Verify model path in `config.yaml` is correct and file exists
2. Check llama-server is in PATH: `llama-server --version`
3. Ensure port 8080 is not already in use: `netstat -ano | findstr :8080`
4. Verify GPU has enough VRAM (check with `nvidia-smi`)
5. Review llama-server compatibility with model format

</details>

<details>
<summary><b>Server already running message</b></summary>

**Behavior:** "llama-server is already running and responding"

**Explanation:** GoCode detected an existing llama-server instance on the configured port and will reuse it. The server won't be stopped when GoCode exits.

**Action:** No action needed. To force a fresh start, manually stop the existing server first.

</details>

<details>
<summary><b>Context window overflow</b></summary>

**Symptoms:** Errors about context being too large

**Solutions:**
1. Increase `ctx_size` in config.yaml
2. Enable more aggressive KV cache quantization (q4_1 instead of q8_0)
3. Lower the context pruning threshold in context manager
4. Reduce `max_tokens` to reserve more space for conversation history

</details>

<details>
<summary><b>LSP tools not working</b></summary>

**Symptoms:** "LSP server not responding" or undefined references

**Solutions:**
1. Verify LSP server is installed: `gopls version` / `pylsp --version`
2. Enable LSP in config: `lsp.enabled: true`
3. Check command is correct in LSP server configuration
4. Ensure workspace has project files (go.mod, package.json, etc.)
5. Check LSP server logs in `.gocode/lsp-<language>.log`

</details>

<details>
<summary><b>Project initialization skipped</b></summary>

**Symptoms:** No project analysis on first run

**Cause:** `.gocode/state.json` exists with `skip_init: true`

**Solution:** Delete `.gocode/state.json` to retrigger initialization

</details>

<details>
<summary><b>Out of memory errors</b></summary>

**Solutions:**
1. Reduce `ctx_size` (e.g., 32768 instead of 102400)
2. Use more aggressive KV cache quantization (q4_1, q5_1)
3. Lower `n_gpu_layers` to use more CPU RAM
4. Reduce `batch_size` and `ubatch_size`
5. Disable flash_attn if causing issues

</details>

<details>
<summary><b>Embedding server connection failed</b></summary>

**Symptoms:** "Failed to connect to embedding endpoint"

**Solutions:**
1. Start an embedding server on configured port (default 8081)
2. Verify endpoint in config: `embeddings.endpoint`
3. Check server is responding: `curl http://localhost:8081/health`
4. Disable embeddings if not needed: `embeddings.enabled: false`

</details>

<details>
<summary><b>Performance issues</b></summary>

**Symptoms:** Slow responses, high latency

**Optimizations:**
1. Enable GPU offloading: increase `n_gpu_layers`
2. Use quantized KV cache: `cache_type_k: q4_1`, `cache_type_v: q4_1`
3. Increase `batch_size` and `ubatch_size`
4. Use Flash Attention: `flash_attn: true`
5. Reduce `context_window` if not using long contexts
6. Disable unused features (retrieval, LSP, telemetry)

</details>

---

## <span style="color: #ff00ff;">▸</span> Resources

- **llama.cpp**: https://github.com/ggerganov/llama.cpp
- **Model Hub**: https://huggingface.co/models?library=gguf
- **OpenTelemetry**: https://opentelemetry.io/
- **LSP Specification**: https://microsoft.github.io/language-server-protocol/
- **Issues**: https://github.com/jwest33/gocode/issues

---

## <span style="color: #ff00ff;">▸</span> License

MIT License - See LICENSE file for details

---

<div align="center">

**Built for local AI development**

<sub>Cyan • Pink • Purple • Green • Red • Yellow</sub>

</div>
