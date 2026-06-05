# ALMS — Go Application Design & Quality Standards v2

## 1. Project Structure

```
alms/
├── cmd/
│   └── alms/
│       └── main.go                 # Entry point — thin: parse flags, init services, start server
├── internal/                       # Not importable by external packages
│   ├── config/
│   │   └── config.go               # YAML/env config loading (stdlib flag + gopkg.in/yaml.v3)
│   ├── models/                     # Pure data structs, no logic
│   │   ├── agent.go                # AgentSpec, AgentState
│   │   ├── learning.go             # LearningRecord
│   │   ├── protocol.go             # ProtocolRecord
│   │   └── errors.go               # Sentinel errors (ErrNotFound, ErrConflict)
│   ├── store/                      # Data access layer (pgx/v5 native pool)
│   │   ├── agent_store.go          # Agent CRUD + sync cursor
│   │   ├── learning_store.go       # Learning CRUD + dedup
│   │   ├── protocol_store.go       # Protocol CRUD
│   │   └── migrations/
│   │       └── 001_initial.sql
│   ├── server/                     # MCP server setup (Streamable HTTP)
│   │   ├── server.go               # mark3labs/mcp-go server init
│   │   ├── tools.go                # Tool handlers
│   │   ├── resources.go            # Resource handlers
│   │   └── middleware.go           # Auth middleware (X-ALMS-TOKEN)
│   ├── service/                    # Business logic layer (testable, no I/O)
│   │   ├── registry.go             # Agent registration + heartbeat
│   │   ├── sync.go                 # Learning sync + gap-safe ack validation
│   │   └── learning.go             # Store, search, dedup, scoring
│   └── validate/                   # Input validation
│       └── validate.go             # AgentID validation, field checks
├── scripts/
│   └── migrate.sh                  # DB migration runner (golang-migrate)
├── deploy/
│   ├── alms.service                # systemd unit
│   └── alms.yaml                   # Default config
├── docker-compose.yml              # PostgreSQL for dev
├── Makefile                        # Quality targets
├── .golangci.yml                   # Linter config
├── .pre-commit-config.yaml         # Pre-commit hooks
├── .editorconfig                   # Cross-editor consistency
├── .gitignore
├── AGENTS.md
├── README.md
├── go.mod                          # go 1.22
├── go.sum
└── tools.go                        # Pin tool deps (deadcode, govulncheck)
```

## 2. Architecture Principles

### 2.1 Layering
```
cmd/alms/main.go
    ↓
internal/server/    →  MCP transport + middleware (thin, no business logic)
    ↓
internal/service/   →  Business logic (testable via store interfaces)
    ↓
internal/store/     →  PostgreSQL via pgx/v5/pgxpool (the only I/O layer)
    ↓
internal/models/    →  Pure structs, shared everywhere
```

**Rules:**
- `store/` never calls `service/`. `service/` calls `store/`. `server/` calls `service/`.
- `models/` has zero imports beyond stdlib + time.
- No cyclic imports — enforced by `go vet`.
- No package imports outside its allowed direction.

### 2.2 Error Handling
- Sentinel errors defined in the package that returns them (e.g., `store.ErrNotFound`, `service.ErrConflict`).
- Errors wrap upstream with context: `fmt.Errorf("register agent %s: %w", id, err)`.
- No panic/recover except in `main.go` for the HTTP server.
- No `else` branches after error checks — early return is the Go idiom.
- Store layer returns sentinel errors. Service layer wraps them. Server layer logs and returns MCP error codes.

### 2.3 Context Propagation
- Every database call takes `context.Context` as first argument.
- Server sets timeout on incoming MCP requests (via `context.WithTimeout`).
- Store layer respects context cancellation for query timeouts.
- No stored contexts in struct fields — context flows through function calls only.
- No `context.Background()` in business logic — only in `main.go` and tests.
- Graceful shutdown via `signal.NotifyContext(ctx, SIGINT, SIGTERM)`.

### 2.4 Logging
- Use `log/slog` (Go 1.21+ stdlib). No third-party logger.
- Structured JSON output by default: `slog.Info("agent registered", "agent_id", id, "type", agentType)`.
- Error level for operational failures: `slog.Error("db query failed", "error", err)`.
- No `fmt.Println` anywhere in production code. Only in `--version` flag output.

