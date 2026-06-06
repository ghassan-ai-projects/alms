# Getting Started with ALMS

A step-by-step tutorial from zero to a running ALMS server with your first agent.

---

## Prerequisites

- Go 1.22+ (`go version`)
- Docker + Docker Compose (for PostgreSQL)
- `golang-migrate` CLI: `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

## 1. Start PostgreSQL

```bash
cd ~/ai-projects/alms
docker compose up -d db
```

Verify:
```bash
docker compose ps
# Name → alms-db-1, Status → Up
```

## 2. Configure

Set the database DSN (secrets go in env vars, not config files):

```bash
export ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable"
```

Optionally set an auth token:

```bash
export ALMS_AUTH_TOKEN="my-secret-token"
```

## 3. Run Migrations

```bash
migrate -path internal/store/migrations -database "$ALMS_PG_DSN" up
```

Expected output: `1/u initial (23.123ms)` — one migration applied.

## 4. Build and Start

```bash
# Build
go build -o bin/alms ./cmd/alms/

# Start
./bin/alms
```

You should see: `INFO starting server addr=127.0.0.1:8001`

## 5. Verify: List MCP Tools

In another terminal:

```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | jq '.result.tools | length'
```

You should get `16` — that's all MCP tools.

## 6. Register an Agent

```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: my-secret-token" \
  -d '{
    "jsonrpc":"2.0","id":2,"method":"tools/call",
    "params":{"name":"agent.register","arguments":{
      "agent_id":"newsletter-scout",
      "agent_type":"systemd",
      "display_name":"Newsletter Scout",
      "capabilities":{"tools":["web_search","web_fetch"],"skills":["newsletter","research"]}
    }}
  }' | jq .
```

## 7. Push a Learning

```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: my-secret-token" \
  -d '{
    "jsonrpc":"2.0","id":3,"method":"tools/call",
    "params":{"name":"learning.store","arguments":{
      "agent_id":"newsletter-scout",
      "title":"Site X needs custom HTML parser",
      "body":"Site X wraps content in malformed divs. Use html.parser with soup-strategy.",
      "type":"pattern",
      "tags":["scraping","workaround"],
      "severity":"medium"
    }}
  }' | jq .
```

## 8. Sync Learnings (as another agent)

```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -H "X-ALMS-TOKEN: my-secret-token" \
  -d '{
    "jsonrpc":"2.0","id":4,"method":"tools/call",
    "params":{"name":"learning.sync","arguments":{
      "agent_id":"research-agent",
      "since":"2026-01-01T00:00:00Z"
    }}
  }' | jq '.result.content[0].text | fromjson | length'
```

You should get `1` — the newsletter-scout's pattern is now available to research-agent.

## 9. View the Dashboard

Open `http://localhost:8001/dashboard` in a browser.

You'll see:
- Number of registered agents
- Recent learnings
- System health

## 10. Clean Up

```bash
# Stop server (Ctrl+C)
docker compose down   # Stops PostgreSQL
```

---

## What's Next

| Topic | Doc |
|-------|-----|
| Full agent lifecycle | [Integration Guide](integration-guide.md) |
| Data model details | [Data Model](data-model.md) |
| Production deploy | [Operations](operations.md) |
| Design decisions | [Architecture](architecture.md) |
