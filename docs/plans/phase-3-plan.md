# Phase 3 Plan — Deployment + Integration

## Current State

### What Already Exists (Phase 1+2 Complete)
- `make ci-check` passes (build → vet → lint → test)
- 16 MCP tools + 7 resources registered (exceeds 15-tool minimum)
- `deploy/alms.service` — systemd unit file (correct)
- `deploy/alms.yaml` — config with open bind + empty auth token
- `deploy/deploy.sh` — scaffold present but has bugs (typo `almps.service`, missing SCP of service file)
- Docker Compose for local PG dev
- Full test suite passes

### What's Missing
- No PostgreSQL on data machine (`pg_isready` → MISSING)
- No `alms` binary on data machine (NO_BINARY)
- Deploy script has bugs (typo, missing file copy)
- No acceptance test for Phase 3
- No newsletter agent agent registration pattern implemented
- No backup cron
- OpenClaw MCP config not updated for remote ALMS

## Implementation Steps

### Step 1: Fix deploy.sh bugs
- Fix `almps.service` → `alms.service` typo
- Add SCP of `alms.service` to remote
- Ensure script exits properly on failures
- Ensure `al-s` is used consistently (not `almps`)

### Step 2: Install PostgreSQL on data machine
- `ssh data "sudo apt-get update && sudo apt-get install -y postgresql postgresql-client"`
- `ssh data "sudo systemctl enable --now postgresql"`
- Create `alms` user and `alms_db` database
- Set password or trust auth for local connections

### Step 3: First deploy + migrate
- Run corrected `deploy/deploy.sh` to push binary, config, service file
- SSH in to run `migrate -path ... up` to create tables
- Verify `systemctl is-active alms` → active
- Verify `curl http://192.168.2.112:8001/mcp` from Mac works (tools/list)

### Step 4: OpenClaw MCP config
- Add `alms` server entry to `~/.openclaw/openclaw.json` under `mcp.servers`
- Use `url: "http://192.168.2.112:8001/mcp"` + `transport: "streamable-http"`
- Set `X-ALMS-TOKEN` header via `headers`
- Verify `tools/list` returns 16 tools from remote

### Step 5: Newsletter agent registration pattern
- Provide example agent registration script (curl-based or Go-based)
- Register `newsletter-scout` agent via `agent.register` tool
- Push a test learning via `learning.store`
- Sync + ack test

### Step 6: Backup cron
- Install daily `pg_dump` at 03:00 to `/opt/alms/backup/`
- 7-day retention logic
- Verify backup works

### Step 7: Acceptance test
- `phase-3-acceptance.sh` — tests deploy, PG setup, service health, remote MCP access, agent registration, learning sync, backup cron

## What Can Be Tested Without Real Deployment
- Unit tests for deploy.sh changes (dry-run verification)
- MCP config format validation (`openclaw config validate`)
- Agent registration tool works locally against Docker PG
- Backup cron script syntax verification
