@echo off
echo Building GoCode...
go mod tidy
go build -o gocode.exe cmd/gocode/main.go
if %errorlevel% equ 0 (
    echo Build successful! Run with: gocode.exe
) else (
    echo Build failed!
    exit /b 1
)
