# ALMS Deployment Guide

## Prerequisites

- macOS with SSH key access to `data` (192.168.2.112)
- `data` running Ubuntu 26.04
- SSH to `data` works passwordlessly

## Step 1: Setup Data Machine

SSH into `data` and run the setup script **interactively** (it needs sudo):

```bash
ssh -t data 'bash -s' < deploy/setup-data-machine.sh
```

This will:
- Install PostgreSQL 16
- Create `alms` user and `alms_db` database
- Create `/opt/alms/` directory
- Install `golang-migrate`

## Step 2: Deploy ALMS

From your Mac:

```bash
cd ~/ai-projects/alms
./deploy/deploy.sh
```

This will:
- Cross-compile for linux/amd64
- SCP binary, config, systemd unit, and migrations to data
- Install + enable + restart systemd service

## Step 3: Run Migrations

```bash
ssh data 'migrate -path /opt/alms/migrations \
  -database "postgres://alms:alms@localhost:5432/alms_db?sslmode=disable" up'
```

## Step 4: Verify

```bash
# Check service
ssh data 'systemctl status alms --no-pager'

# Test from Mac
curl -s -X POST http://192.168.2.112:8001/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' | jq .

# Run acceptance test
./phase-3-acceptance.sh
```

## Step 5: Connect OpenClaw

```bash
openclaw config patch --file deploy/openclaw-mcp-patch.json5
openclaw gateway restart
```

## Step 6: Install Backup Cron

```bash
ssh data 'sudo cp /opt/alms/backup/alms-backup.sh /opt/alms/backup/'
ssh data 'sudo cp deploy/cron-alms-backup /etc/cron.d/alms-backup'
ssh data 'sudo chmod 644 /etc/cron.d/alms-backup'
ssh data 'sudo chown root:root /etc/cron.d/alms-backup'
```

## Files Created/Modified

| File | Purpose |
|------|---------|
| `deploy/deploy.sh` | Cross-compile + SCP + systemctl reload (FIXED) |
| `deploy/setup-data-machine.sh` | One-time PG setup on data machine |
| `deploy/alms.service` | systemd unit (unchanged) |
| `deploy/alms.yaml` | Default config (unchanged) |
| `deploy/openclaw-mcp-patch.json5` | OpenClaw MCP config patch |
| `deploy/alms-backup.sh` | Daily pg_dump + gzip backup script |
| `deploy/cron-alms-backup` | Cron job definition (03:00 daily) |
| `deploy/newsletter-agent-example.sh` | Example agent lifecycle |
| `deploy/README.md` | This file |
| `phase-3-acceptance.sh` | Phase 3 acceptance test |
| `docs/integration-guide.md` | MCP integration guide |
| `docs/operations.md` | Operations runbook |
