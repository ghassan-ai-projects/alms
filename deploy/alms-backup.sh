#!/bin/bash
# ALMS daily backup script
# Installed to /opt/alms/backup/alms-backup.sh, runs via cron at 03:00 daily
set -euo pipefail

BACKUP_DIR="/opt/alms/backup"
DB_NAME="alms_db"
DB_USER="alms"
DB_HOST="localhost"
RETENTION_DAYS=7
TIMESTAMP=$(date +%Y-%m-%d-%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/alms-${TIMESTAMP}.sql.gz"

echo "==> Starting ALMS backup: ${TIMESTAMP}"

# Ensure backup dir exists
mkdir -p "${BACKUP_DIR}"

# Dump and compress
export PGPASSWORD="${ALMS_PG_PASSWORD:-alms}"
pg_dump -U "${DB_USER}" -h "${DB_HOST}" "${DB_NAME}" | gzip > "${BACKUP_FILE}"

# Verify
if [ -s "${BACKUP_FILE}" ]; then
    SIZE=$(stat -c%s "${BACKUP_FILE}" 2>/dev/null || stat -f%z "${BACKUP_FILE}" 2>/dev/null)
    echo "==> Backup created: ${BACKUP_FILE} (${SIZE} bytes)"
else
    echo "ERROR: Backup file is empty!" >&2
    exit 1
fi

# Cleanup old backups
find "${BACKUP_DIR}" -name "alms-*.sql.gz" -mtime +${RETENTION_DAYS} -delete

echo "==> Backup complete. Retention: ${RETENTION_DAYS} days"
