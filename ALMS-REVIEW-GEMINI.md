# ALMS Architecture Plan v2.0 — Critical Review

**Reviewer:** Subagent (Gemini-class analysis)  
**Date:** 2026-06-05  
**Status:** ❌ **DO NOT BUILD AS-IS** (build a trimmed version first)

---

## Executive Summary

The plan is thoughtful, well-researched, and architecturally sound **on paper**. It has serious thinking about states, data models, failure modes, and deployment — which puts it ahead of most first-pass architecture docs.

**But.** It has four fundamental problems that will waste weeks if built as specified:

1. **Language mismatch (Go vs Python).** The plan says Go, then specifies FastAPI/Python for every module. This isn't a trivial detail — it's a 2-week delay rewriting things after Phase 1 fails.
2. **Scope creep disguised as phased delivery.** Week 1 alone has 13+ build items *plus* remote agent registration *plus* MCP host integration *plus* the first deployment to a machine you can't SSH into right now.
3. **No auth will bite immediately.** "LAN-only, single user" ignores that ALMS controls `systemctl start/stop` on production agents. One stray curl from a compromised LAN device and your newsletter pipeline is dead.
4. **MCP over Streamable HTTP for remote agent health.** Heartbeats over HTTP from agents that run hourly cron jobs will register false UNREACHABLE states constantly.

The plan needs **30% cuts and 20% fixes** before it's buildable. Below is the detailed breakdown.

---

## 1. What Works — Strongest Parts

### ✅ State Machine Design (Section 5)
The BOOTING → RUNNING → DEGRADED → FAILED → STOPPED → UNREACHABLE flow is complete. It covers real edge cases (health check failures during boot, heartbeat timeouts for remote agents) without over-engineering. The transition table is one of the cleanest parts of the plan. Keep this as-is.

### ✅ Data Model (Section 4)
AgentSpec, LearningRecord, ToolSpec, LifecycleEvent — each has exactly the right fields. No bloat. The separation of static spec (`AgentSpec`) from runtime state (`AgentState`) is correct and avoids cache-invalidation headaches. `ToolSpec.availability` (online/offline) and `latency_ms` are details most plans miss.

### ✅ Process Isolation Strategy (Section 9)
systemd + venvs is the right call for this hardware. No Docker layer, no container overhead, native resource limits via `MemoryMax=` and `CPUQuota=`. On 7GB RAM with 12 cores, Docker adds ~200MB overhead for no benefit. `BindsTo=alms.service` in the agent unit template is a smart dependency chain.

### ✅ Failure Modes Table (Section 12)
Thorough. Covers ALMS crash, agent crash, DB down, machine reboot, remote disconnect — each with a real mitigation, not just "handle gracefully." The "ALMS handles connection retry" note for PostgreSQL is the right level of concreteness.

### ✅ Learning Transfer Flow (Section 8.5)
The pure-MCP flow (Agent A pushes → ALMS validates → Agent B pulls) is clean, minimal, and fits the "no SSH, no side channels" constraint. Semantic dedup + scoring + GC is the right baseline for a learning store.

---

## 2. What's Missing — Real Gaps

### 🔴 Go vs Python Contradiction
The plan says Go everywhere (Section 1: "Go-first — single binary, ~30MB RAM"), then Phase 1 (Section 7) specifies **FastAPI + asyncpg + Pydantic + mcp (pip)** — all Python. The tech stack table (Section 15) also says Go, but the implementation details are 100% Python.

This is not a minor inconsistency. If you build in Python:
- **Memory:** FastAPI + venv + asyncpg + uvicorn ≈ 80-120MB, not 30MB
- **Dependencies:** pip freeze list, venv management on the remote machine
- **Deployment:** Python version mismatch (Ubuntu 26.04 ships Python 3.14?), pip install on every deploy
- **Recompilation risk:** The only reason to pick Go is the single-binary deployment. If you're shipping Python on a headless machine, Go provides zero benefit.

