# ALMS Architecture Plan v2.0 — Critical Review

**Reviewer:** DeepSeek  
**Date:** 2026-06-05  
**Status:** ❌ BLOCKING ISSUES FOUND — DO NOT START PHASE 1 AS-IS

---

## Executive Summary

ALMS v2.0 is **well-motivated but architecturally inconsistent**. The core idea — a lightweight control plane for agents over MCP — is sound. But the plan has a **fatal identity split**: it describes a Go binary architecture but all Phase 1 artifacts are Python/FastAPI. The state machine is overengineered for the actual operational reality. The MCP surface is good but imbalanced. The plan is useful as a design document but **must be reconciled with itself before a single line of code is written**.

---

## 1. Architecture Quality — 5/10 (Needs Major Cleanup)

### What's Right
- Single binary, zero deps, Go for the control plane — excellent choice for a long-running server on a RAM-constrained machine
- systemd for process isolation is pragmatic and correct for a single Ubuntu machine
- MCP as the canonical interface is the right bet for an agent control plane
- No SSH for agent communication, no Docker overhead — lean and correct
- Module layout in Section 3.2 is clean

### What's Wrong

**CRITICAL: The plan describes two different architectures.** Section 3 consistently shows the entire stack in Go:
- `cmd/alms/main.go`, `go.mod`, Go module tree
- `mark3labs/mcp-go`, `pgx`, `viper`
- Tech Stack Summary (Section 15) says **Go 1.24+**

But **Phase 1** (Section 7) creates:
- `alms/server.py` — FastAPI
- `alms/config.py`
- `alms/models/*.py` — Pydantic models
- `alms/services/registry.py`
- `alms/db/schema.py` — asyncpg
- `alms/mcp/*.py`
- `alms/runners/systemd.py`

This is not a minor inconsistency — it's two completely different tech stacks. The Go architecture was extensively debated and chosen; the Python FastAPI stack in Phase 1 appears to be a copy-paste artifact from an earlier iteration. **This must be resolved before any implementation.**

**My position: Go is the correct choice.** A Go binary with `mark3labs/mcp-go`, `pgx`, and `viper` will use ~30MB RAM, compile to a static binary, and serve the Streamable HTTP transport directly without FastAPI as an extra layer. Python/FastAPI would require the full Python runtime (~150MB), pip deps, virtualenv, and would negate every Go advantage listed in Section 9.

### Missing Architectural Components

| Missing Component | Why It Matters |
|---|---|
| **Circuit breaker / backpressure** | 7GB RAM, 12 agents all calling `learning.search` simultaneously — no protection |
| **Leader election / lock for systemd ops** | Two concurrent `agent.start` calls race on systemctl; need a mutex per agent_id |
| **Rate limiter for MCP endpoints** | A buggy agent spamming `learning.store` can DOS the DB |
| **Graceful shutdown handler** | SIGTERM → finish in-flight MCP requests → drain queue → shutdown |
| **Structured logging package** | `journald` with JSON logs requires a logging library, not `fmt.Println` |
| **Startup readiness probe** | systemd `Type=simple` means systemd marks it "active" immediately even before DB connection succeeds |
| **Config reload (SIGHUP)** | Changing PG DSN or agent paths requires restart; SIGHUP reload is cheap |

---

## 2. Data Model — 7/10 (Good Core, Missing Critical Fields)

### AgentSpec
Solid. But missing:

```json
{
  // Add these fields:
  "schedule": null,              // cron expression for scheduled agents
  "dependencies": [],            // agent_ids this agent depends on
  "resource_limits": {           // explicit overrides for systemd
    "memory_max_mb": 256,
    "cpu_quota_pct": 50
  },
  "retry_policy": {              // agent-level retry, not just systemd RestartSec
    "max_retries": 3,
    "backoff_s": 5
  },
  "last_start_result": null      // "success" | "timeout" | "oom_killed" | "exit_code:1"
}
```

Also: `"config.env"` embeds the PG DSN with password in the registry **and** in the systemd unit file. This is a credential leak — the DSN is readable via `alms://agents/{id}` and visible in `journalctl` process listings. Fix: store secrets externally and reference them (env file with `EnvironmentFile=` in systemd, or a local secrets file ALMS reads but never exposes via MCP).

### AgentState
Missing: `oom_killed` flag, `exit_code` (on failure), `signal` (termination signal). The current model only captures `pid` and `error` — not enough to distinguish OOM from a clean crash from a signal kill.