### 2.5 Configuration
- YAML config file (`alms.yaml`) for static settings. Uses `gopkg.in/yaml.v3` — single-purpose, minimal deps.
- Environment variable overrides for secrets: `ALMS_PG_DSN`, `ALMS_AUTH_TOKEN`.
- No secrets in config files. Config file path in `.gitignore`.
- No `spf13/viper` — it adds 16+ indirect deps. `yaml.v3` + `os.Getenv` is sufficient.

### 2.6 Graceful Shutdown Pattern
```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    cfg := config.Load()
    pool, err := store.NewPool(ctx, cfg.PGDSN)
    if err != nil { slog.Error("db connect", "error", err); os.Exit(1) }
    defer pool.Close()

    srv := server.New(cfg, pool)
    slog.Info("starting ALMS", "addr", cfg.Server.Addr())

    go func() {
        if err := srv.ListenAndServe(); err != nil {
            slog.Error("server error", "error", err)
            os.Exit(1)
        }
    }()

    <-ctx.Done()
    slog.Info("shutting down...")
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(shutdownCtx)
}
```

### 2.7 MCP Transport
Use **Streamable HTTP** (`mark3labs/mcp-go` with SSE/HTTP transport). No stdio transport — ALMS is a long-running server on a headless machine, deployed via systemd.

### 2.8 Middleware
`internal/server/middleware.go` — Bearer token check via `X-ALMS-TOKEN` header. Returns MCP-format JSON-RPC error on auth failure (not plain HTTP 401).

## 3. Dependencies

### Core (Go stdlib + 5 packages)
| Package | Purpose | Justification |
|---------|---------|---------------|
| `net/http` | HTTP server for MCP | Stdlib, Go 1.22 enhanced routing — no chi/gorilla |
| `log/slog` | Structured logging | Stdlib since Go 1.21 |
| `context` | Request-scoped values | Stdlib |
| `github.com/jackc/pgx/v5` | PostgreSQL driver | Best Go PG driver. Use `pgxpool` natively — NOT via `database/sql` |
| `github.com/mark3labs/mcp-go` | MCP protocol | Most active Go MCP SDK. Streamable HTTP. Pin version. |
| `github.com/google/uuid` | UUID generation | Small dep, zero indirect deps. Alternative: 20 lines of stdlib `crypto/rand`. |
| `gopkg.in/yaml.v3` | YAML config parsing | Minimal dep. No viper (16+ indirect deps). |

### Tool-only deps (pinned via `tools.go`)
| Tool | Purpose |
|------|---------|
| `golang.org/x/tools/cmd/deadcode` | Dead code detection |
| `golang.org/x/vuln/cmd/govulncheck` | Vulnerability scanning |
| `github.com/golang-migrate/migrate/v4` | DB migration runner |

### Developer tooling (not in go.mod)
- `golangci-lint` — `brew install golangci-lint`
- `goimports` — `go install golang.org/x/tools/cmd/goimports@latest`

### `go.mod` preamble
```
module github.com/ghassan/alms

go 1.22                      // Minimum Go version — enables enhanced routing + slog
```

### Anti-patterns to avoid
- ❌ Mega-constructors with 80+ parameters — use functional options or a server config struct
- ❌ DB init spread across main.go — one `store.NewPool(ctx, dsn)` call
- ❌ Package-per-feature for trivial logic — one package per layer, not per tool
- ❌ `defer` hidden mid-function — always deferred at line after error check
- ❌ `_` imports in main.go for side effects — pgx is used natively, no driver registration needed
- ❌ Error checks without wrapping — always `fmt.Errorf("...: %w", err)`
- ❌ `context.Background()` in business logic — only in main.go + tests
- ❌ `init()` functions in non-config packages — use explicit constructors
- ❌ `fmt.Sprintf` for SQL queries — use `pgx` parameterised queries ($1, $2)
- ❌ Returning `*[]T` instead of `[]T` — slices are already reference types
- ❌ `any` / `interface{}` in function signatures — every parameter has a concrete type

## 4. Models Design

### Conventions
- All struct fields exported (capitalized). JSON tags on every field.
- `omitempty` on optional fields only.
- No methods on models except `Validate()` and `String()`.
- Constants grouped with typed const blocks, not separate files.
- Sentinel errors in the package that returns them, not in models.

