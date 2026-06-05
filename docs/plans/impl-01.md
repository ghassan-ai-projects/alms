# ALMS — Implementation Plan

## Overview

**Repository:** `github.com/ghassan-ai-projects/alms`
**Language:** Go (1.22+)
**Architecture:** MCP server (Streamable HTTP) + PostgreSQL + systemd
**Machine:** `data` (192.168.2.112, Ubuntu 26.04, 7GB RAM, 12 cores)

The implementation is split into 4 sequential phases. Each phase produces a running, testable milestone. Phases are gated — Phase 2 depends on Phase 1, Phase 3 depends on Phase 2.

---

## File System Targets

All code goes under `~/ai-projects/alms/` and is pushed to GitHub. Deployment SCPs to `data:/opt/alms/`.

### Docs Structure (after this plan is created)
```
alms/docs/
├── plans/
│   └── impl-01.md            # This document — full implementation plan
├── sync-flow.md              # Sequence diagram + gap-safe algorithm (Phase 1)
├── postgres-schema.md        # Full PG schema with indexes (Phase 1)
├── integration-guide.md      # How to connect OpenClaw, Qwen, newsletter agents (Phase 3)
└── operations.md             # Runbooks: deploy, backup, restore, monitor (Phase 4)
```

### Project Structure (Phase 1 delivers)
```
alms/
├── cmd/
│   └── alms/
│       └── main.go                 # Entry point
├── internal/
│   ├── config/
│   │   └── config.go               # YAML + env config
│   ├── models/
│   │   ├── agent.go                # AgentSpec, AgentState
│   │   ├── learning.go             # LearningRecord
│   │   ├── protocol.go             # ProtocolRecord
│   │   └── errors.go               # Sentinel errors
│   ├── store/
│   │   ├── agent_store.go          # Agent CRUD + sync cursor
│   │   ├── learning_store.go       # Learning CRUD + dedup
│   │   ├── protocol_store.go       # Protocol CRUD
│   │   └── migrations/
│   │       └── 001_initial.sql
│   ├── server/
│   │   ├── server.go               # MCP server init (Streamable HTTP)
│   │   ├── tools.go                # Tool handlers
│   │   ├── resources.go            # Resource handlers
│   │   └── middleware.go           # Auth middleware
│   └── service/
│       ├── registry.go             # Agent registration + heartbeat
│       ├── sync.go                 # Learning sync + gap-safe ack
│       └── learning.go             # Store, search, scoring
├── deploy/
│   ├── alms.service                # systemd unit
│   ├── alms.yaml                   # Default config
│   └── deploy.sh                   # scp + reload script
├── docker-compose.yml              # PostgreSQL for dev
├── Makefile                        # Quality targets
├── .golangci.yml                   # Linter config
├── .pre-commit-config.yaml         # Pre-commit hooks
├── .editorconfig
├── .gitignore
├── go.mod
├── tools.go                        # Pin tool deps
├── AGENTS.md
└── README.md
```

---

## Phase 1: Core Bootstrapping (Week 1)

**Deliverable:** ALMS binary runs locally. Agent registry works via MCP tools. Learning sync + ack works.

### 1.1 Project Scaffolding (Day 1)

#### Files to Create
| File | Description |
|------|-------------|
| `go.mod` | Module `github.com/ghassan/alms`, Go 1.22, deps: pgx/v5, mcp-go, yaml.v3, uuid |
| `tools.go` | Pin deadcode, govulncheck, golang-migrate |
| `Makefile` | build, vet, lint, test, ci-check, tidy, deadcode, vulncheck targets |
| `.golangci.yml` | 15 linters enabled (see go-design §5.3) |
| `.pre-commit-config.yaml` | 6 hooks (see go-design §5.4) |
| `.editorconfig` | Go: tabs at 8. YAML/JSON/MD: spaces at 2. |
| `.gitignore` | ignore: `coverage.out`, `bin/`, `.env`, `~/.alms/` |
| `docker-compose.yml` | PostgreSQL 16 service with `alms` user + `alms_db` database |
| `AGENTS.md` | AI assistant guide (see go-design §11) |
| `README.md` | Project description, quick start, badge links |

