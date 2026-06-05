# ALMS Architecture Plan v3.0 — Critical Review

**Reviewer:** DeepSeek (subagent)  
**Date:** 2026-06-05  
**Version Reviewed:** v3.0 (current plan)  
**Status:** ⚠️ **REVISE — Close but not build-ready**

---

## Executive Summary

v3.0 has **fixed the fundamental problems** that made v2.0 unbuildable. The Go/Python contradiction is gone — the architecture is unambiguously Python/FastAPI. The state machine (lifecycle management) is deleted. The scope is sharply focused: registry + learning store + MCP interface. No lifecycle control, no process supervision.

This is a **massively improved plan**. The hard structural problems are solved.

**Remaining issues are refinements, not blockers:** 4 critical gaps in the learning sync design, 3 architecture robustness gaps, and several missing specifications that will cause implementation friction. Fix these and this plan is ready to build.

---

## 1. Architecture — 9/10 (Clean, Focused, Consistent)

### What's Excellent

**No identity crisis.** The plan describes one architecture, consistently Python/FastAPI, PostgreSQL, Streamable HTTP MCP. The v2.0 reviewers both flagged the Go/Python contradiction as blocking. v3.0 cleanly resolves this. This is the single biggest improvement.

**Tight scope.** "No lifecycle management" is stated upfront and reinforced throughout. Every tool and resource in the MCP surface is registration or learning — no `agent.start`, no `agent.stop`, no `agent.restart`. This is exactly right. ALMS is a passive store, not an active controller.

**The agent diagram is honest.** Self-registering agents on data machine (systemd/cron) and remote (Mac/MCP client). No pretense that ALMS controls them. This matches reality.

**Module layout (Section 3) is clean and conventional.** `models/`, `services/`, `mcp/`, `db/`, `tests/` — standard FastAPI project structure. Easy for any Python developer to navigate.

### One Architecture Concern

**The MCP transport choice (Streamable HTTP) needs validation.** The plan assumes agents can connect via HTTP MCP. In practice:
- OpenClaw MCP servers typically use `stdio` transport
- The `modelcontextprotocol/python-sdk` Streamable HTTP support may not be mature
- Remote agents (Qwen CLI, OpenClaw on Mac) need HTTP accessibility, which means ALMS must be reachable from the Mac — either on `0.0.0.0:8001` or via SSH tunnel

**Recommendation:** Add a note in Section 6 (or a new "Transport" subsection under Implementation) specifying:
1. Whether ALMS binds to `0.0.0.0` or `127.0.0.1`
2. How remote agents reach ALMS (SSH tunnel? Tailscale? VPN?)
3. Whether `mcp` python-sdk's Streamable HTTP server runs as a uvicorn ASGI app or wraps FastAPI

This isn't blocking, but it will be day-one friction when setting up the first remote agent connection.

---

## 2. Data Model — 8/10 (Solid foundation, 4 missing fields)

### What's Right

**AgentSpec (Section 4) is well-designed.** The separation of static spec (`capabilities`, `metadata`) from sync state (`learning_sync`) from observed state (`health`) is correct. No lifecycle artifacts left over from v2.0.

**LearningRecord mirrors IS-043 accurately.** The field set (`learning_id`, `type`, `title`, `body`, `tags`, `severity`, `author`, `score`, `is_mandatory`, `ttl_days`, `created_at`) is complete and well-typed. This is the core schema and it's right.

### What's Missing

**1. `src_agent_id`** — Every learning should track which agent created it. Currently `author` is a free-form string like `"agent:newsletter-scout"`. Make this an explicit `src_agent_id` foreign key with a `display_author` field for human-readable names. This enables:
- `learning.sync(agent_id="X")` to filter out learnings that agent X itself created
- Analytics: "which agent contributes the most valuable learnings"
- Audit trail: "who pushed this protocol"

**2. `resolution` field on LearningRecord** — Learnings are not static. An edge case found in June may be fixed in July. Add:
```json
{
  "resolution": "open",         // "open" | "resolved" | "superseded"
  "superseded_by": null,        // learning_id if superseded
  "resolved_at": null
}
```
Without this, there's no way to mark a learning as outdated. GC by TTL alone is insufficient — you might *want* to keep a resolved learning for reference, just not show it in sync results.

**3. LearningRecord needs `ai_generated: bool`** — Distinguish human-authored protocols from auto-captured agent learnings. This matters for:
- Priority sorting (human > agent)
- Display in UIs (protocols vs discoveries)
- Trust weighting in scoring

