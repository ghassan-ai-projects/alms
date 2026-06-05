# ALMS Phase 5 — Deep Quality Audit (DeepSeek Pro Second Opinion)

## Audit Overview
- **Audit Date:** 2026-06-05
- **Auditor:** DeepSeek Pro Review (second opinion)
- **Audit Scope:** All 9 categories from `phase-5-audit.md` — independent analysis
- **Previous Verdict:** ✅ **DEPLOY-READY**
- **DeepSeek Verdict:** ⚠️ **BLOCKED** — 4 critical/P1 findings must be addressed before deployment

---

## Independent Category Scores

| Category | Score | Notes |
|----------|-------|-------|
| 1. Architecture vs Reality | ⚠️ | Major auth-middleware wrapping bug; planned resource templates still missing |
| 2. Dead Code | ❌ | **25+ unreachable symbols** including entire GC, Scoring, Dedup, Storemock packages |
| 3. Complex Logic | ⚠️ | Gap-safe ack is correct in isolation but dead code in GC/Scoring creates trust issues |
| 4. Missing Features | ❌ | Dedup engine exists but is NEVER wired into the MCP tool; GC never started; 2 resources missing |
| 5. Test Quality | ⚠️ | Coverage figures mask the dead-code problem; tests only cover what's not dead |
| 6. Documentation | ✅ | Operations and Integration guides accurate |
| 7. Security | ⚠️ | Auth middleware wraps itself in an infinite loop when auth is enabled |
| 8. Deliverables Status | ⚠️ | GC, Scoring, Dedup are code-complete but non-functional in production |
| 9. Load & Integration Tests | ✅ | Integration tests pass, but don't test GC/Scoring/Dedup paths |

---

## Things the First Review Got Right

1. **Architecture layering:** Server → Service → Store → Models hierarchy is strictly maintained; no circular imports.
2. **Gap-safe ack algorithm:** The core algorithm in `SyncAck()` correctly fetches `ExpectedSyncIDs`, builds a set, checks for gaps, and returns `ErrGapDetected` with missing IDs. The `SyncAck` store layer uses an `ON CONFLICT DO NOTHING` pattern for idempotency.
3. **SQL injection via fmt.Sprintf:** The identified `fmt.Sprintf` usages at `agent_store.go:217` and `learning_store.go:156` are indeed safe — they only parameterize the **value** position (`$1`, `$2`), not user-controlled column names.
4. **Auth middleware error format:** Returns JSON-RPC error with code `-32001` and message `"unauthorized"`.
5. **Test coverage targets:** Models at 100%, Service at 80.7%, Config at 81%, Server at 55.3%.
6. **Missing resource templates:** `alms://agents/{id}` and `alms://learnings/{id}` are indeed unimplemented.
7. **Fuzz testing:** Present in `agent_test.go`.
8. **go vet passes cleanly.**
9. **Build compiles successfully.**

---

## Things the First Review Missed — ⚠️ CRITICAL

This is the most important section. The first review passed 9/9 categories but missed systemic issues.

### 🔴 P1: Auth Middleware Wraps Itself — Infinite Loop Bug

**File:** `internal/server/server.go:54-56`

```go
mux.Handle("/", handler)         // line 54 — mux handles "/" to mcp handler
mux.Handle("/", AuthMiddleware(s.cfg.Auth.Token)(mux))  // line 56 — wraps the ENTIRE mux
```

When `cfg.Auth.Token != ""`, line 56 replaces the `/` route with `AuthMiddleware(...)(mux)`. The **entire mux** (including the auth-protected routes) is passed to the auth middleware as `next`. Since the mux routes `/` to `next` and `next` IS the mux, any request matching `/` will:

1. Enter `AuthMiddleware`
2. If valid token → `next.ServeHTTP(w, r)` where `next = mux`
3. `mux` receives the request → `/` matches → calls the handler registered at line 56 (auth middleware again)
4. **Infinite loop** until stack overflow

**Reproduction:** Set `auth.token` in config, start server, send any MCP request with a valid token. The handler will recursively call itself.

**Actual intent should be:**
```go
var handler http.Handler = streamableHTTPServer
handler = AuthMiddleware(s.cfg.Auth.Token)(handler)  // wrap handler, not mux
mux.Handle("/", handler)
```

**Severity:** If auth is configured (production scenario), the server crashes on the first request.

### 🔴 P1: GC Never Started — Dead Code

**File:** `cmd/alms/main.go`

The `GC` service is imported (`internal/service/gc.go`) but is **never instantiated or started** in `main.go`. Deadcode confirms:

