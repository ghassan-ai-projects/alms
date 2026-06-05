# Phase 5 — P1 Fix Validation Checklist

## Success Criteria

Phase 5 is complete when ALL 4 P1 bugs are fixed, verified, and deployed to `main`.

### P1-1: Auth Middleware Infinite Loop
**Must pass:**
- [ ] `internal/server/server.go`: `AuthMiddleware` wraps the handler, NOT the mux
- [ ] With auth token set: server responds to valid MCP requests without crashing
- [ ] With auth token set: server returns error `-32001` for requests without token
- [ ] With auth token empty (dev mode): server passes requests through
- [ ] Test exists that starts server with token, sends valid request, expects 200
- [ ] Test exists that starts server with token, sends invalid request, expects -32001
- [ ] Test exists that starts server without token (dev mode), expects passthrough

### P1-2: GC Never Started
**Must pass:**
- [ ] `cmd/alms/main.go`: `NewGC(learningStore, DefaultGCConfig())` is called
- [ ] `cmd/alms/main.go`: `.Start(ctx)` is called before server starts
- [ ] `cmd/alms/main.go`: `defer gc.Stop()` is set for graceful shutdown
- [ ] Server logs `"GC started"` on boot
- [ ] `deadcode ./...` shows NO unreachable symbols from `internal/service/gc.go`
- [ ] Test exists that GC goroutine actually runs (log check or mock-based)

### P1-3: Dedup Not Wired
**Must pass:**
- [ ] `internal/server/tools.go`: `learning.store` tool calls dedup-capable method
- [ ] Two identical `learning.store` calls return the same `learning_id`
- [ ] `deadcode ./...` shows NO unreachable symbols from dedup engine
- [ ] Test exists: push duplicate learning, verify single record

### P1-4: Health Check Static
**Must pass:**
- [ ] `internal/server/tools.go`: `health.check` returns actual agent count
- [ ] `internal/server/tools.go`: `health.check` returns dynamic status (not hardcoded `"ok"`)
- [ ] Health check uses a short timeout context for PG ping
- [ ] Test exists: call `health.check`, verify JSON has `agent_count` field

### Overall Phase 5 Gate
- [ ] All 4 P1 fixes pass their criteria above
- [ ] `make ci-check` passes
- [ ] `golangci-lint run ./...` — 0 issues
- [ ] `deadcode ./...` — GC, Dedup, Scoring symbols are LIVE (not unreachable)
- [ ] `go test -race -count=1 ./...` — all pass
- [ ] `go test -race -count=1 -coverprofile=coverage.out ./...` — no regression in coverage
- [ ] All changes committed and pushed to `main`
- [ ] Progress log in `docs/plans/impl-01.md` updated with Phase 5 status
