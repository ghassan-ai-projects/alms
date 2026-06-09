# Skills And Prompts

## Why They Matter

The prompt library and skill definition are part of ALMS’s integration surface. They document how agents are expected to interact with the MCP server, not just how humans read about it.

## Prompt Library

Files:

- [../prompts/README.md](../prompts/README.md)
- [../prompts/prompts.md](../prompts/prompts.md)

The prompt library currently covers:

- storing a new learning
- scoring a learning
- searching and syncing learnings
- updating scoring results
- nudging agents about important learnings

## Skill Definition

File:

- [../skill/SKILL.md](../skill/SKILL.md)
- [../skill/alms-learning-SKILL-v2.1.md](../skill/alms-learning-SKILL-v2.1.md)

The skill packages the prompt patterns into repeatable agent workflows such as:

- store a discovery
- sync fresh knowledge
- evaluate learnings
- perform background checks

The copied `alms-learning-SKILL-v2.1.md` is now the default onboarding skill for new agents, with the helper scripts in `scripts/` serving as the recommended direct MCP bridge.

## Public Documentation Position

For open source, these files should be treated as:

- integration guidance for agent builders
- examples of the intended ALMS operating model
- part of the evolving contract between ALMS and agent clients

## Important Caveat

The prompt library currently references a `learning.score_update` style workflow, while the implemented server exposes `learning.update_enrichment`. That is a real contract mismatch to track after `0.1.0`; the docs should not hide it.

The remaining caution is narrower now: the learning skill and helper scripts are production-verified for sync and publish flows, but the enrichment/scoring contract still needs a later cleanup pass to align terminology and API shape.
