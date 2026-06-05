# ALMS — Agent Learning Management System
## Architecture Plan v3.2

**Date:** 2026-06-05
**Language:** Python
**Status:** Architecture Plan — Ready for Review

---

## 1. What ALMS Is

ALMS is a **learning store + agent registry** exposed as an MCP server. No lifecycle management, no process control. Agents register themselves, push learnings, pull knowledge.

| Component | Description | Status |
|-----------|-------------|--------|
| **Agent Registry** | Who's alive, their capabilities, their tools | New |
| **Learning Store** | Cross-agent learning transfer, SOP governance | IS-043 implemented as LearningRecord |
| **MCP Interface** | MCP server so any agent can interact | New |

ALMS is NOT:
- An agent lifecycle manager (no systemctl wrappers)
- An agent framework (not LangGraph, not CrewAI)
- A task scheduler or process supervisor

**Offline behavior:** If ALMS is down, agents operate offline and re-sync when ALMS comes back. ALMS is not a runtime dependency for agent execution — only for learning transfer and registration.

---

## 2. Architecture

```
┌──────────────────────────────────────────────────────────┐
│                      Data Machine                         │
│                     (192.168.2.112)                       │
│                                                           │
│  ┌──────────────────────────────────────────────────┐    │
│  │              ALMS MCP Server (8001)               │    │
│  │  ┌────────────────┐  ┌──────────────────────┐    │    │
│  │  │ Agent Registry │  │    Learning Store    │    │    │
│  │  └────────────────┘  └──────────────────────┘    │    │
│  │           FastAPI + MCP Streamable HTTP           │    │
│  └──────────────────────┬───────────────────────────┘    │
│                         │                                 │
│              ┌──────────┴──────────┐                      │
│              │     PostgreSQL      │                      │
│              │  - agents           │                      │
│              │  - learnings        │                      │
│              │  - tool_catalog     │                      │
│              │  - protocols        │                      │
│              └─────────────────────┘                      │
│                                                           │
│  Managed (local) + Remote (MCP client) Agents            │
└──────────────────────────────────────────────────────────┘
```

---

## 3. Module Structure

```
alms/
├── server.py              # FastAPI + MCP entrypoint + auth middleware
├── config.py              # YAML config (DB, auth token, paths)
├── models/
│   ├── agent.py           # AgentSpec
│   ├── learning.py        # LearningRecord
│   ├── protocol.py        # ProtocolRecord
│   └── tool.py            # ToolSpec
├── services/
│   ├── registry.py        # Agent CRUD
│   └── learning_store.py  # Learning CRUD, dedup, scoring, GC
├── mcp/
│   ├── resources.py       # MCP resource handlers
│   ├── tools.py           # MCP tool handlers
│   └── prompts.py         # MCP prompt templates
├── middleware/
│   └── auth.py            # Bearer token check
├── db/
│   ├── connection.py      # asyncpg pool
│   ├── migrations/        # alembic migrations
│   └── schema.sql         # PG schema
├── deploy/
│   ├── alms.service       # systemd unit
│   └── deploy.sh          # scp + ssh reload
├── docker-compose.yml     # PostgreSQL for local dev
├── tests/
└── Makefile
```

---

## 4. Data Model

### AgentSpec
```json
{
  "agent_id": "newsletter-scout",
  "display_name": "Newsletter Scout",
  "agent_type": "systemd",
  "capabilities": {
    "tools": ["web_search", "web_fetch", "article_write"],
    "skills": ["newsletter", "research"]
  },
  "metadata": {
    "owner": "ghassan",
    "tags": ["newsletter"]
  },
  "learning_sync": {
    "last_sync_timestamp": null,
    "last_sync_at": null
  },
  "health": {
    "last_heartbeat": null,
    "health_score": 1.0
  },
  "created_at": "2026-06-05T17:00:00Z"
}
```

### LearningRecord
```json
{
  "learning_id": "LRN-001",
  "type": "pattern",
  "title": "Site X returns malformed HTML",
  "body": "Parse with html.parser...",
  "tags": ["scraping", "workaround"],
  "severity": "medium",
  "author": "ghassan",
  "src_agent_id": "newsletter-scout",
  "ai_generated": false,
  "score": 0.85,
  "is_pinned": false,
  "resolution": "open",
  "superseded_by": null,
  "ttl_days": 90,
  "created_at": "2026-06-05T17:00:00Z"
}
```

### ProtocolRecord
```json
{
  "protocol_id": "SOP-001",
  "title": "Always validate newsletter output",
  "body": "Before publishing, run the validation script...",
  "target_tags": ["newsletter", "client-facing"],
  "version": 1,
  "author": "ghassan",
  "is_active": true,
  "created_at": "2026-06-05T17:00:00Z",
  "updated_at": null
}
```

### Sync Cursor & Gap Safety
Sync uses `created_at` timestamps. `learning.sync(agent_id, since_timestamp, type?, tags?)` returns learnings with `created_at > since_timestamp`, optionally filtered by type and/or tags.