#### Verification
```bash
go mod tidy
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
go install golang.org/x/tools/cmd/deadcode@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
make ci-check   # should pass with zero Go files
```

### 1.2 Config Package (Day 1)

#### File
`internal/config/config.go`

#### Responsibilities
- Load YAML config from `~/.alms/alms.yaml` (via `gopkg.in/yaml.v3`)
- Override with env vars: `ALMS_PG_DSN`, `ALMS_AUTH_TOKEN`
- Provide `Config` struct with `Server`, `Database`, `Auth` sub-configs
- `Server.Addr()` method returning `host:port`

#### Tests
- `TestLoadFromFile` — valid YAML → correct struct
- `TestLoadFromEnv` — env var overrides file
- `TestLoadDefaults` — no file + no env → sensible defaults
- `TestLoadMissingFile` — graceful fallback to env/defaults

#### Verification
```go
cfg := config.Load()
fmt.Println(cfg.Server.Addr()) // 127.0.0.1:8001
```

### 1.3 Models Package (Day 1)

#### Files
| File | Contents |
|------|----------|
| `internal/models/agent.go` | `AgentSpec` struct with `Validate()`, `AgentType` constants, `AgentCapabilities`, `AgentMetadata`, `LearningsSync`, `AgentHealth` |
| `internal/models/learning.go` | `LearningRecord` struct with `Validate()`, `LearningType`, `Severity`, `Resolution` constants |
| `internal/models/protocol.go` | `ProtocolRecord` struct with `Validate()` |
| `internal/models/errors.go` | Sentinel errors: `ErrNotFound`, `ErrConflict`, `ErrValidation`, `ErrGapDetected` |

#### Design Rules
- All struct fields exported with JSON tags
- `omitempty` on optional fields only
- Typed constants for all enums
- `Validate()` returns `ErrValidation` with field-level details
- No methods beyond `Validate()` and `String()`

#### Tests
- `TestAgentSpecValidate` — valid, empty ID, invalid type, long ID
- `TestLearningRecordValidate` — valid, missing title, invalid type
- `TestProtocolRecordValidate` — valid, missing title
- Fuzz test: `FuzzValidateAgent` on random inputs (no panic)

#### Verification
```go
a := AgentSpec{AgentID: "test", AgentType: AgentTypeSystemd}
err := a.Validate()  // nil
a2 := AgentSpec{AgentID: ""}
err2 := a2.Validate() // ErrValidation
```

### 1.4 PG Schema + Migrations (Day 2)

#### File
`internal/store/migrations/001_initial.sql`

#### Tables

**`agents`**
```sql
CREATE TABLE agents (
    agent_id       TEXT PRIMARY KEY,
    display_name   TEXT NOT NULL DEFAULT '',
    agent_type     TEXT NOT NULL CHECK (agent_type IN ('systemd', 'mcp_client')),
    capabilities   JSONB NOT NULL DEFAULT '{}',
    metadata       JSONB NOT NULL DEFAULT '{}',
    last_sync_ts   TIMESTAMPTZ,
    last_sync_at   TIMESTAMPTZ,
    last_heartbeat TIMESTAMPTZ,
    health_score   REAL NOT NULL DEFAULT 1.0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

**`learnings`**
```sql
CREATE TABLE learnings (
    learning_id    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type           TEXT NOT NULL CHECK (type IN ('pattern','failure','config','protocol','edge_case')),
    title          TEXT NOT NULL,
    body           TEXT NOT NULL DEFAULT '',
    tags           TEXT[] NOT NULL DEFAULT '{}',
    severity       TEXT NOT NULL DEFAULT 'medium' CHECK (severity IN ('low','medium','high','critical')),
    author         TEXT NOT NULL DEFAULT '',
    src_agent_id   TEXT REFERENCES agents(agent_id) ON DELETE SET NULL,
    ai_generated   BOOLEAN NOT NULL DEFAULT false,
    score          REAL NOT NULL DEFAULT 0.5,
    is_pinned      BOOLEAN NOT NULL DEFAULT false,
    is_deleted     BOOLEAN NOT NULL DEFAULT false,
    resolution     TEXT NOT NULL DEFAULT 'open' CHECK (resolution IN ('open','resolved','superseded')),
    superseded_by  UUID REFERENCES learnings(learning_id) ON DELETE SET NULL,
    ttl_days       INT NOT NULL DEFAULT 90,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    search_vector  TSVECTOR GENERATED ALWAYS AS (to_tsvector('english', title || ' ' || body)) STORED
);

