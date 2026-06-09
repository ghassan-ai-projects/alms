# How ALMS Works

## Runtime Shape

ALMS is a single Go binary that exposes an MCP server over Streamable HTTP and persists state in PostgreSQL.

High-level flow:

1. Agents register with `agent.register`.
2. Agents send heartbeats with `agent.heartbeat`.
3. Agents publish learnings with `learning.store`.
4. Other agents pull new learnings with `learning.sync`.
5. Agents confirm completed processing with `learning.sync_ack`.
6. Operators publish instructions with `protocol.push`.
7. Agents retrieve relevant instructions with `protocol.pull`.

## Core Components

### Agent Registry

The registry tracks:

- `agent_id`
- `agent_type`
- display name
- last heartbeat
- sync-related timestamps

The registry does not assign work or manage processes.

### Learning Store

The learning store holds cross-agent knowledge records. Each learning includes a type, title, body, tags, authoring context, and lifecycle metadata. Records are soft-deleted rather than hard-deleted so historical sync behavior stays coherent.

### Protocol Store

Protocols are operator-authored instructions associated with target tags. They let you shape agent behavior without baking every instruction directly into every prompt or skill.

## Sync Model

The sync contract currently works like this:

1. The client calls `learning.sync` with an `agent_id` and a `since` timestamp.
2. The server returns matching learnings ordered by creation time.
3. The client processes the batch.
4. The client calls `learning.sync_ack` with the full list of returned learning IDs.

This gives ALMS a gap-safe acknowledgement step, but the project should still treat sync semantics as an area to refine after `0.1.0`, because the client-provided `since` cursor and server-side acknowledgement state overlap conceptually.

## Data Model

The main persisted entities are:

- `agents`
- `learnings`
- `learning_acknowledgements`
- `protocols`

The implementation uses PostgreSQL JSONB for flexible metadata and GIN-backed search structures for learnings.

## Background Work

The server starts a background garbage collection service at startup. It is responsible for lifecycle cleanup decisions around stored learnings according to the service configuration.

## Request Path

The code follows a strict layered flow:

1. `internal/server` validates and translates MCP requests.
2. `internal/service` enforces business rules and orchestration.
3. `internal/store` performs PostgreSQL reads and writes.
4. `internal/models` defines shared data structures.

That split is one of the project’s strongest maintainability properties and should be preserved.
