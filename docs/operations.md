# ALMS Operations

## Architecture

- **Server:** `data` (192.168.2.112, Ubuntu 26.04)
- **Binary:** `/opt/alms/alms`
- **Config:** `/opt/alms/alms.yaml`
- **Service:** `alms.service` (systemd)
- **Database:** PostgreSQL 16 on localhost
- **Data dir:** `/opt/alms/backup/`

## Deploy

### Quick Deploy

```bash
cd ~/ai-projects/alms
./deploy/deploy.sh
```

This cross-compiles for linux/amd64, SCPs binary + config + systemd unit, and reloads the service.

### One-Time Setup (Data Machine)

Run this interactively on `data`:

```bash
ssh -t data 'bash -s' < deploy/setup-data-machine.sh
```

Then from Mac:

```bash
cd ~/ai-projects/alms
ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable" ./deploy/deploy.sh
ssh data 'migrate -path /opt/alms/migrations -database "postgres://alms:alms@localhost:5432/alms_db?sslmode=disable" up'
```

### Deploy Without Setup

If PostgreSQL is already running and configured:

```bash
cd ~/ai-projects/alms
./deploy/deploy.sh
ssh data 'migrate -path /opt/alms/migrations -database "$ALMS_PG_DSN" up'
systemctl restart alms
```

## Service Management

```bash
# Status
systemctl status alms

# Logs
journalctl -u alms -f
journalctl -u alms --since "5 min ago"

# Restart
systemctl restart alms

# Stop
systemctl stop alms

# Start after maintenance
systemctl start alms
```

## Backup

### Automatic (cron)

Daily backup at 03:00 via `/etc/cron.d/alms-backup`:
- Dumps to `/opt/alms/backup/alms-YYYY-MM-DD-HHMMSS.sql.gz`
- Retains 7 days of backups
- Logs to `/opt/alms/backup/backup.log`

### Manual

```bash
/opt/alms/backup/alms-backup.sh
```

### Restore

```bash
gunzip -c /opt/alms/backup/alms-2026-06-06-030000.sql.gz | \
  psql -U alms -h localhost alms_db
```

## Monitoring

### Health Check

```bash
curl -s -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"health.check","arguments":{}}}'
```

### Dashboard

Open `http://192.168.2.112:8001/dashboard` in a browser.

### Disk Space

```bash
df -h /opt/alms
du -sh /opt/alms/backup/
du -sh /var/lib/postgresql/
```

## PostgreSQL Maintenance

```bash
# Vacuum
psql -U alms -d alms_db -c "VACUUM ANALYZE;"

# Connection count
psql -U alms -d alms_db -c "SELECT count(*) FROM pg_stat_activity;"

# Table sizes
psql -U alms -d alms_db -c "
SELECT relname, n_live_tup, n_dead_tup,
  pg_size_pretty(pg_total_relation_size(relid))
FROM pg_stat_user_tables
ORDER BY n_live_tup DESC;
"
```

## Migration Rollback

```bash
migrate -path /opt/alms/migrations -database "$ALMS_PG_DSN" down 1
```

## Troubleshooting

### Service fails to start

```bash
journalctl -u alms --no-pager -n 50
```

### Cannot connect to PostgreSQL

Check if PG is running:
```bash
systemctl status postgresql
pg_isready
```

Check auth in `/etc/postgresql/16/main/pg_hba.conf`:
```
# Should have: local all alms md5
```

### MCP connection refused from Mac

```bash
# Test from Mac
curl -s http://192.168.2.112:8001/mcp -X POST \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'

# Check firewall on data
ssh data 'sudo ufw status'
```