CREATE INDEX idx_learnings_created_at ON learnings (created_at DESC);
CREATE INDEX idx_learnings_search ON learnings USING GIN (search_vector);
CREATE INDEX idx_learnings_type ON learnings (type);
CREATE INDEX idx_learnings_tags ON learnings USING GIN (tags);
CREATE INDEX idx_learnings_active ON learnings (created_at DESC) WHERE NOT is_deleted;
```

**`learning_acknowledgements`** (gap-safe join table)
```sql
CREATE TABLE learning_acknowledgements (
    agent_id        TEXT NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    learning_id     UUID NOT NULL REFERENCES learnings(learning_id) ON DELETE CASCADE,
    acknowledged_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (agent_id, learning_id)
);
```

**`protocols`**
```sql
CREATE TABLE protocols (
    protocol_id   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title         TEXT NOT NULL,
    body          TEXT NOT NULL DEFAULT '',
    target_tags   TEXT[] NOT NULL DEFAULT '{}',
    version       INT NOT NULL DEFAULT 1,
    author        TEXT NOT NULL DEFAULT '',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ
);
```

#### Migration Tool
Use `golang-migrate/migrate` CLI. File naming: `001_initial.up.sql`, `001_initial.down.sql`.

```bash
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
```

#### Verification
```bash
docker compose up -d db
migrate -path internal/store/migrations -database "postgres://alms:alms@localhost:5432/alms_db?sslmode=disable" up
psql "postgres://alms:alms@localhost:5432/alms_db" -c "\dt"
# Should show 4 tables
```

### 1.5 Store Layer (Day 2-3)

#### Files
| File | Implements |
|------|------------|
| `internal/store/postgres.go` | `NewPool(ctx, dsn)` — pgxpool init, `Ping()` |
| `internal/store/agent_store.go` | `AgentStore` interface methods |
| `internal/store/learning_store.go` | `LearningStore` interface methods |
| `internal/store/protocol_store.go` | `ProtocolStore` interface methods |

#### AgentStore Methods
```go
Create(ctx, spec) error
Get(ctx, agentID) (AgentSpec, error)
Update(ctx, spec) error
Delete(ctx, agentID) error
Heartbeat(ctx, agentID) (time.Time, error)
List(ctx, filter, limit, offset) ([]AgentSpec, error)
```

#### LearningStore Methods
```go
Create(ctx, record) (learningID string, err error)
Get(ctx, learningID) (LearningRecord, error)
Sync(ctx, agentID, since time.Time, ltype string, tags []string) ([]LearningRecord, error)
SyncAck(ctx, agentID, learningIDs []string) error
Search(ctx, query, ltype string, tags []string, limit int) ([]LearningRecord, error)
SoftDelete(ctx, learningID) error
```

**`SyncAck` algorithm (critical path):**
1. Fetch all learnings with `created_at > agent.last_sync_ts AND NOT is_deleted`
2. Collect their IDs into an ordered list `expected_ids`
3. Verify every ID in `expected_ids` appears in `ack.learning_ids`
4. If gap found → return `ErrGapDetected` with the missing IDs
5. If no gap → insert into `learning_acknowledgements`, update `agents.last_sync_ts`

#### ProtocolStore Methods
```go
Create(ctx, record) (protocolID string, err error)
Get(ctx, protocolID) (ProtocolRecord, error)
Pull(ctx, agentID) ([]ProtocolRecord, error)        // All active + matching agent tags
PullSince(ctx, agentID, sinceID) ([]ProtocolRecord, error)
List(ctx) ([]ProtocolRecord, error)
```

#### Tests
- `TestAgentStoreCRUD` — create, get, update, list, delete against real PG
- `TestLearningStoreSync` — push 3 learnings, sync from 0, verify ordering
- `TestSyncAckNoGap` — valid batch accepted
- `TestSyncAckWithGap` — batch missing a learning → `ErrGapDetected`
- `TestSyncAckIdempotent` — ack same batch twice → 2nd call no-op
- `TestSyncAckCrashedAgent` — agent acks partial batch, sync restores unacked
- `TestLearningSearch` — full-text search query returns results
- `TestProtocolPull` — verify tag-based filtering

### 1.6 Service Layer (Day 3-4)

#### Files
| File | Responsibilities |
|------|-----------------|
| `internal/service/registry.go` | Agent register/unregister/list/heartbeat business rules |
| `internal/service/sync.go` | Sync flow orchestration + gap validation + protocol pull |
| `internal/service/learning.go` | Store, search, scoring, dedup, GC |

**Key: services accept interfaces, not concrete types.** Service constructors take:
```go
func NewRegistry(store AgentStore)
func NewSyncer(store LearningStore, agentStore AgentStore, protoStore ProtocolStore)
```

#### Gap Validation Algorithm (in `sync.go`)
```
func (s *Syncer) Ack(ctx, agentID, learningIDs []string) error:
  agent, err := s.agentStore.Get(ctx, agentID)   // get last_sync_ts
  newLearnings, err := s.learningStore.Sync(ctx, agentID, agent.LearningsSync.LastSyncTimestamp, "", nil)
  expectedIDs := extractIDs(newLearnings)          // ordered by created_at
  missingIDs := subtract(expectedIDs, learningIDs) // set difference preserving order
  if len(missingIDs) > 0:
    return fmt.Errorf("%w: missing IDs %v", models.ErrGapDetected, missingIDs)
  return s.learningStore.SyncAck(ctx, agentID, learningIDs)
