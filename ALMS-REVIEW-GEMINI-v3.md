# ALMS Architecture Plan v3.0 — Critical Review

**Reviewer:** Subagent (Gemini-class analysis, v3 review pass)  
**Date:** 2026-06-05  
**Version Reviewed:** `alms-architecture-plan.md` (v3.0, heavily trimmed from v2)  
**Status:** ✅ **ACCEPT WITH REVISIONS** — Buildable but with 4 concrete changes

---

## Executive Summary

v3.0 is a **dramatic improvement** over v2.0. The scope cut is surgical: no lifecycle management, no state machine, no Go/Python identity crisis. What remains is a clean, minimal MCP server for agent registration + learning transfer. The plan is **buildable today**.

However, the trimming went too far in a few places and left some genuine gaps from v2.0's analysis that were good advice, not scope creep. This review scores each section and flags what must change before Phase 1 begins.

**Bottom line:** Accept the plan. Build it as described. But make 4 targeted revisions (totaling ~2 hours of work) before coding starts.

---

## Section-by-Section Scoring

### Section 1: What ALMS Is — Score: 9/10

**What's right:** The "NOT" list is the single best change from v2. No lifecycle management. No framework. No scheduler. No supervisor. The scope boundary is now crystalline clear.

**What changed from v2:** v2 had a full state machine (BOOTING → RUNNING → DEGRADED → FAILED → STOPPED → UNREACHABLE) and lifecycle tools (`agent.start`, `agent.stop`, `agent.restart`) with systemctl wrappers. v3 removes all of it. This is the correct call — MCP is for learning transfer, not for process control.

**What's missing:** The plan says "no lifecycle management" but doesn't say what happens if ALMS itself crashes or needs a restart. Agents register into ALMS's PostgreSQL. If ALMS is down, agents can't sync learnings. That's fine — but the plan should state it explicitly: **"ALMS being down means agents operate offline and re-sync when ALMS is back."** Minor documentation gap, not a code gap.

**Verdict:** Keep as-is. Add one sentence about offline agent behavior.

---

### Section 2: Architecture Diagram — Score: 10/10

**What's right:** Clean, accurate, no distractions. The diagram clearly shows:
- ALMS as the center
- PostgreSQL as the persistence layer
- Managed agents (data machine) and remote agents (Mac/OpenClaw) as peers
- MCP as the only communication protocol

**What changed from v2:** v2's diagram was busier with lifecycle annotations. v3 strips it to the essentials. Better.

**Verdict:** Keep as-is. No changes needed.

---

### Section 3: Module Structure — Score: 9/10

**What's right:** Flat, predictable Python project layout. `services/`, `models/`, `db/`, `mcp/`, `deploy/`, `tests/` — standard FastAPI structure, instantly navigable. The naming is consistent and descriptive.

**What changed from v2:** v2 had `runners/systemd.py`, `runners/mcp.py`, `mcp/ws_handlers.py`, `services/scheduler.py` — all lifecycle artifacts. v3 removes them cleanly.

**What's missing:**
1. **No `middleware/` or auth layer.** If auth is a single middleware function, it belongs in `server.py` or a lightweight `middleware.py`. The plan doesn't mention auth at all — a regression from v2's review that flagged it as critical. **This must be added.**
2. **No `scripts/` or `Makefile`.** Where does `scp` + `ssh` deployment logic live? v2 had a deployment script. v3's module tree has `deploy/` with only a systemd unit. The deployment workflow (build → scp → enable → start) should be scripted, not manual. Add a root `Makefile` or `scripts/deploy.sh`.
3. **No `docker-compose.yml` or DevContainer for PostgreSQL.** For local development, you need PostgreSQL accessible. The plan assumes the data machine. Developers (including future-you from 3 months after a machine rebuild) need a faster local setup. Add a `docker-compose.yml` with PostgreSQL-only.

**Verdict:** Keep structure, add `server/middleware.py` for auth, add `Makefile` + `docker-compose.yml`.

---

### Section 4: Data Model — Score: 7/10

**What's right:** AgentSpec and LearningRecord are well-designed. AgentSpec has the right minimal fields (agent_id, display_name, capabilities, metadata, learning_sync cursor, heartbeats). LearningRecord correctly mirrors IS-043's structure.

**What changed from v2:** v2 had AgentState (separate from AgentSpec), LifecycleEvent, more fields on everything. v3 collapses to just AgentSpec with a `learning_sync` sub-object. Clean.

**What's wrong — 3 concrete gaps:**

