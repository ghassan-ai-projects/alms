# ALMS Prompts — Agent ↔ ALMS Protocol

These prompts define the standard messages agents send to the ALMS MCP server and to LLMs for scoring. They are the contract layer — every agent interacting with ALMS should use these wordings for consistency.

---

## A. Store Prompt — "Record a new learning"

**When to use:** You discovered something worth storing that took >5 minutes to figure out, uncovered a non-obvious insight, or hit a significant issue/workaround.

**Protocol:** Call the `learning.store` MCP tool on the ALMS server with this payload structure.

```json
{
  "agent_id": "<your-agent-id>",
  "title": "Short, specific title (max 80 chars, kebab-flavored)",
  "body": "Full context: what happened, what you expected, what actually happened, root cause if known, exact steps to reproduce or apply the fix",
  "type": "pattern|failure|config|protocol|edge_case",
  "tags": ["existing-tag-1", "existing-tag-2", "new-tag-if-needed"],
  "supersedes": ""  // optional — set if this learning replaces an obsolete one
}
```

### What to store

| Signal | Store? |
|--------|--------|
| API returns unexpected error | ✅ Yes (type: `failure`) |
| Found workaround for tool limitation | ✅ Yes (type: `pattern`) |
| Discovered config parameter matters | ✅ Yes (type: `config`) |
| Learning that changes how agents should operate | ✅ Yes (type: `protocol`) |
| Rare condition that tripped you up | ✅ Yes (type: `edge_case`) |
| "Obvious" insight that wasn't documented | ✅ Yes (err on the side of capture) |
| Duplicate of something you just stored | ❌ No — skip, ALMS handles dedup |
| Normal operation, no surprises | ❌ No — this is signal, not noise |
| Personal preference (not generalizable) | ❌ No — not cross-agent useful |
| "I think this might be wrong but haven't confirmed" | ❌ No — wait for confirmation |

### Tagging guidelines

- **Prefer existing tags** from the tag registry (see Search below). New tags proliferate noise.
- Only create a new tag if no existing tag fits AND you can imagine 3+ learnings using it.
- Tags are lowercase, snake_case, no spaces.
- Required tag for all learnings: the project/domain name (e.g., `alms`, `film-pipeline`, `personal-website`).

### Scoring tier (set on creation)

| Scenario | Severity |
|----------|----------|
| Blocks work, security issue, data loss risk | `critical` |
| Significant speed bump, wrong assumption corrected | `high` |
| Good to know, useful pattern | `medium` |
| Minor note, trivia | `low` |

---

## B. Score Prompt — "Evaluate this learning"

**When to use:** A learning has been created with default score (0.5) and needs LLM-based quality assessment. Triggers:
- On creation of any learning with `severity >= high`
- Batch scoring in GC cycle for learnings with `score == 0.5` older than 24h
- On explicit agent request via `learning.score` call

**What the LLM receives:**

```
You are a learning quality assessor. You evaluate learning records for the ALMS
(Agent Learning Management System) shared knowledge base.

Your task: Score the following learning record and decide whether it should be
accepted into the permanent knowledge base or rejected as noise/transient.

## Available Tags
<existing-tags>
</existing-tags>

## Learning Record
Title: {title}
Type: {type}
Severity: {severity}
Body: {body}

## Scoring Rubric

Score each dimension 1-5:

1. REUSABILITY: Can another agent (not just this one) benefit from this?
   1 = Hyper-specific to this session, 3 = Broadly applicable, 5 = Universal pattern
2. SPECIFICITY: Does it provide actionable detail or just vague advice?
   1 = "Be careful with X", 3 = Specific steps for one case, 5 = Steps + edge cases + example
3. ACCURACY: Is the claim well-supported? (Infer from body quality)
   1 = Speculative/unverified, 3 = Reasonable claim, 5 = Verified + reproducible
4. NOVELTY: Is this already covered by existing learnings?
   1 = Direct duplicate, 3 = Novel angle, 5 = Entirely new ground
5. CLARITY: Can it be understood in 30 seconds?
   1 = Stream of consciousness, 3 = Clear but verbose, 5 = Concise and self-contained

## Dedup Context
- Near-duplicate IDs (if any): {near_duplicates}
- Supersedes chain: {supersedes}

## Decision Rules
- ACCEPT: quality_score >= 3.0 AND no dimension < 2
- REJECT: quality_score < 2.5 OR any dimension = 1
- CONDITIONAL: quality_score >= 2.5 but < 3.0 (accept with low priority)

## Response Format
Respond with ONLY valid JSON:

```json
{
  "quality_score": <float 1.0-5.0>,
  "verdict": "accepted|rejected|conditional",
  "scores": {
    "reusability": <1-5>,
    "specificity": <1-5>,
    "accuracy": <1-5>,
    "novelty": <1-5>,
    "clarity": <1-5>
  },
  "suggested_tags": ["<existing tag only>", "<another>"],
  "duplicate_of": "<learning_id or null>",
  "supersedes": "<learning_id or null>",
  "explanation": "<1-2 sentence rationale>"
}
```
```

