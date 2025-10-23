# Quick Start Guide

## First Time Setup (5 minutes)

### 1. Build
```powershell
cd C:\Users\Jake\coder
go build -o coder.exe cmd/coder/main.go
```

### 2. Configure Model Path
Edit `config.yaml`:
```yaml
llm:
  server:
    model_path: "C:\\models\\Qwen3-Yoyo-V3-42B-A3B-Thinking\\Qwen3-Yoyo-V3-42B-A3B-Thinking.gguf"
```

### 3. Add to PATH
```powershell
.\setup-path.bat
```

### 4. Restart PowerShell
**IMPORTANT:** Close and reopen PowerShell for PATH changes to take effect.

### 5. Test
```powershell
coder --version
# Should show: Coder Agent v1.0.0
```

## Daily Usage

```powershell
# Navigate to your project
cd C:\Projects\MyAwesomeApp

# Start the agent
coder

# The agent will:
# 1. Auto-start llama-server (if needed)
# 2. Create TODO.md in your project directory
# 3. Save logs to C:\Users\Jake\coder\logs\

# Start coding!
❯ help me implement a user authentication system

# Exit when done
❯ exit
```

## Commands

- `coder` - Start the agent
- `coder --version` - Show version
- `coder --help` - Show help
- `coder --config <path>` - Use custom config

## File Locations

When you run `coder` from `C:\Projects\MyApp`:
- **Config**: `C:\Users\Jake\coder\config.yaml`
- **Logs**: `C:\Users\Jake\coder\logs\` (all projects)
- **TODO.md**: `C:\Projects\MyApp\TODO.md` (this project)

## Troubleshooting

### "coder is not recognized"
1. Did you run `setup-path.bat`?
2. Did you **restart PowerShell** after?
3. Verify: `$env:Path -split ';' | Select-String "coder"`

### llama-server fails to start
1. Check model path in `config.yaml`
2. Verify llama-server is in PATH: `llama-server --version`
3. Check logs in `C:\Users\Jake\coder\logs\`

### Can't find config
Set environment variable:
```powershell
$env:CODER_CONFIG = "C:\Users\Jake\coder\config.yaml"
```

## Tips

- Each project gets its own TODO.md for task tracking
- All logs are centralized for GRPO fine-tuning
- You can have multiple projects using the same agent
- The agent reuses existing llama-server if already running
- Use `read`, `write`, `edit`, `glob`, `grep` tools freely
- Confirmations default to interactive mode (safe)

## Next Steps

- Read [README.md](README.md) for full documentation
- Check [INSTALL.md](INSTALL.md) for advanced installation
- See [config.yaml](config.yaml) for all configuration options
- Review [CLAUDE.md](CLAUDE.md) for architecture details
