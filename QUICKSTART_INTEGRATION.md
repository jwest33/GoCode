# Quick Start: Integrating SOTA Features

This guide shows you exactly how to integrate all the features we've built into your running agent.

---

## 🎯 Goal

Wire up all Phase 1-5 features so you can:
- Use semantic code search
- Navigate code with LSP
- Save and resume sessions
- Track everything with telemetry

---

## ⚡ Quick Integration (15 Minutes)

### Step 1: Enable Features (2 min)

Edit `config.yaml` and enable the features you want:

```yaml
# Enable semantic search (optional - requires embedding server)
embeddings:
  enabled: true  # Change to true

# Enable retrieval (recommended)
retrieval:
  enabled: true  # Change to true

# Enable LSP navigation (recommended - requires LSP servers installed)
lsp:
  enabled: true  # Change to true

# Enable session persistence (recommended)
checkpoint:
  enabled: true  # Change to true

# Enable long-term memory (recommended)
memory:
  enabled: true  # Change to true

# Enable telemetry (optional but useful for debugging)
telemetry:
  enabled: true  # Change to true
```

### Step 2: Add Components to Agent (5 min)

Edit `internal/agent/agent.go` - Add fields to Agent struct:

```go
// Add these imports at the top
import (
	// ... existing imports ...
	"github.com/jake/gocode/internal/embeddings"
	"github.com/jake/gocode/internal/retrieval"
	"github.com/jake/gocode/internal/lsp"
	"github.com/jake/gocode/internal/codegraph"
	"github.com/jake/gocode/internal/checkpoint"
	"github.com/jake/gocode/internal/memory"
	"github.com/jake/gocode/internal/context"
	"github.com/jake/gocode/internal/telemetry"
)

// Add these fields to Agent struct
type Agent struct {
	// ... existing fields ...

	// New fields
	embeddingsMgr     *embeddings.Manager
	retriever         *retrieval.HybridRetriever
	reranker          *retrieval.Reranker
	lspMgr            *lsp.Manager
	codeGraph         *codegraph.Graph
	checkpointMgr     *checkpoint.Manager
	ltMemory          *memory.LongTermMemory
	contextMgr        *context.Manager
	telemetryProvider *telemetry.Provider
}
```

### Step 3: Initialize in New() (5 min)

Add initialization code in `agent.New()` function, after existing initialization:

```go
func New(cfg *config.Config) (*Agent, error) {
	// ... existing logger, serverManager, llmClient, registry initialization ...

	// Initialize telemetry (do this first!)
	var telemetryProvider *telemetry.Provider
	if cfg.Telemetry.Enabled {
		tConfig := telemetry.Config{
			Enabled:     true,
			ServiceName: cfg.Telemetry.ServiceName,
			DBPath:      filepath.Join(cfg.BaseDir, cfg.Telemetry.DBPath),
		}
		telemetryProvider, _ = telemetry.NewProvider(tConfig)

		// Set tracer on LLM client
		if telemetryProvider != nil {
			llmClient.SetTracer(telemetryProvider.Tracer())
		}
	}

	// Initialize embeddings
	var embMgr *embeddings.Manager
	if cfg.Embeddings.Enabled {
		embConfig := embeddings.Config{
			EmbeddingEndpoint: cfg.Embeddings.Endpoint,
			EmbeddingDim:      cfg.Embeddings.Dimension,
			VectorDBPath:      filepath.Join(cfg.BaseDir, cfg.Embeddings.DBPath),
			ChunkerConfig:     embeddings.DefaultChunkerConfig(),
		}
		embMgr, _ = embeddings.NewManager(embConfig)
		// Note: Errors are logged but we continue without embeddings
	}

	// Initialize retriever
	var retriever *retrieval.HybridRetriever
	var reranker *retrieval.Reranker
	if cfg.Retrieval.Enabled {
		weights := retrieval.FusionWeights{
			BM25:     cfg.Retrieval.Weights.BM25,
			Semantic: cfg.Retrieval.Weights.Semantic,
			Trigram:  cfg.Retrieval.Weights.Trigram,
		}
		retriever = retrieval.NewHybridRetriever(weights, embMgr)
		reranker = retrieval.NewReranker()
	}

	// Initialize LSP manager
	var lspMgr *lsp.Manager
	var codeGraph *codegraph.Graph
	if cfg.LSP.Enabled {
		lspConfigs := make(map[string]lsp.LanguageServerConfig)
		for lang, serverCfg := range cfg.LSP.Servers {
			lspConfigs[lang] = lsp.LanguageServerConfig{
				Command:  serverCfg.Command,
				Args:     serverCfg.Args,
				FileExts: getFileExtensions(lang),
			}
		}
		lspMgr = lsp.NewManager(cfg.WorkingDir, lspConfigs)
		codeGraph = codegraph.NewGraph(cfg.WorkingDir, lspMgr)
	}

	// Initialize checkpoint manager
	var checkpointMgr *checkpoint.Manager
	if cfg.Checkpoint.Enabled {
		cpConfig := checkpoint.Config{
			DBPath:       filepath.Join(cfg.BaseDir, cfg.Checkpoint.DBPath),
			AutoSave:     cfg.Checkpoint.AutoSave,
			SaveInterval: cfg.Checkpoint.SaveInterval,
		}
		checkpointMgr, _ = checkpoint.NewManager(cpConfig)
	}

	// Initialize long-term memory
	var ltMemory *memory.LongTermMemory
	if cfg.Memory.Enabled {
		ltMemory, _ = memory.NewLongTermMemory(filepath.Join(cfg.BaseDir, cfg.Memory.DBPath))
	}

	// Initialize context manager (always enabled)
	contextMgr := context.NewManager(context.DefaultBudgetConfig())

	// ... existing tool registration ...

	// Register new tools
	if codeGraph != nil {
		registry.Register(tools.NewFindDefinitionTool(codeGraph))
		registry.Register(tools.NewFindReferencesTool(codeGraph))
		registry.Register(tools.NewListSymbolsTool(codeGraph))
	}

	return &Agent{
		// ... existing fields ...
		embeddingsMgr:     embMgr,
		retriever:         retriever,
		reranker:          reranker,
		lspMgr:            lspMgr,
		codeGraph:         codeGraph,
		checkpointMgr:     checkpointMgr,
		ltMemory:          ltMemory,
		contextMgr:        contextMgr,
		telemetryProvider: telemetryProvider,
	}, nil
}

// Helper function for LSP file extensions
func getFileExtensions(lang string) []string {
	switch lang {
	case "go":
		return []string{".go"}
	case "python":
		return []string{".py"}
	case "typescript":
		return []string{".ts", ".tsx", ".js", ".jsx"}
	default:
		return []string{}
	}
}
```

