# Agent Learning Integration

## Policy

Every new agent integrated with ALMS should include a learning workflow, not just a registration call.

Minimum expected behavior for a new ALMS-connected agent:

1. register itself with `agent.register`
2. check for relevant existing learnings before major work
3. capture meaningful new learnings during or after work
4. send those learnings back to ALMS with the push script or `learning.store`
5. periodically sync remote learnings with the fetch script or `learning.sync` and `learning.sync_ack`

An agent that only registers but never contributes or consumes learnings is using ALMS as a directory, not as a learning system.

## Recommended Default Skill

The copied skill file [../skill/alms-learning-SKILL-v2.1.md](../skill/alms-learning-SKILL-v2.1.md) is a good base pattern for new agents because it covers:

- pre-task learning retrieval
- post-task learning capture
- periodic sync
- enrichment/scoring awareness
- local cursor tracking

For open-source documentation, this should be described as the default learning skill new agents are expected to adopt or adapt.

## Included Helper Scripts

The repository now also contains:

- [../scripts/fetch-remote-learnings.py](../scripts/fetch-remote-learnings.py)
- [../scripts/push-local-learnings.py](../scripts/push-local-learnings.py)

These scripts are the recommended operational bridge for agents that keep a local `learnings/` directory and want repeatable sync and publish behavior without embedding custom HTTP logic in every agent.

They now call ALMS directly over MCP Streamable HTTP, initialize MCP sessions automatically, and were verified against the production ALMS deployment on 2026-06-09.

## Quality Assessment

### What Is Strong

- The skill has a concrete operating model instead of vague guidance.
- It correctly frames learning capture as part of normal agent work.
- It distinguishes ingest, push, sync, cursor tracking, and enrichment.
- The scripts are simple enough for contributors to understand and adapt quickly.

### Current Boundaries

- The push script uses conservative exact-title dedup by source agent before publish; ALMS server-side dedup remains the authority.
- The skill still distinguishes targeted `learning.search` from canonical `learning.sync`, which is correct, but implementers should keep that distinction explicit.
- The enrichment/scoring story still reflects the broader project mismatch between documented scoring concepts and the implemented `learning.update_enrichment` tool.

## Recommended Onboarding Pattern

When adding a new agent to ALMS, document and implement this baseline:

### Startup

1. call `agent.register`
2. run `python3 scripts/fetch-remote-learnings.py --search-query "<domain>"` for domain-specific prior knowledge
3. run `python3 scripts/fetch-remote-learnings.py --apply` for new global learnings since the agent cursor
4. process the batch

### During Work

- when the agent finds a reusable pattern, failure, protocol, config detail, or edge case, create a learning record

### End Of Task

1. review candidate learnings captured during the task
2. publish them with `python3 scripts/push-local-learnings.py --apply`
3. optionally update enrichment metadata if the agent also performs evaluation

## Documentation Position

For the public project, this workflow should be treated as a default integration expectation:

- new agents should not just register
- new agents should both consume and contribute learnings
- the learning skill is part of the product story, not an optional side note