**4. No Protocol schema defined** — The plan introduces `protocol.push` and `protocol.list` tools and an `alms://protocols` resource, and mentions "Protocol[]" as an output type, but there's no JSON schema for `Protocol` anywhere. This is a specification gap. Define it:

```json
{
  "protocol_id": "PRO-001",
  "title": "Email disclaimer required",
  "body": "Every client-facing email must include...",
  "target_agents": ["newsletter-scout", "newsletter-writer"],
  "target_tags": ["newsletter", "client-facing"],
  "version": 3,
  "author": "admin:ghassan",
  "is_active": true,
  "created_at": "2026-06-05T17:00:00Z"
}
```

Key design decision: is `target_agents` an explicit list or a tag-based filter? Both have tradeoffs. Explicit lists scale poorly (N agents = N entries per protocol), but tag-based filtering can miss agents. **Recommendation:** Use tag-based targeting (target_tags) with an optional explicit allowlist (target_agents). Agents call `protocol.pull(agent_id)` and ALMS matches their tags + explicit allowlist.

**Score: 8/10** — Core fields are right. The 4 missing items are implementation refinements, not architecture flaws.

---

## 3. MCP Surface — 8/10 (Well-scoped, but protocol.pull is underspecified)

### What's Right

**18 tools is appropriate.** For a registry + learning store, this is comprehensive without bloat. `learning.store`, `learning.sync`, `learning.sync_ack`, `protocol.pull` are the core value proposition. Everything else is support.

**Resource URIs are clean and conventional.** `alms://agents`, `alms://learnings`, `alms://protocols`, `alms://health` — readable, hierarchical. No confusion about what's a resource vs a tool.

**No lifecycle tools.** This is discipline. v2.0 had `agent.start`/`agent.stop`/`agent.restart` which would have created a split-brain with systemd. v3.0 cleanly removes them.

### The Critical Problem: protocol.pull Semantics

**`protocol.pull(agent_id)` returns "all mandatory protocols"** — but this is underspecified in 3 ways:

1. **Agent ID vs agent role.** If `newsletter-scout` and `newsletter-writer` both call `protocol.pull`, do they get different protocols? The plan says protocols are "Mandatory SOPs," implying they apply broadly. But if *all* agents get *all* protocols, the field grows linearly with no filtering. At 50 agents × 10 protocols each, a `protocol.pull` returns 500 records — most irrelevant.

2. **Already-acknowledged protocols?** Does `protocol.pull` always return everything (as the plan states)? If so, agents re-process the same SOPs every sync, which is wasteful. Consider: `protocol.pull(agent_id)` returns only protocols that have been **updated** or **added** since the agent's last ack (tracked similarly to `learning_sync` but on protocol-level).

3. **Idempotency.** An agent that calls `protocol.pull` twice in a row (due to network retry) should get the same result. The plan needs to specify idempotency guarantees.

**Recommendation:** Add a `protocol.pull_since` field — `protocol.pull(agent_id, since_id?)` — that returns only protocols changed/added since a cursor. Agents call `protocol.pull(agent_id)` unconditionally on first boot (or if `since_id` is missing), and `protocol.pull(agent_id, since_id=<last_protocol_id>)` on subsequent syncs.

### Minor Surface Issues

**`learning.vote` is preemptive.** On a single-user setup with 3 agents, who's voting? Track usage frequency instead (most-pulled learnings get higher score). Votes make sense at 10+ agents or multi-user. Add it in Phase 2, not Phase 1.

**`learning.get` is redundant with resource `alms://learnings/{id}`.** MCP resources serve read-only data. `learning.get` is a tool that returns a single record — identical functionality. Pick one. **Recommendation:** Make `alms://learnings/{id}` the canonical read path, remove `learning.get`. Keep `learning.get` only if it adds query-time dedup or formatting logic that resources can't express.

**`health.check` as a tool is fine, but add a resource.** `alms://health` should exist as a resource (shows current state), while `health.check` as a tool could trigger a fresh probe (force-check PG connectivity, verify systemd status). Use the tool for on-demand deep checks, the resource for passive display.

---

## 4. Learning Sync Design — 6/10 (Functional, but 4 critical gaps)

### The Flow (Section 6) is Correct
```
Agent syncs → gets new learnings → processes → acks → cursor advances
```

This is the right pattern. The cursor-based sync (`since_id`) is appropriate for a learning store — it's simple, robust, and doesn't require timestamps or vector clocks.

### But the Design Has 4 Gaps

