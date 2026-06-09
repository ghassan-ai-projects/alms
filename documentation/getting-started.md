# Getting Started

## Prerequisites

- Go 1.22+
- Docker and Docker Compose
- `golang-migrate`
- `golangci-lint`

## Start PostgreSQL

```bash
docker compose up -d db
```

## Configure Environment

```bash
export ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"
export ALMS_AUTH_TOKEN="change-me"
```

The auth token is optional in code terms, but for any non-local use you should set one.

## Apply Migrations

```bash
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
```

## Build And Run

```bash
make build
./bin/alms
```

Expected startup behavior:

- the server connects to PostgreSQL
- the background GC service starts
- the MCP server listens on `127.0.0.1:8001` by default

## Verify The Server

List tools:

```bash
curl -s -X POST http://127.0.0.1:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
```

Register an agent:

```bash
curl -s -X POST http://127.0.0.1:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"agent.register","arguments":{"agent_id":"test-agent-1","agent_type":"systemd","display_name":"Test Agent"}}}'
```

Store a learning:

```bash
curl -s -X POST http://127.0.0.1:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: change-me" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"learning.store","arguments":{"agent_id":"test-agent-1","title":"HTML parser workaround","body":"Malformed upstream HTML requires lenient parsing before extraction.","type":"pattern","tags":["demo","html"]}}}'
```

## Verify The Script Path

The recommended agent integration path is the script layer rather than raw HTTP calls, because the scripts handle MCP Streamable HTTP session initialization automatically.

Example targeted lookup:

```bash
export ALMS_URL="http://127.0.0.1:8001/mcp"
export ALMS_AUTH_TOKEN="change-me"
export AGENT_ID="test-agent-1"
python3 scripts/fetch-remote-learnings.py --search-query "html parser"
```

Example publish from local markdown learnings:

```bash
export ALMS_URL="http://127.0.0.1:8001/mcp"
export ALMS_AUTH_TOKEN="change-me"
export AGENT_ID="test-agent-1"
python3 scripts/push-local-learnings.py --dir ./learnings --apply
```

## Next Reads

- [configuration.md](configuration.md)
- [mcp-api.md](mcp-api.md)
- [skills-and-prompts.md](skills-and-prompts.md)
- [agent-learning-integration.md](agent-learning-integration.md)
