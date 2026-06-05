# ALMS Go Design v2 — Final Review (DeepSeek)

**Reviewer:** DeepSeek (subagent)
**Date:** 2026-06-05
**Document:** `alms-go-design.md` (v2)

---

## Checklist Verification

| # | Item | Status |
|---|------|--------|
| 1 | Remove `database/sql` from dep table | ✅ Done — pgx native pool documented, `database/sql` explicitly warned against |
| 2 | Pin Go version (1.22) | ✅ `go 1.22` in `go.mod` preamble |
| 3 | Set module path (`github.com/ghassan/alms`) | ✅ `module github.com/ghassan/alms` |
| 4 | Document migration tool (golang-migrate) | ✅ Tool dep in `tools.go`, `scripts/migrate.sh`, migration docs in AGENTS.md |
| 5 | Graceful shutdown pattern (`signal.NotifyContext`) | ✅ Full code example in §2.6 |
| 6 | `go mod tidy` in CI (`make tidy` in ci-check) | ✅ `ci-check: tidy build vet lint-ci test-short` |
| 7 | `-local` flag on goimports | ✅ Makefile and pre-commit hooks both use `-local github.com/ghassan/alms` |
| 8 | Pre-commit: remove duplicate deadcode | ✅ Section 5.4 has no deadcode hook |
| 9 | Pre-commit: remove go-unit-tests | ✅ Section 5.4 has no test hooks |
| 10 | Pre-commit: `--fast` for golangci-lint | ✅ `golangci-lint run --fast --timeout=1m` |
| 11 | `go test -shuffle=on` in CI | ✅ Present in both `test` and `test-short` targets |
| 12 | Document 3 store interfaces | ✅ §6: AgentStore, LearningStore, ProtocolStore |
| 13 | Table-driven test convention | ✅ Mentioned in §8 with example |
| 14 | Disable prealloc linter | ✅ In `.golangci.yml` `disable` list |
| 15 | Add musttag, thelper, wrapcheck, errorlint, nakedret | ✅ All five in `.golangci.yml` `enable` list |
| 16 | ldflags version embed | ✅ §10 with full code + Makefile build target |
| 17 | No `init()`, no `any`, no `fmt.Sprintf` for SQL | ✅ §3 Anti-patterns lists all three |
| 18 | Add `.editorconfig` | ✅ §5.5 with complete config |

**All 18 items verified as present.** ✓

---

## Detailed Review

### Score: **8.5 / 10**

### Recommendation: **ACCEPT** with minor suggestions (see below)

---

## What's Done Well

### Architecture
- Clean 4-layer dependency direction: `server → service → store → models`
- No cyclic imports enforced by `go vet`
- Store interfaces in `service/` package for clean testability — excellent pattern
- Graceful shutdown pattern is idiomatic and complete

### Dependency Management
- Tight dependency footprint: stdlib + only 4 runtime packages (pgx/v5, mcp-go, uuid, yaml.v3)
- Tool deps properly pinned via `tools.go` convention
- Clear anti-pattern list in §3 — especially important for team onboarding

### Quality Tooling
- Linter config is well-curated: 14 enabled, 4 disabled with reasons
- Pre-commit hooks are lean (6 hooks, fast-only) — appropriate for solo project
- CI pipeline includes `go mod tidy` + `git diff --exit-code` check to prevent dependency drift
- `govulncheck` and `deadcode` in separate targets (not in pre-commit) — good judgment

### Testing
- Table-driven tests required + `t.Helper()` enforced via linter
- Manual mocks (no mockgen) — keeps deps minimal and code transparent
- Critical integration test path (`TestGapSafeSyncAck`) documented with concrete steps
- Fuzz testing on `Validate()` is a nice touch
- Coverage expectations are reasonable per layer

### Documentation
- AGENTS.md is comprehensive: naming conventions, structure, quality commands, setup, commit style
- First-run section with curl examples is practical
- Anti-pattern list in §3 and AGENTS.md are consistent

---

## Minor Suggestions (not blockers)

These are observations for further polish, not required changes.

### 1. `deadcode` runs twice in CI
The CI pipeline runs both `make deadcode` (standalone CLI) and `make lint` (which includes `unused` linter in golangci-lint, covering the same purpose). Consider:
- Dropping the standalone `make deadcode` from CI (the `unused` linter in golangci-lint is sufficient)
- Or adding a comment clarifying they catch different things (deadcode finds unused exported vs unused finds unused types/functions)

**Mitigation:** This is harmless — duplicate detection is better than none. No action required.

### 2. `make test` vs `make test-short` overlap in CI
`make ci-check` uses `test-short` while `deadcode` runs standalone. If `test-short` skips integration tests, consider renaming `test-short` to `test-unit` for clarity, or adding a comment explaining the split.

**Mitigation:** The names are self-explanatory. No action required.

### 3. Missing test for middleware.go
The testing strategy covers store, service, and server layers but doesn't explicitly call out middleware tests. Auth middleware is a security boundary — worth documenting a test for it.

**Suggestion (optional):** Add a line to §8 Testing Strategy: `Server middleware: 90% coverage on auth token validation + rejection paths`.

### 4. Docker Compose exposes PostgreSQL port
The `docker-compose.yml` is mentioned for dev but no port mapping is shown. By default, Docker Compose only exposes ports to linked services unless `ports:` is declared. For local `psql` access, this is fine, but if the intent is to run tests from the host, ensure the compose file maps port 5432.

**Suggestion (optional):** Add a comment in the test setup section: "Ensure `docker-compose.yml` maps host port 5432 for local test runs."

### 5. No explicit branch strategy or PR template
The CI supports `on: [push, pull_request]` but there's no branch protection rules or PR checklist. For a solo project this is fine, but if collaboration is expected, consider a `pull_request_template.md`.

**Suggestion (optional):** Add a section or a `PULL_REQUEST_TEMPLATE.md` referencing the quality gates.

---

## Conclusion

**Score: 8.5/10** — All 18 checklist items from the previous review are correctly implemented. The design is production-ready with clean architecture, appropriate tooling, and thorough documentation.

**Decision: ACCEPT**

The v2 design reflects careful attention to every review item. The minor suggestions above are optional improvements that can be addressed incrementally. The project is well-positioned for Phase 1 implementation.