#### Gap 1: What learnings does sync return?

`learning.sync(agent_id, since_id)` returns learnings created **after** `since_id`. But does it filter by relevance to this agent?

- Option A: All learnings after `since_id` regardless of agent → every agent gets everything. Simple but noisy. A film-pipeline agent doesn't need newsletter scraping learnings.
- Option B: Learnings filtered by tag/type matching the agent's capabilities → relevant only. Requires a matching algorithm.
- Option C: Learnings after `since_id`, with optional tag filter → agent controls what it pulls.

**The plan describes Option A (no filter).** This works for 3 agents, breaks at 10+. **Recommendation:** Implement Option C — `learning.sync(agent_id, since_id, tags?: [], types?: [])`. The agent specifies what it's interested in. ALMS defaults to "all" if no filter is given (backward compat).

#### Gap 2: Mandatory learnings and sync interaction

If `is_mandatory: true` on a learning, does it appear in `learning.sync` results or only in `protocol.pull`? The plan has `protocol.pull` for mandatory SOPs and `learning.sync` for regular learnings — but `LearningRecord` has an `is_mandatory` field. Can a regular learning be mandatory?

**Clarification needed:** Is `is_mandatory` on `LearningRecord` a holdover from the IS-043 schema, or does it mean "this learning should be force-injected into all agents"? If the latter, define how `sync` vs `protocol.pull` handles it.

**Recommendation:** Remove `is_mandatory` from `LearningRecord` — it's confusingly named. Rename to `is_pinned` (survives GC) on learning records. Mandatory SOPs live in the `protocols` table, not `learnings`. This cleanly separates "things agents discovered" from "things admin decrees."

#### Gap 3: No dedup strategy for sync_ack

Agent calls `sync_ack(agent_id, last_processed_id=8)`. What if:
- Agent processes LRN-006, LRN-007, LRN-008 successfully and acks `8` → ALMS marks agent as synced through 8.
- Agent processes LRN-006 successfully, crashes before LRN-007, recovers, calls `sync(agent_id, since_id=8)` → sees no new learnings (because 8 is the cursor, not 6). It **skips** LRN-007 and LRN-008.

This is a correctness bug. The cursor design assumes linear, in-order processing with no failures. Real agents will fail mid-batch.

**Fix:** `sync_ack(agent_id, last_processed_id=8)` should validate that the agent has acknowledged *all* learnings between its last ack and `8`. If it hasn't (gap detected), either:
- Reject the ack (agent must re-process the gap)
- Allow the ack but log a warning (optimistic — agent may have processed the gap via another path)

**Recommended approach:** `sync_ack` acknowledges exactly one learning at a time: `sync_ack(agent_id, learning_id_acked: "LRN-007")`. ALMS tracks which specific learnings each agent has processed (via a join table). This handles gaps naturally — an agent can process LRN-006, skip LRN-007 (because it was already known via another channel), and ack LRN-008.

**Tradeoff:** Single-learning acks increase MCP call volume (N calls for N learnings instead of 1). Mitigation: batch acks are fine, but create a secondary index `(agent_id, learning_id)` in a `learning_acknowledgements` table and validate no gaps in the batch.

#### Gap 4: Learning deletion + sync inconsistency

Admin deletes LRN-005 (bad data, duplicate, superseded). Agent B last synced at `since_id=4`. Now when Agent B calls `learning.sync(since_id=4)`, ALMS returns learnings 6, 7, 8 — but the sequence has a **hole** at 5. Agent B may interpret this as a data loss bug.

**Fix:** Don't hard-delete learnings. Soft-delete them (`is_deleted: true`, `deleted_at`). `learning.sync` skips deleted records but the IDs remain in the sequence. Or use timestamps instead of ID cursors. **Recommendation:** Use monotonic timestamps (`created_at` with microsecond precision) instead of integer IDs for the sync cursor. SQL `WHERE created_at > $since_ts AND NOT is_deleted`. This is more robust to deletion.

---

## 5. PostgreSQL Schema — 6/10 (Declared but not specified)

Section 3 shows `db/schema.sql` in the module structure, but there's **no SQL in the plan**. The data model is JSON-only. This means:

**Undefined schema details:**
- Primary key strategy (UUID v4 vs `ULID` vs auto-increment)?
- Index strategy (`(agent_id, created_at)` for sync, GIN for full-text search on learnings)?
- Storing `LearningRecord.body` — full-text search suggests `tsvector` column, but not specified
- How `learning.sync` finds "learnings since last cursor" efficiently — needs composite index on `(created_at, learning_id)`
- `ToolSpec.availability` and `latency_ms` — are these stored or computed?

