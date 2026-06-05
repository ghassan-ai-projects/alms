# ALMS Phase 5 Fix Plan — P1 Bugs

This document outlines the implementation plan for the 4 critical P1 bugs identified in the Phase 5 deep audit.

## 1. Auth Middleware Infinite Loop

**Location:** `internal/server/server.go:54-56` (approximate)

**The Problem:**
In `ListenAndServe`, the code currently wraps the `mux` with the `AuthMiddleware` and then re-registers that wrapped handler into the `mux` itself at the root path `"/"`. When a request comes in, the `mux` matches `"/"`, calls the middleware, which calls `mux.ServeHTTP`, which matches `"/"` again, causing an infinite recursion and stack overflow.

**The Fix:**
Wrap the primary MCP handler (`streamableHTTPServer`) with the `AuthMiddleware` *before* adding it to the `mux`.

**Pseudocode:**
```go
// Inside ListenAndServe
handler = server.NewStreamableHTTPServer(s.mcp)

if s.cfg.Auth.Token != "" {
    handler = AuthMiddleware(s.cfg.Auth.Token)(handler)
}

mux := http.NewServeMux()
mux.Handle("/dashboard", dashboardHandler)
mux.Handle("/", handler) // Handler is now either raw or auth-wrapped
```

**Verification:**
- **Test:** A new test in `internal/server/server_test.go` (or `auth_test.go`) that starts the server with a token, sends a request with the correct token, and verifies it receives a `200 OK` (or successful JSON-RPC response) instead of crashing.
- **Test:** Verify a request without a token or with a wrong token receives the `-32001` "unauthorized" error.

---

## 2. GC Service Never Started

**Location:** `cmd/alms/main.go`

**The Problem:**
The `internal/service/gc.go` service is implemented but never instantiated or started in the main entry point. It remains dead code.

**The Fix:**
1. Instantiate `service.NewGC` in `main.go`.
2. Call `gc.Start(ctx)` before the server's `ListenAndServe` loop.
3. Ensure `gc.Stop()` is called during graceful shutdown.

**Pseudocode:**
```go
// main.go
gcSvc := service.NewGC(learningStore, service.DefaultGCConfig())
gcSvc.Start(ctx)
defer gcSvc.Stop()

// ... server start ...
```

**Verification:**
- **Test:** Check logs for "GC started" and "GC completed" messages on startup.
- **Test:** Unit test in `internal/service/gc_test.go` (if not already present) that mocks the store and verifies `Sweep` deletes expired records.

---

## 3. Dedup Not Wired to Learning Store Tool

**Location:** `internal/server/tools.go:250` (approximate)

**The Problem:**
The `learning.store` MCP tool handler calls `learning.Store(...)` directly, which performs validation but skips the `CheckExactDup` and `CheckNearDup` logic implemented in `StoreLearningWithDedup`.

**The Fix:**
Update the `learning.store` tool handler in `internal/server/tools.go` to call `learning.StoreLearningWithDedup(...)` instead of `learning.Store(...)`.

**Pseudocode:**
```go
// internal/server/tools.go (learning.store handler)
id, dedupResult, err := learning.StoreLearningWithDedup(ctx, rec, supersedes)
if err != nil {
    return mcp.NewToolResultError(err.Error()), nil
}

return marshalResult(map[string]any{
    "learning_id":  id,
    "status":       "created",
    "is_duplicate": dedupResult.IsExactDup || dedupResult.IsNearDup,
    "duplicate_id": dedupResult.ExactMatchID,
})
```

**Verification:**
- **Test:** Integration test that registers two identical learnings via the MCP tool and verifies the second one returns the same ID as the first or flags it as a duplicate.

---

## 4. Health Check Tool Returns Static Data

**Location:** `internal/server/tools.go:413-425` (approximate)

**The Problem:**
The `health.check` tool always returns `{"status":"ok"}`. It does not perform the advertised PostgreSQL ping or agent count check.

**The Fix:**
Update the handler to use the `registry` (and by extension its store) to perform a `Ping` and a `Count`.

**Pseudocode:**
```go
// internal/server/tools.go (health.check handler)
// Note: Need to expose store/pool or add Ping to Registry service
count, err := registry.CountAgents(ctx) // Add this method to registry service
if err != nil {
    return mcp.NewToolResultError("database health check failed"), nil
}

result := map[string]any{
    "status":      "ok",
    "agent_count": count,
    "db_ping":     "ok",
    "version":     "0.1.0",
}
```

**Verification:**
- **Test:** Call `health.check` via MCP and verify the response contains `agent_count`.
- **Test:** (Optional/Manual) Stop PostgreSQL and verify `health.check` returns an error or "unhealthy" status.

---

## Implementation Priority & Effort

| Priority | Bug | Estimated Effort |
| :--- | :--- | :--- |
| **1** | Auth Middleware Infinite Loop | 30 mins |
| **2** | GC Service Startup | 15 mins |
| **3** | Dedup Wiring | 30 mins |
| **4** | Real Health Check | 20 mins |

**Order of execution:**
1. **Auth Fix** first, as it blocks all auth-enabled testing.
2. **GC Startup** next to ensure background processes are active.
3. **Dedup Wiring** to ensure data integrity.
4. **Health Check** to finalize observability.