#### Gap 1: No ProtocolRecord schema
Section 5 has `protocol.pull` and `protocol.push` as MCP tools, but Section 4 never defines the Protocol data structure. What fields does a protocol have? `protocol_id`, `title`, `body`, `version`, `agent_id`? Is it just a LearningRecord with `is_mandatory=true` and `type="protocol"`, or a separate entity? The learning.store tool can create mandatory learnings via `is_mandatory: true`, but `protocol.push` is a separate tool — so they're different entities. **Define the Protocol struct in Section 4.**

**Fix:** Add a Protocol model:
```json
{
  "protocol_id": "SOP-001",
  "title": "Always validate newsletter output",
  "body": "Before publishing...",
  "version": 1,
  "author": "agent:admin",
  "created_at": "2026-06-05T17:00:00Z",
  "updated_at": null
}
```

#### Gap 2: sync cursor semantics
`learning_sync.last_learning_id` stores a scalar ID. If Agent A syncs at ID=8, then Agent B pushes LRN-010 (skipping 9), Agent A calls sync(since_id=8) — does it get LRN-010? Or only records with IDs > 8? The plan says "new learnings since last_id for this agent" but doesn't specify whether IDs are monotonically increasing and gap-free (they won't be if deletions/votes reorder things). **Use `created_at` or a sequential vector clock, not a linear ID.**

**Fix:** Change `since_id` to `since_timestamp` (ISO 8601). This avoids ID-ordering problems entirely. Or use a ULID/timestamp-embedded ID scheme.

#### Gap 3: `agent.list` filter semantics
The tool signature says `agent.list(filter?)` but doesn't define what filters exist. Common filters: `by_capability` (agents with `web_search`), `by_type` (only systemd agents), `by_health` (only agents with heartbeat in last 5 min). **Specify the filter interface** — even if Phase 1 implements only `filter={"health": "alive"}`.

**Verdict:** Add ProtocolRecord definition, fix sync cursor to be timestamp-based, specify agent.list filter fields. These are ~30 minutes of documentation work.

---

### Section 5: MCP Surface — Score: 8/10

**What's right:** Clean separation of resources (read-only) from tools (write). 6 resources + 18 tools is the right surface area. The sync flow (`learning.sync` → process → `learning.sync_ack`) is the most important MCP pattern and it's well-specified. No extraneous endpoints from v2's lifecycle layer.

**What changed from v2:** Removed `agent.start`, `agent.stop`, `agent.restart`, `agent.get_logs`, `health.check_agent`, `agent.batch` — all lifecycle tools that belonged to the old state-machine approach. Good.

**What's missing:**

1. **No `health.check`** — listed in v3's own table but crossed a relationship with v2's health endpoints. Actually it IS in the table (`health.check` returns `HealthReport`). This is fine.

2. **`agent.update` tool** — critical gap. v3 has `agent.register`, `agent.unregister`, `agent.list`, `agent.heartbeat`. No way to update agent metadata (capabilities, display_name, metadata) after registration. Agents grow new tools over time. **Without `agent.update`, an agent would have to unregister and re-register to add a tool, losing its sync cursor and heartbeat history.**

3. **`learning.vote`** — v3 keeps this from v2. I agree with the v2 review: this is unnecessary for a single-user system. Remove it or relegate to Phase 2+.

4. **No pagination on `learning.search`** — `agent.list` and `learning.search` have no `limit`/`offset` params. With 10,000 learnings, a search that matches 500 results returns 500 records in one MCP response. Add `limit` (default 50) and `offset` params.

**Verdict:** Add `agent.update` tool, remove `learning.vote`, add pagination to list/search endpoints. These are small changes.

---

### Section 6: Implementation (Sync Flow + Phases) — Score: 10/10

**What's right:** The learning sync flow diagram is concrete and correct. Three actors (Agent A, ALMS, Other Agents) with call-by-call trace ends at confirmed cursor state. This is the core value proposition of ALMS and it's specified unambiguously.

**What changed from v2:** v2's "Sync Flow" was mixed with lifecycle transitions. v3 focuses purely on learning sync. Better.

**Phases:** Phase 1 (Week 1) has 6 build items — realistic for one developer. Phase 2 (Week 2) adds learning store/tools. The split is clear and independent: Phase 1 = registry + sync, Phase 2 = learning store + protocols.

**Concern:** Phase 2 depends on Phase 1 (PostgreSQL + MCP server must exist), but Phase 1's work (registry, sync) doesn't depend on Phase 2's (learning store, protocols). This is a correct dependency graph.

**Verdict:** Keep as-is. Build phases as specified.

---

### Section 7: Tech Stack — Score: 8/10

**What's right:** Python + FastAPI + uvicorn. PostgreSQL 16 + asyncpg. systemd deployment. YAML config. Clear, correct for this scope.