**Gap:** The data model in Section 4 defines the API payloads, but the storage model in `db/schema.sql` will make or break the performance of `learning.sync` and `learning.search`. This should be at least partially specified, especially the indexing strategy for the two critical queries:

```sql
-- sync query: find new learnings since cursor
CREATE INDEX idx_learnings_created_at ON learnings (created_at DESC);

-- search query: full-text search
ALTER TABLE learnings ADD COLUMN search_vector tsvector
  GENERATED ALWAYS AS (to_tsvector('english', title || ' ' || body)) STORED;
CREATE INDEX idx_learnings_search ON learnings USING GIN (search_vector);
```

**Recommendation:** Add a "Schema Decisions" subsection to Section 4 (Data Model) documenting:
- Primary key format (UUID v4 recommended for distributed agent pushes)
- Index on `learnings.learning_id` for cursor-based sync
- Full-text search via `tsvector` or pgvector (if semantic search needed)
- `learning_acknowledgements` join table design for gap-safe sync_ack

---

## 6. Tech Stack & Dependencies — 8/10 (Sound choices, 2 concerns)

### What's Right

| Layer | Choice | Verdict |
|-------|--------|---------|
| Server | FastAPI + uvicorn | Best-in-class Python HTTP framework |
| MCP | modelcontextprotocol/python-sdk | Only option for Python MCP; well-maintained |
| Database | PostgreSQL 16 | Correct for the query patterns (full-text search, relational, no conflicts) |
| DB driver | asyncpg | Best async PG driver for Python |
| Deploy | systemd | Right for same-machine, no orchestration |

### Concerns

**1. Streamable HTTP transport maturity.** The `modelcontextprotocol/python-sdk` added Streamable HTTP support relatively recently. Its reliability at the scale of 3-10 concurrent agents needs verification. **Risk:** If Streamable HTTP has bugs or performance issues, you may need to fall back to SSE (Server-Sent Events) transport, which changes the deployment topology (persistent connections instead of request-response).

**Mitigation:** Add a transport compatibility note. If both ALMS and agents run on the same machine, `stdio` MCP via subprocess is simpler and more reliable. Reserve Streamable HTTP for remote agents.

**2. No testing strategy.** Section 3 has a `tests/` directory but Section 6 (Implementation) never mentions test approach, CI, or coverage expectations. For a system that stores cross-agent knowledge — where a bug could silently corrupt learnings or lose sync state — tests are not optional.

**Minimum test plan for Phase 1:**
- **Unit tests:** `learning.sync` cursor handling, `sync_ack` gap detection, dedup logic
- **Integration tests:** Against a real PostgreSQL (testcontainer or ephemeral). Verify `agent.register` + `learning.store` + `learning.sync` round-trips in 50ms.
- **No E2E** — acceptable for Phase 1

---

## 7. Deployment & Operations — 7/10 (Solid but undetailed)

### What's Right

**systemd unit for ALMS itself.** `BindsTo=postgresql.service` in the unit ensures correct dependency ordering. `Restart=on-failure` handles crashes.

**SCP binary.** Since it's Python, "binary" is misleading (it's a venv + source), but SCP is the right deploy mechanism for a single machine.

### What's Missing

**1. Start order and readiness.** ALMS starts → connects PostgreSQL → runs migrations → serves MCP. But systemd `Type=simple` marks the unit as "active (running)" the moment the uvicorn process starts, **before** the DB connection succeeds or migrations run. If PG is down, ALMS will accept MCP connections but fail on every request.

**Recommendation:** Use `Type=notify` if the service library supports it, or add a startup health endpoint (`/health/startup`) that returns 200 only after PG connection + migrations. Configure systemd to use `ExecStartPost=/usr/bin/curl -sf http://127.0.0.1:8001/health/startup`.

**2. Log rotation.** 10 agents on journald, each producing log output, will fill disk. Add a note about `journalctl --vacuum-size=1G` or `SystemMaxUse=1G` in `journald.conf`.

**3. Config reload.** The plan has YAML config (`alms.yaml`). Changing config (e.g., PostgreSQL DSN, auth token) currently requires a restart. For a server that's meant to be always-on, SIGHUP config reload is cheap and useful. This is Phase 2 work, but acknowledge it.

