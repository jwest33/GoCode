## Scope / Objectives

* Enhance GoCode with:

  * **Prune** and pack only the most relevant context.
  * **Traverse** large repos to find/compose multi‑file knowledge.
  * **Persist** working state across sessions.
  * **Trace** expose persistent traces in an LLM friendly format for replay between runs.

---

## Core Patterns (high level)

* **Two‑stage retrieval → rerank → compress** before prompting.
* **Graph‑augmented code search** (symbols/defs/refs/imports) + embeddings + keyword.
* **Checkpointed workflow state** (“threads”) + optional long‑term memory.
* **OpenTelemetry (OTel) GenAI spans** for full observability of LLM/tool calls.
* **Critic/selection** at inference time (n attempts → best‑of‑n).

---

## 1) Context Pruning (prompt budget control)

* **Retrieval**: hybrid keyword (BM25/trigram) + semantic (code embeddings), small code‑aware chunks.
* **Reranking**: cross‑encoder or late‑interaction to filter top‑k into budget.
* **Ordering**: place critical snippets at **top/bottom** (mitigate “lost‑in‑the‑middle”).
* **Compression**: add prompt compressors (e.g., LLMLingua‑style) *after* retrieval quality is strong.
* **Selective retrieval**: allow the model to *decide* when to retrieve vs. rely on parametric knowledge.
* **KPIs**: NDCG@k / reranker AUC, tokens per request, latency, pass@1 on coding tasks.

---

## 2) Project Context Traversal (repo‑scale grounding)

* **Multi‑retriever fusion**:

  * Keyword search (e.g., Zoekt/ripgrep) for exact strings/symbols.
  * Semantic search with **code embeddings**.
  * **Code graph** traversal (symbol defs/refs, imports, callers/callees, test↔src).
* **Graph sources**: LSP/SCIP indexes; Tree‑sitter/ctags fallback; optional Code Property Graphs for deeper flow.
* **Traversal loop**: seed → retrieve (kw+vec) → 1–2 hop graph expand → rerank → pack.
* **Caching**: memoize traversal results keyed by repo SHA/paths for reuse.

---

## 3) Context Persistence (across sessions)

* **Checkpointed threads**: orchestrator with **checkpointer** (e.g., SQLite/Redis/Postgres) to time‑travel/replay/branch.
* **State contents** (short‑term): messages, current plan, retrieved snippets, intermediate artifacts.
* **Long‑term memory** (optional): summarized episodic/semantic facts keyed to repo/ticket/user; store **pointers** (artifact IDs, trace IDs), not raw logs.
* **Data hygiene**: TTLs, redaction, PII controls; explicit versioning of models/tools/datasets.

---

## 4) Intelligent Traceability (self‑debug & replay)

* **Instrumentation**: OTel **GenAI semantic conventions** for model/agent/tool spans & events.
* **Per tool/exec step log**: input params, CWD, env/version hashes, stdout/stderr tails, exit code, **file diffs**, artifacts (patches, test reports), duration, verdict.
* **Linkage**: stable **trace IDs** and **artifact IDs** referenced in memory and critiques.
* **Backends**: LangSmith / Langfuse / any OTel‑compatible APM.
* **Trajectory selection**: run n attempts; score with a critic (tests/logs/diffs); choose best patch.

---

## Reference Architecture (modular)

* **UI/IDE**: VS Code/JetBrains/CLI.
* **Orchestrator**: LangGraph‑style workflow with checkpointer (threads, time‑travel).
* **Context Engine**: hybrid retrieval + graph retriever + reranker + optional compressor.
* **Execution Sandbox**: container/VM with tool wrappers; collects stdout/stderr/diffs/artifacts; OTel‑instrumented.
* **Observability & Eval**: OTel + LangSmith/Langfuse; SWE‑bench (Verified/Live) pipelines.

---

## Metrics & Benchmarks

* **Quality**: pass@1 / % resolved (SWE‑bench Verified & Live); unit/integration test pass rate.
* **Retrieval**: NDCG@10, recall@k, reranker AUC.
* **Ops**: token count/request, latency (p50/p95), cost, failure classifications.
* **Reliability**: successful resume/replay rate; deterministic replays under fixed seeds.

---

## Common Pitfalls / Remedies

* **Over‑stuffed prompts** → *Fix*: rerank + edge‑placement + compression.
* **Vector‑only search** → *Fix*: add code graph traversal.
* **Unstructured logs** → *Fix*: adopt OTel GenAI; persist artifacts with IDs.
* **Ephemeral state** → *Fix*: checkpoint threads; summarize into long‑term memory with pointers.

---
