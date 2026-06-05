# ALMS Go Design — DeepSeek Review

**Reviewer:** DeepSeek (subagent)  
**Document:** `alms-go-design.md`  
**Status:** ✅ **APPROVED WITH RECOMMENDATIONS** — The design is sound. Fix the issues below before writing code.

---

## Executive Summary

This is a solid, opinionated Go design document. The author clearly understands Go idioms, has studied the A2A project's mistakes, and is making deliberate, minimal-dependency choices. I would accept this design and let the author write code against it, **provided** the recommendations below are addressed. None are blocking; several would prevent real pain 3–6 months in.

---

## 1. Correctness — Go Pattern Review

### What's right

| Pattern | Verdict | Why |
|---------|---------|-----|
| `internal/` tree | ✅ | Correct. External consumers can't import it. |
| `cmd/alms/main.go` entry point | ✅ | Thin main is canonical Go. |
| Layering direction (store → service → server) | ✅ | Textbook clean architecture. |
| Sentinel errors in `models/errors.go` | ✅ | Correct placement. They belong with the domain, not the store. |
| Error wrapping `fmt.Errorf("...: %w")` | ✅ | Go 1.20+ idiom, works with `errors.Is`/`errors.As`. |
| `context.Context` as first arg | ✅ | Go convention. |
| `log/slog` | ✅ | Stdlib since Go 1.21, structured JSON, zero dependencies. |
| No panics, no stored contexts in structs | ✅ | Correct. |
| Typed constants (`AgentType`, `LearningType`) | ✅ | Type safety > raw strings. |
| `Validate()` method on models | ✅ | Go idiom — `Validate() error` is a recognised pattern. |
| No methods on models beyond Validate/String | ✅ | Keeps models anemic, business logic in `service/`. |
| `go test -race -count=1` | ✅ | Catches races. `-count=1` disables caching. |

### What needs fixing

**1. `database/sql` listed as a dependency but the project uses `pgx/v5`.**
The doc says `database/sql` is a "stdlib" dependency and uses `github.com/jackc/pgx/v5` alongside it. **These are separate stacks.** `pgx` has its own connection pool (`pgxpool`) and does not go through `database/sql`. If you import both you get two connection pools, two health checks, and more complexity.

**Fix:** Remove `database/sql` from the dependency table. The doc correctly has no `sql.Open()` calls in the architecture. The table just needs cleanup.

**2. `models/errors.go` placement is slightly non-standard.**
Go convention places sentinel errors in the package that *returns* them, not in models. If `store/agent_store.go` returns `ErrNotFound`, it's more idiomatic (and easier to maintain) to define `ErrNotFound` in `store/` and have `service/` import it.

**Counter-argument the author might make:** "But then `server/` needs to import `store/` to check errors, which creates an import to a layer below." This is a valid concern. The compromise is to define sentinel errors in `models/` and define helper functions like `store.IsNotFound(err)` in the store package. But that's overengineering for 5 packages.

**Recommendation:** Define sentinel errors in the package that returns them. For a 5-package project, the cross-package import is fine. If the author strongly prefers `models/errors.go`, that's also acceptable — it's a style choice, not a correctness issue.

**3. `goimports` is referenced in pre-commit but there's no `-local` flag.**
Without `-local`, `goimports` sorts all imports together. The Go standard is:
```bash
goimports -w -local github.com/yourorg/alms .
```
This separates stdlib / third-party / internal imports into three groups.

**Fix:** Add `-local` to the pre-commit hook and the Makefile `fmt` target.

**4. No `go mod tidy` in the `fmt` or `ci-check` targets.**
The Makefile has `go mod tidy` only in the pre-commit hook, not in `make ci-check`. CI runs independently of pre-commit. If `go.mod` gets out of sync (e.g., a merge conflict resolved badly), `make ci-check` would pass and the broken module would only be caught on the next commit.

**Fix:** Add `go mod tidy` to the `ci-check` target or make it a separate `make tidy` target called by `ci-check`.

---

## 2. Completeness — What's Missing

### 2.1 Critical