**What changed from v2:** v2's plan was internally contradictory (Go in prose, Python in implementation). v3 is **unambiguously Python**. This single decision eliminates the largest blocker from v2 reviews. Well done.

**What's missing:**

1. **No testing framework specified.** pytest? Unittest? The module tree has `tests/` but the tech stack doesn't list a test framework or coverage tool. Add `pytest` + `pytest-asyncio` + `httpx` (for async test client) + `coverage`.

2. **No linting/formatting tools.** ruff? myPy? black? For a solo project this is optional but the plan's tech stack should include at least ruff for Python consistency.

3. **No migration tool.** The plan's Phase 1 says "PostgreSQL setup on data machine" but doesn't specify how schema changes are versioned. v2's review recommended `golang-migrate/migrate` (Go) or `alembic` (Python). Since this is Python, add **alembic** for schema migrations. Raw SQL files work for Phase 1 but bite you by Phase 3.

**Minor corrections:**
- asyncpg should be `asyncpg` (package name), not a description
- Add `httpx` for testing (FastAPI `TestClient` is synchronous; async MCP handlers need `httpx.AsyncClient`)

**Verdict:** Add alembic, pytest + pytest-asyncio, httpx, and ruff to the tech stack table.

---

### Section 8: Summary — Score: 10/10

**What's right:** Short, definitive, no ambiguity. Three bullet points covering exactly what ALMS does. "No lifecycle management" stated twice (in prose and as the first summary bullet). The restatement of what ALMS isn't is a good close.

**Verdict:** Keep as-is.

---

## Cross-Cutting Concerns (Not Section-Specific)

### 🔴 Auth: Still Missing
v2's review flagged this as critical. v3 omitted auth entirely — not even mentioned as a future concern. For a server on `192.168.2.112:8001` that stores agent metadata and potentially sensitive learnings:

- Any device on the LAN can call `agent.unregister("newsletter-scout")` and wipe the registry
- Any device can call `learning.search()` and read all learnings
- Any device can call `protocol.push()` and inject malicious SOPs

**The fix:** Add a static bearer token (see `middleware.py` suggestion above). 30 lines of code. `X-ALMS-TOKEN` header. Configurable via `alms.yaml`. Worth it in Phase 1, not Phase 2.

### 🟡 OpenClaw MCP Transport
v2's review raised this as a Phase 0 blocker: "Verify OpenClaw supports Streamable HTTP MCP servers." v3 doesn't address it. ALMS is built as Streamable HTTP MCP. If OpenClaw only supports stdio MCP, you need a bridge.

**Recommendation:** Verify this before Phase 1 coding starts. If OpenClaw supports HTTP, proceed. If not, build a 50-line stdio↔HTTP proxy as a local launchd service on the Mac.

### 🟢 IS-043 Integration
v3 references IS-043 for the LearningRecord model but defers integration. Since ALMS is the implementation of IS-043's learning store, and the LearningRecord in Section 4 mirrors IS-043's schema, there's no deferred integration — the learning store IS the implementation. The plan just doesn't say this explicitly.

**Fix:** Change "IS-043 defined" status to "IS-043 implemented as LearningRecord" in Section 1's table.

### 🟢 Deployment Script
v2's review noted the deploy script does `ssh data` for setup. v3 has `deploy/alms.service` but no deploy script. For a solo developer who knows the machine, this is fine — but add a simple `deploy/deploy.sh` that does: build → scp alms/ → ssh data 'systemctl daemon-reload && systemctl restart alms' . This prevents forgetfulness during the 3rd deploy.

---

## v2 → v3 Trimming Assessment: Right Cuts, Wrong Omissions

### Good Cuts (v3 correctly removed)

| Removed from v2 | Why it was right to cut |
|---|---|
| State machine (8 states) | ALMS is a learning store, not a lifecycle manager |
| `agent.start/stop/restart` | No systemd control = no lifecycle tools |
| `agent.get_logs` (journalctl) | Not ALMS's job; agents manage their own logs |
| `agent.batch` | Unnecessary without lifecycle tools |
| `health.check_agent` | Folded into `agent.heartbeat` implicitly |
| Go architecture (cmd/, internal/) | V3 commits to Python, ends the identity crisis |
| systemd unit template for agents | No lifecycle management → no agent unit templates |
| Scheduler / cron integration | Out of scope |
| Leader election / per-agent mutex | No systemctl operations → no races |
| D-Bus / go-systemd | Go-specific, irrelevant to Python/Python |
| golang-migrate/migrate | Replace with alembic for Python |

### Wrong Omissions (v3 trimmed too far)

