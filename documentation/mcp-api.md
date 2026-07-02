# MCP API Reference

## Overview

ALMS exposes MCP tools for writes and workflow operations, plus MCP resources for lightweight read access.

Current public release: `0.1.0`

## Tools

ALMS currently registers 18 tools.

### Agent Tools

- `agent.register`: register a new agent
- `agent.unregister`: remove an agent
- `agent.update`: update agent metadata fields exposed by the tool
- `agent.list`: list agents with optional type filter and pagination
- `agent.heartbeat`: update heartbeat for an agent

### Learning Tools

- `learning.sync`: fetch learnings since a timestamp
- `learning.sync_ack`: acknowledge a complete sync batch
- `learning.store`: create a learning with dedup handling
- `learning.search`: full-text search with filters
- `learning.delete`: soft-delete a learning
- `learning.get`: fetch a learning by ID
- `learning.update_enrichment`: merge enrichment metadata for a learning

### Protocol Tools

- `protocol.pull`: get active protocols for an agent
- `protocol.pull_since`: get protocols after a known protocol ID
- `protocol.push`: create a protocol
- `protocol.list`: list protocols

### Health Tool

- `health.check`: return server health and agent count

### OKF Tool

- `okf.export`: export accepted, high-confidence ALMS learnings as an OKF v0.1 bundle payload

## Resources

ALMS currently registers 5 resources.

- `alms://agents`
- `alms://health`
- `alms://learnings`
- `alms://tools`
- `alms://protocols`

## Example Requests

### Register An Agent

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "agent.register",
    "arguments": {
      "agent_id": "research-agent",
      "agent_type": "systemd",
      "display_name": "Research Agent"
    }
  }
}
```

### Sync Learnings

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "learning.sync",
    "arguments": {
      "agent_id": "research-agent",
      "since": "2026-01-01T00:00:00Z",
      "type": "",
      "tags": ["research"]
    }
  }
}
```

### Acknowledge A Batch

```json
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "learning.sync_ack",
    "arguments": {
      "agent_id": "research-agent",
      "learning_ids": [
        "uuid-1",
        "uuid-2"
      ]
    }
  }
}
```

### Store A Learning

```json
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "learning.store",
    "arguments": {
      "agent_id": "research-agent",
      "title": "Retry malformed JSON responses",
      "body": "Some upstream responses include a trailing comma. Strip it before decoding.",
      "type": "failure",
      "tags": ["api", "json"]
    }
  }
}
```

### Export Learnings As OKF

```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "okf.export",
    "arguments": {
      "query": "retry malformed JSON",
      "status": "accepted",
      "min_score": 4.0,
      "tags": ["api"],
      "limit": 50
    }
  }
}
```

## API Notes

- `learning.search` requires a non-empty `query`.
- `learning.sync_ack` requires a non-empty ordered `learning_ids` list.
- `learning.update_enrichment` exists for enrichment and scoring workflows; this is relevant when integrating ALMS with asynchronous agent-side evaluation.
- `okf.export` returns a JSON payload containing OKF file paths and file contents. It does not write files to disk. By default it exports learnings with enrichment status `accepted` and score `>= 4.0`. `query` is optional; callers can export by tags, type, status, and score alone.

## Compatibility Note

The sync model is functional in `0.1.0`, but it should be treated as an early contract. External adopters should avoid assuming long-term stability around cursor semantics until a later release tightens the protocol.
