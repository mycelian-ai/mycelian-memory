Plan → Act:
- Scan related files to validate assumptions behind feedback: `api/health.go`, factory for embeddings and search index.
- Then provide a concise, actionable code review with specific line references and fixes.


- Scanned `server/memoryservice/run.go` plus the factory and handler layers to validate optional dependency behavior and health wiring. Found one hard compile issue and a few startup/health/shutdown problems. Here’s a focused review with concrete fixes.

### High-impact issues
- Compile-time type mismatch for search handler
  - `NewSearchHandler` expects `emb.EmbeddingProvider` but `embedding` is `searchindex.Embeddings`.
```136:138:server/memoryservice/run.go
// Search (reuse embeddings and index from factory)
search := api.NewSearchHandler(embedding, searchIdx, cfg.SearchAlpha)
root.HandleFunc("/v0/search", search.HandleSearch).Methods("POST")
```
  - Fix: either change factory to return `emb.EmbeddingProvider`, or assert before use:
```go
var embedProvider emb.EmbeddingProvider
if p, ok := any(embedding).(emb.EmbeddingProvider); ok {
    embedProvider = p
} else {
    log.Warn().Msg("embedding provider does not implement embeddings.EmbeddingProvider")
}
search := api.NewSearchHandler(embedProvider, searchIdx, cfg.SearchAlpha)
```
  - Prefer unifying on one interface (see “Reduce interface duplication” below).

- Optional dependencies must not fail startup
```70:77:server/memoryservice/run.go
if searchIdx == nil {
    return fmt.Errorf("search index not configured")
}
// Initialize embeddings layer
embedding := factory.NewEmbeddingProvider(ctx, cfg, log)
if embedding == nil {
    return fmt.Errorf("embedding provider not configured")
}
```
  - These should log and continue. Let the `/v0/search` route return 503 when unconfigured (it already does), or only register the route when both are present.

### Correctness and robustness
- Tie background goroutines to a cancelable root context
  - You pass `context.Background()` to health checkers, so they won’t stop on shutdown. Use `signal.NotifyContext` and pass that `ctx` everywhere; cancel on shutdown to stop tickers cleanly.
  - Optionally use `server.RegisterOnShutdown(func(){ cancel() })`.

- Only register search route when usable (optional)
  - Current handler self-guards; alternatively, conditionally register the route when both embedder and index are present to avoid a discoverable-but-disabled endpoint.

### Design and maintainability
- Reduce interface duplication
  - There are two identical embedding interfaces: `embeddings.EmbeddingProvider` and `searchindex.Embeddings`. Pick one (recommend `embeddings.EmbeddingProvider`) and standardize factory/handlers to it to avoid type assertions and reduce confusion.

- Logging clarity for optional deps
  - On startup, log whether search index and embeddings are enabled/disabled with key config (provider, model, Weaviate URL). Replace hard errors with `Info`/`Warn` as appropriate.

- Improve HTTP server hardening (minor)
  - Consider adding `ReadHeaderTimeout` and, if desired, `MaxHeaderBytes`. Current read/write/idle timeouts are fine.

- Health warmup goroutine
  - The warmup wait loop is fine, but it does nothing observable. Consider logging when service first becomes healthy.

- Naming polish
  - Rename local `embedding` to `embedProvider` for clarity and to reduce confusion with the `emb` import alias.

### Example adjustments (minimal)
- Make ctx cancelable and wire shutdown:
```go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

// pass ctx to health checkers
go storeChecker.Start(ctx, interval)
// ...
go svcHealth.Start(ctx, interval)

// shutdown branch
case <-ctx.Done():
    // perform server.Shutdown as you already do
```

- Don’t fail when optional deps are absent:
```go
if searchIdx == nil {
    log.Info().Msg("Search index not configured; search disabled")
}
if embedProvider == nil {
    log.Info().Msg("Embedding provider not configured; search disabled")
}
```

- Register search route only when both present (optional):
```go
if embedProvider != nil && searchIdx != nil {
    search := api.NewSearchHandler(embedProvider, searchIdx, cfg.SearchAlpha)
    root.HandleFunc("/v0/search", search.HandleSearch).Methods("POST")
}
```

- Unify the interface (preferred)
  - Change `factory.NewEmbeddingProvider` to return `emb.EmbeddingProvider`, update its internal type to `ollama.New(...)` (which already matches), and remove the run-time assertions in `run.go`.

- Minor: add `ReadHeaderTimeout`:
```go
ReadHeaderTimeout: 10 * time.Second,
```

- Test additions
  - Add a build check or unit test that instantiates the server without Weaviate/embeddings to ensure startup succeeds and `/v0/health` returns healthy once store is OK.

- Status: Completed scan; identified a compile error, optional-dependency startup blockers, and shutdown context leak. Proposed concrete code changes to fix build, make optional deps truly optional, and improve shutdown and logging.

- Key changes to make:
  - Fix `NewSearchHandler` arg type or factory return type.
  - Don’t return errors when search/embeddings are missing; log and continue.
  - Use a cancelable root context for health checker goroutines.
  - Optional: register `/v0/search` only when both deps exist.
  - Optional: unify embedding interfaces and add `ReadHeaderTimeout`.