**4. The `deploy/alms.service` file is not included.** The module tree references it but the document doesn't contain it. For a plan that describes systemd deployment, including the actual unit file would be valuable for review. What `User=` runs it? What's `WorkingDirectory=`? Is there an `EnvironmentFile=` for the PostgreSQL DSN?

---

## 8. Security — 6/10 (Plan says nothing)

The plan has **zero security considerations**. No auth, no secrets management, no network access control, no mention of PostgreSQL `pg_hba.conf`.

**This is acceptable for a home-lab MVP**, but only if acknowledged. The plan should state:

> **Security Notice (Phase 1):** ALMS runs on a LAN, bound to `127.0.0.1:8001` on the data machine, with no authentication. Agents connect from the same machine or via SSH tunnel. This is acceptable for a single-user setup with no external access. Authentication (bearer token in X-ALMS-TOKEN header) and PostgreSQL security hardening will be added in Phase 2.

Without this notice, the plan implies security was not considered, which undermines confidence.

**Immediate risk:** If ALMS binds to `0.0.0.0` (so remote Mac agents can reach it), any device on the LAN can:
- List all agents (`agent.list`)
- Delete learnings (`learning.delete`)
- Create fake learnings (`learning.store`)

PostgreSQL on the data machine defaults to listening on `0.0.0.0:5432` in Ubuntu. If ALMS connects using a DSN with a password over TCP, any LAN device that discovers the port can attempt brute force.

**Recommendation:** Add a "Security Considerations" subsection to Section 7 (or a new Section 9). At minimum:
- Bind ALMS to `127.0.0.1` by default; expose via SSH tunnel for remote agents
- Lock PostgreSQL to `127.0.0.1` in `pg_hba.conf`
- Create a dedicated `alma` PostgreSQL role with per-table permissions
- Add a config flag for `auth_token` for when external access is needed

---

## 9. Gaps & Risks

### 🔴 HIGH: sync gap vulnerability (no gap handling)
*Already detailed in Section 4 (Gap 3 above). This is the single most dangerous implementation detail.* Without gap-safe ack logic, agent crashes mid-sync will cause permanent learning loss. **Must be fixed before Phase 1 ships.**

### 🔴 HIGH: No test strategy for sync correctness
The sync flow is the core value proposition. A bug here means agents miss critical learnings or consume duplicate data. The plan should specify a test: "register 2 agents, push 3 learnings, sync agent A, ack, delete learning 2, sync agent B — verify agent B sees only learnings 1 and 3 and doesn't crash on the gap."

### 🟡 MEDIUM: Protocol schema not defined
`protocol.push` and `protocol.list` are referenced but the Protocol struct is never specified. This is a specification gap — implementers will guess and get it wrong.

### 🟡 MEDIUM: No filtering in learning.sync
As agents grow (5, 10, 20+), returning *all* learnings since cursor is increasingly noisy. A film-pipeline agent doesn't need 200 newsletter scraping learnings. Add tag/type filtering parameters now (optional, backward-compatible default = all).

### 🟡 MEDIUM: Soft-delete strategy unspecified
`learning.delete` exists but the plan doesn't say whether it's soft or hard delete. Hard-delete creates gaps in sync sequences (Discussed in Gap 4). Soft-delete adds complexity (filtering in every query). **Pick one and document it.**

### 🟢 LOW: Duplicate `learning.get` tool
Redundant with resource `alms://learnings/{id}`. Either remove the tool or give it a distinct purpose (rich formatting, resolved superseded links).

### 🟢 LOW: No agent.update in Phase 1
`agent.register` creates an agent at first heartbeat, but what if capabilities change? Currently you must unregister and re-register. Add `agent.update(agent_id, fields)` — it's a trivial endpoint and prevents stale capability data.

### 🟢 LOW: `learning.vote` preemptive
Single-user setup doesn't need voting. Remove from Phase 1, add usage-tracking scoring instead.

---

## Section-by-Section Scoring

| Section | Score | Summary |
|---------|-------|---------|
| 1. What ALMS Is | **9** | Clear, honest, no scope creep |
| 2. Architecture | **9** | Clean diagram, correct distinction of local vs remote agents |
| 3. Module Structure | **8** | Standard and navigable; missing test detail |
| 4. Data Model | **8** | Core fields right; 4 missing fields (src_agent_id, resolution, ai_generated, Protocol schema) |
| 5. MCP Surface | **8** | Well-scoped; `protocol.pull` underspecified; `learning.get` redundant |
| 6. Implementation | **6** | Sync flow correct but has 4 critical gaps (no filter, mandatory/protocol overlap, gap-safe ack, soft-delete) |
| 7. Tech Stack | **8** | Solid; Streamable HTTP maturity unvalidated; no test strategy |
| 8. Summary | **8** | Accurate summary of what ALMS is and isn't |

