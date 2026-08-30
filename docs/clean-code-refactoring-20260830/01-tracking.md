# Tracking

Status values: `queued`, `reviewing`, `in_progress`, `reviewed`, `committed`, `bar_met`, `bar_gap`, `inspected`.

Hard constraint: no production `.go` file may exceed 250 lines after the task. A split is required for every queued file currently above that limit.

Tests remain deferred until every production file has reached a terminal status.

The final validation phase below supersedes the per-loop `tests deferred` notes.

| Order | File | Lines | Status | Commit | Notes |
|---:|---|---:|---|---|---|
| 1 | `internal/service/storemock/mock.go` | 694 | bar_met | `49ad345` | Split into six cohesive files; resulting max is 163 lines; tests deferred. |
| 2 | `internal/server/tools.go` | 554 | bar_met | `8c83674`, `0130693` | Split into domain and focused tool-registration files; restored package-private compatibility aliases; max resulting file 79 lines; tests deferred. |
| 3 | `internal/store/learning_store.go` | 439 | bar_met | `e353800` | Split into CRUD, sync, search, mutation, and row-scanning files; max changed file 142 lines; tests deferred. |
| 4 | `internal/service/okf.go` | 325 | bar_met | `eaec891` | Split into export, options, document, index, and helper files; max changed file 104 lines; tests deferred. |
| 5 | `internal/store/agent_store.go` | 275 | bar_met | `759d8a0` | Split into mutation, query, and encoding files; max changed file 176 lines; tests deferred. |
| 6 | `internal/service/learning.go` | 214 | bar_met | `e8b415a` | Split learning lifecycle, dedup, enrichment, and protocol responsibilities; resulting max 139 lines; tests deferred. |
| 7 | `internal/service/gc.go` | 201 | bar_met | `9f55d5d` | Split lifecycle, sweep policy, and decay responsibilities; max changed file 108 lines; tests deferred. |
| 8 | `internal/service/dedup.go` | 201 | bar_met | `f0ce74d` | Split exact, near, and supersession workflows; resulting max 128 lines; tests deferred. |
| 9 | `internal/store/protocol_store.go` | 180 | bar_met | `4d79cf8` | Consolidated query execution and isolated row scanning; final file 169 lines; tests deferred. |
| 10 | `internal/service/scoring.go` | 167 | bar_met | `cd04c6d` | Split score adjustments from decay workflows; resulting max 91 lines; tests deferred. |
| 11 | `internal/server/resources.go` | 164 | bar_met | `5f051c7` | Split five resource domains; max resulting file 42 lines; tests deferred. |
| 12 | `internal/models/learning.go` | 134 | inspected | — | Cohesive cross-layer model contract; no behavior-preserving refactor justified; tests deferred. |
| 13 | `internal/service/sync.go` | 122 | bar_met | `4d3df2f` | Split learning acknowledgement and protocol pulls; max resulting file 57 lines; tests deferred. |
| 14 | `internal/config/config.go` | 101 | bar_met | `4fc3105`, `1c4a004` | Split file loading and environment overrides; removed stale duplicate helper; max resulting file 66 lines; tests deferred. |
| 15 | `internal/service/registry.go` | 99 | inspected | — | Cohesive agent-lifecycle façade; no behavior-preserving refactor justified; tests deferred. |
| 16 | `cmd/alms/main.go` | 97 | bar_met | `4149328` | Extracted runtime dependency wiring; max resulting file 85 lines; tests deferred. |
| 17 | `internal/server/server.go` | 87 | bar_met | `55f4755` | Extracted HTTP handler/server assembly; max resulting file 64 lines; tests deferred. |
| 18 | `internal/models/agent.go` | 81 | inspected | — | Cohesive agent contract and validator; no behavior-preserving refactor justified; tests deferred. |
| 19 | `internal/service/interfaces.go` | 46 | inspected | — | Cohesive store boundaries; interface segregation recorded as follow-up; tests deferred. |
| 20 | `internal/server/middleware.go` | 39 | bar_met | `8b8d835` | Extracted unauthorized response writer; final file 43 lines; tests deferred. |
| 21 | `internal/store/postgres.go` | 37 | inspected | — | Cohesive pool setup/readiness function; no behavior-preserving refactor justified; tests deferred. |
| 22 | `internal/models/protocol.go` | 33 | bar_met | `429d7e0` | Simplified single-rule validation; file is 28 lines; tests deferred. |
| 23 | `internal/server/dashboard.go` | 22 | inspected | — | Cohesive static-page handler; no behavior-preserving refactor justified; tests deferred. |
| 24 | `internal/models/errors.go` | 18 | inspected | — | Cohesive sentinel definitions; no behavior-preserving refactor justified; tests deferred. |
| 25 | `tools.go` | 8 | bar_met | `2a8cd08` | Retained tool dependency anchor and removed stale unused import caught by final build; file is 5 lines; tests deferred. |

Files with no justified refactor will be marked `inspected` with evidence rather than changed for change's sake.

## Final validation

- `gofmt -l` over all non-test Go files: passed with no output.
- `make build`: passed. The sandbox emitted a module stat-cache permission warning; it did not affect the successful build.
- `make vet`: passed after restoring the package-private registration aliases used by same-package tests.
- `make lint`: passed with `0 issues`; the sandbox emitted golangci-lint cache-write warnings.
- `make test`: passed with race detection, shuffled execution, and coverage. Final package coverage: `cmd/alms` 24.2%, `internal/config` 92.6%, `internal/models` 100.0%, `internal/server` 62.0%, `internal/service` 79.0%, `internal/service/storemock` 20.1%, `internal/store` 8.4%; total 46.7%.
- Scope check: 80 non-test Go files are documented in `05-resulting-file-inventory.md`; no file exceeds 250 lines; no existing test file was refactored. Two focused coverage test files were added for newly extracted runtime/storemock wiring.
- Residual coverage gap: `internal/store` remains below the repository’s 60% layer target; improving that would require broader test work outside this refactor’s requested scope.