```

#### Tests
- `TestRegisterAgent` — valid agent → stored + returned
- `TestDuplicateRegister` — same agent_id twice → `ErrConflict`
- `TestHeartbeatUpdatesTimestamp` — heartbeat advances last_heartbeat
- `TestSyncFlowEndToEnd` — register agent → push learning → sync → ack → sync empty
- `TestGapSafeAckRejectsMissing` — ack with missing IDs → `ErrGapDetected`
- `TestGapSafeAckAcceptsCorrect` — ack with all IDs → OK
- `TestCrashedAgentRecovers` — ack partial, resync, remaining learnings returned

### 1.7 Server Layer (Day 4-5)

#### Files
| File | Responsibilities |
|------|-----------------|
| `internal/server/server.go` | `mark3labs/mcp-go` server init, `ListenAndServe(ctx)`, `Shutdown(ctx)` |
| `internal/server/tools.go` | Tool handler functions (15 tools) |
| `internal/server/resources.go` | Resource handler functions (7 resources) |
| `internal/server/middleware.go` | `X-ALMS-TOKEN` bearer check middleware |

#### Server setup
```go
func New(cfg *config.Config, regSvc *service.Registry, syncSvc *service.Syncer, learnSvc *service.Learning) *Server
func (s *Server) ListenAndServe(ctx context.Context) error
```

#### Tools to register (Phase 1 subset = 9 tools)
| Tool | Phase 1 | Notes |
|------|---------|-------|
| `agent.register` | ✅ | |
| `agent.unregister` | ✅ | |
| `agent.update` | ✅ | |
| `agent.list` | ✅ | |
| `agent.heartbeat` | ✅ | |
| `learning.sync` | ✅ | Core sync |
| `learning.sync_ack` | ✅ | Gap-safe ack |
| `protocol.pull` | ✅ | |
| `protocol.pull_since` | ✅ | |
| `learning.store` | Phase 2 | |
| `learning.search` | Phase 2 | |
| `learning.delete` | Phase 2 | |
| `protocol.push` | Phase 2 | |
| `protocol.list` | Phase 2 | |
| `health.check` | Phase 2 | |

#### Resources to register (Phase 1 subset = 4 resources)
| Resource | Phase 1 | Notes |
|----------|---------|-------|
| `alms://agents` | ✅ | |
| `alms://agents/{id}` | ✅ | |
| `alms://health` | ✅ | |
| `alms://learnings` | Phase 2 | |
| `alms://learnings/{id}` | Phase 2 | |
| `alms://tools` | Phase 2 | |
| `alms://protocols` | Phase 2 | |

