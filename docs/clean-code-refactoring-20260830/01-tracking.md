# Tracking

Status values: `queued`, `reviewing`, `in_progress`, `reviewed`, `committed`, `bar_met`, `bar_gap`, `inspected`.

Hard constraint: no production `.go` file may exceed 250 lines after the task. A split is required for every queued file currently above that limit.

Tests remain deferred until every production file has reached a terminal status.

| Order | File | Lines | Status | Commit | Notes |
|---:|---|---:|---|---|---|
| 1 | `internal/service/storemock/mock.go` | 694 | bar_met | `49ad345` | Split into six cohesive files; resulting max is 163 lines; tests deferred. |
| 2 | `internal/server/tools.go` | 554 | bar_met | `8c83674` | Split into domain and focused tool-registration files; max changed file 79 lines; tests deferred. |
| 3 | `internal/store/learning_store.go` | 439 | bar_met | `e353800` | Split into CRUD, sync, search, mutation, and row-scanning files; max changed file 142 lines; tests deferred. |
| 4 | `internal/service/okf.go` | 325 | reviewed | — | Split into export, options, document, index, and helper files; tests deferred. |
| 5 | `internal/store/agent_store.go` | 275 | queued | — | Split required. |
| 6 | `internal/service/learning.go` | 214 | queued | — | — |
| 7 | `internal/service/gc.go` | 201 | queued | — | — |
| 8 | `internal/service/dedup.go` | 201 | queued | — | — |
| 9 | `internal/store/protocol_store.go` | 180 | queued | — | — |
| 10 | `internal/service/scoring.go` | 167 | queued | — | — |
| 11 | `internal/server/resources.go` | 164 | queued | — | — |
| 12 | `internal/models/learning.go` | 134 | queued | — | — |
| 13 | `internal/service/sync.go` | 122 | queued | — | — |
| 14 | `internal/config/config.go` | 101 | queued | — | — |
| 15 | `internal/service/registry.go` | 99 | queued | — | — |
| 16 | `cmd/alms/main.go` | 97 | queued | — | — |
| 17 | `internal/server/server.go` | 87 | queued | — | — |
| 18 | `internal/models/agent.go` | 81 | queued | — | — |
| 19 | `internal/service/interfaces.go` | 46 | queued | — | — |
| 20 | `internal/server/middleware.go` | 39 | queued | — | — |
| 21 | `internal/store/postgres.go` | 37 | queued | — | — |
| 22 | `internal/models/protocol.go` | 33 | queued | — | — |
| 23 | `internal/server/dashboard.go` | 22 | queued | — | — |
| 24 | `internal/models/errors.go` | 18 | queued | — | — |
| 25 | `tools.go` | 8 | queued | — | — |

Files with no justified refactor will be marked `inspected` with evidence rather than changed for change's sake.
