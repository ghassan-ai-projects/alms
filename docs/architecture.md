# ALMS Architecture & Design

---

## Design Principles

### Layered Architecture

```
cmd/alms/main.go
    ↓
internal/server/    →  MCP transport + middleware (thin, no business logic)
    ↓
internal/service/   →  Business logic (testable via store interfaces)
    ↓
internal/store/     →  PostgreSQL via pgx/v5/pgxpool (only I/O layer)
    ↓
internal/models/    →  Pure structs, shared everywhere
```

**Direction rules (enforced by `go vet` — no circular imports):**
- `store/` never imports `service/`
- `service/` imports `store/` interfaces (not implementations)
- `server/` imports `service/`
- `models/` has zero imports beyond stdlib + `time`

### Offline-First

ALMS is **not a runtime dependency** for agents. If ALMS is down:

1. Agents operate independently using their local state
2. They retry sync/registration on a backoff schedule
3. When ALMS comes back, agents re-sync from their last cursor
4. Gap-safe ack validation ensures no data loss across the outage

This is the most important architectural property. ALMS is advisory and persistent, not a bottleneck.

### Stdlib-First Dependency Policy

| Package | Purpose | Why not an alternative |
|---------|---------|----------------------|
| `net/http` | HTTP server | Go 1.22 enhanced routing eliminates chi/gorilla |
| `log/slog` | Logging | Structured, leveled, stdlib since Go 1.21 |
| `context` | Request propagation | Stdlib, no alternative |
| `pgx/v5` | PostgreSQL | Best Go PG driver — native pool, not `database/sql` |
| `mark3labs/mcp-go` | MCP protocol | Most active Go SDK, Streamable HTTP support |
| `google/uuid` | UUID generation | Or 20 lines of `crypto/rand` — negligible weight |
| `gopkg.in/yaml.v3` | Config parsing | Minimal dep — no viper (16+ indirect deps) |

Total: **5 third-party packages** in `go.mod`. Zero frameworks.

---

## Core Concepts

### Agent Registry

The registry tracks agent identity, capabilities, and health. It does NOT:

- Manage agent processes (no systemctl wrappers)
- Assign work or queue tasks
- Authenticate agents (agents authenticate to ALMS via bearer token)

Agents self-register on startup. ALMS detects stale agents via missing heartbeats.

### Learning Store

The learning store is a centralized knowledge base that agents read from and write to.

**Learning types:**

| Type | When to use | Example |
|------|-------------|---------|
| `pattern` | Reusable approach | "Always validate newsletter output with HTML Tidy" |
| `failure` | Bug or crash | "Site X API returns 500 every 1000th request — retry with 3s backoff" |
| `config` | Config insight | "Database pool size should be 10 for cron agents, 25 for interactive" |
| `protocol` | SOP | "Before publishing, run validation script in /opt/scripts/validate.sh" |
| `edge_case` | Rare condition | "User names with '&' break the RSS feed — html.EscapeString before render" |

### Sync Protocol (Gap-Safe)

```
Agent A registers → cursor = null
Agent A calls learning.sync(cursor=T0)
ALMS returns [LRN-001, LRN-002, LRN-003]

Agent A processes LRN-001, LRN-002
Agent A crashes

Agent A restarts → cursor = T0 (last acked)
Agent A calls learning.sync(cursor=T0)
ALMS returns [LRN-001, LRN-002, LRN-003]  (none were acked — no data loss!)

Agent A processes all 3
Agent A calls learning.sync_ack(ids=[LRN-001,LRN-002,LRN-003])
ALMS validates: no gaps → advances cursor to T1
```

**Why gap-safe matters:** Without it, a crash mid-batch would cause permanent knowledge loss. With it, the agent simply re-fetches from its last confirmed point.

**Gap detection:** When agent acks `[LRN-001, LRN-003]` but not `LRN-002`, ALMS rejects the ack with `ErrGapDetected`, listing the missing ID. The agent must re-fetch the gap.

