@echo off
echo Building Coder Agent...
go mod tidy
go build -o coder.exe cmd/coder/main.go
if %errorlevel% equ 0 (
    echo Build successful! Run with: coder.exe
) else (
    echo Build failed!
    exit /b 1
)