| Missing from v3 | Why it should be restored or added |
|---|---|
| **Auth (bearer token)** | v2 flagged as critical; v3 has zero auth discussion |
| **ProtocolRecord definition** | `protocol.push/pull` tools exist but Protocol struct is undefined |
| **`agent.update` tool** | No way to update agent metadata after registration |
| **Migration tooling** | Schema version tracking (alembic) |
| **Deployment script** | Manual `scp` + `ssh` workflow is error-prone |
| **Pagination for list/search** | Large result sets break MCP response limits |
| **Timeout/rate-limit discussion** | 7GB RAM, no protection against spam (minor concern) |
| **OpenClaw transport verification** | v2 raised as Phase 0 blocker; v3 doesn't address it |

### Verdict on Trimming
The trimming was **80% correct** — removing the state machine and lifecycle tools was the right call. The 20% over-trimming removes genuinely useful things (auth, update tool, ProtocolRecord) that aren't scope creep. v3 should be trimmed back from v2's 40-page plan but not this far. Add back the 4 critical items above.

---

## Updated Recommendations

### Must-Fix Before Phase 1 (4 changes, ~2 hours total)

1. **Add static bearer token auth.** Middleware function checking `X-ALMS-TOKEN` header. Config field in `alms.yaml`. 30 lines of code.

2. **Define ProtocolRecord in Section 4.** Copy the LearningRecord pattern with `protocol_id`, `title`, `body`, `version`, `author`, timestamps. Resolves the dangling reference from Section 5.

3. **Add `agent.update` tool.** Allow agents to update capabilities, metadata, display_name. Without this, agents that grow new tools must unregister/re-register.

4. **Add alembic to the tech stack.** Schema migration management. `alembic init migrations`, then `alembic revision --autogenerate` after schema changes. Prevents Phase 2 schema drift.

### Strongly Recommended Before Phase 2

5. **Verify OpenClaw MCP transport.** If OpenClaw only supports stdio, build a local proxy before Phase 2.

6. **Add pagination (`limit`/`offset`) to `learning.search` and `agent.list`.** Default limit 50.

7. **Replace sync cursor with timestamp-based sync** (`since_timestamp` instead of `since_id`). Avoids ID-ordering bugs.

8. **Remove `learning.vote`.** Unnecessary for a single-user system.

9. **Add `deploy/deploy.sh`.** scp + ssh reload script.

10. **Add `docker-compose.yml`** with PostgreSQL for local development.

---

## Final Scorecard

| Section | Score | Verdict |
|---------|-------|---------|
| §1: What ALMS Is | 9/10 | Keep, add offline agent sentence |
| §2: Architecture Diagram | 10/10 | Keep as-is |
| §3: Module Structure | 9/10 | Add middleware, Makefile, docker-compose |
| §4: Data Model | 7/10 | Define ProtocolRecord, fix sync cursor, specify filters |
| §5: MCP Surface | 8/10 | Add agent.update, remove learning.vote, add pagination |
| §6: Implementation | 10/10 | Keep as-is |
| §7: Tech Stack | 8/10 | Add alembic, pytest, httpx, ruff |
| §8: Summary | 10/10 | Keep as-is |
| **Overall** | **8.9/10** | **ACCEPT WITH REVISIONS** |

**Overall verdict: ACCEPT WITH REVISIONS.**

The plan is buildable. The v2→v3 trim was mostly correct. Fix the 4 must-fix items (auth, ProtocolRecord, agent.update, alembic) and the 4 strong-recommends (transport verify, pagination, timestamp sync, remove vote), and Phase 1 is ready to code. Do not chase v2's state machine or lifecycle features — v3's scope is the right scope.

---

## Appendix: Quick Sanity Check

| Test | Pass/Fail |
|------|-----------|
| Can a new agent register, sync learnings, and ack? | ✅ Yes (sync flow in §6) |
| Can an agent push a mandatory protocol? | ✅ Yes (`protocol.push` in §5) |
| Does the plan fit on 7GB RAM with 6 agents? | ✅ Yes (~2GB total) |
| Can you deploy to Data Machine via systemd? | ✅ Yes (`deploy/alms.service` exists) |
| Is the scope clearly bounded? | ✅ Yes (3 explicit NOTs in §1) |
| Is there a data model for every entity named in the MCP surface? | ❌ No (ProtocolRecord missing) |
| Can agents update their own metadata? | ❌ No (`agent.update` missing) |
| Is there ANY security mechanism? | ❌ No (auth completely absent) |
| Can you run this locally without Data Machine? | ❌ No (no docker-compose or pg setup) |

4/9 fails are from the 4 must-fix items above. Fix those and this plan is ready for implementation.