### Dedup Strategy

| Type | Method | What happens |
|------|--------|-------------|
| **Exact** | SHA256 of `title + body` | Return existing ID, skip insert |
| **Near** | Levenshtein ratio on title (threshold 0.85) | Insert but flag as potential dupe |
| **Supersession** | Manual via `superseded_by` field | Old learning marked `superseded`, scored down |

### Garbage Collection

Runs periodically. Conditions for deletion:

```
created_at + ttl_days < now()
AND score < 0.3
AND NOT is_pinned
AND NOT is_deleted  (idempotent)
```

---

## MCP Surface

### Tools (16)

| Tool | Input | Output | Idempotent? |
|------|-------|--------|-------------|
| `agent.register` | agent_id, agent_type, capabilities... | AgentSpec | UPSERT |
| `agent.unregister` | agent_id | Success | Yes |
| `agent.update` | agent_id, fields | AgentSpec | Yes |
| `agent.list` | filter?, limit?, offset? | AgentSpec[] | Yes |
| `agent.heartbeat` | agent_id | Timestamp | Yes |
| `learning.sync` | agent_id, since, type?, tags? | LearningRecord[] | Yes |
| `learning.sync_ack` | agent_id, learning_ids | Success (rejects gaps) | Idempotent |
| `learning.store` | title, body, type, tags... | LearningID | No (creates new) |
| `learning.search` | query, type?, tags?, limit? | LearningRecord[] | Yes |
| `learning.delete` | learning_id | Success (soft) | Yes |
| `learning.get` | learning_id | LearningRecord | Yes |
| `protocol.pull` | agent_id | Protocol[] | Yes |
| `protocol.pull_since` | agent_id, since_id | Protocol[] | Yes |
| `protocol.push` | title, body, target_tags | ProtocolID | No |
| `protocol.list` | agent_id? | Protocol[] | Yes |
| `health.check` | — | HealthReport | Yes |

### Resources (6)

| URI | Content |
|-----|---------|
| `alms://agents` | All registered agents |
| `alms://agents/{id}` | Single agent spec |
| `alms://tools` | Aggregated tool catalog across all agents |
| `alms://learnings` | All learnings (paginated) |
| `alms://protocols` | Active SOPs |
| `alms://health` | ALMS system health (DB status, uptime) |

---

## Key Architectural Decisions

### Why PostgreSQL? (not SQLite, not file-based)

- ALMS runs on a server, not embedded — SQLite's single-writer model is a bottleneck
- Agents need concurrent read/write access (sync + ack + store can overlap)
- Partial indexes + GIN + tsvector provide real full-text search without Elasticsearch
- `pgx` native driver is fast enough for our scale (~100 agents, ~10K learnings)

### Why UUIDs? (not auto-increment)

- UUID eliminates DB-generated ID bottlenecks — the store layer just writes
- Distributed safe — if an agent generates an ID and ALMS is temporarily unreachable, no conflict
- The sort issue is irrelevant: sync uses `created_at DESC`, not UUID ordering

### Why JSONB for capabilities/metadata?

- Each agent has different capability shapes — relational normalization would be N join tables
- JSONB is indexable (GIN) for queries like "find all agents with tool X"
- It's rarely queried in complex ways — mostly read-as-a-whole per agent

### Why soft-delete for learnings?

Hard-delete creates holes in sync sequences. If learning LRN-003 is hard-deleted:
- Agent syncs from cursor → expects [LRN-001, LRN-002, LRN-003]
- LRN-003 is missing → gap-safe ack validation fires falsely
- Soft-delete with `NOT is_deleted` filter solves this cleanly

### Why gap validation at the server level? (not cursors only)

Timestamp cursors are monotonic but not gap-safe — an agent could miss learnings between T1 and T2 due to crash. The ack validation layer adds explicit gap detection by comparing expected IDs (pulled from DB) vs acked IDs (provided by agent). This is the server's responsibility, not the agent's.