**Either:** commit to Go (net/http + pgx + mcp-go) with ~15MB binary, ~30MB RAM  
**Or:** commit to Python (FastAPI) and remove all Go references

Given the constraint of 7GB RAM and a machine you SSH into for deployment, **Python is fine**. The memory difference (80MB vs 30MB) is noise when the newsletter-writer agent alone gets 256MB. But pick one.

### 🔴 OpenClaw Channel Constraint
ALMS is the control plane, but the **primary human interface** to your agents is Telegram via OpenClaw. The plan mentions OpenClaw as an MCP host but doesn't address:
- How does a Telegram user start/stop an agent? Through a Telegram command that OpenClaw translates to an MCP call?
- How does ALMS notify the human about agent failures? Currently OpenClaw sends Telegram messages. If ALMS detects a crash, does it push to OpenClaw's MCP, or does OpenClaw poll?
- The orchestration loop: if the newsletter-scout crashes during a human-initiated run, who retries? ALMS (via systemd auto-restart) or OpenClaw (via retry logic)?

You need a **notification contract**: ALMS pushes lifecycle events → OpenClaw MCP subscription → Telegram message. The plan has no push mechanism.

### 🔴 Agent Logs Access Pattern
`alms://agents/{agent_id}/logs` reads `journalctl`. This is elegant but **latency-uncapped** — journalctl can take 2-5 seconds to return 100 lines on a busy system. For an MCP resource (which expects fast reads), this will time out or block the MCP server.

Solution: stream logs async, or cap at 50 lines with a 1-second timeout. The plan should specify this.

### 🔴 "No SSH" Stance Overstated
Section 1 says "No SSH for agent control — agent communication is pure MCP over HTTP." Then Phase 1's deployment script uses `ssh data` for *everything* — installing PostgreSQL, creating databases, copying binaries, enabling systemd services. The deploy script does 10+ SSH commands.

This is fine (SSH for deploy, MCP for runtime) but the plan's messaging is misleading. Be precise: **SSH is deploy-time only. Runtime agent control is MCP over HTTP.** The current phrasing will confuse implementers.

### 🔴 Cron → systemd Migration
Section 8.3 says "Current newsletter cron jobs become systemd-managed services on the data machine." But ALMS's role in triggering them is an open question (Section 14: "How does ALMS trigger them? systemd timers? Internal scheduler?").

If they're systemd services with `OnCalendar=` timers, ALMS doesn't need to trigger them — but then ALMS has no control over scheduling. If ALMS schedules them, you need an internal scheduler (`time.AfterFunc` in Go, `asyncio.sleep` loops in Python) that adds complexity.

This needs resolution before Phase 2. I'd recommend: **systemd timers for cron-like agents, ALMS just monitors.** For complex workflows (chain agent → writer → publisher), ALMS or OpenClaw orchestrates.

---

## 3. Scalability Concerns — How Far Does 7GB Stretch?

### The Honest Answer
For your current workload (2-3 newsletter agents, 1 film pipeline, ALMS itself), **7GB is comfortable** — you're at ~2.4GB as the plan calculated.

### Where It Breaks
- **10+ systemd agents** each with Python venvs: each venv is 50-150MB on disk, plus 100-250MB RAM per agent. 10 agents = 2.5GB RAM just for agents.
- **PostgreSQL data growth:** learnings with full-text search on 10,000+ records will consume 500MB-1GB in PostgreSQL shared_buffers before it's fast. The plan allocates only 200MB to PostgreSQL.
- **Concurrent MCP calls:** Streamable HTTP MCP on FastAPI with uvicorn workers → each worker is a separate Python process. 1 worker = 100MB. 4 workers = 400MB for the MCP layer alone.
- **Logs in journald:** Without log rotation configured, 10 agents producing stderr will fill the 84GB free disk in months.

