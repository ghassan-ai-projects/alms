# GO Design Review — ALMS

**Reviewer:** Gemini (senior Go infrastructure engineer, 10+ yrs)
**Document:** `alms-go-design.md`
**Date:** 2026-06-05
**Project:** ALMS — Agent Learning Management System (Go MCP server)
**Team size:** Single developer (👤 solo)

---

## Overall Score: 6.5 / 10

**TL;DR:** Solid fundamentals, clearly informed by real-world Go experience. But it's about 20% over-engineered for a single-developer project with 5-7 packages. The pre-commit stack, linter config, and Makefile are a cargo-cult from a 50-person team. The design documents well what it *should* do but skips important practical details (interface shape, actual config parsing, MCP testing strategy).

---

## 1. Pragmatism vs Over-Engineering

### Pre-commit hooks: 7 hooks is too many for this project

The `.pre-commit-config.yaml` defines **7 hooks from external repos + 4 local hooks = 11 steps** in the pre-commit pipeline. For a 5-package project cared for by one person:

| Hook | Verdict | Rationale |
|------|---------|-----------|
| `trailing-whitespace` | ✅ Keep | Free, fast, catches sloppy edits |
| `end-of-file-fixer` | ✅ Keep | One blank line at EOF, no trailing newlines |
| `check-yaml` | ✅ Keep | Cheap, catches busted YAML before golangci-lint |
| `check-added-large-files` | ✅ Keep | Good habit |
| `check-merge-conflict` | ✅ Keep | Valid for any project with branches |
| `go-fmt` | 🔶 Move to editor | `gofmt` should run on save, not pre-commit |
| `go-vet` | 🔶 Merge into golangci-lint | `govet` is already enabled in `.golangci.yml` |
| `go-imports` | 🔶 Merge into golangci-lint | `goimports` is already enabled |
| `go-unit-tests` | ❌ Remove from pre-commit | Tests belong in CI. Pre-commit should be < 5 seconds. Tests take 10-30s |
| `validate-toml` | ❌ Remove | You have no TOML files. The `docs/` doesn't exist yet |
| `golangci-lint` | ✅ Keep but make faster | `--timeout=3m` is defensive; real run is ~15s |
| `deadcode` | ❌ Remove | Redundant — `unused` linter in golangci-lint covers this |
| `go-mod-tidy` | ❌ Remove | `go mod tidy` on every commit is overkill. Run it manually after dep changes |
| `govulncheck` | ❌ Remove from pre-commit | Network call on every commit. Belongs in weekly CI cron |

**Recommended pre-commit:** 3 hooks, ~3 seconds:
```yaml
hooks:
  - id: trailing-whitespace
  - id: end-of-file-fixer
  - id: check-yaml
  - id: golangci-lint       # --fast --timeout=30s, already includes gofmt+govet+goimports
```

Put `go-unit-tests`, `deadcode`, and `govulncheck` in the Makefile as optional targets. Run them before push, not every commit.

### Makefile has too many targets for solo dev

16 targets is heavy. For practice:
- `make build`, `make lint`, `make test` — these are the daily drivers
- `make ci-check` — bundles them for CI
- `make vulncheck`, `make deadcode` — weekly checks, not daily

The `clean` target is essentially vestigial for Go (deleting `coverage.out`). That's fine — it's a convention.

**Bottom line:** The Makefile is fine but the pre-commit hook assembly is the most over-engineered part of this design. For a solo dev, 3 hooks + `make ci-check` before push is plenty.

---

## 2. Dependency Minimalism

### "3 external packages" claim is unrealistic

The design claims "stdlib + 3 packages" but lists 3 dependencies in the table:

| Declared | Actual behavior |
|----------|-----------------|
| `pgx/v5` | ✅ This is necessary. Realistic. |
| `mark3labs/mcp-go` | ✅ Necessary — MCP protocol is complex. |
| `google/uuid` | ⚠️ Could use `crypto/rand` + `fmt.Sprintf` instead. 30 lines. See below. |

**Missing from the list:**
1. **Config parser** — Section 2.5 says "use `spf13/viper` for config merging, or stdlib `flag` + env parsing for minimal deps." This is a binary XOR choice that matters a lot. `viper` pulls in ~16 indirect dependencies. If you roll your own YAML+env parsing, you need `gopkg.in/yaml.v3` — that's a 4th dep. The design waffles on this.