### LearningRecord
Good schema. Missing:
- `resolution`: "pending" | "resolved" | "superseded" — so you can mark learnings as stale
- `source_url`: direct link to the source that triggered this learning
- `related_learnings`: array of learning_ids for linking related discoveries
- `ai_generated: bool` — distinguish human-authored protocol from auto-captured agent learnings

### ProtocolRecord
The plan mentions it (in `protocol.push` tool) but never defines the ProtocolRecord struct. This is a gap — if protocols are mandatory SOPs that agents must read, they need versioning, approval workflow, and a "last_acknowledged_by_agent" tracking field.

---

## 3. State Machine — 6/10 (Too Many States, Wrong Default)

### The Problem

8 states for a system with two types of agents:
1. **systemd agents** (on the data machine) — systemd handles their lifecycle almost entirely
2. **mcp_client agents** (remote) — ALMS has no process control; it only monitors heartbeats

For systemd agents, the only ALMS-controlled transitions are `agent.start`, `agent.stop`, `agent.restart`. Everything else (`RUNNING → FAILED`, `RUNNING → DEGRADED`) is ALMS *observing* systemd state, not *driving* it. Systemd already has `Restart=on-failure`. So ALMS is adding a second state layer on top of systemd's existing one. This creates a **split-brain risk**: systemd thinks the agent is `active (running)`, ALMS thinks it's `FAILED` because health check 1.0 is slightly below threshold.

For mcp_client agents, only three states matter: `REGISTERED`, `RUNNING`, `UNREACHABLE`. The entire `BOOTING → RUNNING → DEGRADED → STOPPING → STOPPED` axis is irrelevant because ALMS can't systemctl a remote Mac process.

### Recommendation

**Split the state machine:**

**systemd agents (7 states → 5 states):**
```
REGISTERED → STARTING → RUNNING → STOPPED
                              ↘ FAILED → (auto-restart via systemd, ALMS just logs)
```
Remove `DEGRADED` and `STOPPING`:
- `DEGRADED` is an operational metric (health score < 0.8), not a state. Track it in AgentState.health_score.
- `STOPPING` is transient and systemd handles it. ALMS polls until `is-active` returns inactive, then sets state to `STOPPED`.

**mcp_client agents (3 states):**
```
REGISTERED → RUNNING → UNREACHABLE
```
That's it. Simplify the whole diagram.

### Edge Cases Missed
1. **Double-stop**: `agent.stop` called while already STOPPING/STOPPED → no-op or error
2. **Zombie agent**: systemd unit removed but ALMS still has it in DB → no GC
3. **Race on boot**: ALMS starts before PostgreSQL → BOOTING → FAILED immediately, then systemd restarts ALMS, which starts before PG again → infinite restart loop
4. **systemd agent from another session**: User manually runs `systemctl stop agent-X.service` from SSH while ALMS thinks it's RUNNING → ALMS is now stale

---

## 4. MCP Surface — 8/10 (Good Scope, But Incomplete in Detail)

### Overall Assessment
18 tools + 11 resources is **not too many**. For a control plane, this is lean. A good sign is there's no unnecessary CRUD for things that should be resources.

### What's Missing

**MCP Health resources:**
- `alms://agents/{agent_id}/learnings` — get all learnings from a specific agent
- `alms://alerts` — recent critical events (FAILED, UNREACHABLE transitions in last 24h)

**Missing MCP tools:**
- `agent.update` — update AgentSpec fields (description, config, capabilities). Currently you'd have to unregister and re-register.
- `agent.get_events` — currently only available as a resource; having it as a tool with filters (event_type, from_date, to_date) is better for programmatic use.
- `learning.delete` — currently no way to remove a bad learning record
- `agent.set_schedule` — add/modify cron schedule for a registered agent
- `system.status` — show system-level info (disk usage, RAM, agent count, DB size)

### What's Slightly Overbuilt
`health.check_agent` duplicates what `agent.get_status` already provides. Merge: `agent.get_status` always includes health info. Remove `health.check_agent`.

### Resource vs Tool Boundary
Resources: state/read-only data. Tools: actions with side effects. The plan mostly gets this right, but `alms://agents/{agent_id}/logs` is an unusual resource — reading logs is an action (running journalctl). If journalctl takes 2s, that blocks an MCP request. Consider making it a tool (`agent.get_logs`) instead, or cache log output for 10s.

