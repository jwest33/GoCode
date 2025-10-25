# GoCode SWE-bench Benchmark Tool

A standalone CLI tool for evaluating the GoCode agent against SWE-bench Verified benchmark.

## Quick Start

```bash
# 1. Setup Python evaluation harness
./scripts/setup-swebench-harness.sh  # Linux/macOS
# OR
scripts\setup-swebench-harness.bat   # Windows

# 2. Download benchmark dataset
go run cmd/benchmark/main.go setup

# 3. Run benchmark (test on 10 tasks first)
go run cmd/benchmark/main.go run --run-id test --limit 10

# 4. Evaluate results
go run cmd/benchmark/main.go evaluate --run-id test

# 5. View report
go run cmd/benchmark/main.go report --run-id test
```

## Building

Build a standalone binary:

```bash
go build -o benchmark cmd/benchmark/main.go
```

Then use it directly:

```bash
./benchmark setup
./benchmark run --run-id my-run
./benchmark evaluate --run-id my-run
./benchmark report --run-id my-run
```

## Documentation

See [docs/BENCHMARK.md](../../docs/BENCHMARK.md) for comprehensive documentation including:

- Architecture and design
- Complete CLI reference
- Configuration options
- Troubleshooting guide
- Advanced usage patterns

## What is SWE-bench Verified?

SWE-bench Verified is a benchmark of 500 curated, real-world GitHub issues from popular open-source Python projects. It tests an AI agent's ability to:

- Understand bug reports and feature requests
- Navigate unfamiliar codebases
- Implement correct fixes
- Pass existing and new tests

**Current state-of-the-art** (October 2025):
- Claude Sonnet 4.5: 77.2%
- GPT-5: 74.9%
- Claude Opus 4.1: 74.5%

## Project Structure

```
cmd/benchmark/          # CLI tool
internal/benchmark/     # Core benchmark logic
  ├── dataset.go        # Dataset loading
  ├── task.go           # Task models
  ├── runner.go         # Agent execution
  ├── predictions.go    # Patch generation
  ├── evaluator.go      # Evaluation harness
  └── reporter.go       # Results reporting
benchmarks/             # Data and results
  └── swebench-verified/
      ├── dataset.json
      ├── predictions/
      ├── results/
      └── swebench-harness/
```

## Contributing

The benchmark module is designed to be extensible. Key areas for contribution:

1. **Full Agent Integration**: Complete the agent execution logic in `runner.go`
2. **Parallel Execution**: Implement multi-worker support
3. **Resume Support**: Add checkpoint/resume for interrupted runs
4. **Additional Benchmarks**: Add support for SWE-bench Lite, Pro, or LiveCodeBench

## License

Part of GoCode. See main project license.
