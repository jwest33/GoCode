@echo off
REM SWE-bench Evaluation Harness Setup Script (Windows)
REM This script sets up the official SWE-bench evaluation harness

setlocal enabledelayedexpansion

set "SCRIPT_DIR=%~dp0"
set "PROJECT_ROOT=%SCRIPT_DIR%.."
set "BENCHMARK_DIR=%PROJECT_ROOT%\benchmarks\swebench-verified"
set "HARNESS_DIR=%BENCHMARK_DIR%\swebench-harness"

echo ==========================================
echo   SWE-bench Harness Setup (Windows)
echo ==========================================
echo.

REM Check for Python
python --version >nul 2>&1
if errorlevel 1 (
    echo Error: Python is not installed or not in PATH.
    echo Please install Python 3.8 or higher and try again.
    exit /b 1
)

for /f "tokens=2" %%i in ('python --version 2^>^&1') do set PYTHON_VERSION=%%i
echo [OK] Found Python %PYTHON_VERSION%

REM Check for git
git --version >nul 2>&1
if errorlevel 1 (
    echo Error: Git is not installed or not in PATH.
    echo Please install Git and try again.
    exit /b 1
)
echo [OK] Found Git

REM Check for Docker (optional)
docker --version >nul 2>&1
if errorlevel 1 (
    echo [WARNING] Docker not found (recommended for full evaluation)
    echo           You can still run evaluations, but some features may be limited
) else (
    echo [OK] Found Docker (recommended for evaluation)
)

echo.
echo Creating benchmark directory structure...
if not exist "%BENCHMARK_DIR%" mkdir "%BENCHMARK_DIR%"

REM Clone SWE-bench repository if not already present
if exist "%HARNESS_DIR%" (
    echo SWE-bench harness already exists at %HARNESS_DIR%
    set /p UPDATE="Do you want to update it? (y/n): "
    if /i "!UPDATE!"=="y" (
        cd /d "%HARNESS_DIR%"
        git pull origin main
    )
) else (
    echo Cloning SWE-bench repository...
    git clone https://github.com/SWE-bench/SWE-bench.git "%HARNESS_DIR%"
)

echo.
echo Setting up Python virtual environment...
cd /d "%HARNESS_DIR%"

if not exist "venv" (
    python -m venv venv
    echo [OK] Created virtual environment
) else (
    echo [OK] Virtual environment already exists
)

REM Activate virtual environment
call venv\Scripts\activate.bat

echo.
echo Installing SWE-bench dependencies...
python -m pip install --upgrade pip
pip install -e .

echo.
echo Installing additional dependencies...
pip install datasets

echo.
echo ==========================================
echo   Setup Complete!
echo ==========================================
echo.
echo SWE-bench harness installed at: %HARNESS_DIR%
echo.
echo To use the harness, activate the virtual environment:
echo   %HARNESS_DIR%\venv\Scripts\activate.bat
echo.
echo You can now run the benchmark tool:
echo   go run cmd\benchmark\main.go setup
echo   go run cmd\benchmark\main.go run --run-id test
echo   go run cmd\benchmark\main.go evaluate --run-id test
echo.
echo For more information, see:
echo   - SWE-bench: https://github.com/SWE-bench/SWE-bench
echo   - Documentation: %PROJECT_ROOT%\docs\BENCHMARK.md
echo.

pause