2. **`database/sql` is listed but you won't use it with pgx directly.** If you use `pgx/v5/pgxpool` directly (recommended), you don't need `database/sql`. If you use `database/sql` with `pgx/v5/stdlib` driver, you get the stdlib interface but lose pgx-native features (copy protocol, `pgtype`, `Listen/Notify`). The design should pick a lane.

3. **Test dependencies** — `testcontainers-go` or `dockertest` for integration tests. If you mock the store interface (good practice), you don't need this for unit tests, but the design says "Integration tests against real PostgreSQL via testcontainers" — that adds at least one test-only dependency.

### UUID: Is it worth importing?

```go
google/uuid.New()
```

Replace with:
```go
import "crypto/rand"

func NewUUID() string {
    b := make([]byte, 16)
    rand.Read(b)
    b[6] = (b[6] & 0x0f) | 0x40  // UUID v4
    b[8] = (b[8] & 0x3f) | 0x80  // variant bits
    return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}
```

**Verdict:** `google/uuid` is a small, stable dep (zero indirect deps) — it's fine to keep. But if you want to hit "3 external deps total," this is the easiest one to eliminate. Takes 20 lines of code. I'd keep it and accept 4 deps.

### Realistic dependency count: 4-5

| Dep | Can you cut it? | Should you? |
|-----|-----------------|-------------|
| `pgx/v5` | No | PostgreSQL driver is non-negotiable |
| `mcp-go` | No | MCP protocol handshake + transport is non-trivial |
| `google/uuid` | Yes, with 20 lines of stdlib code | Borderline. Keep for clarity. |
| `yaml.v3` (or viper) | No if using YAML config | Needed unless you go all-env-vars |
| Test deps (testcontainers) | Yes, if mocking store | ✅ Mock the store. Skip testcontainers. |

**Score honesty:** The design says "3 packages." In practice, you'll have **4-6 dependencies** (including test-only and config parsing). That's still excellent minimalism — far better than most Go projects. But don't claim 3 when it's 5.

---

## 3. golangci-lint Config

### Enabled linters: Good core, 2 problematic

Let me walk through all 15:

| Linter | Verdict | Rationale |
|--------|---------|-----------|
| `errcheck` | ✅ Must-have | Standard for any Go project |
| `gosimple` | ✅ Must-have | Catches `if len(x) > 0` → use `x != nil` etc. |
| `govet` | ✅ Must-have | Built-in, catches real bugs |
| `ineffassign` | ✅ Must-have | Catches dead code patterns |
| `staticcheck` | ✅ Must-have | SA series checks are the gold standard |
| `unused` | ✅ Must-have | Replaces `deadcode` CLI |
| `gosec` | ✅ Good | Catches SQL injection, XSS, hardcoded creds |
| `revive` | ✅ Good | Replacement for `golint` |
| `misspell` | ✅ Good | Cheap, catches comment typos |
| `goimports` | ⚠️ Low value | `goimports` is better as `gofumpt` + IDE save hook. In CI it just checks ordering |
| `prealloc` | 🔶 Edge-of-noisy | Pre-allocation advice is often premature. It fires on trivial loops |
| `noctx` | ✅ Good for this project | Catches missing `context.Context` on DB calls — directly relevant |
| `unconvert` | 🔶 Low value | Fires maybe once a month. Rarely catches bugs |
| `wastedassign` | 🔶 Low value | Catches `x := x` patterns that are almost never in real code |

### What's missing:

1. **`gofumpt`** (not `goimports`) — More strict formatting than `gofmt`. Forces line breaks, removes empty `()`s. One formatting standard is better than two (`goimports` + `gofmt` are both running).

2. **`thelper`** — Marks helper functions in tests so line numbers are correct. Critical for test readability.

3. **`wrapcheck`** — Ensures errors from external packages are wrapped. Complements `errcheck`. The design says "always wrap with context" — this linter enforces it.

4. **`whitespace`** — Catches blank lines before/after blocks. Personal style, but useful for consistency.

### What to cut:

`prealloc`, `unconvert`, `wastedassign` — these fire infrequently and rarely find bugs. They add noise. With 15 linters, 3 are borderline useless.

**Recommended enabled:** `errcheck`, `gosimple`, `govet`, `ineffassign`, `staticcheck`, `unused`, `gosec`, `revive`, `misspell`, `noctx`, `thelper`, `wrapcheck` = 12 focused linters.

