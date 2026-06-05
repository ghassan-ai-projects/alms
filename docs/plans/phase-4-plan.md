# ALMS Phase 4 — Polish + CI + Integration Tests

## Plan

### Phase 4 Deliverables

1. **GitHub Actions CI** — `.github/workflows/ci.yml`
   - Trigger: push/PR to main
   - Go 1.22+ setup (using go.mod's `go 1.25.5`)
   - PostgreSQL service container
   - Steps: go mod tidy → build → vet → lint → test (short) → deadcode → vulncheck
   - Cross-compile linux binary at the end
   - Store tests skip automatically when PG DSN not set (existing `t.Skip` pattern)

2. **Integration Tests** — `internal/integration/alms_test.go`
   - Build tag: `//go:build integration`
   - Uses `os.Getenv("ALMS_PG_DSN")` — skip if not set
   - 4 test cases:
     - Full E2E: register agent → push 3 learnings → sync → ack → verify empty
     - Crash recovery: ack partial → resync → remaining returned
     - Concurrent sync: 10 agents syncing simultaneously
     - Protocol matching: create SOP with tags, verify agent gets it
   - Uses real PostgreSQL via pgxpool + real store layer
   - Spins up/down its own test data (prefix tables with `test_` or use separate schema)
   - Matches `*_test.go` requirement with build-tag skip

3. **Load Test Script** — `test/load-test.sh`
   - Starts ALMS binary in background
   - Connects to PG from docker-compose
   - Spawns 10 concurrent agents syncing
   - Measures latency p50/p95/p99, passes/fails
   - Exits 0 if p99 < 1000ms, 1 otherwise
   - Idempotent: cleanup on exit

4. **Phase 4 Acceptance Test** — `test/phase-4-acceptance.sh`
   - CI check passes (`make ci-check`)
   - Integration tests pass (if PG available)
   - Load test passes (if PG available)
   - Operations doc exists
   - Coverage: service >80%, store >60%, server >40%

### Implementation Order

1. `.github/workflows/ci.yml` — no Go files, no test needed
2. `internal/integration/alms_test.go` — needs `internal/integration/alms_test.go` (single file)
3. `test/load-test.sh` — standalone shell script
4. `test/phase-4-acceptance.sh` — standalone shell script

### Notes from Existing Code

- go.mod says `go 1.25.5` (not 1.22 — update CI accordingly)
- Store tests all use `t.Skip("PostgreSQL required")` — CI with PG service container will run them
- Service tests use mocks from `internal/service/storemock/` — no PG needed
- Server tests use service mocks — no PG needed
- `Makefile` already has `ci-check`, `deadcode`, `vulncheck` targets
- No `.github/` directory exists yet
- No `internal/integration/` directory exists yet
- No `test/` directory exists yet
- `docs/operations.md` already exists (Gate 4 checklist item ✅)

### Coverage Targets

- service: >80% ✅ (already high from Phase 1-2 service tests)
- store: >60% (needs PG — CI will run these)
- server: >40% ✅ (already covered by server tests)
