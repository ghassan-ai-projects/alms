# ALMS Phase 5 — Improved Fix Plan (P1 Criticals)

This document provides a refined, detailed implementation strategy for the 4 P1 bugs identified in the Phase 5 DeepSeek Audit. This plan replaces the previous shallow plan and includes root cause analysis, precise fixes, and verification steps.

## 1. Auth Middleware Infinite Loop

**Location:** `alms/internal/server/server.go:54-56`

### Root Cause Analysis
The server currently registers the `AuthMiddleware` by wrapping the entire `http.NewServeMux`. When auth is enabled, the `mux` is configured to handle the root path `"/"` by calling `AuthMiddleware(...)(mux)`. 
1. Request for `"/"` arrives.
2. `mux` matches `"/"` and calls the registered handler: the middleware.
3. Middleware validates token and calls `next.ServeHTTP(w, r)`, where `next` is the `mux`.
4. `mux` receives the request again, matches `"/"`, and calls the middleware again.
**Result:** Infinite recursion and stack overflow.

### Precise Fix
Wrap only the MCP handler (`streamableHTTPServer`) with the middleware before registering it to the `mux`.

**Code Change:**
```go
// alms/internal/server/server.go:ListenAndServe

// 1. Build the base handler
var handler http.Handler = server.NewStreamableHTTPServer(s.mcp)

// 2. Wrap it if auth is enabled
if s.cfg.Auth.Token != "" {
    handler = AuthMiddleware(s.cfg.Auth.Token)(handler)
}

// 3. Register specialized routes and the main handler
mux := http.NewServeMux()
mux.Handle("/dashboard", DashboardHandler())
mux.Handle("/", handler) // This correctly routes to the (possibly wrapped) handler
```

### Risk Assessment
- **Risk:** If other routes are added later that *also* need auth, they must be wrapped individually or the whole mux must be wrapped correctly (by wrapping a separate router or using a middleware chain).
- **Mitigation:** Standardizing on wrapping the `handler` before `mux.Handle` is safe for this architecture.

### Verification
- **Test:** `TestServerAuthLoop`: Start a test server with `Auth.Token` set. Use `net/http/httptest` to send a request. Confirm it returns `200 OK` (with token) or `401/JSON-RPC -32001` (without token) rather than crashing.

---

## 2. GC Service Never Started

**Location:** `alms/cmd/alms/main.go`

### Root Cause Analysis
The `GC` service (Garbage Collector) is implemented in `internal/service/gc.go` but the instantiation and lifecycle management (Start/Stop) are missing from the `main` entry point. The symbols exist but are unreachable at runtime.

### Precise Fix
Initialize the `GC` service in `main.go` using the existing `learningStore`, start it in the background, and ensure it stops gracefully.

**Code Change:**
```go
// alms/cmd/alms/main.go

// 1. Initialize GC service
gcSvc := service.NewGC(learningStore, service.DefaultGCConfig())

// 2. Start GC in background
gcSvc.Start(ctx)

// 3. Add Stop to graceful shutdown
defer gcSvc.Stop()
```

### Risk Assessment
- **Risk:** GC runs in a goroutine and performs `UpdateScore` and `SoftDelete` operations. If the interval is too short or the DB is slow, it could create contention.
- **Mitigation:** Use `DefaultGCConfig()` which sets a safe 24-hour interval.

### Verification
- **Test:** Log Check. Verify `slog` output contains `"GC started"` on boot and `"GC completed"` after the initial sweep.
- **Test:** Unit test `internal/service/gc_test.go` confirms `Sweep` logic correctly identifies and deletes records based on TTL/Score.

---

## 3. Dedup Not Wired to Tool

**Location:** `alms/internal/server/tools.go:250`

### Root Cause Analysis
The MCP tool `learning.store` calls `learning.Store(...)`. While `learning.go` contains a more advanced `StoreLearningWithDedup(...)` method that invokes the `DedupEngine`, this method is never used. Consequently, duplicate learnings are stored without check.

### Precise Fix
Update the `learning.store` tool handler to use `StoreLearningWithDedup`. This requires updating the response to include dedup status if desired, though the immediate priority is functional dedup.

**Code Change:**
```go
// alms/internal/server/tools.go: registerLearningStoreTools

// Change this:
// id, err := learning.Store(ctx, rec, supersedes)

// To this:
id, dedupResult, err := learning.StoreLearningWithDedup(ctx, rec, supersedes)
if err != nil {
    return mcp.NewToolResultError(err.Error()), nil
}

// (Optional but recommended) Update result map to show dedup status
return marshalResult(map[string]any{
    "learning_id":  id,
    "status":       "created",
    "is_duplicate": dedupResult.IsExactDup || dedupResult.IsNearDup,
})
```

### Risk Assessment
- **Risk:** `CheckNearDup` uses full-text search which might be slow on very large datasets.
- **Mitigation:** The current implementation uses a `limit` of 100 for search results; verify GIN index is healthy.

### Verification
- **Test:** Send two identical `learning.store` requests via MCP. The second request should return the same `learning_id` as the first (if exact match) or be flagged.

---

## 4. Health Check Static

**Location:** `alms/internal/server/tools.go:413-425`

### Root Cause Analysis
The `health.check` tool returns a hardcoded `{"status":"ok"}`. It accepts a `Registry` service but never uses it to verify the database connection or the actual state of the system (agent count).

### Precise Fix
1. Add a `Ping` check to the tool using a new method in the store layer or exposing it through a service.
2. Call `registry.AgentCount(ctx)` to return the actual count.

**Code Change:**
```go
// alms/internal/server/tools.go: registerHealthTools

// Handler implementation:
count, countErr := registry.AgentCount(ctx)
// Note: We need a way to ping the DB. Since Registry has the store, 
// we can add a Ping() method to Registry or use a store-level check.

status := "ok"
if countErr != nil {
    status = "degraded"
}

result := map[string]any{
    "status":      status,
    "agent_count": count,
    "version":     "0.1.0",
}
```

### Risk Assessment
- **Risk:** A slow database could cause the health check to hang or timeout.
- **Mitigation:** Use a context with a short timeout (e.g., 2s) for the health check.

### Verification
- **Test:** Call `health.check`. Confirm the JSON contains a numeric `agent_count`.

---

## Implementation Summary

### Fix Order
1. **Auth Fix:** Essential for any testing with auth enabled.
2. **Dedup Wiring:** Critical for data integrity during early ingest.
3. **GC Startup:** Ensures background lifecycle is active.
4. **Health Check:** Finalizes observability.

### Dependencies
- **Health Check** depends on `Registry.AgentCount` (already exists in `service/registry.go`).
- **Dedup Wiring** depends on `Learning.StoreLearningWithDedup` (already exists in `service/learning.go`).

### Effort Estimate
- Total time: ~2 hours.
- Complexity: Low (Wiring existing logic).