### That said: 15 linters is fine

For a solo dev, 15 linters is not a problem. golangci-lint runs in ~10-15s on a 5-package project. The config is clean and well-organized. This is not over-engineered — it's a solid baseline.

**Score:** 7/10 for the linter config. Good but could be tightened.

---

## 4. Testing Viability

### "Mock the store interface in service tests" — what's the interface?

The document says:
```
internal/service/
    registry_test.go
    sync_test.go
    learning_test.go

Mock the store interface in service tests:
    type AgentStore interface { ... }
```

**Problem:** The interface is not defined anywhere in the design. This is the single most important gap in the document.

A service-layer test for `registry.go` would need something like:

```go
// internal/service/registry.go (or internal/store/interfaces.go)
type AgentStore interface {
    Create(ctx context.Context, spec models.AgentSpec) error
    Get(ctx context.Context, agentID string) (models.AgentSpec, error)
    Update(ctx context.Context, spec models.AgentSpec) error
    Delete(ctx context.Context, agentID string) error
    Heartbeat(ctx context.Context, agentID string) (time.Time, error)
    List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error)
}

type LearningStore interface {
    Create(ctx context.Context, record models.LearningRecord) (string, error)
    Get(ctx context.Context, learningID string) (models.LearningRecord, error)
    Sync(ctx context.Context, agentID string, since time.Time, ltype, tags string) ([]models.LearningRecord, error)
    SyncAck(ctx context.Context, agentID string, learningIDs []string) error
    Search(ctx context.Context, query string, ltype, tags string, limit int) ([]models.LearningRecord, error)
    SoftDelete(ctx context.Context, learningID string) error
}

type ProtocolStore interface {
    Create(ctx context.Context, record models.ProtocolRecord) (string, error)
    Pull(ctx context.Context, agentID string, sinceID string) ([]models.ProtocolRecord, error)
    List(ctx context.Context, agentID string) ([]models.ProtocolRecord, error)
}
```

### The gap-safety test is the critical one

The "gap-safe ack validation" is the trickiest business logic in this system. The design mentions it in `internal/service/`:

> `sync.go` — Learning sync + gap-safe ack validation

But doesn't detail the gap-safe algorithm. For a proper test:

```go
// The ack validates that learning_ids form a contiguous range
// with no gaps relative to the agent's last acknowledged batch.
// Example: agent acks [LRN-006, LRN-007] and previously acked [LRN-005].
// If agent acks [LRN-006, LRN-008] without 007 → REJECT.
```

This validation logic is **~15 lines of Go** but crucial to test. The design doesn't specify what happens on gap detection (reject entire batch? accept contiguous prefix?).

### Coverage targets are reasonable

- Service layer 80%+ ✅ Realistic for pure logic with mocked store
- Store layer 60%+ ✅ Integration tests or `pgx` mock. 60% is achievable
- Server layer 40%+ ✅ MCP handler testing is painful. 40% is honest

### Test tooling

Missing from the design: what mock framework? Options:

| Approach | Verdict |
|----------|---------|
| Interface-based manual mocks in `storemock/` | ✅ Best for small projects. No external deps. |
| `golang/mock` / `mockgen` | ❌ Adds a codegen step. Overkill for 3 interfaces. |
| `testify/mock` | ⚠️ Brings in testify. If you use testify for assertions anyway, fine. |
| `stretchr/testify` for assertions only | ✅ Bring in `testify/assert` — it's worth the dep for table-driven tests |

**Recommendation:** Write 3 manual mock structs in `internal/service/registry_test.go` et al. No mock framework. Use `testify/assert` for assertions (one small dep worth keeping).

### Verdict on testing section: 5/10

The section is aspirational but vague. The undefined interfaces, undefined mock strategy, and unspecified gap-safe logic mean a developer reading this would still have to design 30-40% of the test infrastructure from scratch. For a solo project, that's fine — but the design doc should at least sketch the interface signatures.

---

## 5. AGENTS.md Template

### Is it useful? Yes, absolutely.

For AI-assisted development (Cline, Copilot, Codex), an AGENTS.md is the single highest-ROI documentation investment. A well-written AGENTS.md:

1. Eliminates "what framework/language/db?" questions on every new task
2. Provides the import path conventions so the AI generates correct code
3. Prevents the AI from adding unnecessary dependencies

### Current sections:

