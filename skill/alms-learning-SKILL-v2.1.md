---
name: learning
description: "ALMS learning workflow for agents: search prior knowledge, sync remote learnings, capture new learnings, and publish them back to ALMS."
version: "2.2.0"
author: "ghassan-ai"
tags: [knowledge, alms, memory, synchronization, cron]
provides: ["learn-store", "learn-search", "learn-sync", "learn-score"]
---

# Learning Skill (ALMS)

Connect OpenClaw agents to the **Agent Learning Management System (ALMS)** for cross-agent knowledge persistence, discovery, and synchronization.

This skill is intended to be the default learning workflow for any new ALMS-connected agent. An agent should not only register itself. It should also read existing knowledge before major work and publish meaningful learnings after major work.

Use the helper scripts in `scripts/` as the default MCP bridge for repeatable learning sync and publish operations. Raw tool calls remain the underlying contract, but the scripts are now the recommended production path because they handle MCP Streamable HTTP session setup and the `notifications/initialized` handshake automatically.

## Architecture

```
┌─────────────────┐     MCP (Streamable HTTP)     ┌──────────────┐
│  OpenClaw Agent  │ ◄──────────────────────────► │  ALMS Server  │
│  (this machine)  │                              │  (Postgres)   │
└─────────────────┘                               └──────┬───────┘
                                                          │
                                                  ┌───────┴───────┐
                                                  │  PostgreSQL    │
                                                  │  (GIN index)   │
                                                  └───────────────┘
```

## Protocol

### Step 0: Targeted Knowledge Check (Start of major tasks)

Before beginning any major task, search for relevant prior knowledge:

1. Use `python3 scripts/fetch-remote-learnings.py --search-query "<project or topic>"` for targeted discovery, or call `alms__learning-search` directly only if your environment cannot use the helper scripts
2. Read the returned learnings before starting implementation
3. Apply relevant patterns, failures, and edge cases to the current task
4. Treat `protocol` learnings as candidate operating instructions that may require local workflow updates

Use `learning.search` here for targeted discovery, not as the primary fleet sync mechanism. The canonical fleet sync path is `learning.sync` plus `learning.sync_ack`.

**Decide what to ingest locally:**
| Learning Type | Local Action | Notes |
|---------------|-------------|-------|
| `pattern` | ✅ Store to `learnings/` | Useful cross-agent patterns |
| `failure` | ✅ Store to `learnings/` | Valuable failure knowledge |
| `config` | ⚠️ Evaluate first | Apply if relevant to this workspace |
| `protocol` | ⚠️ Evaluate first | Could be SOP update or skill change |
| `edge_case` | ✅ Store to `learnings/` | Edge cases are always useful |

### Step 1: Capture and Push New Learnings to ALMS

At end of every major task, publish any new learnings generated.

**What counts as a "learning":**
- Lessons learned from bugs/mistakes (type: `failure`)
- Patterns discovered (type: `pattern`)
- Configuration decisions (type: `config`)
- Protocols or SOP changes (type: `protocol`)
- Edge cases handled (type: `edge_case`)

**Preferred publish path:** use the helper script to push local markdown learnings with MCP-aware dedup checks:
```bash
export ALMS_URL="http://127.0.0.1:8001/mcp"
export ALMS_AUTH_TOKEN="change-me"
export AGENT_ID="<your-agent-id>"
python3 scripts/push-local-learnings.py          # dry-run
python3 scripts/push-local-learnings.py --apply
```

**Raw tool path:** before storing directly, search ALMS for an existing learning with the same title and source agent. Skip if a result with matching title AND same `src_agent_id` exists.

**Storage command:**
```
alms__learning-store(
  agent_id="<your-agent-id>",
  title="<descriptive title>",
  type="pattern|failure|config|protocol|edge_case",
  tags=["tag1", "tag2", ...],
  body="<detailed body - what happened, root cause, fix/decision>"
)
```

**Bulk push (historical catch-up):**
Scan all files in `learnings/*.md`, extract title/type/body/tags, and push them via `python3 scripts/push-local-learnings.py --apply`.

### Step 2: Sync Remote Learnings (Canonical Pull Path)

**Automated cron (recommended: every 6h):**
1. Call `python3 scripts/fetch-remote-learnings.py --apply`, which performs `learning.sync` and optional `learning.sync_ack`
2. If you are not using the script, call `alms__learning-sync` with:
   - `agent_id`: your agent identifier
   - `since`: timestamp of last sync (stored in cursor file)
3. For each new learning returned, apply the Learning Decision Matrix
4. Call `alms__learning-sync_ack` with processed learning IDs
5. Update cursor