```
internal/service/gc.go:21:6  unreachable func: DefaultGCConfig
internal/service/gc.go:39:6  unreachable func: NewGC
internal/service/gc.go:48:14 unreachable func: GC.Start
internal/service/gc.go:89:14 unreachable func: GC.Stop
```

The plan says "GC scheduler" → "async goroutine" → "✅" but the code path is never reached from `main()`. The goroutine in `GC.Start()` literally never executes in production.

**Also:** `ScoringEngine` (8 unreachable symbols) and `DedupEngine.CheckNearDup`/`CheckExactDup` (4 unreachable symbols) — all never called from any running code path.

### 🔴 P1: Dedup Not Wired Into Learning.Store Tool

**File:** `internal/server/tools.go:250` calls `learning.Store()`, not `learning.StoreLearningWithDedup()`

The MCP tool `learning.store` (the primary insertion endpoint) calls `Learning.Store()` directly. `StoreLearningWithDedup()` exists but is **never invoked from any handler**, test, or entry point.

- `deadcode` confirms `StoreLearningWithDedup` is unreachable
- There is **no SHA256 hash column** in the DB schema
- There is **no UNIQUE constraint** on title+body
- Exact and near-duplicate detection is **completely non-functional** in production

**The `checkExactDup` implementation also has a design flaw:** It does a `Search` first (which uses `plainto_tsquery`), then iterates results calculating hashes. This is O(n) on search results and fragile — if a title is so different that `plainto_tsquery` doesn't match it, the hash check will miss the exact duplicate.

### 🔴 P1: Health.Check Tool Returns Static "OK" Without PG Ping

**File:** `internal/server/tools.go:413-425`

```go
func registerHealthTools(...) {
    // ...
    result := map[string]any{
        "status":  "ok",
        "version": "0.1.0",
    }
    return marshalResult(result)
}
```

The `health.check` tool **never pings PostgreSQL** and **never queries the agent count**, despite its description saying "Check server health: PG ping + agent count". It always returns `{"status":"ok"}` even if the database is down. The `registry` parameter is accepted but unused.

### 🟠 P2: ExpectedSyncIDs Does Not Filter by Agent Tags

**File:** `internal/store/learning_store.go:199-220`

`ExpectedSyncIDs` returns ALL learnings since the timestamp, regardless of whether they were scoped to the agent's tags. This means if learning records have tag-based routing (like protocols do), the gap-safe ack will still require all learnings to be acknowledged, even ones the agent wouldn't normally receive.

This isn't an immediate bug because the current sync doesn't filter by agent tags either. But if tag-based sync is ever added, `ExpectedSyncIDs` must be updated to match the same filtering.

### 🟠 P2: SyncAck Cursor Uses Last ID in Array, Not Max Timestamp

**File:** `internal/store/learning_store.go:174`

```go
lastID := learningIDs[len(learningIDs)-1]
query := `
    UPDATE agents a
    SET last_sync_ts = (
        SELECT l.created_at FROM learnings l
        WHERE l.learning_id = $2
    ),
```

If learning IDs are provided in non-chronological order (the SyncAck validates gaps but doesn't enforce order), the cursor may be set to a timestamp earlier than some acknowledged learnings. This means on the next sync, those "earlier" learnings would be re-sent, causing duplicate work.

**Fix:** Either enforce chronological ordering in `SyncAck` validation or use `MAX(l.created_at)` instead of looking up the last element.

### 🟠 P2: AgentStore List Ignores Errors on JSON Unmarshal

**File:** `internal/store/agent_store.go:252-255`

```go
if len(capBytes) > 0 {
    _ = json.Unmarshal(capBytes, &spec.Capabilities)
}
if len(metaBytes) > 0 {
    _ = json.Unmarshal(metaBytes, &spec.Metadata)
}
```

The errors from `json.Unmarshal` are silently discarded with `_ =`. While `Get` properly checks these errors, `List` does not. If corrupt JSON data lands in the database (e.g., from an external tool writing directly to the DB), `List` will silently return agents with zero-value capabilities/metadata, potentially confusing tools that rely on these fields.

### 🟠 P2: AgentStore Create Uses UPSERT Instead of INSERT

**File:** `internal/store/agent_store.go:40-53`

The `Create` method uses `ON CONFLICT DO UPDATE`, meaning it **silently overwrites** existing agents instead of returning `ErrConflict`. The `Registry.Register()` method does a `Get` before `Create` to check for conflicts, but this creates a TOCTOU race condition: if two concurrent registrations happen for the same agent ID, both Get checks pass, and one silently overwrites the other.

**Fix:** Use `ON CONFLICT DO NOTHING` and check `RowsAffected()` in the store layer, or use `ON CONFLICT DO UPDATE` only in a separate `Upsert` method.

### 🟡 P3: Missing Template Resources (Confirmed First Review Finding)

**File:** `internal/server/resources.go`

Planned 7 resources, only 5 implemented. Missing:
- `alms://agents/{id}` (individual agent detail)
- `alms://learnings/{id}` (individual learning detail)