| Section | Verdict | Problem |
|---------|---------|---------|
| Project | ✅ Good | Clear scope and language |
| Structure | ✅ Good | Maps dirs to package names |
| Quality | ✅ Good | `make` targets are unambiguous |
| Rules | ✅ Good | Actionable constraints |
| First Run | 🔶 Has a gap | See section 7 |

### Missing sections to add:

1. **"Naming conventions"** — AIs generate inconsistent naming:
   ```markdown
   ## Naming
   - Files: snake_case.go (e.g., `agent_store.go`)
   - Types: PascalCase for exported, camelCase internal
   - Interface names: `Methoder` suffix (e.g., `AgentCreator`, not `IAgent`)
   - Error vars: `ErrXxx` (e.g., `ErrNotFound`)
   - Acronyms: all caps in Go (`HTTP`, `ID`, `URL`, `MCP`)
   ```

2. **"Testing conventions"**
   ```markdown
   ## Testing
   - Table-driven tests with `t.Run(name, func(t *testing.T) {})`
   - Store interfaces mocked manually (no mockgen)
   - Use `testify/assert` for assertions
   - Integration tests: `go test -tags=integration` with `test.main` check for `*testing.T`
   ```

3. **"DB migration conventions"**
   ```markdown
   ## Migrations
   - Files: `internal/store/migrations/NNNN_name.sql`
   - Use `golang-migrate/migrate` or manual `scripts/migrate.sh`
   - Never modify an existing migration after merge
   ```

4. **"Commit style"** (optional but useful for AIs generating commits)
   ```
   feat: add agent heartbeat tool
   fix: handle nil context in store
   chore: bump pgx to v5.7
   ```

### Verdict: 7/10

The AGENTS.md template is a good start and the highest-ROI part of this design. Adding *naming conventions* and *testing conventions* would make it excellent.

---

## 6. The 5 Anti-Patterns — Which is Most Dangerous?

The design lists:
1. Mega-constructors with 80+ parameters
2. DB init spread across main.go
3. Package-per-feature for trivial logic
4. `defer` hidden mid-function
5. `_` imports in main.go for side effects
6. *(bonus)* Error checks without wrapping

### #3 is the most dangerous for this project

**Why:** In a 5-package project, the natural temptation is to create `internal/registry/`, `internal/sync/`, `internal/learning/`, `internal/protocol/` — each with one file and 50 lines of code. This is the "package-per-feature" anti-pattern.

**The consequence for ALMS:**
```go
// ❌ Package-per-feature (bad)
internal/
  registry/
    registry.go    // 40 lines
  sync/
    sync.go        // 50 lines
  learning/
    store.go       // 60 lines
    search.go      // 30 lines
  protocol/
    protocol.go    // 45 lines

// ✅ Layer-per-package (good — the design's intent)
internal/
  service/         // All business logic
    registry.go
    sync.go
    learning.go
  store/           // All data access
    agent_store.go
    learning_store.go
    protocol_store.go
  models/
```

**For a solo developer on a small project, anti-pattern #3 is seductive** because each feature feels "self-contained." You start with `internal/registry/` and by Week 2 you have 8 packages with 3 files each. Then you realize `learning/sync.go` needs to call `registry/GetAgent()` and you're fighting import cycles.

### Honorable mention: #5 (`_` imports in main.go)

This is subtle but dangerous for ALMS because `pgx` driver registration happens via `_` import if using `database/sql`:

```go
import (
    _ "github.com/jackc/pgx/v5/stdlib"  // implicit driver registration
)
```

If someone refactors and removes the import thinking it's unused, the DB connection silently breaks at runtime. `go vet` won't catch it. `deadcode` won't catch it. **You get a runtime nil pointer on `sql.Open()`.** That's a hard-to-debug 30-minute panic.

**If using pgx natively** (no `database/sql`), anti-pattern #5 is irrelevant. The design should explicitly state: "Use pgx native pool, not `database/sql` + driver import."

---

## 7. Bootstrap Path — "First Run" in AGENTS.md

### Current First Run:

