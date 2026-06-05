# ALMS — Phase 5: Deep Quality Audit

## Objective
Exhaustively review what was built across all 4 phases. Find dead code, complex logic, missing tests, architectural drift, and anything that doesn't match the approved plans.

## Audit Checklist

### 1. Architecture vs Reality
- [ ] Does the actual module tree match `docs/plans/impl-01.md` Project Structure?
- [ ] Are there any packages or files that exist but aren't in any plan?
- [ ] Are there planned packages that were never created? (e.g., `internal/middleware/`, `internal/validate/`)
- [ ] Does layering still hold? (server → service → store → models, never circular)
- [ ] Run `go vet ./...` — verify no circular imports

### 2. Dead Code Detection
- [ ] Run `deadcode ./...` — list unused functions, types, methods
- [ ] Run `golangci-lint run ./...` with `unused` linter — compare results
- [ ] Manually check `internal/service/storemock/mock.go` — all mock methods actually used?
- [ ] Check for `_` imports in main.go — pgx driver registration pattern
- [ ] Check for any `tools.go` deps that aren't actually used

### 3. Complex Logic Audit
- [ ] **Gap-safe ack** in `internal/service/sync.go` — is the algorithm correct? Check edge cases: empty batch, duplicate IDs, out-of-order IDs, already-acked IDs
- [ ] **Dedup engine** in `internal/service/dedup.go` — SHA256 exact dedup correct? Levenshtein threshold 0.85 appropriate?
- [ ] **GC scheduler** in `internal/service/gc.go` — does it actually run? Is the goroutine started from main.go?
- [ ] **Scoring** in `internal/service/scoring.go` — does `BatchApplyDecay` handle empty tables?
- [ ] **Resource templates** in `internal/server/resources.go` — template regex matching works? Are `alms://learnings/{id}` templates correctly registered?
- [ ] **Auth middleware** in `internal/server/middleware.go` — dev mode passthrough correct? MCP error format matches spec?
- [ ] **Sync flow E2E** — walk through the full flow: register → push → sync → ack → second sync empty

### 4. Missing Features vs Architecture Plan
Compare `alms-architecture-plan.md` (v3.2) against actual code:

| Feature | Plan says | Actual | Status |
|---------|-----------|--------|--------|
| `agent.update` tool | MCP tool | ✅ | Check exists |
| `learning.search` full-text | GIN vector index | ✅ | Check exists |
| `learning.sync` timestamp cursor | created_at based | ✅ | Check |
| `learning.sync_ack` gap-safe | Array of IDs, reject gaps | ✅ | Check |
| `protocol.pull` by agent tags | Match target_tags | ✅ | Check |
| `protocol.pull_since` | Change cursor | ✅ | Check |
| `health.check` | PG ping + agent count | ✅ | Check |
| Auth: X-ALMS-TOKEN header | Bearer check | ✅ | Check |
| Soft-delete on learnings | is_deleted flag | ✅ | Check |
| `learning_acknowledgements` table | Join table | ✅ | Check |
| All 15 MCP tools (P1+P2) | Listed in plan | ⚠️ | Verify count |
| All 7 MCP resources | Listed in plan | ⚠️ | Phase 2 dropped some templates |
| `internal/validate/` package | In plan module tree | ❌ | Was planned but never created |
| Web dashboard at /dashboard | HTML page | ✅ | Check |

### 5. Test Quality
- [ ] Run `go test -race -count=1 -coverprofile=coverage.out ./...` — record coverage by package
- [ ] Targets vs actual: models 100%✅, config 81%✅, service 75%+✅, server 40%+✅, store 60%? (skips without PG)
- [ ] Are there `t.Helper()` calls in all test helpers? (thelper linter checks this)
- [ ] Are there fuzz tests for Validate() functions?
- [ ] Do store tests actually work with PostgreSQL? (manual check)
- [ ] Is there a test that proves the gap-safe ack works end-to-end?

### 6. Documentation Audit
- [ ] `AGENTS.md` — accurate? Up to date with current structure?
- [ ] `docs/operations.md` — covers: deploy, backup, restore, migrate, log inspect, disk monitor?
- [ ] `docs/integration-guide.md` — covers: OpenClaw setup, Qwen CLI setup, newsletter agent example?
- [ ] `docs/plans/impl-01.md` — progress log up to date? Phase 4 marked done?
- [ ] `README.md` — basic project description? Quick start?
- [ ] `LICENSE` — MIT? Present?

### 7. Security Review
- [ ] Auth token: is it actually checked? Is there a fallback path without it?
- [ ] PG DSN: exposed in logs? In MCP responses?
- [ ] SQL injection: all queries use `$1` parameterised style? Check `fmt.Sprintf` usage in store/
- [ ] gosec linter: run with `golangci-lint run --enable-all ./...` to check G-series warnings

### 8. Deliverables Status

| Deliverable | Phase | Code | Test | Doc | Deployed | Notes |
|------------|-------|------|------|-----|----------|-------|
| Agent registry | P1 | ✅ | ✅ | ✅ | ⏳ | Needs data machine |
| Learning sync | P1 | ✅ | ✅ | ✅ | ⏳ | Core flow |
| Gap-safe ack | P1 | ✅ | ✅ | ✅ | ⏳ | Most critical logic |
| Protocol pull | P1 | ✅ | ✅ | ✅ | ⏳ | |
| learning.store | P2 | ✅ | ✅ | ✅ | ⏳ | |
| learning.search | P2 | ✅ | ✅ | ✅ | ⏳ | Full-text |
| Dedup engine | P2 | ✅ | ✅ | ✅ | ⏳ | SHA256 + Levenshtein |
| Scoring engine | P2 | ✅ | ✅ | ✅ | ⏳ | |
| GC scheduler | P2 | ✅ | ✅ | ✅ | ⏳ | async goroutine |
| Dashboard | P2 | ✅ | ✅ | ✅ | ⏳ | |
| systemd deploy | P3 | ✅ | — | ✅ | ⏳ | Needs sudo |
| Backup cron | P3 | ✅ | — | ✅ | ⏳ | |
| Newsletter agent | P3 | ✅ | — | ✅ | ⏳ | Example script |
| CI pipeline | P4 | ⏳ | — | — | — | Phase 4 running |
| Integration tests | P4 | ⏳ | ⏳ | — | — | Phase 4 running |
| Load testing | P4 | ⏳ | — | — | — | Phase 4 running |
| Operations docs | P3 | — | — | ✅ | — | |
| Integration guide | P3 | — | — | ✅ | — | |

### 9. Final Recommendations
After all checks pass, produce a summary:
- Top 3 quality issues found (if any)
- Recommended fixes with priority
- Things that are fine as-is
- Whether the project is ready for deployment to data machine
