# Integration Guide: SOTA Features

This guide explains how to integrate all the new features into your GoCode agent.

## 🎯 What We've Built

### Phase 1: Context Engine
- **Local embeddings** with vector search
- **Hybrid retrieval** (BM25 + semantic + trigram)
- **Intelligent reranking** with heuristics
- **Context budgeting** and message pruning

### Phase 2: Code Graph Navigation
- **LSP integration** for multiple languages
- **Fallback parser** for when LSP unavailable
- **Symbol graph** with definitions/references/calls
- **Navigation tools** for the agent

### Phase 3: Session Persistence
- **SQLite checkpointing** for conversation state
- **Thread management** (create/resume/branch)
- **Long-term memory** for facts and artifacts

---

## 📦 Step 1: Update Configuration

Add to `internal/config/config.go`:

```go
type Config struct {
	// ... existing fields ...

	Embeddings  EmbeddingsConfig  `yaml:"embeddings"`
	Retrieval   RetrievalConfig   `yaml:"retrieval"`
	LSP         LSPConfig         `yaml:"lsp"`
	Checkpoint  CheckpointConfig  `yaml:"checkpoint"`
	Memory      MemoryConfig      `yaml:"memory"`
}

type EmbeddingsConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Endpoint  string `yaml:"endpoint"`
	Dimension int    `yaml:"dimension"`
	DBPath    string `yaml:"db_path"`
}

type RetrievalConfig struct {
	Enabled bool `yaml:"enabled"`
	Weights struct {
		BM25     float32 `yaml:"bm25"`
		Semantic float32 `yaml:"semantic"`
		Trigram  float32 `yaml:"trigram"`
	} `yaml:"weights"`
}

type LSPConfig struct {
	Enabled bool                            `yaml:"enabled"`
	Servers map[string]LSPServerConfig     `yaml:"servers"`
}

type LSPServerConfig struct {
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

type CheckpointConfig struct {
	Enabled      bool   `yaml:"enabled"`
	DBPath       string `yaml:"db_path"`
	AutoSave     bool   `yaml:"auto_save"`
	SaveInterval int    `yaml:"save_interval"`
}

type MemoryConfig struct {
	Enabled bool   `yaml:"enabled"`
	DBPath  string `yaml:"db_path"`
}
```

Add to `config.yaml`:

```yaml
# Embeddings Configuration
embeddings:
  enabled: true
  endpoint: "http://localhost:8081"  # Run llama-server with embedding model
  dimension: 384                      # For nomic-embed-text or bge-small-en-v1.5
  db_path: "embeddings.db"

# Retrieval Configuration
retrieval:
  enabled: true
  weights:
    bm25: 0.4
    semantic: 0.5
    trigram: 0.1

# LSP Configuration
lsp:
  enabled: true
  servers:
    go:
      command: "gopls"
      args: []
    python:
      command: "pylsp"
      args: []
    typescript:
      command: "typescript-language-server"
      args: ["--stdio"]

# Checkpoint Configuration
checkpoint:
  enabled: true
  db_path: "checkpoints.db"
  auto_save: true
  save_interval: 5  # Every 5 messages

# Memory Configuration
memory:
  enabled: true
  db_path: "memory.db"
```

---

## 📦 Step 2: Update Agent Initialization

Modify `internal/agent/agent.go`:

