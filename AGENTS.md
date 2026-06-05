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

## 🚨 Test Mandate (Hard Requirement)
Every code-writing task MUST produce `*_test.go` files alongside production code. This is not optional.

**Requirements:**
- Every package modified must have corresponding test files committed in the same push
- Table-driven tests with `t.Run()` for all new functions
- Target minimum coverage by layer: service 80%+, store 60%+, server 40%+, models 90%+
- `make test` must pass before push
- `go test -race -count=1 -coverprofile=coverage.out ./...` must show >0% coverage for all modified packages
- No commit is valid without tests — verify with `make ci-check` before push

**Rejection process:** If `make test` fails or coverage is 0%, the work is rejected and must be fixed before merging.

## Rules
- No mega-constructors — use functional options or config structs
- No package-per-feature — one package per layer (service/, store/, server/)
- `store/` → `service/` → `server/` — never circular
- `context.Context` is first arg on all DB calls
- `log/slog` everywhere — no fmt.Println
- No `database/sql` — use pgx native pool
- No `context.Background()` in business logic — only in main.go
- No `init()` outside config package
- Table-driven tests with `t.Run()` for every function
- Every file you create: write a `*_test.go` alongside it. No exceptions.
- `t.Helper()` in all test helper functions
- `make test` must pass before any commit is valid

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
