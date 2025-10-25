# GoCode Dev Playground

An interactive REPL environment for testing and debugging GoCode agent components in isolation.

## Quick Start

```bash
go run cmd/playground/main.go
```

## Features

- **Interactive Testing**: Test individual components without full agent startup
- **Rapid Debugging**: Isolate issues to specific components
- **Learning Tool**: Explore the codebase interactively
- **Development Aid**: Iterate quickly on new features

## Common Workflows

### 1. Test Basic Setup

```
> config create
✓ Config created
  BaseDir: C:\Users\Jake\gocode\playground
  WorkingDir: C:\Users\Jake\gocode\playground
  LogDir: C:\Users\Jake\gocode\playground\logs

> config validate
✓ BaseDir: C:\Users\Jake\gocode\playground
✓ Log directory: C:\Users\Jake\gocode\playground\logs
✓ LLM endpoint: http://localhost:8080/v1

Validation: 3/3 checks passed
```

### 2. Test Logger

```
> logger init
✓ Logger initialized
  Log directory: C:\Users\Jake\gocode\playground\logs

> logger test
ℹ Testing logger...
✓ Test entries logged
```

### 3. Test Individual Tools

```
> tool read go.mod
ℹ Testing read tool with: go.mod
✓ File read successfully
Preview:
module github.com/jake/gocode

go 1.23
...

> tool bash go version
ℹ Testing bash tool with: go version
✓ Command executed
Output:
go version go1.23.0 windows/amd64

> tool list
═══ Available Tools ═══
   1. read
   2. write
   3. edit
   ...
```

### 4. Test LLM Connection

```
> llm ping
ℹ Testing LLM endpoint: http://localhost:8080/v1
✓ LLM endpoint reachable

> llm complete Hello, how are you?
ℹ Sending completion request...
✓ Completion received
Response: I'm doing well, thank you for asking!
Tokens: 15
```

### 5. Test Agent Creation

```
> agent create
ℹ Creating agent...
✓ Agent created
ℹ Tools registered: 7
```

### 6. Test Autonomous Execution

```
> autonomous simple
ℹ Running simple autonomous task: List the files in the current directory
✓ Task completed successfully
  Iterations: 2
  Tool calls: 1
  Tokens: 234
  Duration: 1.5s
  Final message: Found 42 files in the current directory...

> autonomous task Create a hello.txt file with "Hello World"
ℹ Running custom task: Create a hello.txt file with "Hello World"
✓ Task completed
  Result: I've created hello.txt with the content "Hello World"
```

## Command Reference

### Configuration Commands

- `config create` - Create a minimal config for testing
- `config validate` - Validate the current configuration
- `config show` - Display current configuration

### Logger Commands

- `logger init` - Initialize the logging system
- `logger test` - Test logging with sample entries
- `logger show` - Show log file location

### Tool Commands

- `tool list` - List all available tools
- `tool read <file>` - Test reading a file
- `tool bash <command>` - Test running a bash command
- `tool <name>` - Interactive testing (coming soon)

### LLM Commands

- `llm ping` - Test connection to LLM endpoint
- `llm complete <prompt>` - Test completion with prompt

### Agent Commands

- `agent create` - Create a minimal agent instance
- `agent status` - Show agent status
- `agent prompt <text>` - Send prompt to agent (coming soon)

### Autonomous Commands

- `autonomous simple` - Test with a simple pre-defined task
- `autonomous task <description>` - Run a custom autonomous task

### Utility Commands

- `clear` - Clear the screen
- `help` - Show help message
- `exit` - Exit the playground

## Troubleshooting

### "Failed to create logs directory"

This usually means `BaseDir` is not set. Run `config create` first to initialize a proper configuration.

### "LLM endpoint unreachable"

Make sure llama-server is running:
```bash
# In another terminal
llama-server --model path/to/model.gguf --host localhost --port 8080
```

### "No agent created"

Run the setup sequence:
```
> config create
> logger init
> agent create
```

## Development Tips

1. **Start with config**: Always begin with `config create` to set up a proper environment

2. **Test incrementally**: Test each component before moving to the next:
   ```
   config create → logger init → agent create → autonomous simple
   ```

3. **Use tool shortcuts**: Quick test tools with direct commands:
   ```
   tool read go.mod
   tool bash echo test
   ```

4. **Check logs**: After testing, check the playground/logs directory for detailed logs

5. **Iterate quickly**: The playground lets you test changes without restarting the full agent

## Examples

### Debug a Config Issue

```
> config create
> config validate
✗ Cannot create log directory: ...
# Fix the issue
> config validate
✓ All checks passed
```

### Test a New Tool

```
> tool list
# Find your tool in the list
> tool mytool arg1 arg2
# See if it works
```

### Develop Autonomous Features

```
> agent create
> autonomous task test my feature
# Observe behavior
# Make code changes
# Test again without restarting
```

## Architecture

The playground creates an isolated environment in `playground/` directory:

```
playground/
├── logs/                    # Log files
├── test_files/             # Test data
└── workspaces/             # Temporary workspaces
```

All playground activity is contained here and won't affect your main project files.

## Future Enhancements

- [ ] Batch mode for running commands from file
- [ ] Script mode for automation
- [ ] JSON output for programmatic use
- [ ] Step-by-step agent execution
- [ ] Breakpoint debugging
- [ ] Performance profiling
- [ ] Memory usage tracking

## Contributing

The playground is designed to be extensible. To add new commands:

1. Add command handler in `commands.go`
2. Add case in main switch statement
3. Update help text
4. Document in this README

## License

Part of GoCode project.