```go
import (
	// ... existing imports ...
	"github.com/jake/gocode/internal/embeddings"
	"github.com/jake/gocode/internal/retrieval"
	"github.com/jake/gocode/internal/lsp"
	"github.com/jake/gocode/internal/codegraph"
	"github.com/jake/gocode/internal/checkpoint"
	"github.com/jake/gocode/internal/memory"
	"github.com/jake/gocode/internal/context"
)

type Agent struct {
	// ... existing fields ...

	// New fields
	embeddingsMgr   *embeddings.Manager
	retriever       *retrieval.HybridRetriever
	reranker        *retrieval.Reranker
	lspMgr          *lsp.Manager
	codeGraph       *codegraph.Graph
	checkpointMgr   *checkpoint.Manager
	ltMemory        *memory.LongTermMemory
	contextMgr      *context.Manager
}

func New(cfg *config.Config) (*Agent, error) {
	// ... existing initialization ...

	// Initialize embeddings manager (if enabled)
	var embMgr *embeddings.Manager
	if cfg.Embeddings.Enabled {
		embConfig := embeddings.Config{
			EmbeddingEndpoint: cfg.Embeddings.Endpoint,
			EmbeddingDim:      cfg.Embeddings.Dimension,
			VectorDBPath:      filepath.Join(cfg.BaseDir, cfg.Embeddings.DBPath),
			ChunkerConfig:     embeddings.DefaultChunkerConfig(),
		}
		var err error
		embMgr, err = embeddings.NewManager(embConfig)
		if err != nil {
			// Log warning but continue without embeddings
			fmt.Printf("Warning: Failed to initialize embeddings: %v\n", err)
		}
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
	}

	// Initialize code graph
	var codeGraph *codegraph.Graph
	if lspMgr != nil {
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
		var err error
		checkpointMgr, err = checkpoint.NewManager(cpConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize checkpoint manager: %w", err)
		}
	}

	// Initialize long-term memory
	var ltMemory *memory.LongTermMemory
	if cfg.Memory.Enabled {
		var err error
		ltMemory, err = memory.NewLongTermMemory(filepath.Join(cfg.BaseDir, cfg.Memory.DBPath))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize long-term memory: %w", err)
		}
	}

	// Initialize context manager
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
		embeddingsMgr: embMgr,
		retriever:     retriever,
		reranker:      reranker,
		lspMgr:        lspMgr,
		codeGraph:     codeGraph,
		checkpointMgr: checkpointMgr,
		ltMemory:      ltMemory,
		contextMgr:    contextMgr,
	}, nil
}

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

---

## 📦 Step 3: Integrate Retrieval into Agent Loop

Modify `processInput` in `agent.go`:

```go
func (a *Agent) processInput(input string) error {
	a.logger.LogUserInput(input)

	// Check if we should resume a thread
	if a.checkpointMgr != nil && len(a.messages) == 1 {
		// Try to resume default thread
		threads, _ := a.checkpointMgr.ListThreads()
		if len(threads) > 0 {
			msgs, _ := a.checkpointMgr.ResumeThread(threads[0].ID)
			if len(msgs) > 0 {
				// Convert to non-pointer slice
				for _, msg := range msgs {
					a.messages = append(a.messages, *msg)
				}
			}
		} else {
			// Start new thread
			a.checkpointMgr.StartNewThread("default")
		}
	}

	// Add user message
	a.messages = append(a.messages, llm.Message{
		Role:    "user",
		Content: input,
	})

	// Check if retrieval is needed
	var retrievedContexts []string
	if a.shouldRetrieve(input) {
		retrievedContexts = a.performRetrieval(input)
	}

	// Set messages in context manager
	a.contextMgr.SetMessages(a.messages)

	// Main conversation loop
	for {
		// Prepare messages with retrieval and pruning
		messages := a.contextMgr.PrepareMessagesForLLM(retrievedContexts)

		// ... rest of existing loop logic ...

		// After completing turn, save checkpoint
		if a.checkpointMgr != nil {
			a.checkpointMgr.OnMessage(a.messages)
		}

		if resp.FinishReason == "stop" {
			break
		}
	}

	// Update context manager
	a.contextMgr.SetMessages(a.messages)

	return nil
}

func (a *Agent) shouldRetrieve(input string) bool {
	if a.retriever == nil {
		return false
	}

	// Simple heuristics - can be enhanced
	keywords := []string{"find", "search", "where", "what", "how", "show me", "explain"}
	lowerInput := strings.ToLower(input)

	for _, keyword := range keywords {
		if strings.Contains(lowerInput, keyword) {
			return true
		}
	}

	return false
}

func (a *Agent) performRetrieval(query string) []string {
	ctx := context.Background()

	// Hybrid retrieval
	results, err := a.retriever.Search(ctx, query, 20)
	if err != nil {
		fmt.Printf("Retrieval failed: %v\n", err)
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

	// Order for optimal placement
	ordered := retrieval.OrderContext(filtered, len(filtered))

	return ordered
}
```

---

## 📦 Step 4: Add Session Management Commands

Add command handling in `processInput`:

```go
func (a *Agent) processInput(input string) error {
	// Check for session commands
	if strings.HasPrefix(input, "/") {
		return a.handleCommand(input)
	}

	// ... rest of existing logic ...
}

func (a *Agent) handleCommand(cmd string) error {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/save":
		return a.handleSave(parts[1:])
	case "/resume":
		return a.handleResume(parts[1:])
	case "/branch":
		return a.handleBranch(parts[1:])
	case "/threads":
		return a.handleListThreads()
	case "/memory":
		return a.handleMemorySearch(parts[1:])
	default:
		fmt.Printf("Unknown command: %s\n", parts[0])
	}

	return nil
}

func (a *Agent) handleSave(args []string) error {
	if a.checkpointMgr == nil {
		fmt.Println("Checkpointing not enabled")
		return nil
	}

	description := "Manual save"
	if len(args) > 0 {
		description = strings.Join(args, " ")
	}

	cp, err := a.checkpointMgr.SaveCheckpoint(a.messages, description)
	if err != nil {
		return err
	}

	fmt.Printf("✓ Saved checkpoint: %s\n", cp.ID)
	return nil
}