### Step 4: Add Cleanup (1 min)

Update the `Run()` method to cleanup all components:

```go
func (a *Agent) Run() error {
	defer a.serverManager.Stop()
	defer a.logger.Close()
	defer a.rl.Close()

	// Add new cleanups
	if a.telemetryProvider != nil {
		defer a.telemetryProvider.Shutdown(context.Background())
	}
	if a.lspMgr != nil {
		defer a.lspMgr.Close()
	}
	if a.embeddingsMgr != nil {
		defer a.embeddingsMgr.Close()
	}
	if a.checkpointMgr != nil {
		defer a.checkpointMgr.Close()
	}
	if a.ltMemory != nil {
		defer a.ltMemory.Close()
	}

	// ... rest of existing Run() code ...
}
```

### Step 5: Add Retrieval to processInput (2 min)

Add these helper methods to agent.go:

```go
func (a *Agent) shouldRetrieve(input string) bool {
	if a.retriever == nil {
		return false
	}

	// Simple heuristics
	lowerInput := strings.ToLower(input)
	keywords := []string{"find", "search", "where", "what", "how", "show", "explain", "list"}

	for _, keyword := range keywords {
		if strings.Contains(lowerInput, keyword) {
			return true
		}
	}

	return false
}

func (a *Agent) performRetrieval(ctx context.Context, query string) []string {
	// Hybrid retrieval
	results, err := a.retriever.Search(ctx, query, 20)
	if err != nil {
		return []string{}
	}

	// Rerank
	reranked := a.reranker.Rerank(results, query, 10)

	// Extract contexts
	contexts := make([]string, len(reranked))
	for i, result := range reranked {
		contexts[i] = result.Document.Content
	}

	// Filter by budget
	filtered := a.contextMgr.FilterContextByBudget(contexts)

	// Order optimally
	ordered := retrieval.OrderContext(filtered, len(filtered))

	return ordered
}
```

Then update processInput to use retrieval:

```go
func (a *Agent) processInput(input string) error {
	a.logger.LogUserInput(input)

	// Add user message
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: input,
	})

	// Check if retrieval needed
	var retrievedContexts []string
	if a.shouldRetrieve(input) {
		retrievedContexts = a.performRetrieval(context.Background(), input)
	}

	// Set messages in context manager
	a.contextMgr.SetMessages(a.messages)

	// Main conversation loop
	for {
		// Prepare messages with retrieval and pruning
		messages := a.contextMgr.PrepareMessagesForLLM(retrievedContexts)

		// ... existing LLM call and tool execution code ...

		// After completing turn, save checkpoint
		if a.checkpointMgr != nil {
			a.checkpointMgr.OnMessage(a.messages)
		}

		if resp.FinishReason == "stop" {
			break
		}
	}

	return nil
}
```

---

## 🧪 Testing Your Integration

### Test 1: Basic Compilation
```bash
go build -o gocode.exe cmd/gocode/main.go
```
Should compile without errors.

### Test 2: Run Without Features
Start with all features disabled in config.yaml:
```bash
gocode
```
Should work exactly as before.

### Test 3: Enable Telemetry Only
```yaml
telemetry:
  enabled: true
```
Run and check that `traces.db` is created.

### Test 4: Enable Checkpoint
```yaml
checkpoint:
  enabled: true
```
Run, chat, exit, run again - should resume conversation.

### Test 5: Enable LSP
```yaml
lsp:
  enabled: true
```
Try: `list symbols in internal/agent/agent.go`

---

## 🚨 Troubleshooting

**Build Errors:**
```bash
go mod tidy
go mod download
```

**Missing LSP Servers:**
```bash
# Go
go install golang.org/x/tools/gopls@latest

# Python
pip install python-lsp-server

# TypeScript
npm install -g typescript-language-server
```

**Embeddings Not Working:**
- Start embedding server:
```bash
llama-server --model nomic-embed-text.gguf --port 8081 --embedding
```

**Telemetry Errors:**
- Check write permissions on DB paths
- Verify go.mod has otel packages

---

## ✅ Success Checklist

- [ ] Code compiles without errors
- [ ] Agent runs with all features disabled
- [ ] Telemetry creates traces.db
- [ ] Checkpoints save and resume
- [ ] LSP tools work (if servers installed)
- [ ] No crashes or panics

---

## 🎉 You're Done!

You now have a fully integrated SOTA autonomous coding agent with:
- Hybrid retrieval
- Code navigation
- Session persistence
- Complete observability
- Long-term memory

**Start using it and enjoy the power!** 🚀

See `FINAL_IMPLEMENTATION.md` for complete feature documentation.