### Type system
```go
type AgentType string
const (
    AgentTypeSystemd   AgentType = "systemd"
    AgentTypeMCPClient AgentType = "mcp_client"
)

type LearningType string
const (
    LearningTypePattern  LearningType = "pattern"
    LearningTypeFailure  LearningType = "failure"
    LearningTypeConfig   LearningType = "config"
    LearningTypeProtocol LearningType = "protocol"
    LearningTypeEdgeCase LearningType = "edge_case"
)

type Severity string
const (
    SeverityLow      Severity = "low"
    SeverityMedium   Severity = "medium"
    SeverityHigh     Severity = "high"
    SeverityCritical Severity = "critical"
)

type Resolution string
const (
    ResolutionOpen       Resolution = "open"
    ResolutionResolved   Resolution = "resolved"
    ResolutionSuperseded Resolution = "superseded"
)
```

### Validation
```go
func (a AgentSpec) Validate() error {
    if a.AgentID == "" {
        return fmt.Errorf("agent_id is required")
    }
    if len(a.AgentID) > 64 {
        return fmt.Errorf("agent_id too long: max 64 chars")
    }
    switch a.AgentType {
    case AgentTypeSystemd, AgentTypeMCPClient:
    default:
        return fmt.Errorf("invalid agent_type: %s", a.AgentType)
    }
    return nil
}
```

## 5. Quality Control

### 5.1 Makefile Targets
```makefile
.PHONY: all build vet lint test ci-check clean fmt tidy

all: build test lint

build:
    go build -ldflags="-X main.Version=$(git describe --tags 2>/dev/null || echo dev)" ./...

vet:
    go vet ./...

fmt:
    goimports -w -local github.com/ghassan/alms .

tidy:
    go mod tidy

lint:
    golangci-lint run ./... --timeout=3m

lint-ci:
    golangci-lint run ./... --timeout=3m --out-format=github-actions

test:
    go test -race -count=1 -shuffle=on -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | grep total

test-short:
    go test -short -race -count=1 -shuffle=on ./...

ci-check: tidy build vet lint-ci test-short
    @echo "✅ CI check passed"

vulncheck:
    govulncheck ./...

deadcode:
    deadcode ./...

clean:
    rm -f coverage.out
```

### 5.2 Static Analysis Tools
| Tool | Install | Purpose |
|------|---------|---------|
| `go vet` | Built-in | Nil pointers, unreachable code, bad build tags |
| `golangci-lint` | `brew install golangci-lint` | Meta-linter: 50+ linters |
| `govulncheck` | `go install golang.org/x/vuln/cmd/govulncheck@latest` | Dep vulnerability scanning |
| `deadcode` | `go install golang.org/x/tools/cmd/deadcode@latest` | Unused functions/types |
| `go mod tidy` | Built-in | Remove unused deps |

### 5.3 golangci-lint Config (`.golangci.yml`)
```yaml
linters:
  enable:
    - errcheck       # Check unchecked errors
    - gosimple       # Simplify code
    - govet          # Go's built-in vet (includes gofmt check)
    - ineffassign    # Detect unused assignments
    - staticcheck    # Go static analysis suite
    - unused         # Detect unused code (replaces deadcode CLI)
    - gosec          # Security checks
    - revive         # Style checker (replacement for golint)
    - misspell       # Spelling in comments
    - noctx          # Detect missing context.Context
    - thelper        # Ensure t.Helper() in test helpers
    - musttag        # Enforce JSON tags on struct fields used with json.Marshal/Unmarshal
    - wrapcheck      # Ensure errors from external packages are wrapped
    - errorlint      # Catch errors.Is/Fmt without %w
    - nakedret       # Prevent naked returns in long functions
  disable:
    - paralleltest   # Too noisy for small projects
    - prealloc       # Often wrong, negligible performance gain
    - unconvert      # Rarely catches bugs
    - wastedassign   # Rarely catches bugs
  settings:
    revive:
      rules:
        - name: exported
          severity: warning
        - name: package-comments
          severity: warning
    gosec:
      excludes:
        - G304  # Allow file path reads in CLI tools
    wrapcheck:
      ignoreSigs:
        - fmt.Errorf  # Already wraps
        - errors.New  # Creates sentinel
```

### 5.4 Pre-commit Hooks (`.pre-commit-config.yaml`)
```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v5.0.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-merge-conflict

  - repo: https://github.com/dnephin/pre-commit-golang
    rev: v0.5.1
    hooks:
      - id: go-fmt
      - id: go-vet
      - id: go-imports
        args: [-local, github.com/ghassan/alms]

  - repo: local
    hooks:
      - id: golangci-lint
        name: golangci-lint (fast)
        entry: golangci-lint run --fast --timeout=1m
        language: system
        types: [go]
        pass_filenames: false
```