### Recommendations
- **Set `SharedBuffers=512MB`** for PostgreSQL, not the 128MB default. Your 7GB can spare it.
- **Add log rotation:** `journalctl --vacuum-size=1G` in a daily cron. Not in the plan.
- **Stay at 5-6 agents max** on this machine. Beyond that, spin up a second data machine.
- **Go binary vs Python server:** At scale, Go's 30MB flat memory (vs Python's 80MB + per-worker) matters. If you plan to grow beyond 10 agents, Go is the right choice. If this stays at 3-5 agents, Python wins on developer speed.

### Verdict
Fine for today. Marginal for 12+ agents. Plan for a second machine before you hit 8 agents.

---

## 4. Security Considerations

### 🔴 The "No Auth" Assumption Is Wrong for Your Setup
The plan says "LAN-only, single user. Add security layer if/when it goes external." But:

1. **A compromised LAN device** (IoT light bulb, smart TV, guest WiFi device) can call `agent.stop("newsletter-scout")` or `agent.unregister("film-pipeline")` and take down production agents.
2. **MCP tools include systemctl wrappers.** `agent.restart` calls `systemctl restart`. This is root-level power exposed over HTTP. No auth = no accountability.
3. **Your data machine runs PostgreSQL with a password.** If ALMS has no auth, anyone on the LAN who discovers the MCP endpoint can query `learning.search()` and read all your learnings (including potential business-sensitive data).

### The Minimal Fix
You don't need OAuth2 or JWTs. You need two things:
- **Bearer token in `X-ALMS-TOKEN` header.** A 32-char hex string from `alms.yaml`. Static, simple, blocks 99% of LAN attackers. 20 lines of code.
- **systemd `Restart=on-failure`** with ALMS's own unit. If ALMS is down, agents continue running (they're separate systemd units with `BindsTo=alms.service`, but the agent unit keeps running even if ALMS restarts — verify this).

### Additional Protections
- **PostgreSQL: `pg_hba.conf`** should only allow connections from localhost (127.0.0.1). Currently the plan shows `ALMS_URL=http://127.0.0.1:8001` for agents, but PostgreSQL config isn't mentioned. Default PostgreSQL on Ubuntu listens on `0.0.0.0:5432`.
- **ALMS port should not be on `0.0.0.0`** unless Mac agents need it. If all agents on the data machine connect via 127.0.0.1, bind ALMS to `127.0.0.1:8001` and use a reverse proxy (or just expose MCP port via SSH tunnel for remote agents).
- **`alarm` user** in PostgreSQL should not be a superuser. Create a dedicated `alarms` role with SELECT/INSERT on specific tables only.

---

## 5. MCP Interface Design — Tools and Resources

### What's Right
- The resource URI scheme (`alms://agents/{id}/state`, `alms://learnings/{id}`) is clean and follows MCP conventions.
- `alms://health` as a top-level resource is a good pattern.
- Separating reads (resources) from writes (tools) is correct MCP design.

### What Needs Fixing

**⚠️ `agent.get_logs` as a tool, not a resource.** The plan puts "Get agent logs" as a tool. But logs are a read operation — they should be a resource: `alms://agents/{id}/logs?lines=50`. The current tool-based approach forces MCP hosts to call a tool to get data that should be fetchable as a resource. This blocks OpenClaw from showing logs in its resource browser. **Move to resource with query params.**

**⚠️ No batch operations.** If you have 5 agents and want to restart all of them, you need 5 MCP calls. Add `agent.batch` or `agent.restart_many(agent_ids: ["a","b","c"])`. Even for a single-user setup, this matters when recovering from a data machine reboot.

**⚠️ `learning.vote` is unnecessary.** Upvoting learnings is a nice social feature for multi-tenant ALMS. For a single-user setup with 3 agents, scoring by usage frequency is better than explicit votes. Vote data is sparse and adds a table + endpoint for no value in the MVP.

**⚠️ Missing `health.get_config` resource.** The plan's `alms://health` endpoint returns system health. Add a `alms://health/config` resource that returns the active ALMS config (redacted passwords). Useful for debugging without SSH'ing.