This is the canonical background sync path for fleet-wide learning transfer. Use it for periodic catch-up. Use `learning.search` separately for targeted, task-specific lookups.

**Manual fetch:**
```bash
export ALMS_URL="http://127.0.0.1:8001/mcp"
export ALMS_AUTH_TOKEN="change-me"
export AGENT_ID="<your-agent-id>"
python3 scripts/fetch-remote-learnings.py --apply
```

### Step 3: Cursor Tracking

The cursor tracks what has been ingested to avoid duplicates. Stored at a configurable location (e.g. `learnings/.cursor.json`):

```json
{
  "last_learning_id": "abc123",
  "last_timestamp": "2026-06-09T11:16:32.320777Z",
  "ingested_count": 35,
  "ingested_at": "2026-06-09T13:59:54.522270+00:00"
}
```

Fields:
- `last_learning_id`: backup identifier for last ingested learning
- `last_timestamp`: primary cursor — only learnings with `created_at > last_timestamp` are new
- `ingested_count`: cumulative total
- `ingested_at`: when the cursor was last updated

### Step 4: Scoring & Enrichment (Admin)

Used to maintain ALMS database quality:
- **Input:** Pending/un-scored learnings
- **Action:** Run scoring via model, update entries via `alms__learning-update_enrichment`

The enrichment API is currently implemented as `learning.update_enrichment`. If your prompts or wrappers refer to `score_update`, treat that as a conceptual name, not the current concrete ALMS tool name.

## ALMS MCP Tool Mapping

| Gateway Tool | ALMS Method | Use |
|-------------|------------|-----|
| `alms__learning-store` | `learning.store` | Create learning |
| `alms__learning-search` | `learning.search` | Search learnings |
| `alms__learning-get` | `learning.get` | Get single record |
| `alms__learning-sync` | `learning.sync` | Pull new learnings |
| `alms__learning-sync_ack` | `learning.sync_ack` | Ack processed learnings |
| `alms__learning-delete` | `learning.delete` | Soft-delete learning |
| `alms__learning-update_enrichment` | `learning.update_enrichment` | Score/annotate |
| `alms__health-check` | `health.check` | Server health |

## Integration: PM Orchestrator
- When a PM stage fails, call `learn-store` to record the "fail" pattern.
- Before starting a new PM project, call `learn-search` to find relevant "fail" patterns or research facts from previous projects.

## Integration: Memory Protocol
Updates to `MEMORY.md` should be preceded by a `learn-search` to ensure the agent isn't overwriting global facts with session-local assumptions.

## Scripts (recommended)

Two helper scripts that should be used by agents where possible:

### `scripts/fetch-remote-learnings.py`
Calls ALMS directly by default, performs `learning.sync`, optionally calls `learning.sync_ack`, filters to remote-only, dedups by cursor, and writes new learnings to the local `learnings/` directory.

```
python3 scripts/fetch-remote-learnings.py --apply
python3 scripts/fetch-remote-learnings.py --search-query "postgres" --apply
```

The script is production-ready for direct ALMS calls: it initializes an MCP Streamable HTTP session, sends the required `notifications/initialized` notification, performs `learning.sync`, and acknowledges the full sync batch before advancing the cursor.

Compatibility mode still exists for stdin or file-fed MCP output when needed.

### `scripts/push-local-learnings.py`
Scans `learnings/*.md` files, extracts learnings, performs exact-title dedup lookup through ALMS, and pushes them via `learning.store`.

```
python3 scripts/push-local-learnings.py          # dry-run
python3 scripts/push-local-learnings.py --apply   # actually push
```

Note: the script performs a conservative exact-title check by source agent before publish. ALMS server-side dedup and human review remain the real authority.

This direct publish path was verified against the production ALMS deployment on 2026-06-09.

Environment variable for ALMS connection:
- `ALMS_URL` (defaults to `http://localhost:8001/mcp`)
- `ALMS_AUTH_TOKEN` (optional; sent as `X-ALMS-TOKEN` when set)

## Cron Setup

Recommended cron job for automated sync (every 6 hours):
```yaml
name: alms-learning-sync
schedule:
  kind: cron
  expr: "0 */6 * * *"
sessionTarget: isolated
payload:
  kind: agentTurn
  message: "Run ALMS learning sync: fetch remote learnings, apply decision matrix, ack processed IDs, update cursor."
  model: "deepseek/deepseek-chat"
  lightContext: true
```

## Tags
alms, learning, sync, pipeline, protocol, cross-agent, remote-ingest
