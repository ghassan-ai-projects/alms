# ALMS Integration Guide

## OpenClaw MCP Integration

ALMS runs as a Streamable HTTP MCP server on the data machine (`192.168.2.112:8001`).

### Registering ALMS as OpenClaw MCP Server

Apply the included patch:

```bash
openclaw config patch --file deploy/openclaw-mcp-patch.json5
```

Or manually add to `~/.openclaw/openclaw.json`:

```json
{
  "mcp": {
    "servers": {
      "alms": {
        "url": "http://192.168.2.112:8001/mcp",
        "transport": "streamable-http",
        "headers": {
          "X-ALMS-TOKEN": ""
        },
        "enabled": true
      }
    }
  }
}
```

Then verify:

```bash
openclaw config validate
# Ensure no errors, then restart gateway
openclaw gateway restart
```

### Verify Tools

Once connected, you should see 16 tools:

```
agent.register   — Register a new agent
agent.unregister — Unregister an agent
agent.update     — Update agent details
agent.list       — List all agents
agent.heartbeat  — Send agent heartbeat
learning.sync    — Pull new learnings for an agent
learning.sync_ack — Acknowledge received learnings (gap-safe)
learning.store   — Create a new learning (with dedup)
learning.search  — Full-text search via GIN index
learning.delete  — Soft-delete a learning
learning.get     — Get single learning by ID
protocol.pull    — Pull active protocols for an agent
protocol.pull_since — Pull protocols since a given ID
protocol.push    — Create a new protocol
protocol.list    — List all protocols
health.check     — Server health check
```

## Agent Workflow

### Registration on Startup

When an agent starts, it should:

1. Call `agent.register` with its agent_id and type
2. Call `learning.sync` to pull any new learnings since its last sync
3. Call `protocol.pull` to get any active protocols
4. Process learnings, then call `learning.sync_ack` to confirm receipt

### Periodic Heartbeat

Agents should call `agent.heartbeat` every 5 minutes.

### Pushing Learnings

After processing, agents can push new learnings:

```bash
curl -X POST http://192.168.2.112:8001/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"learning.store","arguments":{"agent_id":"my-agent","title":"Found pattern","body":"Details...","type":"pattern","tags":["discovery"]}}}'
```

### Example: Newsletter Agent Lifecycle

See `deploy/newsletter-agent-example.sh` for a complete example.

## MCP API Details

### Auth Header

If configured, send `X-ALMS-TOKEN: <token>` with every request.
In dev mode (empty token), auth is bypassed.

### Error Handling

Errors return MCP JSON-RPC error responses:
```json
{"jsonrpc":"2.0","id":1,"error":{"code":-32001,"message":"unauthorized"}}
```

### Dashboard

A web dashboard is available at `http://192.168.2.112:8001/dashboard`.
