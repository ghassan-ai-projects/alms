# Product Overview

## What ALMS Is

ALMS is an MCP server for agent coordination primitives:

- agent registry
- shared learning store
- protocol distribution

It exists to make autonomous agents less isolated from each other without forcing them into a shared runtime.

## What ALMS Is Not

ALMS is not:

- an agent framework
- a process supervisor
- a job scheduler
- a message bus
- a real-time collaboration system

That scope discipline is intentional. The server stores and exposes operational knowledge; it does not orchestrate the full lifecycle of your agents.

## Primary Use Case

The clearest public wedge for ALMS is shared operational memory for autonomous agents.

Example:

1. An ingestion agent discovers that a third-party API intermittently returns malformed payloads.
2. The agent stores that as a learning with tags and context.
3. A different agent syncs ALMS later and receives the learning.
4. The second agent avoids the same failure path without rediscovering it itself.

## Secondary Capabilities

- Registry: operators can see which agents are registered and sending heartbeats.
- Protocols: operators can publish instructions to agents grouped by tag.
- Search: agents and humans can query stored learnings later.

## Design Principles

- Offline-first: agents keep working without ALMS and resync later.
- Small dependency surface: Go, PostgreSQL, MCP, no ORM.
- Layered codebase: `server -> service -> store -> models`.
- Operational clarity: system behavior is explicit and inspectable.

## Who Should Use It

ALMS fits teams that:

- run more than one autonomous or semi-autonomous agent
- want learnings to persist outside a single process
- prefer self-hosted infrastructure
- already use MCP or want an MCP-native integration point

It is a weaker fit if you need task orchestration, real-time eventing, or a full multi-agent framework.
