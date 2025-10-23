# Installation Guide

## Quick Setup

### 1. Build the Agent
```powershell
cd C:\Users\Jake\coder
go build -o coder.exe cmd/coder/main.go
```

### 2. Add to PATH

**Option A: Run the Setup Script (Easiest)**
```powershell
.\setup-path.bat
```
Then **restart PowerShell**.

**Option B: Manual PATH Setup**
1. Press `Win + X` and select "System"
2. Click "Advanced system settings"
3. Click "Environment Variables"
4. Under "User variables", find and select "Path"
5. Click "Edit"
6. Click "New"
7. Add: `C:\Users\Jake\coder`
8. Click OK on all dialogs
9. **Restart PowerShell**

**Option C: PowerShell Command (Admin Required)**
```powershell
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Users\Jake\coder", "User")
```
Then **restart PowerShell**.

### 3. Verify Installation

After restarting PowerShell:
```powershell
coder --version
# Should output: Coder Agent v1.0.0
```

### 4. Configure Model Path

Edit `config.yaml` and set your model path:
```yaml
llm:
  server:
    model_path: "C:\\path\\to\\your\\model.gguf"
```

### 5. Run from Any Directory!

```powershell
cd C:\Projects\MyApp
coder
```

The agent will:
- Find config at `C:\Users\Jake\coder\config.yaml`
- Start llama-server automatically
- Save logs to `C:\Users\Jake\coder\logs\`
- Save TODO.md to `C:\Projects\MyApp\TODO.md`

## Troubleshooting

### "coder is not recognized"

**Cause**: PATH not updated or PowerShell not restarted.

**Solution**:
1. Verify PATH was added:
   ```powershell
   $env:Path -split ';' | Select-String "coder"
   ```
2. If not found, add it using one of the methods above
3. **Close and reopen PowerShell** (this is critical!)

### Test without PATH

If you don't want to modify PATH, you can use the full path:
```powershell
cd C:\Projects\MyApp
C:\Users\Jake\coder\coder.exe
```

Or create a PowerShell alias in your profile:
```powershell
# Edit profile
notepad $PROFILE

# Add this line:
Set-Alias coder C:\Users\Jake\coder\coder.exe

# Save and reload
. $PROFILE
```

## Uninstall

To remove from PATH:
1. Open Environment Variables (same as setup)
2. Edit "Path" variable
3. Find and remove `C:\Users\Jake\coder`
4. Click OK
5. Restart PowerShell

## Alternative Installation Locations

### Install to Home Directory
```powershell
mkdir $env:USERPROFILE\.coder
copy coder.exe $env:USERPROFILE\.coder\
copy config.yaml $env:USERPROFILE\.coder\

# Add to PATH
setx PATH "%PATH%;$env:USERPROFILE\.coder"
```

### System-Wide Installation (Admin)
```powershell
mkdir C:\Program Files\Coder
copy coder.exe "C:\Program Files\Coder\"
copy config.yaml "C:\Program Files\Coder\"

# Add to System PATH (requires admin)
[Environment]::SetEnvironmentVariable("Path", $env:Path + ";C:\Program Files\Coder", "Machine")
```
