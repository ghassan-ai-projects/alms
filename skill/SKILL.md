---
name: alms-agent
description: "ALMS learning lifecycle for agents: store discoveries, sync fresh knowledge, score quality, stay nudged."
version: "1.0.0"
author: "openclaw-alms"
tags: [alms, learning, knowledge-management, protocol, agent-sync]
provides: ["alms-store", "alms-sync", "alms-score", "alms-medicate"]
references:
  - pm-orchestrator (multi-model quality-gate pattern for high-severity scoring)
  - learning (filesystem-based learning protocol — alms replaces it for multi-agent)
---

# ALMS Agent Learning Skill v1.0

> **The knowledge lifecycle for agents.** ALMS is the shared brain. This skill tells agents how to use it.

This skill wraps the four ALMS prompt categories into three concrete workflows. Every agent that interacts with ALMS should reference this skill.

---

## Dependencies

- **ALMS MCP server** running and reachable (tools: `learning.store`, `learning.sync`, `learning.sync_ack`, `learning.search`, `learning.get`)
- **Your agent registered** in ALMS (already done during first heartbeat)
- **Prompts library** at `alms/prompts/prompts.md` for the exact prompt text
- **Existing tags** from ALMS (fetch via `learning.search` with empty query to get tag registry)
- For high-severity multi-model scoring: access to DeepSeek and Gemini LLMs

---

## Workflow 1: "I Found Something Important" → Store

Triggered when you encounter a non-obvious insight, workaround, or failure.

```
1. COMPOSE the learning:
   - title: max 80 chars, descriptive
   - body: problem → expectation → reality → root cause → fix/workaround
   - type: pick from [pattern, failure, config, protocol, edge_case]
   - tags: check existing tags first; only create new if necessary
   - severity: calibrate using the prompt's scoring tier table
   - supersedes: set to an existing learning_id if this replaces obsolete info

2. CALL learning.store with the composed record
   - Returns: learning_id + dedup status

3. IF severity >= high:
   → Run Workflow 3 (Score) on the newly created learning
   → This gives us an LLM quality assessment immediately

4. IF severity = critical:
   → Notify: agent should escalate to human via Telegram
   → Message: "ALMS: Critical learning registered — {title} ({learning_id})"

5. LOG: Record the learning_id in local state for dedup avoidance
```

### What NOT to store (refresher)

| Don't store | Why |
|-------------|-----|
| Duplicate of something you stored this session | ALMS dedup handles it, but don't waste the call |
| Unconfirmed speculation | "I think X might be wrong" — wait for proof |
| 5-min trivial fix with obvious root cause | Already in your brain; no cross-agent value |
| Personal config preferences | Not generalizable to other agents |

---

## Workflow 2: "Check for New Knowledge" → Sync

Triggered at session startup, after task completion, or on explicit request.

### Step-by-step

```
PHASE 1 — PREP
├── Load agent state → get last_sync_timestamp
├── Determine filter:
│   ├── Full sync: since = last_sync_ts, type = "", tags = []
│   └── Targeted: since = last_sync_ts, type = "failure", tags = ["my-project"]
├── Read project list from agent capabilities → use as tag filter candidates

PHASE 2 — FETCH
├── Call learning.sync with (agent_id, since, type, tags)
├── Expect: array of LearningRecord objects
├── If empty: done — ack empty and return

PHASE 3 — PROCESS
├── Sort by severity: critical first, then high, then medium/low
├── For each learning:
│   ├── severity=critical → process immediately
│   │   ├── Read body thoroughly
│   │   ├── If type=protocol: integrate into agent behavior
│   │   ├── If actionable fix: apply immediately
│   │   └── Notify human if needed
│   ├── severity=high → read within same batch
│   │   ├── Apply if relevant to current context
│   │   └── File for next-session reference
│   ├── Medium/low → batch-read after all critical/high processed
│   │   └── Add to session context as "known recent learnings"
│   └── If type=protocol → treat as mandatory instruction
│       └── Store in local protocol registry for agent behavior shaping

PHASE 4 — ACK
├── Collect all returned learning_ids into a list (ordered by created_at ASC)
├── Call learning.sync_ack(agent_id, learning_ids)
├── If gap error: re-sync from last_sync_ts (unacked = redelivered)
└── Update local agent state: last_sync_timestamp = now

PHASE 5 — CONTEXT INTEGRATION
├── Add learned facts to session context memory
├── Pin any critical learnings as "FYI — must reference during this session"
└── Log sync summary for audit
```

### Sync summary format

```
ALMS Sync Summary
├── New learnings fetched: 12
├── Critical: 1 — immediately processed
├── High: 3 — applied where relevant
├── Medium/Low: 8 — queued for batch reading
├── Protocols: 1 — integrated into agent behavior
└── Sync timestamp updated: 2026-06-07T23:41:00Z
```

---

## Workflow 3: "Score Pending Learnings" → Evaluate

Triggered:
- Immediately after storing a `severity >= high` learning
- During GC cycle (batch score all learnings with `score == 0.5` that are >24h old)
- On explicit agent command

### For a single learning