#### Middleware
```go
func AuthMiddleware(token string) func(http.Handler) http.Handler
```
- Reads `X-ALMS-TOKEN` header
- If config token is empty → pass through (dev mode)
- If config token is set + header missing/wrong → return MCP JSON-RPC error `{"error":{"code":-32001,"message":"unauthorized"}}`

#### Tests
- `TestAgentRegisterTool` — MCP call returns agent spec
- `TestAgentListTool` — multiple agents listed
- `TestSyncFlowOverMCP` — full MCP round-trip
- `TestAuthMiddleware` — missing token returns error
- `TestAuthMiddlewareDevMode` — empty token = pass through

### 1.8 Main Entry Point (Day 5)

#### File
`cmd/alms/main.go`

```go
func main() {
    // Parse flags: --config, --migrate, --version
    // Load config
    // Connect PG pool
    // If --migrate: run migrations, exit
    // Init services (registry, syncer, learning)
    // Init MCP server
    // Start HTTP server in goroutine
    // Wait for SIGINT/SIGTERM via signal.NotifyContext
    // Graceful shutdown: 10s timeout
}
```

#### Verification
```bash
go run ./cmd/alms/ --version       # prints version + commit
go run ./cmd/alms/ --migrate       # runs migrations, exits
go run ./cmd/alms/                 # starts server on :8001
curl -X POST http://localhost:8001/mcp ...  # tools/list works
```

### 1.9 Phase 1 Integration Test

```bash
docker compose up -d db
make migrate
make build
./bin/alms --migrate &
sleep 2

# Register agent
curl -X POST ... -d '{"method":"tools/call","params":{"name":"agent.register","arguments":{"agent_id":"test","agent_type":"systemd"}}}'

# Push learning
# Sync
# Ack
# Verify sync returns empty

kill %1
docker compose down
```

### Phase 1 Exit Criteria
- [ ] `make ci-check` passes (tidy → build → vet → lint → test)
- [ ] 4 PostgreSQL tables with proper indexes
- [ ] 3 store interfaces fully implemented with PG queries
- [ ] 3 services with business logic
- [ ] MCP server with 9 tools + 4 resources
- [ ] Auth middleware working (token passthrough in dev)
- [ ] Gap-safe sync flow E2E test passes
- [ ] Graceful shutdown works (SIGTERM → drain → exit)
- [ ] `make test` coverage > 70% (service), > 50% (store)

---

## Phase 2: Learning Store (Week 2)

**Deliverable:** Full learning CRUD + search + dedup + protocol management. Web dashboard.

### 2.1 Remaining Tools + Resources
| Tool | Status | Implementation |
|------|--------|----------------|
| `learning.store` | New | Create learning, SHA256 dedup check |
| `learning.search` | New | Full-text search via GIN index |
| `learning.delete` | New | Soft-delete via `is_deleted` |
| `protocol.push` | New | Create protocol |
| `protocol.list` | New | List all protocols |
| `health.check` | New | PG ping + agent count |

### 2.2 Dedup Engine
File: `internal/service/dedup.go`

- **Exact dedup**: SHA256 of `title + body` → `INSERT ... ON CONFLICT DO NOTHING`
- **Near dedup**: Levenshtein ratio on title (threshold 0.85) → flag for human review
- **Supersession**: `learning.store` accepts optional `supersedes` param

