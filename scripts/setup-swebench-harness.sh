#!/bin/bash
# SWE-bench Evaluation Harness Setup Script
# This script sets up the official SWE-bench evaluation harness

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
BENCHMARK_DIR="$PROJECT_ROOT/benchmarks/swebench-verified"
HARNESS_DIR="$BENCHMARK_DIR/swebench-harness"

echo "=========================================="
echo "  SWE-bench Harness Setup"
echo "=========================================="
echo ""

# Check for Python
if ! command -v python3 &> /dev/null; then
    echo "Error: Python 3 is not installed."
    echo "Please install Python 3.8 or higher and try again."
    exit 1
fi

PYTHON_VERSION=$(python3 --version | cut -d' ' -f2 | cut -d'.' -f1,2)
echo "✓ Found Python $PYTHON_VERSION"

# Check for git
if ! command -v git &> /dev/null; then
    echo "Error: Git is not installed."
    echo "Please install Git and try again."
    exit 1
fi
echo "✓ Found Git"

# Check for Docker (recommended but not required)
if command -v docker &> /dev/null; then
    echo "✓ Found Docker (recommended for evaluation)"
else
    echo "⚠ Docker not found (recommended for full evaluation)"
    echo "  You can still run evaluations, but some features may be limited"
fi

echo ""
echo "Creating benchmark directory structure..."
mkdir -p "$BENCHMARK_DIR"

# Clone SWE-bench repository if not already present
if [ -d "$HARNESS_DIR" ]; then
    echo "SWE-bench harness already exists at $HARNESS_DIR"
    read -p "Do you want to update it? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        cd "$HARNESS_DIR"
        git pull origin main
    fi
else
    echo "Cloning SWE-bench repository..."
    git clone https://github.com/SWE-bench/SWE-bench.git "$HARNESS_DIR"
fi

echo ""
echo "Setting up Python virtual environment..."
cd "$HARNESS_DIR"

if [ ! -d "venv" ]; then
    python3 -m venv venv
    echo "✓ Created virtual environment"
else
    echo "✓ Virtual environment already exists"
fi

# Activate virtual environment
source venv/bin/activate

echo ""
echo "Installing SWE-bench dependencies..."
pip install --upgrade pip
pip install -e .

echo ""
echo "Installing additional dependencies..."
pip install datasets

echo ""
echo "=========================================="
echo "  Setup Complete!"
echo "=========================================="
echo ""
echo "SWE-bench harness installed at: $HARNESS_DIR"
echo ""
echo "To use the harness, activate the virtual environment:"
echo "  source $HARNESS_DIR/venv/bin/activate"
echo ""
echo "You can now run the benchmark tool:"
echo "  go run cmd/benchmark/main.go setup"
echo "  go run cmd/benchmark/main.go run --run-id test"
echo "  go run cmd/benchmark/main.go evaluate --run-id test"
echo ""
echo "For more information, see:"
echo "  - SWE-bench: https://github.com/SWE-bench/SWE-bench"
echo "  - Documentation: $PROJECT_ROOT/docs/BENCHMARK.md"
echo ""