func (a *Agent) handleResume(args []string) error {
	if a.checkpointMgr == nil {
		fmt.Println("Checkpointing not enabled")
		return nil
	}

	if len(args) == 0 {
		fmt.Println("Usage: /resume <thread_id>")
		return nil
	}

	msgs, err := a.checkpointMgr.ResumeThread(args[0])
	if err != nil {
		return err
	}

	// Convert to non-pointer slice
	a.messages = []llm.Message{}
	for _, msg := range msgs {
		a.messages = append(a.messages, *msg)
	}

	fmt.Printf("✓ Resumed thread with %d messages\n", len(a.messages))
	return nil
}

func (a *Agent) handleBranch(args []string) error {
	if a.checkpointMgr == nil {
		fmt.Println("Checkpointing not enabled")
		return nil
	}

	if len(args) < 2 {
		fmt.Println("Usage: /branch <checkpoint_id> <new_thread_name>")
		return nil
	}

	thread, err := a.checkpointMgr.BranchThread(args[0], strings.Join(args[1:], " "))
	if err != nil {
		return err
	}

	fmt.Printf("✓ Created branch: %s (ID: %s)\n", thread.Name, thread.ID)
	return nil
}

func (a *Agent) handleListThreads() error {
	if a.checkpointMgr == nil {
		fmt.Println("Checkpointing not enabled")
		return nil
	}

	threads, err := a.checkpointMgr.ListThreads()
	if err != nil {
		return err
	}

	fmt.Printf("\nThreads (%d):\n\n", len(threads))
	for _, thread := range threads {
		fmt.Printf("  %s - %s (updated: %s)\n", thread.ID, thread.Name, thread.UpdatedAt.Format("2006-01-02 15:04"))
	}

	return nil
}

func (a *Agent) handleMemorySearch(args []string) error {
	if a.ltMemory == nil {
		fmt.Println("Long-term memory not enabled")
		return nil
	}

	if len(args) == 0 {
		fmt.Println("Usage: /memory <search query>")
		return nil
	}

	query := strings.Join(args, " ")
	memories, err := a.ltMemory.Search(query, 5)
	if err != nil {
		return err
	}

	fmt.Printf("\nFound %d memories:\n\n", len(memories))
	for i, mem := range memories {
		fmt.Printf("%d. [%s] %s\n", i+1, mem.Type, mem.Summary)
		fmt.Printf("   Created: %s | Importance: %.2f\n\n", mem.CreatedAt.Format("2006-01-02"), mem.Importance)
	}

	return nil
}
```

---

## 🚀 Step 5: Run Embedding Server

Start a separate llama-server instance for embeddings:

```bash
llama-server --model path/to/nomic-embed-text-v1.5.Q4_K_M.gguf --port 8081 --embedding
```

Or use any compatible embedding server (OpenAI-compatible API).

---

## 🧪 Testing the Integration

1. **Start GoCode:**
   ```bash
   gocode
   ```

2. **Test retrieval:**
   ```
   > Find all functions that handle user authentication
   ```

3. **Test code navigation:**
   ```
   > list symbols in internal/agent/agent.go
   ```

4. **Test checkpointing:**
   ```
   > /save "Before refactoring"
   > /threads
   > /branch <checkpoint_id> experiment
   ```

5. **Test memory:**
   ```
   > /memory authentication
   ```

---

## 📊 Expected Behavior

- **Context Retrieval**: Automatically triggered on questions about code
- **Smart Pruning**: Messages auto-pruned when approaching token limit
- **LSP Navigation**: Jump to definitions, find references across codebase
- **Auto-Save**: Checkpoints saved every 5 messages
- **Resumable**: Sessions resume from last checkpoint on restart

---

## 🔧 Troubleshooting

**Embeddings not working:**
- Check embedding server is running on port 8081
- Verify embedding dimension matches model

**LSP not working:**
- Ensure language servers (gopls, pylsp, etc.) are installed
- Check PATH includes language server binaries

**Checkpoints not saving:**
- Verify write permissions on checkpoint DB path
- Check SQLite is properly installed

---

## 🎯 Next Steps

Optional enhancements:
1. **OpenTelemetry**: Full observability (Phase 4)
2. **Test Execution**: Automated testing framework (Phase 6)
3. **Best-of-N**: Multiple solutions with selection (Phase 6)
4. **Background Indexing**: Auto-index repo on startup (Phase 7)

This implementation provides all core SOTA features from RESEARCH.md while remaining fully local and practical!