### 2.3 Scoring Engine
File: `internal/service/scoring.go`

- Default score: 0.5
- Score increments: +0.1 per successful sync (how many agents pulled it)
- Score decrements: -0.1 per TTL day without updates
- Pinned learnings skip score decay

### 2.4 GC Scheduler
File: `internal/service/gc.go`

- Background goroutine runs every 24h
- Soft-deletes learnings where `created_at + ttl_days < now()` AND `score < 0.3` AND `NOT is_pinned`
- Logs `slog.Info("gc completed", "deleted", count)`
- Configurable via `alms.yaml`: `gc.enabled: true`, `gc.interval: 24h`

### 2.5 Web Dashboard
File: `internal/server/dashboard.go`

- Static HTML page served at `/dashboard`
- Shows: agent list with health, learning count, recent syncs
- JavaScript fetches data via MCP HTTP calls
- Refresh button, no auto-refresh

### Phase 2 Exit Criteria
- [ ] All 15 MCP tools + 7 resources registered
- [ ] Dedup engine tested: exact + near matches
- [ ] Scoring engine tested: increment, decrement, decay
- [ ] GC tested: manual trigger, verify soft-deletes
- [ ] Web dashboard renders agent list + learnings
- [ ] `learning.search` returns relevant results from GIN index
- [ ] `make test` coverage > 75% (service), > 55% (store)

---

## Phase 3: Deployment + Integration (Week 3)

**Deliverable:** ALMS runs on data machine. First agents connect.

### 3.1 systemd Deployment

#### File
`deploy/alms.service`
```
[Unit]
Description=ALMS Agent Control Plane
After=network.target postgresql.service
Requires=postgresql.service

[Service]
Type=simple
User=ghassan
WorkingDirectory=/opt/alms
ExecStart=/opt/alms/alms --config /opt/alms/alms.yaml
Restart=on-failure
RestartSec=5
MemoryMax=512M
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

#### File
`deploy/deploy.sh`
```bash
#!/bin/bash
set -e
GOOS=linux GOARCH=amd64 go build -ldflags="..." -o bin/alms-linux ./cmd/alms/
scp bin/alms-linux data:/opt/alms/alms
scp deploy/alms.yaml data:/opt/alms/
ssh data "sudo cp deploy/alms.service /etc/systemd/system/ && sudo systemctl daemon-reload && sudo systemctl restart alms"
ssh data "systemctl status alms --no-pager"
```

### 3.2 PostgreSQL Setup on Data Machine
```bash
ssh data "
  sudo apt-get install -y postgresql postgresql-client
  sudo systemctl enable --now postgresql
  sudo -u postgres createdb alms_db
  sudo -u postgres createuser alms -P
  sudo -u postgres psql -c \"GRANT ALL ON DATABASE alms_db TO alms;\"
