# ALMS — Agent Learning Management System

**Registry + learning store for autonomous AI agents, exposed as an MCP server.**

ALMS is a lightweight, self-hosted backbone for multi-agent setups. Agents register themselves, share learnings, acknowledge protocols — no lifecycle management, no process control, no agent framework.

---

## What

ALMS is an **MCP server** (Model Context Protocol) that provides two core services:

| Service | What it does |
|---------|-------------|
| **Agent Registry** | Tracks live agents: who they are, their capabilities, their tools, their health |
| **Learning Store** | Cross-agent knowledge transfer: agents push learnings, pull what others discovered, acknowledge sync batches (gap-safe) |

**It is NOT:**
- ❌ An agent lifecycle manager (no process control, no systemctl wrappers)
- ❌ An agent framework (not LangGraph, not CrewAI)
- ❌ A task scheduler or message bus

---

## Why

If you run multiple AI agents (newsletter writers, researchers, automation scripts), each operates in isolation. When one agent discovers a workaround, a config pattern, or a failure mode, the others have no way to learn from it.

**ALMS solves:**
- **Cross-agent memory** — One agent's discovery becomes every agent's knowledge
- **Agent discovery** — Know who's alive and what each agent can do
- **Protocol enforcement** — Push mandatory SOPs to agents by capability tags
- **Crash recovery** — Gap-safe sync ack ensures no data loss on agent crash
- **Offline-first** — ALMS is not a runtime dependency; agents operate offline and re-sync when available

**Example:** Your newsletter-scout agent finds that Site X returns malformed HTML. It pushes a `pattern` learning with a parsing workaround. Next time the research-agent processes a learning sync, it pulls that pattern and avoids the same bug.

---

## How

### Architecture

```
┌──────────────────────────────────────────────────────────┐
│                      Data Machine                         │
│                     (192.168.2.112, Linux x86_64)         │
│                                                           │
│  ┌──────────────────────────────────────────────────┐    │
│  │              ALMS MCP Server (:8001)              │    │
│  │  ┌────────────────┐  ┌──────────────────────┐    │    │
│  │  │ Agent Registry │  │    Learning Store    │    │    │
│  │  └────────────────┘  └──────────────────────┘    │    │
│  │           Go 1.22, mark3labs/mcp-go, pgx/v5      │    │
│  └──────────────────────┬───────────────────────────┘    │
│                         │                                 │
│              ┌──────────┴──────────┐                      │
│              │     PostgreSQL 16   │                      │
│              │                     │                      │
│              │  agents             │                      │
│              │  learnings          │ ← GIN-indexed FTS   │
│              │  learning_acks      │                      │
│              │  protocols          │                      │
│              └─────────────────────┘                      │
└──────────────────────────────────────────────────────────┘
        ▲                           ▲
        │ MCP Streamable HTTP       │ MCP Streamable HTTP
        │                           │
┌───────┴──────────┐      ┌────────┴──────────┐
│   OpenClaw (Mac)  │      │   cron agents      │
│                   │      │   (data machine)   │
│  main orchestrator│      │   newsletter, etc. │
└───────────────────┘      └───────────────────┘
```

### Stack

| Layer | Choice |
|-------|--------|
| Language | **Go 1.22+** — single binary, minimal deps, fast startup |
| Database | **PostgreSQL 16** — GIN-indexed full-text search, time-series sync |
| Transport | **MCP Streamable HTTP** — standard protocol for AI agent tooling |
| Deploy | **systemd** single binary on headless Linux |
| DB driver | **pgx/v5** — native PostgreSQL driver, no ORM |

### Dependencies (Go)

| Package | Purpose | Justification |
|---------|---------|---------------|
| `net/http` | HTTP server | Stdlib — no chi/gorilla |
| `log/slog` | Logging | Stdlib (Go 1.21+) |
| `pgx/v5` | PostgreSQL | Best Go PG driver |
| `mark3labs/mcp-go` | MCP protocol | Most active Go MCP SDK |
| `google/uuid` | UUID gen | Small, zero indirect deps |
| `gopkg.in/yaml.v3` | Config parsing | Minimal — no viper |

Total: **5 third-party packages**.

### Data Model (4 tables)

- **agents** — `agent_id`, `type`, `capabilities` (JSONB), `metadata` (JSONB), sync cursors, heartbeat, health score
- **learnings** — UUID key, `type` (pattern/failure/config/protocol/edge_case), `title`, `body`, `tags[]`, `severity`, `score` (0.0–1.0), `ttl_days`, soft-delete, search_vector (tsvector + GIN)
- **learning_acknowledgements** — `(agent_id, learning_id)` join table, prevents crash data loss
- **protocols** — `title`, `body`, `target_tags[]`, versioned, active/inactive