| Missing | Why it matters | Fix |
|---------|----------------|-----|
| **Go version policy** | The AGENTS.md says "Go 1.26" but Go 1.26 doesn't exist yet (current is 1.24). Version pinning in `go.mod` is essential. | Pin `go 1.22` or `go 1.23` in `go.mod`. Use that version's `net/http` routing if Go 1.22+. |
| **Module path** | The doc doesn't specify the Go module path. | Use `github.com/yourorg/alms` or whatever the actual module path is. This affects `-local` in `goimports`. |
| **Zero-value initialisation pattern** | Not specified. Go initialises things to zero. Services that receive a nil config or nil store should panic early, not limp along. | Add a convention: "All constructor functions must panic on nil dependencies. No silent nil structs." |

### 2.2 Important

| Missing | Why it matters |
|---------|----------------|
| **Release/versioning** | No `VERSION` file, no goreleaser config, no git tag convention. Even a single-binary service benefits from `ldflags` to embed version+commit. |
| **Health check endpoint** | MCP servers benefit from a `/healthz` or similar for orchestrators. Not mentioned. |
| **Graceful shutdown** | The doc mentions HTTP server panic/recover in main.go but doesn't mention `signal.NotifyContext` for SIGINT/SIGTERM. |
| **Documentation generation** | `go doc` works out of the box, but no convention for package-level doc comments. Every package should have a `// Package server ...` doc comment. |
| **Dependency injection** | Not mentioned explicitly. For 5-package project, manual DI in `main.go` is fine. But the doc should *state* that out loud so nobody reaches for wire/dig. |
| **Migrations tool** | `scripts/migrate.sh` is mentioned but no detail. Will it use `golang-migrate/migrate`, `pressly/goose`, or raw `psql`? The decision should be documented. |

### 2.3 Nice-to-have

| Missing | Note |
|---------|------|
| `.gitattributes` | `*.go text diff=golang` improves diffs. Cheap to add. |
| `Dockerfile` | Multi-stage Go build with `distroless` or `scratch` base. |
| `CONTRIBUTING.md` | If this is open source. If internal, skip. |
| `CHANGELOG.md` or `RELEASES.md` | Keep-a-changelog format. |
| `.editorconfig` | Cross-editor consistency. |

---

## 3. Package Choices — Alternatives Analysis

### `mark3labs/mcp-go` ✅ Good choice

| Criterion | Verdict |
|-----------|---------|
| GitHub stars | ~800+ (most popular Go MCP SDK) |
| Activity | Active (March 2025+) |
| Streamable HTTP | Supported |
| Codegen-based? | Yes, but the API is clean enough to use directly |
| Alternative considered | `metoro-io/mcp-golang` — less mature, fewer tools support |

**Verdict:** Stick with it. But be aware: the SDK is changing rapidly. Pin to a specific version in `go.mod` and test upgrades explicitly.

### `pgx/v5` ✅ Best-in-class

| Criterion | Verdict |
|-----------|---------|
| Performance | Best Go PG driver, 2-5× faster than `lib/pq` |
| Pool | Built-in `pgxpool`, no extra deps |
| Prepared statements | Automatic |
| Type mapping | Handles `uuid`, `jsonb`, arrays natively |
| Alternative considered | `lib/pq` — deprecated, slower, no connection pool. `bun` — ORM, too heavy for this project. |

**Verdict:** Correct choice. Recommend using `pgxpool` directly (not `database/sql`) for the reasons in section 1.

### `google/uuid` ✅ Fine, but consider alternatives

| Criterion | Verdict |
|-----------|---------|
| Maturity | De facto standard, 10+ years |
| Performance | Good (no allocations in v7 path) |
| API | `uuid.New()` returns `uuid.UUID` (16 bytes) |
| Alternative 1 | `github.com/gofrs/uuid/v5` — faster, but no v7 support |
| Alternative 2 | Roll your own: `crypto/rand` + `fmt.Sprintf` — 5 lines. Fine for 1000 IDs/day |
| Alternative 3 | Let PostgreSQL generate UUIDs: `INSERT ... RETURNING id` — one less dependency |

**Verdict:** `google/uuid` is fine. But for this scale (~1000 agents, ~10K learnings), `uuid.New()` from `crypto/rand` is only 5 lines and removes the dependency entirely. Consider removing it unless you need UUIDv7 time-sorting.

### Missing from dependency table