"
```

### 3.3 OpenClaw MCP Integration
- OpenClaw registers ALMS as an MCP server in its config
- Verify `tools/list` returns 15 tools
- Verify OpenClaw can call `agent.register`, `learning.sync`

### 3.4 Newsletter Agent Registration
- First real agent: newsletter-scout
- Registers via MCP on startup
- Pushes learnings after each newsletter run
- Syncs learnings on startup

### 3.5 Backup Cron
```bash
# /etc/cron.d/alms-backup
0 3 * * * ghassan pg_dump -U alms alms_db > /opt/alms/backup/alms-$(date +\%F).sql
```

### Phase 3 Exit Criteria
- [ ] ALMS running on data:8001
- [ ] systemd unit active, auto-restarts on crash
- [ ] Deploy script pushes new binary in < 10s
- [ ] OpenClaw connected as MCP host
- [ ] At least one real agent registered + syncing
- [ ] Daily backup cron installed + verified

---

## Phase 4: Polish + Operations (Week 4)

**Deliverable:** Production-ready. Tests, docs, monitoring.

### 4.1 Integration Tests
- `testcontainers-go` for PostgreSQL in CI
- Full sync flow E2E: register agent → push 3 learnings → sync → ack → verify
- Recovery flow: agent crashes, restarts, sync returns unacked learnings

### 4.2 Operations Documentation
`docs/operations.md`
- Deploy procedure
- Backup + restore
- Migration rollback
- Log inspection via journalctl
- Disk space monitoring
- PostgreSQL maintenance

### 4.3 Load Testing
- Simulate 10 agents syncing concurrently
- Measure: sync latency, pg CPU, memory usage
- Document limits for this 7GB machine

### 4.4 Performance Indexes
- Verify query plans for critical paths:
  ```sql
  EXPLAIN ANALYZE SELECT ... FROM learnings WHERE created_at > $1 AND NOT is_deleted ORDER BY created_at;
  EXPLAIN ANALYZE SELECT ... FROM learnings WHERE search_vector @@ plainto_tsquery('english', $1);
  ```
- Add missing indexes if needed

---

## Appendices

### A: Full Tool → Phase Mapping

| Tool | Phase 1 | Phase 2 | Phase 3 | Phase 4 |
|------|---------|---------|---------|---------|
| `agent.register` | ✅ | | | |
| `agent.unregister` | ✅ | | | |
| `agent.update` | ✅ | | | |
| `agent.list` | ✅ | | | |
| `agent.heartbeat` | ✅ | | | |
| `learning.sync` | ✅ | | | |
| `learning.sync_ack` | ✅ | | | |
| `protocol.pull` | ✅ | | | |
| `protocol.pull_since` | ✅ | | | |
| `learning.store` | | ✅ | | |
| `learning.search` | | ✅ | | |
| `learning.delete` | | ✅ | | |
| `protocol.push` | | ✅ | | |
| `protocol.list` | | ✅ | | |
| `health.check` | | ✅ | | |
| Deployment to data | | | ✅ | |
| OpenClaw integration | | | ✅ | |
| E2E integration tests | | | | ✅ |
| Load testing | | | | ✅ |
| Operations docs | | | | ✅ |

### B: Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| pgx connection pool exhausted | Low | Medium | `MaxConns=10` default, monitor with health.check |
| MCP SDK API changes | Medium | High | Pin version in go.mod, test upgrades |
| Sync ack race condition | Low | High | PG `SERIALIZABLE` isolation for ack transaction |
| Disk full from logs | Medium | Medium | journald `SystemMaxUse=1G`, monitor with health.check |
| Cross-compilation failure | Low | High | `CGO_ENABLED=0` in deploy.sh, CI builds linux binary |
| Gap-safe algorithm incorrect | Low | Critical | Fuzz testing on SyncAck, E2E crash recovery test |

### C: Developer Workflow

```bash
# Daily
git checkout -b feat/xxx
# code...
make lint        # fast pre-commit check
make test-short  # unit tests only
git commit -m "feat: xxx"
# repeat

# Before push
make ci-check    # full build + lint + test
git push

# Before deploy
make build-linux
./deploy/deploy.sh
```

### D: File Creation Order (Phase 1)

To minimize blocking dependencies, create files in this order:

1. `go.mod`, `.golangci.yml`, `.pre-commit-config.yaml`, `Makefile`, `.editorconfig`, `.gitignore`, `AGENTS.md` — scaffolding
2. `internal/config/config.go` + tests — no dependencies
3. `internal/models/*.go` + tests — depends only on config (Validate patterns)
4. `internal/store/postgres.go` — depends on config
5. `internal/store/migrations/001_initial.sql` — depends on nothing
6. `internal/store/agent_store.go`, `learning_store.go`, `protocol_store.go` + tests — depends on models + postgres
7. `internal/service/registry.go`, `sync.go`, `learning.go` + tests — depends on store interfaces
8. `internal/server/middleware.go` — depends on config
9. `internal/server/server.go`, `tools.go`, `resources.go` — depends on services
10. `cmd/alms/main.go` — depends on everything
11. `docker-compose.yml`, `deploy/*` — operational files
