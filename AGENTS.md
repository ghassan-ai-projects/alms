# AGENTS.md - ALMS

This is the canonical operating guide for coding agents working in this repository. Keep it specific, enforceable, and aligned with the actual repo automation.

## Project Identity

ALMS is a self-hosted MCP server for shared agent memory. It provides an agent registry, a cross-agent learning store, and protocol distribution for multi-agent systems without becoming an orchestration runtime.

The codebase is a Go 1.22+ single-binary service with PostgreSQL persistence, Streamable HTTP MCP transport, and systemd-oriented deployment assets. Agents should preserve that shape: durable business logic in `internal/service`, pgx-backed persistence in `internal/store`, and thin transport handling in `internal/server`.

## Canonical Instruction Strategy

- `AGENTS.md` is the cross-agent source of truth.
- Codex reads `AGENTS.md` natively. Do not add `CODEX.md`.
- `CLAUDE.md`, `GEMINI.md`, and `.github/copilot-instructions.md` should stay as thin bridge files that point back here.
- Keep durable repo rules here; keep personal preferences out of committed files.
- Use path-scoped or workflow-specific agent files only when a narrow rule should not load in every session.

## Before Editing

1. Read this file and [README.md](README.md).
2. Check the worktree with `git status --short`; never overwrite user changes.
3. Read the relevant package before editing and keep existing package boundaries intact.
4. Run the narrowest useful validation before changes when feasible. Prefer `make ci-check` for substantial work.
5. Follow the Test Mandate for every production-code change.

## Default Workflow

1. Restate the goal, constraints, affected files, and risks.
2. Pick the smallest cohesive change that solves the task.
3. Add or update tests first for production behavior changes.
4. Implement without broad refactors unless explicitly requested.
5. Validate with the relevant repo commands.
6. Handoff with changed files, checks run, skipped checks, and residual risk.

Pause for human review before broad architectural changes, destructive actions, new dependencies, security-sensitive edits, or ambiguous behavior changes.

## Definition Of Done

- The requested scope is complete without unrelated refactors.
- Production-code changes include meaningful tests, and modified packages do not have 0% coverage.
- Formatting, vet, lint, and test commands relevant to the change pass.
- Documentation is updated when behavior, setup, commands, or agent workflow changes.
- No secrets or environment-specific private data are added.
- Final handoff names the checks run, skipped checks with reasons, and any follow-up risk.

## Naming

- Files: `snake_case.go`
- Test files: `<file>_test.go`
- Types: `PascalCase` exported, `camelCase` unexported
- Interfaces: descriptive noun or `-er` suffix; do not use `I*`
- Error vars: `ErrXxx`
- Error types: `XxxError`
- Packages: short, lowercase, singular, no underscores
- Acronyms: all caps in Go: `HTTP`, `ID`, `URL`, `JSON`, `API`, `MCP`, `SQL`, `YAML`, `TLS`, `DNS`, `URI`, `UUID`

## Structure

- `cmd/alms/main.go` - entry point, flag parsing, DI, graceful shutdown
- `internal/config/` - YAML + env config loading
- `internal/models/` - data structs, typed constants, validation
- `internal/service/` - business logic, testable through store interfaces
- `internal/store/` - PostgreSQL access via `pgx/v5/pgxpool`
- `internal/server/` - MCP handlers and auth middleware
- `internal/store/migrations/` - SQL migrations for persistent schema changes
- `deploy/` - deployment assets and example config
- `documentation/` - user and operator documentation
- `prompts/` and `skill/` - agent integration assets

Layer rule:

- `cmd/` -> `server/` -> `service/` -> `store/` -> `models/`
- Imports flow downward only. No circular dependencies.
- `service/` depends on store interfaces, not concrete store types.
- `models/` should not depend on other internal layers.
- `context.Context` is the first argument on DB and request-scoped calls.

## Quality Gates

Use the commands that actually exist in this repo:

- `make fmt` - `goimports` formatting
- `make vet` - `go vet ./...`
- `make lint` - `golangci-lint run ./... --timeout=3m`
- `make test` - race + shuffle + coverage profile
- `make test-short` - short test suite for faster validation
- `make ci-check` - `tidy -> build -> vet -> lint-ci -> test-short`
- `make deadcode` - detect unused functions
- `make vulncheck` - vulnerability scan
- `make deploy-linux` - cross-compile Linux binary

If a change is substantial, run `make ci-check`. For targeted or docs-only work, run the narrowest checks that prove the change is correct and call out anything skipped.

## Test Mandate

Every production-code change must include `*_test.go` files in the same package set. No exceptions.

Requirements:

- Every modified production package must have tests committed in the same push.
- Use table-driven tests with `t.Run()` for new functions and meaningful branches.
- Use `t.Helper()` in helper functions.
- Do not use `context.Background()` in business logic; in tests prefer cancellable contexts.
- `make test` must pass before a production-code change is considered complete.
- `go test -race -count=1 -coverprofile=coverage.out ./...` must show non-zero coverage for modified packages.

Coverage targets by layer:

- `internal/models/` - 90%+
- `internal/service/` - 80%+
- `internal/store/` - 60%+
- `internal/server/` - 40%+
- `cmd/` - best effort

If `make test` fails or a modified package has 0% coverage, the work is rejected until fixed.

## Repository Rules

- Use `log/slog`; do not introduce `fmt.Println`.
- Use pgx native pool APIs, not `database/sql`.
- Do not use `init()` outside config package needs that already exist.
- Do not modify an existing migration after merge; add a new migration instead.
- Keep migrations under `internal/store/migrations/NNNN_name.up.sql`.
- Do not add mega-constructors; prefer config structs or functional options.
- Keep generated files clearly marked and document how they are regenerated.

## First Run

Prerequisites:

- Go 1.22+ (`go version`)
- Docker + Docker Compose
- `golangci-lint`
- `golang-migrate`

Setup:

```bash
docker compose up -d db
cp deploy/alms.yaml ~/.alms/alms.yaml
export ALMS_PG_DSN=postgres://alms:alms@localhost:5432/alms_db?sslmode=disable
export ALMS_AUTH_TOKEN=change-me
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
go run ./cmd/alms/
```

Verify:

```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | jq .
```

Quick test:

```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"agent.register","arguments":{"agent_id":"test-agent-1","agent_type":"systemd"}}}' | jq .
```

Tear down:

```bash
docker compose down
```

## Commit Style

- `feat:` new feature for user or agent
- `fix:` bug fix
- `chore:` dependency bumps, tooling, CI
- `docs:` documentation only
- `refactor:` code change with no functional change

## Validation And Handoff

When handing off work:

- State what changed and why.
- List checks run.
- List skipped checks and why they were skipped.
- Call out config changes, migrations, security impact, and residual risk.