| Should be there | Why |
|----------------|-----|
| `github.com/jackc/pgx/v5/pgxpool` | The actual pool import. `pgx/v5` alone is just the driver. |
| `github.com/golang-migrate/migrate/v4` or `github.com/pressly/goose` | If migrations are done from Go code rather than shell script. |
| `golang.org/x/tools/cmd/deadcode` | Listed as a quality tool but not in the dep table. Add to `tools.go`. |

---

## 4. Quality Tools — Linter Analysis

### 4.1 Enable these additional linters

| Linter | Why add it |
|--------|------------|
| **`thelper`** | Ensures `t.Helper()` is called in test helper functions. Catches confusing test failures. |
| **`musttag`** | Ensures struct fields used with `encoding/json` have JSON tags. Your models convention already requires this, but `musttag` enforces it. |
| **`errorlint`** | Catches `errors.Wrap` without `%w` (not relevant here since you wrap manually), but also catches `errors.Is(err, ErrFoo)` when you should use `errors.Is`. |
| **`nakedret`** | Prevents naked returns in functions longer than N lines. Naked returns are confusing. |
| **`nilnil`** | Catches functions that return `nil, nil` (both nil value and nil error). That's a design smell. |

### 4.2 Disable or reduce these

| Linter | Rationale |
|--------|-----------|
| `prealloc` | **Disable.** It's often wrong. It suggests pre-allocation even when the slice is conditional. For a 5-package project, the performance gain is negligible and the noise is real. |
| `noctx` | Keep enabled but use `exclude-functions` to whitelist `context.Background()` and `context.TODO()` calls. |
| `gosec` G304 exclusion | The exclusion is fine (file path reads in CLI tools). But add a note: "G304 is excluded only for CLI scripts, not for the server binary." |

### 4.3 Pre-commit hooks — concerns