**Overall: 7.8/10** — Good architecture, implementation details need tightening.

---

## Verdict: REVISE

**Recommendation:** Revise the plan with the following changes, then proceed to Phase 1.

### Must-Fix Before Phase 1

1. **Add `tags` and `types` filter params to `learning.sync`.** Backward-compatible (default: all), enables agents to filter by relevance.

2. **Fix the sync_ack gap problem.** Either acknowledge single learnings or validate batch gaps. Document which approach.

3. **Define the Protocol schema.** At minimum: `protocol_id`, `title`, `body`, `target_agents` (or `target_tags`), `version`, `author`, `created_at`, `updated_at`.

4. **State soft-delete or hard-delete explicitly for learnings.** If soft-delete, update `learning.sync` to skip deleted records.

5. **Add a "Security Considerations" section.** Even if the answer is "none for Phase 1," it should be explicit.

6. **Remove `learning.get`** (redundant with resource) **and `learning.vote`** (preemptive). Reduce surface complexity.

7. **Add an `agent.update` tool.** It's cheap and prevents stale capability data.

### Strongly Recommended for Phase 1 Planning

8. **Specify the PostgreSQL schema.** At minimum: indexes for sync (`created_at DESC`), search (`tsvector` GIN), and the `learning_acknowledgements` join table design.

9. **Add a test strategy sketch.** What's the critical path test? ("agent.register → learning.store → learning.sync → learning.sync_ack → learning.sync returns empty")

10. **Define transport details.** Is ALMS on `127.0.0.1` or `0.0.0.0`? How do remote agents connect? Is Streamable HTTP validated with OpenClaw?

11. **Add `resolution` and `src_agent_id` to LearningRecord.** These are cheap now, expensive later.

### Carry Over: Phase 2 Work

- `learning.vote` — after 10+ agents or multi-user
- Config reload via SIGHUP
- Web dashboard (Phase 4 plan)
- Semantic dedup (embedding-based, not exact-match)
- Pagination for `learning.search` with large result sets

---

## Appendix: v3.0 vs Previous Reviews

| Issue | v2.0 Review (DeepSeek) | v2.0 Review (Gemini) | v3.0 Status |
|-------|------------------------|----------------------|-------------|
| Go/Python contradiction | 🔴 Blocking | 🔴 Blocking | ✅ **Fixed** — Python only |
| State machine overengineered | 🟡 High | 🟢 Fine as-is | ✅ **Fixed** — Deleted entirely |
| Credential exposure | 🔴 Blocking | 🟡 High | ⚠️ **Not addressed** — no security section at all |
| No auth | 🟡 High | 🟡 High | ⚠️ **Not addressed** — plan says nothing |
| No migration strategy | 🟡 High | Not flagged | ⚠️ **Not addressed** — schema.sql exists but no migration versioning |
| OpenClaw transport unknown | — | 🟡 High | ⚠️ **Not addressed** — validation deferred |
| Test strategy absent | 🟡 High | — | ⚠️ **Not addressed** — no test plan |
| Learning sync gap handling | — | — | 🔴 **New issue in v3.0** |
| Protocol schema undefined | — | — | 🟡 **New issue in v3.0** |
| Soft-delete unspecified | — | — | 🟡 **New issue in v3.0** |
| No scope creep | 🟢 Fine | 🟢 Fine | ✅ **Still clean** |

The v3.0 plan eliminated the two blocking issues from v2.0 (Go/Python, state machine). But it introduced 3 new issues that didn't exist in v2.0 (sync gap handling, undefined Protocol schema, no soft-delete strategy) because v3.0 is a substantially different architecture — no lifecycle management = different sync concerns.

**Net progress:** Significant. v2.0 was "don't build this," v3.0 is "fix 4 things and build it."

---

## Conclusion

ALMS v3.0 is a **focused, well-motivated architecture plan** that knows what it is and isn't. After the v2.0 identity crisis, the team has produced a coherent Python/FastAPI design with a clean registry + learning store surface.

The 4 critical gaps in the learning sync design (filters, mandatory overlap, gap-safe ack, soft-delete) must be resolved before Phase 1. The missing Protocol schema and undefined security posture are important specification gaps but not blockers.

**Score: 7.8/10**  
**Verdict: Revise → Build** (after 4 fixes)