**Justification:** 6 hooks for a solo project. Fast linters only (`--fast`), full lint runs in CI. No `deadcode`, `govulncheck`, or tests in pre-commit — they belong in `make ci-check` before push.

### 5.5 `.editorconfig`
```ini
root = true

[*.go]
indent_style = tab
indent_size = 8

[*.{yaml,yml,json,md}]
indent_style = space
indent_size = 2
```

## 6. Store Interfaces (Service-Layer Contracts)

These interfaces live in `internal/service/` and are implemented by `internal/store/`. Service tests mock them.

```go
// internal/service/agent.go
type AgentStore interface {
    Create(ctx context.Context, spec models.AgentSpec) error
    Get(ctx context.Context, agentID string) (models.AgentSpec, error)
    Update(ctx context.Context, spec models.AgentSpec) error
    Delete(ctx context.Context, agentID string) error
    Heartbeat(ctx context.Context, agentID string) (time.Time, error)
    List(ctx context.Context, filter map[string]string, limit, offset int) ([]models.AgentSpec, error)
}

// internal/service/sync.go
type LearningStore interface {
    Create(ctx context.Context, record models.LearningRecord) (string, error)
    Sync(ctx context.Context, agentID string, since time.Time, ltype string, tags []string) ([]models.LearningRecord, error)
    SyncAck(ctx context.Context, agentID string, learningIDs []string) error
    Search(ctx context.Context, query string, ltype string, tags []string, limit int) ([]models.LearningRecord, error)
    SoftDelete(ctx context.Context, learningID string) error
}

type ProtocolStore interface {
    Create(ctx context.Context, record models.ProtocolRecord) (string, error)
    Pull(ctx context.Context, agentID string, sinceID string) ([]models.ProtocolRecord, error)
    List(ctx context.Context) ([]models.ProtocolRecord, error)
    PullAll(ctx context.Context, agentTags []string) ([]models.ProtocolRecord, error)
}
```

## 7. Deduplication & Dead Code Removal

### Static Dead Code Detection
```bash
make deadcode          # Shows unused functions, types, methods
make lint              # unused linter in golangci-lint
```

### Learning Dedup Strategy (Phase 2)
- **Exact dedup**: SHA256 of `title + body` → skip on conflict
- **Near dedup**: Levenshtein ratio on title (threshold 0.85) → flag
- **Supersession**: Manual via `superseded_by` field on LearningRecord
- **GC**: Periodically delete learnings where `created_at + ttl_days < now()` and `resolution != "superseded"` and `score < 0.3`

### Import Cleanup
```bash
make tidy              # go mod tidy
make fmt               # goimports -local with proper sorting
```

## 8. Testing Strategy

### Structure
```
internal/store/
    agent_store_test.go
    learning_store_test.go
internal/service/
    registry_test.go
    sync_test.go           # Critical: gap-safe ack validation
    learning_test.go
```

### Requirements
- `make test` must pass before push (`-race -count=1 -shuffle=on`)
- Table-driven tests with `t.Run(name, func(t *testing.T) {})`
- `t.Helper()` in all test helper functions (enforced by `thelper` linter)
- Manual mocks for store interfaces (no mockgen) — write `storemock/` package locally
- Use `testify/assert` for assertions (one small dep worth keeping)
- Integration tests via `docker-compose up -d db` for local, `go test -tags=integration` in CI
- Fuzz tests on `Validate()` methods: `go test -fuzz=FuzzValidateAgent -fuzztime=10s`

### Critical test path (must pass before Phase 1 ships)
```go
func TestGapSafeSyncAck(t *testing.T) {
    // Register agent A
    // Push LRN-001, LRN-002, LRN-003
    // Agent syncs from 0 → gets LRN-001, LRN-002, LRN-003
    // Agent acks [LRN-001, LRN-002, LRN-003] → OK
    // Push LRN-004 (gap: LRN-004 id jumps to 005)
    // Agent syncs from last ack → gets LRN-004
    // Agent acks [LRN-004] → OK (no gap)
    // Agent acks [LRN-004] again → OK (idempotent)
    // Agent acks [LRN-006] without LRN-005 → REJECT (gap detected)
}
```

### Minimum coverage
- Service layer: 80%+
- Store layer: 60%+
- Server layer: 40%+

## 9. CI Pipeline