```markdown
## First Run
1. `docker compose up -d db`     # Start PostgreSQL
2. `cp deploy/alms.yaml ~/.alms/alms.yaml`
3. `export ALMS_PG_DSN=postgres://alms:alms@localhost:5432/alms_db`
4. `go run ./cmd/alms/ --migrate`
5. `go run ./cmd/alms/`
```

### Completeness check: 5/10

**What's fine:**
- Docker compose for PostgreSQL ✅ Correct
- Config file copy ✅ Correct
- Environment variable per design ✅
- `--migrate` flag ✅ Good design choice

**What's missing — the "first five minutes" gaps:**

1. **Go version check.** If someone has Go 1.21, `slog` (1.21+) and routing (1.22+) might work, but Go 1.26 features won't.
   ```markdown
   # Prerequisites:
   - Go 1.22+ (for enhanced routing + slog)
   - Docker (for dev PostgreSQL)
   - golangci-lint (optional, for linting)
   ```

2. **Database creation.** The DSN assumes `alms_db` exists. Docker compose creates the container but not the database unless you define it in the compose file.
   ```markdown
   1. `docker compose up -d db`
   2. `createdb -h localhost -U postgres alms_db`   # ← Missing step
   ```

3. **Config file contents.** What does `alms.yaml` look like? A quick example:
   ```markdown
   # deploy/alms.yaml
   server:
     port: 8001
   database:
     dsn: "postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"
   auth:
     token: "change-me-in-production"
   ```

4. **Migration tool install.** `--migrate` implies a migration tool. Is it `golang-migrate/migrate` CLI? A custom `scripts/migrate.sh`? An in-process migration in `main.go`? The design should specify:
   ```markdown
   # Option A: Use golang-migrate CLI
   go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
   migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up

   # Option B: Built-in --migrate flag (runs on startup)
   go run ./cmd/alms/ --migrate
   ```

5. **Verification step.** How does the developer know it worked?
   ```markdown
   # Verify:
   curl -X POST http://localhost:8001/mcp \
     -H "X-ALMS-TOKEN: change-me-in-production" \
     -H "Content-Type: application/json" \
     -d '{"method":"tools/list","params":{}}'
   # Should return JSON with tool list
   ```

6. **Tear down.** How to clean up:
   ```markdown
   # Stop:
   docker compose down
   ```

### Enhanced First Run would be:

```markdown
## First Run

### Prerequisites
- Go 1.22+ (`go version`)
- Docker + Docker Compose
- PostgreSQL 16 client (`psql --version`) — optional, for manual inspection

### Setup (5 minutes)
```bash
# 1. Clone + enter project
git clone <repo> && cd alms

# 2. Start PostgreSQL
docker compose up -d db          # Creates 'alms' user + 'alms_db' database

# 3. Create default config
cp deploy/alms.yaml ~/.alms/alms.yaml
# Edit ~/.alms/alms.yaml to set auth token if desired

# 4. Set environment (overrides config file secrets)
export ALMS_PG_DSN=postgres://alms:alms@localhost:5432/alms_db?sslmode=disable
export ALMS_AUTH_TOKEN=dev-token-abc123

# 5. Run migrations
go run ./cmd/alms/ --migrate     # Applies internal/store/migrations/*.sql

# 6. Start server
go run ./cmd/alms/               # Listens on :8001

# 7. Verify
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-token-abc123" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' \
  | jq .
```

### Quick Test
```bash
# Register a test agent
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev-token-abc123" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"agent.register","arguments":{"agent_id":"test-agent-1","agent_type":"systemd"}}}' \
  | jq .
```