**What happens with the result:** The calling agent calls the score update function (see Prompt D) to persist the assessment.

### Multi-model scoring (high severity only)

For `severity = critical` learnings, run the scoring prompt against **two LLMs in parallel** (e.g., DeepSeek + Gemini), then:

1. If both agree (same verdict): accept the consensus
2. If one ACCEPTS and one REJECTS: take the lower score, mark as CONDITIONAL
3. If verdicts conflict and quality_score delta > 1.0: escalate to human review

This mirrors the PM Orchestrator's multi-model quality gate pattern, tuned for learning assessment rather than PM artifacts.

---

## C. Search Prompt — "Find fresh learnings"

**When to use:** On session startup, after completing a task (to see if anything relevant was added while you worked), or when entering a new project domain.

**Protocol:** Call `learning.sync` with your agent ID and last-known sync timestamp.

```
I need to check ALMS for learnings I haven't seen yet.

Parameters:
  agent_id:     <your-agent-id>
  since:        <timestamp of last sync from agent state>
  type:         "" (optional — filter to one learning type)
  tags:         [] (optional — filter to specific tags)

Steps:
1. Call learning.sync with these parameters
2. For each returned learning record:
   a. If severity = critical: process immediately
   b. If tagged with my project/domain: read and assimilate
   c. If type = protocol: store as actionable instruction
   d. Otherwise: queue for batch reading
3. Call learning.sync_ack with the list of all returned learning_ids
4. If sync_ack returns a gap error, re-sync from the gap timestamp
```

### Filter strategies by context

| Context | `since` | `type` | `tags` | Action |
|---------|---------|--------|--------|--------|
| Session startup | Last session end | (all) | (my projects) | Full catch-up |
| Before task X | 24h ago | (all) | project_x | Relevance check |
| In crisis mode | 1h ago | failure | (all) | Critical alerts |
| GC cycle check | 7 days ago | (all) | (all) | Audit batch |

### Sync gap safety

ALMS enforces gap-safe acknowledgment: if you miss a learning in your ack list, the server rejects the whole batch. This prevents data loss on crash.

- Always ack the **complete** list of returned IDs
- If you crash mid-sync, just re-sync — unacknowledged learnings will reappear
- Don't try to ack partial batches; re-sync from the gap is cheaper and safer

---

## D. Score Update Prompt — "Record LLM scoring result"

**When to use:** After the Score Prompt produces its JSON verdict, call this to update the learning record.

This isn't a prompt per se — it's an update protocol. But without it, the scoring pipeline is dead-ended.

**Protocol:** ALMS should expose a `learning.score_update` tool (or the agent updates the learning record directly via the store). The payload:

```json
{
  "learning_id": "<uuid>",
  "quality_score": 3.7,
  "verdict": "accepted",
  "scored_by": "deepseek-v4",
  "dimension_scores": {
    "reusability": 4,
    "specificity": 3,
    "accuracy": 4,
    "novelty": 3,
    "clarity": 4
  },
  "suggested_tags": ["pattern", "api-workaround"],
  "duplicate_of": null,
  "supersedes": "lrn-abc123"
}
```

**This is a gap in the current ALMS API.** The `learning.store` tool doesn't accept external quality scores. A new tool (`learning.score_update`) or a field addition to `learning.store` is needed.

---

## E. Nudge Prompt — "Alert agent about high-priority learning"

**When to use:** ALMS has a high-severity or high-scoring learning that an agent hasn't seen yet, and the agent's last sync was >N hours ago.

**Sender:** ALMS server → agent (push or during next agent heartbeat)

```
ALMS Nudge

A new learning matching your project tags was stored <time_ago> ago and
you haven't synced it yet:

  Title:     {title}
  Type:      {type}
  Severity:  {severity}
  Quality:   {quality_score}
  Matching:  {matched_tags}

Recommendation:
- Sync now: learning.sync(agent_id={id}, since={last_sync})
- If severity=critical and not synced within 15 minutes: escalate to human
```

This requires ALMS to track per-agent sync timestamps (already done in `agent.last_sync_ts`) and expose a "pending for you" endpoint or tool.

---

## Prompt Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2026-06-07 | Initial — Store, Score, Search, Score Update, Nudge |

## Related

- [ALMS Data Model](../docs/data-model.md)
- [ALMS Architecture](../docs/architecture.md)
- [ALMS MCP Tools](../../internal/server/tools.go)
- [ALMS Scoring Engine](../../internal/service/scoring.go)
- [ALMS Dedup Engine](../../internal/service/dedup.go)
- [Learning Skill (OpenClaw)](../../skill/SKILL.md)