| Hook | Problem |
|------|---------|
| `go-unit-tests` from `pre-commit-golang` | This runs **all** tests on every commit. For a project with DB integration tests, this will be slow (and fail if PG isn't running). Replace with a script that runs only unit tests (`-short` flag) or remove it from pre-commit and rely on `make ci-check` in CI. |
| `deadcode` as pre-commit hook | `deadcode` runs in ~1s on 5 packages, fine. But it's a duplicate — you already have `golangci-lint` with the `unused` linter enabled. They overlap 95%. Drop the deadcode hook and let `golangci-lint` catch unused code. |
| `golangci-lint` as pre-commit hook | The `--timeout=3m` is generous. On 5 packages it will finish in <5s. That's fine. But consider adding `--fast` so only fast linters run on commit; full lint runs in CI. |

**Recommended pre-commit set:**
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
        args: [-local, github.com/yourorg/alms]
      - id: validate-toml

  - repo: local
    hooks:
      - id: golangci-lint
        name: golangci-lint (fast)
        entry: golangci-lint run --fast --timeout=1m
        language: system
        types: [go]
        pass_filenames: false

      - id: go-mod-tidy
        name: go-mod-tidy
        entry: bash -c 'go mod tidy && git diff --exit-code go.mod go.sum'
        language: system
        files: '(go\.mod|go\.sum)$'
        pass_filenames: false

      - id: govulncheck
        name: govulncheck
        entry: bash -c 'govulncheck ./...'
        language: system
        types: [go]
        pass_filenames: false
```

**Removed:** `go-unit-tests` (too heavy for pre-commit), `deadcode` (overlaps with `unused` linter).

---

## 5. Test Strategy — Coverage Split Analysis

### 5.1 Coverage targets

| Layer | Target | Assessment |
|-------|--------|------------|
| Service | 80%+ | ✅ **Correct.** Business logic is where bugs live. 80% is achievable and meaningful. |
| Store | 60%+ | ✅ **Correct.** Store tests require a real DB (or testcontainers). 60% with a real DB is respectable. |
| Server | 40%+ | ✅ **Agree.** MCP handlers are hard to unit test. 40% forces some testing without being punitive. |

### 5.2 Gaps in the test strategy

**1. No contract/fixture strategy.**
The doc says "mock the store interface in service tests" but doesn't show the interface. For a 5-package project, define the interfaces in `internal/service/`:
```go
// internal/service/registry.go
type AgentStore interface {
    Get(ctx context.Context, id string) (*models.AgentSpec, error)
    Upsert(ctx context.Context, a *models.AgentSpec) error
    // ...
}
```

Then `store/agent_store.go` implements it. This lets service tests use `store/mock_agent_store.go` (hand-written or via `mockgen`).

**2. No table-driven tests convention.**
Table-driven tests are the canonical Go pattern. The doc should mention:
```go
func TestRegisterAgent(t *testing.T) {
    tests := []struct {
        name    string
        agent   models.AgentSpec
        wantErr bool
    }{
        {"valid systemd agent", models.AgentSpec{...}, false},
        {"empty agent_id", models.AgentSpec{...}, true},
        // ...
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // ...
        })
    }
}
```

**3. No mention of `testcontainers-go`.**
For integration tests, `testcontainers-go` (github.com/testcontainers/testcontainers-go) is the standard. Spin up PostgreSQL in a container, run migrations, run tests, tear down. The doc mentions "docker-compose up -d db" which is fine for local dev but not for CI. Document that CI should use testcontainers or a service container in GitHub Actions.

**4. No mention of `go test -shuffle=on`.**
Go 1.17+ supports `-shuffle=on`. This randomises test order and catches hidden test order dependencies. Add it to `ci-check`.

**5. Fuzz testing not mentioned.**
For validation logic (`Validate()` methods), fuzz testing is cheap and valuable:
```go
func FuzzValidateAgent(f *testing.F) {
    f.Fuzz(func(t *testing.T, id string, agentType string) {
        a := models.AgentSpec{AgentID: id, AgentType: models.AgentType(agentType)}
        _ = a.Validate() // No panic
    })
}
```
Add a `make fuzz` target.

---

## 6. Anti-Patterns Section — Additions

The doc lists 5 anti-patterns (mega-constructors, spread DB init, package-per-feature, hidden defer, `_` side-effect imports). These are all good A2A lessons. Here's what I'd add:

### ❌ Naked `context.Background()` in business logic
Services should receive context, not create it. `context.Background()` belongs in `main.go`. If a service function needs `context.Background()`, it's either a sign that context isn't being threaded through, or the function should accept a context parameter.

**Catch it:** `noctx` linter catches missing context propagation.

### ❌ `init()` functions in packages other than `config`
`init()` makes code untestable — you can't control when it runs. Use explicit constructor functions (`NewXxx`) called from `main.go`.

**Exception:** `init()` in `config/config.go` is fine if it sets default flags from env vars. But the doc should say: "No `init()` in any package except `config/`."

### ❌ Overusing `any` (interface{}) in function signatures
Go 1.18+ generics exist. If a function takes `any`, there should be a reason. For this project, no function signature should use `any` — every parameter has a concrete type.

**Catch it:** `golangci-lint` → enable the `forbidigo` linter with a pattern for `interface{}` or `any` in function parameters.

### ❌ Returning `*[]T` instead of `[]T`
A slice is already a reference type. Returning `*[]T` is unnecessary indirection. Every store `List` function should return `[]models.AgentSpec, error`.

### ❌ Not closing `rows` after `pgx` queries
```go
rows, _ := pool.Query(ctx, "SELECT ...")
defer rows.Close() // ← MUST be right after error check
```
The doc says "deferred at line after successful resource acquisition" which is correct. But add an explicit note: "All `rows` and `conn` acquired from pgxpool must have `defer rows.Close()` immediately after the error check."

### ❌ `else if` chains for error handling
```go
// BAD
if err != nil {
    return "", err
} else {
    // do work
}
// GOOD
if err != nil {
    return "", err
}
// do work
```
The Go idiom is early return, not `else`.

### ❌ Using `fmt.Sprintf` for SQL queries
SQL injection is still a thing. Use `pgx` parameterised queries:
```go
// BAD
sql := fmt.Sprintf("SELECT * FROM agents WHERE id = '%s'", id)
// GOOD
row := pool.QueryRow(ctx, "SELECT * FROM agents WHERE id = $1", id)
```

---

## 7. Specific Recommendations (Ordered by Priority)

### 🔴 Must fix before writing code

| # | Issue | Fix |
|---|-------|-----|
| 1 | Remove `database/sql` from dep table | pgxpool doesn't use it. Having both is confusing. |
| 2 | Pin Go version in `go.mod` | `go 1.22` or `go 1.23`. Not 1.26. |
| 3 | Add module path | `github.com/yourorg/alms` or whatever is real. |
| 4 | Document migration tool choice | pick `golang-migrate`, `pressly/goose`, or raw `psql`. |
| 5 | Document graceful shutdown pattern | `signal.NotifyContext` + `server.Shutdown()`. |
| 6 | Add `go mod tidy` to `make ci-check` | Or create `make tidy` and call it from `ci-check`. |

### 🟡 Should fix before first release

| # | Issue | Fix |
|---|-------|-----|
| 7 | Add `-local` flag to `goimports` | `goimports -w -local github.com/yourorg/alms .` |
| 8 | Pre-commit: remove duplicate `deadcode` | Redundant with `unused` linter in golangci-lint. |
| 9 | Pre-commit: remove `go-unit-tests` | Too heavy. Move to CI. |
| 10 | Pre-commit: use `--fast` in golangci-lint | Full lint in CI. |
| 11 | Add `go test -shuffle=on` to ci-check | Detects hidden order dependencies. |
| 12 | Document interface convention for mocking | `type AgentStore interface` in `service/`, not in `store/`. |
| 13 | Document table-driven test convention | Mention `[]struct{...}` pattern. |
| 14 | Disable `prealloc` linter | Wrong too often for the value it provides. |
| 15 | Add `musttag` linter | Enforces JSON tags on all struct fields. |
| 16 | Add `goreleaser` config or ldflags | `-X main.Version=$(git describe --tags)` |
| 17 | Document: no `init()`, no `any`, no `fmt.Sprintf` for SQL | Add to anti-patterns section. |

### 🟢 Nice-to-have (any time)

| # | Issue | Fix |
|---|-------|-----|
| 18 | Add `.editorconfig` | 5 lines, big DX improvement. |
| 19 | Add Dockerfile | Multi-stage build with `gcr.io/distroless/base`. |
| 20 | Add fuzz target to Makefile | `go test -fuzz=FuzzValidateAgent -fuzztime=10s ./internal/models/` |
| 21 | Consider removing `google/uuid` | `crypto/rand` + `fmt.Sprintf` is 5 lines. For this scale, it's enough. |
| 22 | Add `thelper` to linter config | Ensures `t.Helper()` in test helpers. |
| 23 | Add `nakedret` to linter config | Prevents confusing naked returns. |
| 24 | Document: services panic on nil deps | "All NewXxx functions panic on nil arguments." |

---

## 8. Final Verdict

**APPROVED WITH 6 MUST-FIX ITEMS.**

The design document is clean, opinionated, and shows a developer who has learned from past mistakes (the A2A anti-patterns in particular). The dependency choices are sound, the linter config is reasonable, and the test strategy has the right priorities.

The six must-fix items (Remove `database/sql`, pin Go version, add module path, pick migration tool, document graceful shutdown, add `go mod tidy` to CI) are all quick edits. The should-fix items (pre-commit cleanup, add `shuffle=on`, add `musttag`, remove `prealloc`) will prevent real annoyances during development.

The doc does not need a second review. The author can implement the fixes directly and start writing code.

---

### Appendix A: Quick Fix Patch for Versioning

For the `main.go` version embed:

```go
// cmd/alms/main.go
var (
    Version = "dev"
    Commit  = "none"
)
```

```makefile
# Makefile
build:
    go build -ldflags="-X main.Version=$(git describe --tags 2>/dev/null || echo dev) -X main.Commit=$(git rev-parse --short HEAD)" ./cmd/alms
```

```go
// main.go usage
func main() {
    flagVersion := flag.Bool("version", false, "print version")
    flag.Parse()
    if *flagVersion {
        fmt.Printf("alms %s (%s)\n", Version, Commit)
        os.Exit(0)
    }
}
```

### Appendix B: `.editorconfig` (5 lines)

```ini
root = true

[*.go]
indent_style = tab
indent_size = 8

[*.{yaml,yml,json,md}]
indent_style = space
indent_size = 2
```

### Appendix C: Graceful Shutdown Pattern

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    pool, err := store.NewPool(ctx, cfg.DSN)
    if err != nil {
        slog.Error("failed to connect to db", "error", err)
        os.Exit(1)
    }
    defer pool.Close()

    srv := server.New(pool) // starts MCP server in goroutine
    <-ctx.Done()
    slog.Info("shutting down...")

    shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    srv.Shutdown(shutdownCtx)
}
```