### Protocol Design Issues

**Streamable HTTP is still experimental for MCP.** The official MCP spec defines Streamable HTTP as "[spreadsheet](
https://spec.modelcontextprotocol.io/specification/2025-03-26/basic/transports/#streamable-http)." The Python `mcp` SDK supports it, but:
- OpenClaw may not support Streamable HTTP yet (check OpenClaw's MCP config — current servers use `stdio` transport).
- If OpenClaw expects `stdio` MCP, you need an adapter layer or a different transport.

**Recommendation:** Before building, verify OpenClaw supports Streamable HTTP MCP servers. If not, you'll need a local MCP bridge (lightweight stdio process on Mac that proxies HTTP to data:8001). This adds latency (~2ms LAN vs 0ms local) but eliminates the transport risk.

---

## 6. Integration Risks — How This Fits the Current Setup

### 🔴 OpenClaw MCP Transport Unknown
Current OpenClaw config (`~/.openclaw/openclaw.json`) has two MCP servers, both using `stdio` transport (command + args). No HTTP/Streamable HTTP servers are registered. This means:

- **You can't just add `{"name": "alms", "url": "http://192.168.2.112:8001/mcp"}`** — OpenClaw's MCP runtime may not support URL-based server registration.
- **Worst case:** You build the entire Streamable HTTP server, then discover OpenClaw only supports stdio. You'd need a local MCP bridge process.

**Check this first.** If OpenClaw supports HTTP MCP servers, great. If not, Phase 1 becomes Phase 0: "build a lightweight stdio MCP proxy."

### 🔴 Dual Connection Architecture
Your agents need two connections:
1. **MCP to ALMS** (learning store, registry)
2. **MCP to OpenClaw** (tools for the human session)

This creates a split-brain scenario: the newsletter-scout agent on the data machine pushes learnings to ALMS, but also needs to register its tools with OpenClaw for the human to use. The plan doesn't address how these two MCP connections coexist in one agent.

### 🔴 IS-043 Integration Is Deferred to Phase 3
The plan says "Integrate with existing IS-043 learning schema" in Phase 3. This is risky — IS-043 was the research phase for the *learning* side. If Phase 1 builds the registry without the store, and Phase 3 then discovers IS-043's schema doesn't map cleanly to ALMS's design, you'll refactor migrations.

**Recommendation:** Define the PostgreSQL schema for learnings in Phase 1 (based on IS-043's existing model), even if the MCP endpoints for learnings come in Phase 3. The schema rarely changes; APIs change frequently.

### 🔴 Newsletter Cron Migration
Current newsletter workflow:
1. Human triggers via Telegram/OpenClaw
2. OpenClaw runs the newsletter skill (web search → write → publish)
3. Articles are published

Under ALMS:
1. Newsletter-scout runs on schedule (systemd timer)
2. Agent calls ALMS for learnings
3. Agent publishes results

The gap: **who triggers the newsletter?** If it's a systemd timer on data machine, the human loses Telegram control. If it's OpenClaw on Mac (current setup), the triggering agent is a remote MCP client, which depends on the Mac being online.

This is solvable (systemd timer triggers, agent reports results to ALMS, ALMS notifies via OpenClaw MCP → Telegram), but it needs explicit design. The plan doesn't have it.

---

## 7. Priority Recommendations — Top 3 Changes

### 🥇 #1: Resolve the Go vs Python Contradiction (Before Any Coding)

**Position:** Pick Python (FastAPI) for the MVP. Drop Go.

**Why:**
- Every module description says FastAPI, Pydantic, asyncpg, mcp (pip). Someone wrote the plan in Go then described it in Python. The implementation intent is clearly Python.
- Your deploy machine has Python 3.x. No compilation, no cross-compilation flags, no `CGO_ENABLED=0`.
- Dev speed matters more than 50MB RAM for a single-user system.
- The "30MB Go binary" benefit is real but irrelevant when the newsletter-scout agent alone is a 256MB Python venv.

**If** you later need Go (10+ agents, sub-100MB footprint), you can rewrite ALMS in Go as a v3.0. For v2.0 MVP, **write it in Python, own it, stop waffling.**

### 🥇 #2: Add Minimal Auth Before Deployment (Day 1, Not Day 100)

**Position:** Static bearer token in config. Full stop.

**Implementation:** 30 minutes of work.
```yaml
# alms.yaml
server:
  host: "127.0.0.1"  # or "0.0.0.0" for remote agents
  port: 8001
  auth_token: "a1b2c3d4e5f6..."  # 32-char hex
```

```python
# In MCP middleware
if config.auth_token:
    token = request.headers.get("X-ALMS-TOKEN")
    if token != config.auth_token:
        raise HTTPException(403, "invalid token")
```

If remote agents (Mac) need access without the token, add a **same-LAN bypass** (check if source IP is 127.0.0.1 or 192.168.2.0/24). But start with the token. The risk of leaving MCP endpoints unprotected outweighs the trivial setup cost.

### 🥇 #3: Verify OpenClaw MCP Transport, Build Bridge If Needed (Phase 0, Not Phase 1)

**Position:** Before writing a single line of ALMS server code, confirm OpenClaw can connect to an HTTP MCP server.

**Action:**
1. Read OpenClaw docs / config schema for MCP server registration format
2. If HTTP MCP is supported: use Streamable HTTP as planned
3. If only stdio is supported: write a 50-line stdio ↔ HTTP proxy that runs on Mac:
   - Opens a `stdio` MCP server on localhost
   - Forwards all JSON-RPC calls via HTTP to `data:8001`
   - Adds the auth token header automatically
   - Launched as a systemd user service or launchd plist

This avoids the worst-case integration failure: building a full Streamable HTTP server that no one can connect to.

---

## Summary

| Area | Grade | Key Issue |
|------|-------|-----------|
| State machine | **A** | Complete, correct edge cases |
| Data model | **A-** | Right fields, minor: add `ttl_days` to LearningRecord defaults |
| Process isolation | **A** | systemd + venvs is correct for this hardware |
| Failure modes | **B+** | Missing: log rotation, disk space monitoring |
| Scalability analysis | **B** | Fine for now, boundary at 8 agents (not 50) |
| Security | **D** | No auth on agent lifecycle controls is dangerous |
| MCP design | **B-** | Mixing tools/resources, missing batch ops, Streamable HTTP risk |
| Integration analysis | **C** | OpenClaw transport unknown, dual-MCP split brain not addressed |
| Go vs Python | **F** | Plan says Go, implementation says Python. Decide. |
| Overall practicality | **C+** | Thoughtful structure undermined by contradictions and scope |

**Bottom line:** This plan has the right bones. Clean up the Go/Python mismatch, bolt on 30 minutes of auth, and confirm your MCP transport. Then build Phase 1 as specified. You'll have ALMS running on real hardware in a week. Without these fixes, you'll spend Week 1 debugging integration problems and wondering why the "30MB Go binary" requires a Python runtime.

---

## Appendix: Quick Wins (Build These First)

Not core to the review, but cheap to add:

1. **`alms://health/version`** resource — returns the build version + git hash. Debugging gold.
2. **Config reload via SIGHUP** — `alms --reload` or `kill -HUP <pid>` re-reads YAML without restart. For changing auth tokens or database connections.
3. **`agent.list` filter by status** — `agent.list(status="running")` returns only active agents. Useful from OpenClaw when checking "are my agents alive?"
4. **Dashboard as static HTML (not Jinja2)** — Phase 4's web dashboard doesn't need FastAPI + Jinja2. One static HTML page with vanilla JS that calls `alms://health` and `alms://agents` and renders tables. Deployable as a single file. Save the server-side rendering for if/when you need auth.
