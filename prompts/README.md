# ALMS Prompt Library

Standardized prompts that agents use to interact with the ALMS MCP server.

## Architecture Decision Record

### Q1: Where do prompts live?
**Decision: Inside the ALMS repo** (`alms/prompts/`).

Rationale: Prompts define the *contract* between agents and ALMS. They belong with the system they interact with, not scattered among consuming skills. The ALMS API changes → prompts update in the same commit. Skills reference them by path.

### Q2: One file or three?
**Decision: One file** (`prompts.md`).

Rationale: Three files means three reads, three edit targets, harder to diff across versions. A single well-structured file with clear H2 sections is self-documenting and easier to maintain.

### Q3: What's missing from the 3 prompt categories?
**Decision: Add a 4th — "Score Update" callback prompt.**

The three original prompts (Store, Score, Search) are correct but incomplete. There's a missing feedback loop: after the LLM scores a pending learning, something needs to *apply* that score back to ALMS. ALMS's internal scoring engine (TTL decay, sync increment) is server-side; the LLM quality score is separate. The Score Update prompt handles: "learning X was scored by LLM as Y — update its quality_score field and set verdict."

Also missing: **"Nudge" prompt** — when ALMS detects an agent hasn't synced for >N hours, or a high-severity learning is pending, how to alert the agent.

### Q4: Standalone skill or chain from PM Orchestrator?
**Decision: Standalone skill, but references PM Orchestrator's quality-gate pattern.**

The PM Orchestrator is for product planning quality review — a different domain. ALMS learning is a universal agent protocol layer. A standalone `alms-agent` skill makes it invocable by any agent regardless of PM context. However, the LLM scoring prompt borrows the multi-model pattern from PM Orchestrator (parallel DeepSeek + Gemini scoring for high-severity learnings).

### Q5: JSON structured or Markdown templates?
**Decision: Markdown with embedded JSON schema blocks.**

Markdown for readability (agents can read and understand the intent). JSON schema blocks within each prompt define the exact I/O contract — the LLM can parse these to format its structured output. This is the PM Orchestrator's proven pattern: readable prose + machine-parseable examples.

## File Structure

```
prompts/
  README.md        ← this file
  prompts.md       ← the three core prompts + two extras
```