The first review correctly flagged this as missing but downgraded it to P3. It's consistent and well-documented.

### 🟡 P3: Storemock Package Is Entirely Dead Code

**File:** `internal/service/storemock/mock.go`

All 50+ exported functions in the `storemock` package are marked unreachable by `deadcode`. The test files in `internal/service/` directly create mock instances via `storemock.NewAgentStore()`, `storemock.NewLearningStore()`, etc., but those are test-only references, and `deadcode` treats test files as a separate scope.

**Verdict from DeepSeek:** This is expected for test-only packages. The first review was correct to flag it as P3/minor. Not a blocker.

### 🟡 P3: redundant `Ensure *pgx.Rows` comment

**File:** `internal/store/learning_store.go:257-258`

```go
// Ensure *pgx.Rows implements the scanLearnings scanner interface.
// keep imports
```

This appears to be a placeholder comment from development. If the intent was to use `tools.go` or an import guard, it doesn't accomplish anything — `pgx.Rows` is already used in the function signature.

---

## Categories Summary (Independent Scores)

### 1. Architecture vs Reality
**Score: ⚠️** 
- ✅ Module tree matches plan
- ✅ Layering maintained (server → service → store → models)
- ✅ No circular imports
- ❌ Auth middleware routing bug (mux wrapping itself)
- ❌ `health.check` MCP tool doesn't do what it says (PG ping + agent count)
- ✅ Dependency injection pattern is clean