### Tear Down
```bash
docker compose down
```
```

### Verdict: 5/10

The current "First Run" is 5 steps that assume a lot of unstated context (pre-installed tools, DB already created, config format known). For a solo dev who wrote the design, it's fine. For anyone else picking up the project, it's incomplete. Add Go version requirements, config file example, verification step, and migration tool instructions.

---

## 8. Additional Observations

### 8.1 Server boundary vs MCP specifics

The design describes `internal/server/` as "MCP server setup" but doesn't specify which MCP transport. `mark3labs/mcp-go` supports:
- **Streamable HTTP** (the 2025+ MCP standard)
- **SSE** (Server-Sent Events)
- **stdio** (child process protocol)

For a systemd-deployed long-running server, **Streamable HTTP** is the right choice. The design should state this explicitly. If someone accidentally implements SSE, they'll have a harder time with load balancing and health checks.

### 8.2 `cmd/alms/main.go` is under-described

The design says "Entry point — thin: parse flags, init services, start server." For a single-developer project, this is sufficient. But if service init is thin, the config loading + dependency injection should be in `internal/config/` and `internal/server/` respectively. A sketch of `main.go` would help:

```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer cancel()

    cfg := config.Load()  // from flag, env, and yaml
    pool := store.NewPool(ctx, cfg.PGDSN)
    defer pool.Close()

    regSvc := service.NewRegistry(pool)   // or store interface
    learnSvc := service.NewLearning(pool)
    srv := server.New(cfg, regSvc, learnSvc)

    slog.Info("starting ALMS", "port", cfg.Port)
    if err := srv.ListenAndServe(ctx); err != nil {
        slog.Error("server exited", "error", err)
        os.Exit(1)
    }
}
```

This level of detail would make the "thin main" constraint concrete rather than abstract.

### 8.3 Middleware is mentioned but not designed

`internal/server/middleware.go` — "Auth middleware." For a static bearer token with one check, this is ~30 lines. But MCP servers have specific error response formats. The middleware should map an auth failure to the MCP JSON-RPC error format:

```json
{"jsonrpc":"2.0","id":1,"error":{"code":-32001,"message":"Unauthorized"}}
```

This is easy to get wrong. The design should note that middleware must return MCP-formatted errors, not plain HTTP 401.

### 8.4 No mention of `go 1.22` routing vs chi

Section 3 says "Go 1.22+ has enhanced routing" — yes, `net/http` in Go 1.22 supports `GET /mcp/{tool}` patterns natively. The design should state: **"Use stdlib `net/http` mux, no chi/gorilla/mux."** That's a concrete minimalism win.

### 8.5 The architecture plan is Python, the design is Go

The `alms-architecture-plan.md` specifies Python + FastAPI. The Go design is clearly a technology pivot. This isn't a problem per se, but the Go design should explicitly address the differences:
- Python version has `tool_catalog` (aggregated tool catalog from all agents)
- Python version has `prompts.py` for MCP prompt templates
- Go design omits both — intentionally or as an oversight?

---

## Summary Score Card

| Dimension | Score | Notes |
|-----------|-------|-------|
| **Structure & layering** | 8/10 | Good Go-idiomatic layout. Clean dependency direction. |
| **Dependency minimalism** | 5/10 | Claims 3, realistically 4-6. Waffles on config parser. |
| **Pre-commit & CI** | 4/10 | 11 hooks for a solo project. 3 would suffice. |
| **Linter config** | 7/10 | 15 linters, good choices. Some noise (prealloc, unconvert, wastedassign). |
| **Testing strategy** | 5/10 | Vague interfaces, no mock plan, no gap-safe algorithm. |
| **AGENTS.md** | 7/10 | Good template. Missing naming + testing conventions. |
| **First Run** | 5/10 | Missing prerequisites, config example, verification. |
| **Anti-pattern awareness** | 8/10 | Correctly identified. #3 (package-per-feature) is the real threat. |
| **Pragmatism** | 5/10 | Good instincts, but about 20% over-engineered for solo dev. |
| **Overall** | **6.5/10** | Solid skeleton, needs practical density added. |

---

## What I'd Actually Change (Priority Order)

1. **Cut pre-commit hooks to 3** — `trailing-whitespace`, `end-of-file-fixer`, `golangci-lint`. Move tests + vulncheck to CI.

2. **Define the store interfaces** — Even a sketch in the design doc. Three interfaces, 15 methods total. The service tests depend on these shapes.

3. **Pick a config parsing strategy** — Either viper (with its dependency cost acknowledged) or `gopkg.in/yaml.v3` + manual env parsing. Don't leave it as a TODO.

4. **Add golangci-lint rules** `thelper` and `wrapcheck` — They enforce patterns directly relevant to this project. Cut `prealloc` and `unconvert`.

5. **Flesh out AGENTS.md** — Add naming conventions, testing conventions, and a longer First Run with verification.

6. **Add an `examples/` or `docs/sync-flow.md`** — The gap-safe ack validation is the most complex business logic. A sequence diagram or worked example would clarify what the store interface's `SyncAck` method actually validates.

7. **Specify MCP transport** — "Streamable HTTP" explicitly. This affects which `mark3labs/mcp-go` constructor you use.

8. **Add a 10-line `main.go` sketch** — Shows how dependency injection works end-to-end.

None of these are fundamental design flaws. The document shows real Go experience. The over-engineering is mostly in the pre-commit heavy-handedness — a common trap for experienced engineers moving from team projects to solo work. Everything else is solid.
