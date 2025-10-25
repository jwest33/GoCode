# SWE-bench Verified Benchmark Module

This module provides a complete framework for evaluating the GoCode agent against SWE-bench Verified, a state-of-the-art benchmark for agentic coding systems.

## Overview

**SWE-bench Verified** is a curated benchmark consisting of 500 real-world GitHub issues from popular open-source projects. Each task requires an AI agent to:

1. Understand a bug report or feature request
2. Navigate and analyze an unfamiliar codebase
3. Implement a fix or feature
4. Ensure the solution passes existing and new tests

As of October 2025, the best models achieve:
- **Claude Sonnet 4.5**: 77.2%
- **GPT-5**: 74.9%
- **Claude Opus 4.1**: 74.5%

## Quick Start

### 1. Prerequisites

- **Go 1.23+** (for GoCode)
- **Python 3.8+** (for evaluation harness)
- **Git** (for repository cloning)
- **Docker** (recommended, for containerized evaluation)
- **120GB+ free disk space** (for repositories and evaluation)

### 2. Setup Evaluation Harness

#### On Linux/macOS:
```bash
chmod +x scripts/setup-swebench-harness.sh
./scripts/setup-swebench-harness.sh
```

#### On Windows:
```cmd
scripts\setup-swebench-harness.bat
```

This script will:
- Clone the official SWE-bench repository
- Create a Python virtual environment
- Install all required dependencies

### 3. Download Benchmark Dataset

```bash
go run cmd/benchmark/main.go setup
```

This downloads the SWE-bench Verified dataset (500 tasks) from HuggingFace and saves it locally to `benchmarks/swebench-verified/dataset.json`.

### 4. Run the Benchmark

```bash
# Run on a small subset for testing (first 10 tasks)
go run cmd/benchmark/main.go run --run-id test-run --limit 10

# Run on all 500 tasks (takes 24-48 hours)
go run cmd/benchmark/main.go run --run-id full-run
```

### 5. Evaluate Results

```bash
# Activate Python environment
source benchmarks/swebench-verified/swebench-harness/venv/bin/activate  # Linux/macOS
# OR
benchmarks\swebench-verified\swebench-harness\venv\Scripts\activate.bat  # Windows

# Run evaluation
go run cmd/benchmark/main.go evaluate --run-id test-run
```

### 6. Generate Report

```bash
# Text report (default)
go run cmd/benchmark/main.go report --run-id test-run

# JSON report
go run cmd/benchmark/main.go report --run-id test-run --format json

# Markdown report
go run cmd/benchmark/main.go report --run-id test-run --format markdown
```

## Architecture

### Directory Structure

```
gocode/
├── cmd/benchmark/
│   └── main.go                    # CLI entry point
├── internal/benchmark/
│   ├── dataset.go                 # Dataset loading from HuggingFace
│   ├── task.go                    # Task models and structures
│   ├── runner.go                  # Agent execution engine
│   ├── predictions.go             # Patch extraction and formatting
│   ├── evaluator.go               # Python harness integration
│   └── reporter.go                # Results analysis and reporting
├── benchmarks/
│   └── swebench-verified/
│       ├── dataset.json           # Downloaded dataset (500 tasks)
│       ├── workspaces/            # Temporary cloned repositories
│       │   └── <run_id>/
│       │       └── <instance_id>/ # Per-task workspace
│       ├── predictions/           # Generated patches
│       │   └── <run_id>.jsonl     # Predictions file for harness
│       ├── results/               # Task execution results
│       │   └── <run_id>_results.json
│       ├── evaluation_results/    # Evaluation harness outputs
│       │   └── <run_id>/
│       └── swebench-harness/      # Official Python evaluation code
└── scripts/
    ├── setup-swebench-harness.sh  # Setup script (Linux/macOS)
    └── setup-swebench-harness.bat # Setup script (Windows)
```

### Workflow

1. **Setup Phase**:
   - Downloads SWE-bench Verified dataset via HuggingFace API
   - Stores 500 task definitions locally

2. **Run Phase**:
   - For each task:
     - Clones target repository at base commit
     - Creates isolated workspace
     - Initializes GoCode agent with problem statement
     - Runs agent with configured timeout and token budget
     - Extracts git diff as patch
     - Saves predictions in SWE-bench format