---

## 5. Go Tech Choices — 9/10 (Solid)

### mark3labs/mcp-go
The right pick. It's the most active Go MCP SDK. Supports Streamable HTTP transport cleanly. No need for a separate FastAPI layer wrapping MCP — mcp-go handles HTTP transport natively.

### pgx (jackc/pgx v5)
Best-in-class Go PostgreSQL driver. Connection pooling built in (`pgxpool`). Native `COPY` support will matter if learning records grow. Correct choice.

### viper (spf13/viper)
Reasonable, but viper is heavy (~400KB binary overhead for a 5-field config). On a 30MB Go binary this doesn't matter, but consider `caarlos0/env` for a simpler approach — YAML config baked at build time, env vars for runtime overrides. Viper's multi-source layering is overkill for a single-machine setup.

### Go 1.24
Yes. `net/http` routing improvements, `slog` structured logging in stdlib, better generics. No need for Gin/Chi.

### One Contention
The plan's Section 10 decision log lists "FastAPI" as the server framework. This should say "Go + net/http + mark3labs/mcp-go". The decision log row was never reconciled when the architecture switched from Python to Go.

---

## 6. Gaps & Risks

### 🔴 CRITICAL: Credential Exposure
The plan embeds `PG_DSN` (including password) in:
1. AgentSpec.config.env (visible via MCP resource `alms://agents/{agent_id}`)
2. systemd unit files (visible via `journalctl -u agent-X.service`)
3. Agent YAML config on disk

**Fix:** Use `EnvironmentFile=` in systemd units pointing to a `0600`-permission file. ALMS reads it on start, stores only a hash in the registry, and never exposes raw credentials via MCP resources.

### 🟡 HIGH: No Migration Strategy
Schema migrations via custom SQL files (`db/migrations/001_initial.sql`) is fine for Phase 1. But there's no version tracking, no rollback, no idempotent `CREATE TABLE IF NOT EXISTS`. If `--migrate` runs twice, does it error? Skip? Re-run all files? What happens when you have 15 migration files and deploy v2 schema while agents are running?

**Fix:** Use `golang-migrate/migrate` or `pressly/goose`. Both are tiny Go libraries, manage version tracking via a `schema_migrations` table, and support up/down migrations. For 5-15 tables, this is not overhead — it's insurance.

### 🟡 HIGH: No Testing Strategy
Section 13 (Week 4) says "Tests for all services" but gives zero detail. For a control plane that starts/stops agents and manages critical state:
- **Unit tests**: state machine transitions, dedup logic, scoring
- **Integration tests**: against a real PG (testcontainers-go), verify state transitions actually persist
- **No E2E**: fine for Phase 1

Without this, a single state machine bug puts agents in an unrecoverable state.

### 🟡 HIGH: systemd CLI Parsing Is Brittle
The plan says `agent.get_logs` uses `journalctl -u agent-X.service -n N` and parses output. Journalctl output changes between Ubuntu versions, locale settings, and pager configurations. Parsing `systemctl is-active` returns strings like `"active"`, `"inactive"`, `"activating"`, `"deactivating"`, `"failed"`. A typo in parsing puts the state machine in an unrecoverable path.

**Fix:** Use the D-Bus API (via `go-systemd` package) instead of CLI parsing. More reliable, no locale issues, structured data.

### 🟡 MEDIUM: Single-Point-of-Failure Design
One ALMS server, one PostgreSQL, one machine. If:
- ALMS crashes → systemd restarts it, but open MCP connections drop
- PG goes OOM → ALMS can't recover agents
- Disk fills → journalctl stops, learnings can't be stored

For a home-lab agent control plane this is acceptable. For anything beyond: it's not HA. Document this explicitly as a design constraint.

### 🟡 MEDIUM: No Agent → ALMS Auth
"No auth, LAN-only" is fine until one agent goes rogue (infinite loop calling `learning.store`) or a misconfigured agent starts calling `agent.stop` on other agents. **At minimum, agents should authenticate with a shared token or API key.** Not for security — for accountability.

### 🟢 LOW: Go Cross-Compilation
`GOOS=linux GOARCH=amd64 go build` — the deployment steps say "build on Mac, binary on data". This works but there's no mention of CGO. If any Go dependency uses CGO (e.g., SQLite driver), the binary won't cross-compile. Since ALMS uses pgx (pure Go) and mark3labs/mcp-go (pure Go), this is fine, but `CGO_ENABLED=0` should be explicit in the build step.