```yaml
# .github/workflows/ci.yml
name: ci
on: [push, pull_request]
jobs:
  ci:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env: { POSTGRES_DB: alms_test, POSTGRES_USER: alms, POSTGRES_PASSWORD: alms }
        options: >-
          --health-cmd pg_isready
          --health-interval 10s
          --health-timeout 5s
          --health-retries 5
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.22" }
      - run: go mod tidy && git diff --exit-code go.mod go.sum
      - run: make lint-ci
      - run: make deadcode
      - run: make ci-check
        env: { ALMS_PG_DSN: "postgres://alms:alms@localhost:5432/alms_test?sslmode=disable" }
```

## 10. Versioning & Release

```go
// cmd/alms/main.go
var (
    Version = "dev"
    Commit  = "none"
)

func main() {
    flagVersion := flag.Bool("version", false, "print version")
    flag.Parse()
    if *flagVersion {
        fmt.Printf("alms %s (%s)\n", Version, Commit)
        os.Exit(0)
    }
}
```

```makefile
build:
    go build -ldflags="-X main.Version=$(git describe --tags 2>/dev/null || echo dev) -X main.Commit=$(git rev-parse --short HEAD)" ./...
```

## 11. AGENTS.md

```markdown
# ALMS — Agent Learning Management System

## Project
MCP server for agent registry + cross-agent learning transfer.
Written in Go 1.22+. PostgreSQL for persistence. Single binary, systemd deploy.

## Naming
- Files: `snake_case.go` (e.g., `agent_store.go`)
- Types: PascalCase exported, camelCase unexported
- Interface names: `Methoder` suffix (e.g., `AgentStore`, not `IAgent`)
- Error vars: `ErrXxx` (e.g., `ErrNotFound`)
- Acronyms: all caps in Go (`HTTP`, `ID`, `URL`, `MCP`)

## Structure
- `cmd/alms/main.go` — entry point, flag parsing, DI, graceful shutdown
- `internal/server/` — MCP handlers + auth middleware
- `internal/service/` — business logic (testable via store interfaces)
- `internal/store/` — PostgreSQL access via pgx/v5/pgxpool
- `internal/models/` — data structs + typed constants + validation
- `internal/config/` — YAML + env config loading

## Quality
- `make lint` — golangci-lint (errcheck, gosimple, staticcheck, gosec, revive, thelper)
- `make test` — go test with race detector + shuffle + coverage
- `make ci-check` — tidy → build → vet → lint → test
- `make deadcode` — detect unused functions
- Pre-commit hooks: whitespace, EOF, YAML, gofmt, govet, goimports, golangci-lint-fast

## Rules
- No mega-constructors — use functional options or config structs
- No package-per-feature — one package per layer (service/, store/, server/)
- `store/` → `service/` → `server/` — never circular
- `context.Context` is first arg on all DB calls
- `log/slog` everywhere — no fmt.Println
- No `database/sql` — use pgx native pool
- No `context.Background()` in business logic — only in main.go
- No `init()` outside config package
- Table-driven tests with `t.Run()`
- `t.Helper()` in all test helpers

## Migrations
- Tool: `golang-migrate/migrate` CLI
- Files: `internal/store/migrations/NNNN_name.up.sql`
- Never modify a migration after merge — only add new ones
- Run: `migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up`

## First Run
### Prerequisites
- Go 1.22+ (`go version`)
- Docker + Docker Compose
- golangci-lint (`brew install golangci-lint`)
- golang-migrate CLI (`brew install golang-migrate`)

### Setup
```bash
# 1. Start PostgreSQL
docker compose up -d db

# 2. Create default config
cp deploy/alms.yaml ~/.alms/alms.yaml
# Edit ~/.alms/alms.yaml to set auth token if desired

# 3. Set env vars for secrets
export ALMS_PG_DSN=postgres://alms:alms@localhost:5432/alms_db?sslmode=disable
export ALMS_AUTH_TOKEN=change-me

# 4. Run migrations
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up

# 5. Start server
go run ./cmd/alms/

# 6. Verify
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | jq .
```

### Quick Test
```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"agent.register","arguments":{"agent_id":"test-agent-1","agent_type":"systemd"}}}' | jq .
```

### Tear Down
```bash
docker compose down
```

## Commit Style
- `feat:` new feature for user/agent
- `fix:` bug fix
- `chore:` dep bumps, tooling, CI
- `docs:` documentation only
- `refactor:` code change with no functional change
```