### 2. Dead Code Detection
**Score: ❌**
- 25+ unreachable functions found by `deadcode` (beyond just storemock)
- **Entire GC, ScoringEngine, DedupEngine (except HandleSupersession) packages** are unreachable from binary entry point
- `postgres.go:Ping` is unreachable (only `NewPool` uses inline ping)
- `golangci-lint` reports 0 issues (unused linter apparently doesn't flag unreachable via deadcode analysis)

### 3. Complex Logic Audit
**Score: ⚠️**
- Gap-safe ack algorithm: ✅ Correct in isolation
- Gap-safe ack edge case: ⚠️ SyncAck cursor uses last array element, not max timestamp
- Dedup: ⚠️ Implementation exists but never wired to MCP tools; SHA256 exact dedup at app-level only (no DB constraint)
- GC: ⚠️ Code is correct but unreachable
- Scoring: ⚠️ Code is correct but unreachable
- Auth middleware: ❌ Infinite loop when auth is enabled

### 4. Missing Features
**Score: ❌**
- Dedup is **completely non-functional** in production (16 tools found but dedup tool path unused)
- GC is **never started** in main.go
- 2 resource templates still missing (agents/{id}, learnings/{id})
- `internal/validate` package merged into models (as first review noted — acceptable)

### 5. Test Quality
**Score: ⚠️**
- ✅ Models: 100% coverage
- ✅ Service: 80.7% coverage (with good gap-safe ack tests)
- ⚠️ Server: 55.3% (no test for auth middleware being wired into server startup)
- ✅ Fuzz tests exist for Validate
- ⚠️ **No tests confirm GC is actually started from main** (it never is, and tests don't catch this)
- ⚠️ No tests confirm dedup is wired into `learning.store` tool
- ⚠️ Integration tests don't cover GC or Scoring paths
- ✅ gap-safe ack end-to-end test exists (`TestSyncerCrashedAgentRecovers`)

### 6. Documentation
**Score: ✅**
- Operations and Integration guides are accurate
- `impl-01.md` progress log up to date
- `README.md` has basic project description

### 7. Security
**Score: ⚠️**
- ✅ SQL injection: All queries properly parameterized
- ✅ Auth middleware returns proper MCP error format (-32001)
- ✅ Dev mode passthrough (empty token) correct
- ❌ **Auth middleware wraps mux, not handler — auth breaks server entirely when enabled**
- ⚠️ DSN config embeds default password in source (`postgres://alms:alms@...`) — should use env vars only
- ⚠️ `List()` silently swallows JSON unmarshal errors (data corruption risk)

### 8. Deliverables Status
**Score: ⚠️**

| Deliverable | Code Status | Verdict |
|------------|-------------|---------|
| Agent registry | ✅ | Functional |
| Learning sync | ✅ | Functional |
| Gap-safe ack | ✅ | Functional |
| Protocol pull | ✅ | Functional |
| learning.store | ⚠️ | Stores without dedup |
| learning.search | ✅ | Functional |
| Dedup engine | ❌ | **Never called from MCP tools** |
| Scoring engine | ❌ | **Never instantiated** |
| GC scheduler | ❌ | **Never started from main** |
| Dashboard | ✅ | Functional |
| systemd deploy | ✅ | Scripts exist |
| Backup cron | ✅ | Scripts exist |
| Newsletter agent | ✅ | Scripts exist |
| CI pipeline | ✅ | Present |
| Integration tests | ✅ | Pass |
| Load testing | ✅ | Pass |

### 9. Final Recommendations

Issues marked with P1 **must** be fixed before deployment.

---

## Priority-Ordered Fix List

### P1 (Must Fix Before Deploy)

| # | Issue | File | Fix |
|---|-------|------|-----|
| 1 | Auth middleware infinite loop | `internal/server/server.go:54-56` | Wrap `handler`, not `mux` in `AuthMiddleware()` |
| 2 | GC never started | `cmd/alms/main.go` | Instantiate `NewGC(learningStore, DefaultGCConfig())` and call `.Start(ctx)` before `srv.ListenAndServe()` alongside defer `gc.Stop()` |
| 3 | Dedup not wired to `learning.store` tool | `internal/server/tools.go:250` | Call `learning.StoreLearningWithDedup()` instead of `learning.Store()`, or integrate dedup check into `learning.Store()` |
| 4 | `health.check` returns static ok | `internal/server/tools.go:413-425` | Actually call PG ping and agent count, return real values |

### P2 (Fix Before Phase 6)

| # | Issue | File | Fix |
|---|-------|------|-----|
| 5 | SyncAck cursor uses last array element, not max timestamp | `internal/store/learning_store.go:174` | Use `SELECT MAX(l.created_at) FROM learnings WHERE learning_id = ANY($2)` |
| 6 | AgentStore Create uses UPSERT (TOCTOU race) | `internal/store/agent_store.go:40-53` | Use `ON CONFLICT DO NOTHING` + check `RowsAffected` |
| 7 | List() silently discards JSON errors | `internal/store/agent_store.go:252-255` | Return unmarshal errors instead of `_ =` |
| 8 | ExpectedSyncIDs doesn't filter by agent context | `internal/store/learning_store.go:199-220` | Add tag/type filtering parameters |
| 9 | ScoringEngine not wired into GC | `internal/service/gc.go` | Have GC delegate score decay to ScoringEngine.BatchApplyDecay |

### P3 (Nice-to-Have)

| # | Issue | File | Fix |
|---|-------|------|-----|
| 10 | Missing resource templates | `internal/server/resources.go` | Add `alms://agents/{id}` and `alms://learnings/{id}` |
| 11 | Placeholder comment in learning_store | `internal/store/learning_store.go:257` | Remove dev comment |
| 12 | Default DSN in source code | `internal/config/config.go:37` | Remove default password from source; document env-only setup |

---

## Final Verdict: ⚠️ BLOCKED — Do Not Deploy

**Reasoning:**

The first review found the project "deploy-ready" with minor P2/P3 notes. **This second opinion disagrees.**

While the code quality is generally strong (clean layering, proper testing patterns, good error handling), there are **4 critical issues** that make the production deployment non-functional or dangerous:

1. **Auth infinite loop** means the server crashes on the first production request.
2. **GC never starts** means learning records accumulate forever with no cleanup.
3. **Dedup is never called** means duplicate learning records flood the database.
4. **Health check lies** means monitoring tools report the system as healthy when it's not.

These are not style issues — they are runtime bugs that make the promised features non-functional.

**Recommended Workflow:**
1. Apply the 4 P1 fixes (estimated 1-2 hours)
2. Apply the 4 P2 fixes (estimated 1-2 hours)
3. Re-run `deadcode ./...` to confirm GC, Scoring, and Dedup paths are live
4. Re-run test suite and integration tests
5. Then deploy

**For posterity:** Future audits should write a black-box integration test that starts `main.go` and sends actual MCP requests to verify end-to-end registration, GC startup, and dedup enforcement — not just unit tests of isolated packages.

---

*DeepSeek Pro Review — 2026-06-05*