---

## 7. Specific Recommendations (Before Phase 1 Starts)

### Must-Fix (Blocking)

1. **RECONCILE THE ARCHITECTURE.** Delete Python/FastAPI from Phase 1. Create:
   - `cmd/alms/main.go` — Go entry point
   - `internal/mcp/server.go` — mark3labs/mcp-go Streamable HTTP server
   - `internal/db/postgres.go` — pgx pool
   - No `server.py`, no `config.py`, no asyncpg. Go only. The document describes two architectures; pick the Go one and commit.

2. **SIMPLIFY THE STATE MACHINE.** Split into systemd agents (5 states) and mcp_client agents (3 states). Remove DEGRADED and STOPPING as states — they're metrics and transient observations, not states.

3. **FIX CREDENTIAL EXPOSURE.** Use `EnvironmentFile=` in systemd units. Store only hashed references in the AgentSpec. Strip raw secrets from MCP resource responses.

4. **ADD go-systemd INSTEAD OF CLI PARSING.** Use `github.com/coreos/go-systemd` for D-Bus interaction with systemd. This eliminates brittle journalctl/systemctl output parsing.

5. **ADD golang-migrate/migrate** for schema migrations. Don't write a custom migration runner.

### Strongly Recommended

6. **Reduce to 3 golang.org/x/ dependencies:** `golang.org/x/sync` (singleflight + semaphore), `golang.org/x/net` (for HTTP/2 if needed). Keep Go stdlib plus the three explicit deps: `mark3labs/mcp-go`, `jackc/pgx/v5`, `golang-migrate/migrate`. Skip viper — use `caarlos0/env` (or even just flag + JSON/forerunner struct).

7. **Add `agent.update` tool** before you have 10 registered agents and need to change a config field. This is cheap now, expensive later.

8. **Add a simple startup sequence:**
   ```
   alms start → connect PG → run migrations → start MCP server → report ready
   ```
   Use `slog.Info("ALMS ready on :8001")` as the final line. systemd `Type=simple` waits for nothing — use `Type=notify` with `sd_notify("READY=1")` from `coreos/go-systemd/activation`, so systemd only marks the unit active after ALMS is genuinely ready.

9. **Add a per-agent mutex** for lifecycle operations. Two concurrent `agent.start` calls will race on `systemctl`. Use a `sync.Mutex` (or better, `sync.Map` of per-agentID mutexes) to serialize systemd operations for the same agent.

10. **Remove `health.check_agent` tool.** Fold health info into `agent.get_status`.

### Nice-to-Have (Post-Phase 1)

11. **Web dashboard** — Phase 4 is right to defer this. A dashboard adds UI complexity. Verify the MCP surface works first, then build a thin HTML frontend.

12. **Scheduled agent triggers** — Phase 1 says "how does ALMS trigger cron agents?" My recommendation: use systemd timers (`agent-newsletter-scout.timer`). This keeps scheduling at the OS level. ALMS should only manage presence and learning, not cron. If you need dynamic scheduling, add it in Phase 3.

13. **Learning dedup algorithm** — Phase 3 mentions "semantic dedup" but doesn't define the algorithm. For v1: exact title match → skip. Exact body hash (SHA256) → skip. Semantic dedup (embedding similarity) is Phase 4+ work.

---

## Conclusion

This architecture plan is **structurally sound at the concept level** but **self-contradictory at the implementation level**. A Go + systemd + MCP control plane is the right design. But Phase 1 as written implements Python/FastAPI/asyncpg — a different architecture entirely. The state machine is over-modeled. Credentials are exposed. Migration strategy is absent.

**Score breakdown:**
- Architecture concept: 8/10
- Architecture consistency: 4/10 (halved by Go/Python split)
- Data model: 7/10
- State machine: 6/10
- MCP surface: 8/10
- Tech choices: 9/10
- Gaps & risks identified: 3/10 (plan doesn't identify its own gaps)
- Documentation clarity: 7/10

**Verdict:** Revise the plan to resolve the Go/Python inconsistency, simplify the state machine, add the missing security and reliability infrastructure, then start Phase 1 in Go. The plan should take one day of cleanup before it's ready to build.

*This review was intentionally sharp. ALMS is a good idea that deserves a clean implementation.*