3. **Evaluate Phase**:
   - Calls official Python evaluation harness
   - Harness applies patches to clean repositories
   - Runs FAIL_TO_PASS tests (must pass after fix)
   - Runs PASS_TO_PASS tests (must still pass)
   - Determines if issue is "resolved"

4. **Report Phase**:
   - Aggregates execution statistics
   - Calculates resolve rate (% of issues fixed)
   - Compares to leaderboard baselines
   - Generates formatted reports

## CLI Reference

### `benchmark setup`

Downloads the SWE-bench Verified dataset.

**Options:**
- `--data-dir <path>`: Directory to store dataset (default: `benchmarks/swebench-verified`)
- `--force`: Force re-download if dataset exists

**Example:**
```bash
go run cmd/benchmark/main.go setup --data-dir ./my-benchmarks
```

### `benchmark run`

Executes the GoCode agent against benchmark tasks.

**Options:**
- `--run-id <id>`: **Required.** Unique identifier for this run
- `--data-dir <path>`: Dataset directory (default: `benchmarks/swebench-verified`)
- `--limit <n>`: Limit to first N tasks (0 = all 500, default: 0)
- `--filter <pattern>`: Filter tasks by instance ID pattern
- `--timeout <seconds>`: Timeout per task (default: 600)
- `--token-budget <n>`: Max tokens per task (default: 100000)
- `--workers <n>`: Parallel workers (default: 1)

**Examples:**
```bash
# Test run on 5 tasks
go run cmd/benchmark/main.go run --run-id quick-test --limit 5

# Run on Django tasks only
go run cmd/benchmark/main.go run --run-id django-only --filter django

# Full run with 4 parallel workers
go run cmd/benchmark/main.go run --run-id full --workers 4
```

### `benchmark evaluate`

Runs the official evaluation harness on generated predictions.

**Options:**
- `--run-id <id>`: **Required.** Run ID to evaluate
- `--data-dir <path>`: Dataset directory (default: `benchmarks/swebench-verified`)
- `--max-workers <n>`: Parallel workers for harness (default: 4)

**Example:**
```bash
go run cmd/benchmark/main.go evaluate --run-id full --max-workers 8
```

### `benchmark report`

Generates a results report.

**Options:**
- `--run-id <id>`: **Required.** Run ID to report on
- `--data-dir <path>`: Dataset directory (default: `benchmarks/swebench-verified`)
- `--format <format>`: Output format: `text`, `json`, `markdown` (default: `text`)

**Examples:**
```bash
# Human-readable text report
go run cmd/benchmark/main.go report --run-id full

# JSON for programmatic analysis
go run cmd/benchmark/main.go report --run-id full --format json > results.json

# Markdown for documentation
go run cmd/benchmark/main.go report --run-id full --format markdown > RESULTS.md
```

## Configuration

The benchmark runner creates agent configurations optimized for evaluation:

- **Confirmation Policy**: `never` (auto-approve all tools)
- **Context Window**: Set via `--token-budget` flag
- **Checkpoint**: Disabled (not needed for single-task execution)
- **Memory**: Disabled (fresh context per task)
- **Telemetry**: Disabled (reduce overhead)
- **Enabled Tools**: `read`, `write`, `edit`, `bash`, `glob`, `grep`, `bash_output`, `kill_shell`, `todo_write`

## Interpreting Results

### Resolve Rate

The primary metric is **resolve rate**: the percentage of tasks where your agent's patch causes all FAIL_TO_PASS tests to pass while keeping PASS_TO_PASS tests passing.

**Example output:**
```
Resolved:      42 / 100
Resolve Rate:  42.00%
```

This means 42 out of 100 tasks were successfully fixed.

### Leaderboard Comparison

Reports include comparison to known baselines:

```
--- Leaderboard Comparison ---
GoCode Score:  42.00%
Est. Rank:     #6

Baseline Comparison:
  1. Claude Sonnet 4.5: 77.20%
  2. GPT-5: 74.90%
  3. Claude Opus 4.1: 74.50%
  4. o3: 69.10%
  5. Gemini 2.5 Pro: 63.80%
✓ 6. GoCode: 42.00%
```

### Execution Statistics

- **Avg Time**: Average time per task (helps identify timeout issues)
- **Avg Tokens**: Average LLM tokens used (helps with cost estimation)
- **Tool Calls**: Number of tool invocations (indicates agent activity)

## Best Practices