```
1. READ the learning record from ALMS: learning.get(learning_id)
2. BUILD the scoring prompt from prompts.md Section B
   - Inject: title, body, type, severity, existing tags list, near-dup IDs, supersedes chain
3. SEND to LLM (use appropriate model based on severity):
   - severity=critical: DeepSeek Pro + Gemini Pro (parallel, follow reconciliation rules)
   - severity=high: DeepSeek Flash (single pass)
   - severity=medium/low: skip scoring (TTL decay handles these)
4. PARSE LLM JSON response:
   - Validate: quality_score 1-5, valid verdict, dimension scores present
   - If response invalid: retry once with stricter instruction
5. APPLY result via learning score update:
   - Update quality_score field
   - If verdict=rejected and score < 0.3: consider soft-delete
   - If supersedes: handle supersession chain
   - If suggested_tags: append to learning record
6. LOG scoring to local state
```

### Batch scoring (GC cycle)

```
1. SEARCH: learning.search(query="", type="", tags=[], limit=500)
2. FILTER: records where score == 0.5 AND created_at < 24h_ago
3. BATCH: process up to 20 per scoring cycle (rate-limit to avoid cost)
4. For each: run Step 1-5 above
5. REPORT: "Scored 12/45 pending learnings — 8 accepted, 2 conditional, 2 rejected"
```

### Multi-model reconciliation (critical only)

When both DeepSeek and Gemini score the same critical learning:

| Scenario | Action |
|----------|--------|
| Both accept, delta < 0.5 | Average scores, take consensus tags |
| Both accept, delta >= 0.5 | Take lower score, merge tag suggestions |
| One accept, one reject | Conditonal pass, flag for human review |
| Both reject | Soft-delete with note: "LLM consensus: noise" |
| Models disagree on supersedes | Skip supersession, flag for manual audit |

---

## Workflow 4: "Stay Nudged" → Background Check

This is an optional background loop — don't run unless explicitly configured.

```
EVERY 30 MINUTES (or at heartbeat interval):
1. Call learning.sync(agent_id, since=last_sync_ts, type="failure", tags=[])
2. If any critical learnings returned:
   - Process immediately
   - Notify human via Telegram: "🚨 ALMS Critical: {title}"
3. If any protocols returned:
   - Process as mandatory instruction
4. ELSE: sync_ack and continue

EVERY 6 HOURS (or after task completion):
5. Run batch scoring on pending learnings (Workflow 3)
```

This is deliberately low-frequency. ALMS is not real-time. Don't poll it — let sync timestamps handle freshness.

---

## Prompts Reference

| Prompt | File | Section |
|--------|------|---------|
| Store | `alms/prompts/prompts.md` | Section A |
| Score | `alms/prompts/prompts.md` | Section B |
| Search | `alms/prompts/prompts.md` | Section C |
| Score Update | `alms/prompts/prompts.md` | Section D |
| Nudge | `alms/prompts/prompts.md` | Section E |

---

## Differences from `learning` skill

The existing [learning skill](../skills/learning/SKILL.md) governs the **filesystem-based** `learnings/` folder — single-agent, session-scoped, human-reading-oriented. This skill governs **ALMS** — multi-agent, persistent, MCP-tool-based.

| Aspect | learning skill | alms-agent skill (this) |
|--------|---------------|------------------------|
| Storage | Filesystem (`learnings/`) | PostgreSQL via MCP tools |
| Audience | Single agent (self) | All registered agents |
| Discovery | Session-start file scan | Sync cursor + timestamp |
| Dedup | Manual (naming convention) | Auto (SHA256 + Levenshtein) |
| Scoring | None | LLM-driven quality assessment |
| Protocols | Not supported | First-class via `protocols` table |
| Nudges | Not supported | Heartbeat-based alerts |

They are complementary: the filesystem skill is for quick capture; ALMS is for shared, durable, cross-agent knowledge.

---

## Anti-Patterns

| # | Anti-pattern | Detection | Fix |
|---|-------------|-----------|-----|
| 1 | **Store-first, think-later** — using ALMS as scratchpad | High store volume, low quality scores | Use filesystem `learnings/` for raw notes, ALMS for curated |
| 2 | **Polling too frequently** — syncing every minute | High server load, no new learnings | Sync on session start + task end only |
| 3 | **Dual-maintenance** — writing to both filesystem and ALMS | Duplicate work | Filesystem = first draft, ALMS = publish |
| 4 | **Ignoring protocols** — not processing protocol.type | Stale protocols count | Mandatory: all protocol learnings become agent instructions |
| 5 | **No ack** — syncing without acknowledging | Sync never advances | Always ack after processing; gap safety protects against crashes |
| 6 | **Over-tagging** — creating 50+ unique tags | Tag explosion in metadata | Max 1 new tag per 10 learnings; prefer existing |

---

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-06-07 | Initial — 4 workflows, prompt cross-references, anti-patterns |

## Related

- [ALMS Prompt Library](../prompts/prompts.md)
- [ALMS Data Model](../docs/data-model.md)
- [ALMS MCP Tools](../internal/server/tools.go)
- [Learning Skill (filesystem)](../../skills/learning/SKILL.md) — complementary
- [PM Orchestrator Skill](../../skills/pm-orchestrator/SKILL.md) — multi-model scoring pattern inspiration
