# ALMS Quality Standards

Makefile targets, linting configuration, and testing strategy.

---

## Makefile

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

test:
    go test -race -count=1 -shuffle=on -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out | grep total

ci-check: tidy build vet lint-ci test-short
    @echo "✅ CI check passed"

vulncheck:
    govulncheck ./...

deadcode:
    deadcode ./...

clean:
    rm -f coverage.out
```

---

## Linting (golangci-lint)

14 linters enabled in `.golangci.yml`:

| Linter | Purpose |
|--------|---------|
| `errcheck` | Unchecked errors |
| `gosimple` | Simplify code |
| `govet` | Go's built-in vet (includes gofmt) |
| `ineffassign` | Unused assignments |
| `staticcheck` | Go static analysis suite |
| `unused` | Dead code detection |
| `gosec` | Security checks |
| `revive` | Style (replaces golint) |
| `misspell` | Spelling in comments |
| `noctx` | Missing `context.Context` |
| `thelper` | Missing `t.Helper()` in test helpers |
| `musttag` | Enforce JSON tags on marshalled structs |
| `wrapcheck` | Errors from external packages must be wrapped |
| `errorlint` | Catch `errors.Is`/`Fmt` without `%w` |
| `nakedret` | Prevent naked returns in long functions |

Pre-commit hooks run `golangci-lint --fast` only. Full lint runs in CI.

---

## Testing Strategy

### Structure

```
internal/store/
    agent_store_test.go       — Tests against real Postgres (docker-compose)
    learning_store_test.go
    protocol_store_test.go

internal/service/
    registry_test.go          — Tests via storemock (no DB)
    sync_test.go              — Gap-safe ack validation (critical!)
    learning_test.go          — Dedup, scoring, supersession
    gc_test.go                — TTL expiry

internal/server/
    server_test.go            — MCP protocol test via Streamable HTTP
    tools_test.go             — Tool handler integration
    resources_test.go         — Resource handler integration
    middleware_test.go        — Auth header validation
    dashboard_test.go         — Dashboard endpoint

internal/config/
    config_test.go            — YAML loading + env override

internal/models/
    agent_test.go             — Validate() table-driven tests
    learning_test.go
    protocol_test.go
```

### Requirements

- `make test` must pass before push (`-race -count=1 -shuffle=on`)
- Table-driven tests with `t.Run(name, func(t *testing.T) {})`
- `t.Helper()` in all test helpers (enforced by `thelper` linter)
- Manual mocks in `internal/service/storemock/` (no mockgen)
- `testify/assert` for assertions (one small dep)
- Integration tests via `docker-compose up -d db`

### Coverage Targets

| Layer | Target |
|-------|--------|
| Service | 80%+ |
| Store | 60%+ |
| Server | 40%+ |

### Critical Test: Gap-Safe Sync Ack

```go
func TestGapSafeSyncAck(t *testing.T) {
    // Register agent A
    // Push LRN-001, LRN-002, LRN-003
    // Agent acks [LRN-001, LRN-002, LRN-003] → OK
    // Push LRN-004
    // Agent acks [LRN-004] → OK (no gap)
    // Agent acks [LRN-004] again → OK (idempotent)
    // Agent acks [LRN-006] without LRN-005 → REJECT (ErrGapDetected)
}
```

---

## CI Pipeline

```yaml
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

---

## Pre-commit Hooks (`.pre-commit-config.yaml`)

6 hooks for a solo project:

| Hook | When it runs | What it checks |
|------|-------------|----------------|
| `trailing-whitespace` | Every file | No trailing spaces |
| `end-of-file-fixer` | Every file | Newline at EOF |
| `check-yaml` | .yaml/.yml | Valid YAML |
| `check-merge-conflict` | Every file | No unresolved merge |
| `go-fmt` | .go | `gofmt` compliance |
| `go-vet` | .go | `go vet` |
| `go-imports` | .go | Import ordering |
| `golangci-lint --fast` | .go | Fast lint pass |

Heavy checks (full lint, tests, vulnerability scan) run in CI, not pre-commit.

---

## Style Rules

- **Naming:** `snake_case.go` files, PascalCase exported types, camelCase unexported
- **Interface names:** `Methoder` suffix (`AgentStore`, not `IAgent`)
- **Error vars:** `ErrXxx` (`ErrNotFound`, `ErrConflict`)
- **Acronyms:** All caps in Go (`HTTP`, `ID`, `URL`, `MCP`)
- **No `else` after error check** — early return is the Go idiom

### Anti-Patterns

- ❌ Mega-constructors with 80+ parameters — use functional options or config structs
- ❌ Package-per-feature — one package per layer, not per tool
- ❌ `defer` hidden mid-function — always deferred at line after error check
- ❌ `context.Background()` in business logic — only in `main.go` and tests
- ❌ `init()` in non-config packages — use explicit constructors
- ❌ `fmt.Sprintf` for SQL — use `pgx` parameterised queries (`$1`, `$2`)
- ❌ Returning `*[]T` — slices are already reference types