### 1. Start Small

Always test on a small subset before running all 500 tasks:

```bash
go run cmd/benchmark/main.go run --run-id test --limit 10
```

### 2. Monitor Progress

The runner saves results incrementally. If interrupted, you can resume by re-running with the same `--run-id` (future enhancement).

### 3. Resource Management

- **Disk Space**: Each repository clone can be 100MB-1GB. Budget ~100GB for workspaces.
- **Memory**: Parallel workers multiply memory usage. Start with `--workers 1`.
- **Time**: Budget ~5-10 minutes per task (500 tasks = 40-80 hours at 1 worker).

### 4. Cost Estimation

If using a paid LLM API:
- Average tokens per task: ~50,000-100,000
- 500 tasks × 100,000 tokens = 50M tokens
- At $3/M tokens (Claude Sonnet): ~$150 for full benchmark

For local models via llama.cpp: cost is zero, but runtime may be longer.

## Troubleshooting

### "Dataset not found"

Run setup first:
```bash
go run cmd/benchmark/main.go setup
```

### "SWE-bench harness not found"

Run the setup script:
```bash
./scripts/setup-swebench-harness.sh  # Linux/macOS
scripts\setup-swebench-harness.bat    # Windows
```

### "Git clone failed"

Ensure you have:
- Active internet connection
- Git installed and in PATH
- Sufficient disk space

### "Timeout exceeded"

Increase timeout per task:
```bash
go run cmd/benchmark/main.go run --run-id retry --timeout 1200  # 20 minutes
```

### "Token budget exceeded"

Increase token budget:
```bash
go run cmd/benchmark/main.go run --run-id retry --token-budget 200000
```

## Advanced Usage

### Custom Filtering

Run on specific repositories:
```bash
# Django tasks only
go run cmd/benchmark/main.go run --run-id django --filter django__django

# Pytest tasks only
go run cmd/benchmark/main.go run --run-id pytest --filter pytest-dev__pytest
```

### Parallel Execution

Speed up by running multiple tasks concurrently:
```bash
go run cmd/benchmark/main.go run --run-id parallel --workers 4
```

**Note**: Each worker needs its own workspace, LLM context, and system resources.

### Manual Evaluation

If you prefer to run evaluation separately:

1. Generate predictions:
```bash
go run cmd/benchmark/main.go run --run-id my-run
```

2. Activate Python environment:
```bash
source benchmarks/swebench-verified/swebench-harness/venv/bin/activate
```

3. Run harness manually:
```bash
cd benchmarks/swebench-verified/swebench-harness
python -m swebench.harness.run_evaluation \
  --dataset_name princeton-nlp/SWE-bench_Verified \
  --predictions_path ../predictions/my-run.jsonl \
  --max_workers 4 \
  --run_id my-run
```

## Dataset Statistics

**SWE-bench Verified** (500 tasks):
- Manually verified by software engineers
- Real GitHub issues from popular open-source projects
- Includes Python repositories: Django, Flask, Pytest, Requests, Scikit-learn, SymPy, and more
- Average issue age: 1-3 years (battle-tested, stable codebases)
- Difficulty: Ranges from simple bug fixes to complex feature additions

## Contributing

To improve the benchmark module:

1. **Agent Integration**: The runner currently has a placeholder for agent execution (see `runner.go:runAgent`). Full integration with the GoCode agent package is needed.

2. **Resume Support**: Add checkpoint support to resume interrupted runs.

3. **Better Parsing**: Improve evaluation results parsing for richer reports.

4. **Visualization**: Add charts/graphs for results visualization.

5. **Alternative Benchmarks**: Support SWE-bench Lite, SWE-bench Pro, or LiveCodeBench.

## References

- **SWE-bench Paper**: [https://arxiv.org/abs/2310.06770](https://arxiv.org/abs/2310.06770)
- **Official Repository**: [https://github.com/SWE-bench/SWE-bench](https://github.com/SWE-bench/SWE-bench)
- **Leaderboard**: [https://www.swebench.com/](https://www.swebench.com/)
- **HuggingFace Dataset**: [https://huggingface.co/datasets/princeton-nlp/SWE-bench_Verified](https://huggingface.co/datasets/princeton-nlp/SWE-bench_Verified)

## License

This benchmark module is part of GoCode and follows the same license. The SWE-bench evaluation harness has its own license (MIT).