### MCP Surface: 16 tools + 6 resources

**Tools** `agent.(register|unregister|update|list|heartbeat)` · `learning.(store|sync|sync_ack|search|delete|get)` · `protocol.(pull|pull_since|push|list)` · `health.check`

**Resources** `alms://agents`, `alms://agents/{id}`, `alms://tools`, `alms://learnings`, `alms://protocols`, `alms://health`

---

## Project Structure

```
alms/
├── cmd/alms/main.go          ← Entry point: flag parse, DI, graceful shutdown
├── internal/
│   ├── config/               ← YAML + env config (yaml.v3, no viper)
│   ├── models/               ← Pure data structs + typed constants + validation
│   ├── store/                ← PostgreSQL via pgx/v5/pgxpool
│   │   └── migrations/       ← golang-migrate SQL files
│   ├── service/              ← Business logic (testable via store interfaces)
│   │   └── storemock/        ← Manual mocks for unit tests
│   └── server/               ← MCP handlers + auth middleware
├── deploy/                   ← systemd unit, deploy script, backup cron
├── docs/                     ← Reference documentation
├── test/                     ← Integration + load tests
├── Makefile                  ← build, lint, test, ci-check
├── docker-compose.yml        ← PostgreSQL for dev
└── .golangci.yml             ← Lint config (14 linters enabled)
```

---

## Quick Start

```bash
# Prerequisites
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
brew install golangci-lint

# Start PostgreSQL
docker compose up -d db

# Set env vars
export ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"

# Run migrations
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up

# Build and run
make build
./bin/alms

# Verify — list MCP tools
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | jq .
```

See **[docs/getting-started.md](docs/getting-started.md)** for a tutorial walkthrough.

---

## Quality

```bash
make test        # go test -race -shuffle=on -coverprofile=coverage.out
make lint        # golangci-lint (14 linters: errcheck, staticcheck, gosec, revive, etc.)
make ci-check    # tidy → build → vet → lint → test (what CI runs)
make deadcode    # Detect unused functions
```

CI runs on every push via GitHub Actions with a PostgreSQL 16 service container.

---

## Deploy

ALMS deploys to a headless Linux machine (`192.168.2.112`):

```bash
./deploy/deploy.sh   # Cross-compile → SCP → systemd reload
```

See **[docs/operations.md](docs/operations.md)** for service management, backup, and monitoring.

---

## OpenClaw Integration

Apply the MCP server patch so OpenClaw can talk to ALMS:

```bash
openclaw config patch --file deploy/openclaw-mcp-patch.json5
```

This exposes all ALMS tools and resources directly in the OpenClaw session. See **[docs/integration-guide.md](docs/integration-guide.md)** for the full agent lifecycle workflow.

---

## Agent Workflow (TL;DR)

```
Startup → agent.register
         → learning.sync (pull new knowledge)
         → protocol.pull (get SOPs)
         → process learnings
         → learning.sync_ack

Every 5min → agent.heartbeat

On discovery → learning.store

Shutdown → (nothing; ALMS detects stale via heartbeat timeout)
```

ALMS is **offline-safe** — agents operate independently and re-sync when it comes back.

---

## Design Principles

- **Layered architecture** — `cmd → server → service → store → models` (no circular imports)
- **Interface-driven** — services depend on interfaces, not implementations (testable)
- **PG-native** — no ORM, no `database/sql` — pgx native pool with parameterised queries
- **Stdlib-first** — `net/http` router, `log/slog` logging, `context` propagation
- **Minimal deps** — 5 third-party packages, zero frameworks, 60-line `go.mod`
- **Offline-first** — ALMS is advisory, not a runtime dependency

---

## License

MIT — see [LICENSE](LICENSE).

---

## Related Docs

| Document | What it covers |
|----------|---------------|
| [Getting Started](docs/getting-started.md) | Step-by-step tutorial from zero to running |
| [Data Model](docs/data-model.md) | Tables, columns, indexes, SQL queries |
| [Agent Integration](docs/integration-guide.md) | OpenClaw MCP setup + agent lifecycle |
| [Operations](docs/operations.md) | Deploy, systemd, backup, monitoring |
| [Design & Architecture](docs/architecture.md) | Full design decisions, layering, patterns |
| [Quality](docs/quality.md) | Makefile targets, linting, testing strategy |