`learning.sync_ack` accepts a batch of learning IDs: `["LRN-006", "LRN-007", "LRN-008"]`. ALMS validates that the batch contains no gaps relative to the agent's last acknowledged ID. If a gap is detected, the ack is rejected and the agent must re-fetch the gap. A `learning_acknowledgements` join table tracks per-learning ack state, enabling crash recovery: if the agent crashes mid-batch, unacknowledged learnings reappear on next sync.

Learnings use soft-delete: `is_deleted: bool`. Hard-delete would create holes in sync sequences and is prohibited.

---

## 5. MCP Surface

### Resources
| URI | Description |
|-----|-------------|
| `alms://agents` | All registered agents |
| `alms://agents/{id}` | Agent spec |
| `alms://tools` | Aggregated tool catalog |
| `alms://learnings` | All learnings |
| `alms://learnings/{id}` | Single learning |
| `alms://protocols` | Mandatory SOPs |
| `alms://health` | ALMS system health |

### Tools
| Tool | Description | Input | Output |
|------|-------------|-------|--------|
| `agent.register` | Register agent | agent_id, capabilities | AgentSpec |
| `agent.unregister` | Remove agent | agent_id | Success |
| `agent.update` | Update agent metadata | agent_id, fields | AgentSpec |
| `agent.list` | List agents | filter?, limit?, offset? | AgentSpec[] |
| `agent.heartbeat` | Agent reports alive | agent_id | Timestamp |
| `learning.store` | Push a learning | title, body, type, tags | LearningID |
| `learning.sync` | Get learnings since timestamp | agent_id, since_timestamp, type?, tags? | LearningRecord[] |
| `learning.sync_ack` | Confirm agent processed a batch | agent_id, learning_ids | Success (rejects gaps) |
| `protocol.pull` | Get all mandatory protocols for agent | agent_id | Protocol[] |
| `protocol.pull_since` | Get protocols changed since cursor | agent_id, since_id | Protocol[] |
| `learning.search` | Ad-hoc search | query, type?, tags?, limit? | LearningRecord[] |
| `learning.delete` | Soft-delete a learning | learning_id | Success |
| `protocol.push` | Create mandatory SOP | title, body, target_tags | ProtocolID |
| `protocol.list` | List protocols | agent_id? | Protocol[] |
| `health.check` | ALMS health | — | HealthReport |

---

## 6. Implementation

### Authentication
Static bearer token in `X-ALMS-TOKEN` header. Configurable via `alms.yaml`. Protects against unauthorized LAN access. 30 lines of middleware.

### The Sync Flow
```
Agent A stores: last_sync = "2026-06-04T00:00:00Z"

Agent A calls: learning.sync(agent_id="A", since_timestamp="2026-06-04T00:00:00Z")
ALMS returns: [LRN-006, LRN-007, LRN-008]
Agent A processes them
Agent A calls: learning.sync_ack(agent_id="A", learning_ids=["LRN-006","LRN-007","LRN-008"])
ALMS validates no gaps, updates agent_A.learning_sync to "2026-06-05T08:00:00Z"

Agent A calls: protocol.pull(agent_id="A")
ALMS returns: [all active SOPs targeting agent's tags]

── mid-batch crash scenario ──
Agent A processes LRN-006, crashes before LRN-007
Agent A restarts, calls: learning.sync(agent_id="A", since_timestamp="2026-06-04T00:00:00Z")
ALMS returns: [LRN-006, LRN-007, LRN-008] (LRN-006 was never acked)
Agent A processes, acks all 3 together. No data loss.
```

### Schema Decisions (PostgreSQL)
- Primary keys: UUID v4 for distributed safety
- Index on `learnings (created_at DESC)` for sync queries
- Full-text search via `tsvector` + GIN index on `title || ' ' || body`
- `learning_acknowledgements` join table: `(agent_id, learning_id, acknowledged_at)`
- Soft-delete: `is_deleted` boolean, `deleted_at` timestamp

### Phase 1 (Week 1)
- PostgreSQL setup, alembic init, schema
- ALMS server with FastAPI + MCP + auth middleware
- Agent registry (register, list, update, heartbeat)
- `learning.sync` + `learning.sync_ack`
- `protocol.pull`, `protocol.pull_since`
- Systemd deploy + deploy script

### Phase 2 (Week 2)
- `learning.store`, `learning.search`, `learning.delete`
- Learning dedup + scoring
- `protocol.push`, `protocol.list`
- Web dashboard

---

## 7. Tech Stack

| Layer | Choice |
|-------|--------|
| Server | Python + FastAPI + uvicorn |
| MCP | modelcontextprotocol/python-sdk |
| Database | PostgreSQL 16 |
| DB driver | asyncpg |
| Migrations | alembic |
| Auth | Static bearer token |
| Testing | pytest + pytest-asyncio + httpx |
| Linting | ruff |
| Dev DB | docker-compose.yml (PostgreSQL) |
| Deploy | systemd, deploy.sh |

---

## 8. Summary

ALMS is:
- An MCP server for agents to **register** themselves
- A **learning store** for cross-agent knowledge transfer
- A **tool catalog** showing what each agent can do

Agents manage their own lifecycle. ALMS just knows who's alive and what they know.
