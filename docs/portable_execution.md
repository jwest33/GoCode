# Portable Execution Enhancement

## Problem
The agent could only be run from `C:\Users\Jake\coder` because config.yaml was hardcoded to be loaded from the current directory.

## Solution
Implemented smart config file discovery with multiple fallback locations.

## Changes Made

### 1. Config Path Search (`cmd/coder/main.go`)

**Search Order:**
1. `--config` command-line flag
2. `CODER_CONFIG` environment variable
3. Current working directory (`./config.yaml`)
4. Executable's directory (`<exe-dir>/config.yaml`)
5. User's home directory (`~/.coder/config.yaml`)

**New Flags:**
- `--config <path>` - Specify custom config location
- `--version` - Show version and exit
- `--help` - Show usage (default flag behavior)

### 2. Base Directory Tracking (`internal/config/config.go`)

Added `BaseDir` field to Config struct:
```go
type Config struct {
    ...
    BaseDir string `yaml:"-"` // Set at runtime
}
```

This stores the directory containing the config file, used to resolve relative paths.

### 3. Path-Aware Logging (`internal/logging/logging.go`)

Modified `New()` signature:
```go
func New(cfg *config.LoggingConfig, baseDir string) (*Logger, error)
```

Resolves log directory relative to config location:
- Config: `C:\Users\Jake\coder\config.yaml`
- Log dir: `logs`
- Result: `C:\Users\Jake\coder\logs\`

### 4. Path-Aware TODO.md (`internal/agent/agent.go`)

TODO.md path resolved relative to **current working directory**:
```go
todoPath := filepath.Join(cfg.WorkingDir, "TODO.md")
```

This means each project gets its own TODO.md while logs are centralized.

## Usage Examples

### Example 1: Run from Executable Directory
```bash
cd C:\Users\Jake\coder
coder.exe
# Config: C:\Users\Jake\coder\config.yaml
# Logs: C:\Users\Jake\coder\logs\
# TODO.md: C:\Users\Jake\coder\TODO.md
```

### Example 2: Run from Any Directory (Executable in PATH)
```bash
cd C:\Projects\MyCode
coder
# Config: C:\Users\Jake\coder\config.yaml
# Logs: C:\Users\Jake\coder\logs\
# TODO.md: C:\Projects\MyCode\TODO.md ← Project-specific!
```

### Example 3: Custom Config Path
```bash
cd C:\anywhere
coder --config D:\my-configs\coder-config.yaml
# Config: D:\my-configs\coder-config.yaml
# Logs: D:\my-configs\logs\
# TODO.md: C:\anywhere\TODO.md
```

### Example 4: Environment Variable
```bash
set CODER_CONFIG=C:\Users\Jake\coder\config.yaml
cd C:\MyProject
coder
# Config: C:\Users\Jake\coder\config.yaml
# Logs: C:\Users\Jake\coder\logs\
# TODO.md: C:\MyProject\TODO.md
```

### Example 5: Home Directory Installation
```bash
mkdir %USERPROFILE%\.coder
copy config.yaml %USERPROFILE%\.coder\
copy coder.exe %USERPROFILE%\.coder\
cd C:\Projects\WebApp
coder
# Config: C:\Users\Jake\.coder\config.yaml
# Logs: C:\Users\Jake\.coder\logs\
# TODO.md: C:\Projects\WebApp\TODO.md
```

## Benefits

1. **Portable**: Run from any directory
2. **Flexible**: Multiple config location options
3. **Project-Specific**: Each project gets its own TODO.md for task tracking
4. **Centralized Logs**: All session logs in one searchable location
5. **User-Friendly**: Clear error messages if config not found
6. **Standard**: Follows common CLI tool patterns (--config, env vars, ~/.app/)

## Files Modified

1. `cmd/coder/main.go` - Config search logic
2. `internal/config/config.go` - Added BaseDir field
3. `internal/logging/logging.go` - Path-aware logging
4. `internal/agent/agent.go` - Path-aware TODO.md
5. `README.md` - Usage documentation
6. `docs/portable_execution.md` - This file

## Testing Checklist

- [ ] Run from executable directory
- [ ] Run from different directory with exe in PATH
- [ ] Run with --config flag
- [ ] Run with CODER_CONFIG env var
- [ ] Verify logs go to config directory
- [ ] Verify TODO.md in **working directory**
- [ ] Test multiple projects have separate TODO.md files
- [ ] --version flag works
- [ ] --help flag works
- [ ] Clear error if config not found

## Backwards Compatibility

✅ **Fully backwards compatible**

If you were running `./coder.exe` from `C:\Users\Jake\coder`, it still works exactly the same:
- Finds `./config.yaml` (current directory)
- Logs to `./logs/` (same directory as config)
- TODO.md at `./TODO.md` (working directory = config directory)

No changes required for existing workflows!

## Key Insight

**File Placement Strategy:**
- **Logs**: Config directory (centralized, for analysis/fine-tuning across all projects)
- **TODO.md**: Working directory (project-specific, tracks tasks per codebase)

This provides the best of both worlds - centralized logging for GRPO training data, but project-specific task tracking.
