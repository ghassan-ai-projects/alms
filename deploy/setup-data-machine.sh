#!/bin/bash
# Run this script interactively on the data machine (192.168.2.112)
# One-time setup: installs PostgreSQL, creates DB/user, runs migrations, installs backup cron
set -euo pipefail

echo "============================================"
echo "ALMS — Data Machine Setup"
echo "============================================"

# --- Install PostgreSQL ---
echo ""
echo "==> Installing PostgreSQL..."
sudo apt-get update -qq
sudo apt-get install -y -qq postgresql postgresql-client

echo "==> Starting PostgreSQL..."
sudo systemctl enable --now postgresql

# --- Create database and user ---
echo ""
echo "==> Creating alms database user and database..."
sudo -u postgres psql -c "CREATE USER alms WITH PASSWORD 'alms';" 2>/dev/null || echo "User 'alms' already exists"
sudo -u postgres psql -c "CREATE DATABASE alms_db OWNER alms;" 2>/dev/null || echo "Database 'alms_db' already exists"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE alms_db TO alms;" 2>/dev/null || true
# Grant schema permissions (pg 15+ requires explicit schema grants)
sudo -u postgres psql -d alms_db -c "GRANT ALL ON SCHEMA public TO alms;" 2>/dev/null || true

echo ""
echo "==> Verifying connection..."
PGPASSWORD=alms psql -U alms -d alms_db -h localhost -c "SELECT 1 AS connected;" || {
    echo "ERROR: Cannot connect as alms user. Check pg_hba.conf."
    exit 1
}

# --- Install migration tool ---
echo ""
echo "==> Installing golang-migrate..."
if ! which migrate &>/dev/null; then
    # Try apt, then curl download
    sudo apt-get install -y -qq golang-migrate 2>/dev/null || {
        echo "Installing migrate from GitHub..."
        curl -sL https://github.com/golang-migrate/migrate/releases/download/v4.18.1/migrate.linux-amd64.tar.gz | \
            sudo tar xz -C /usr/local/bin migrate
    }
fi

# --- Verify /opt/alms exists ---
echo ""
echo "==> Creating /opt/alms directory..."
sudo mkdir -p /opt/alms/backup
sudo chown -R ghassan:ghassan /opt/alms

echo ""
echo "============================================"
echo "Data machine setup complete!"
echo ""
echo "Next steps from your Mac:"
echo "  1. cd ~/ai-projects/alms && ALMS_PG_DSN=postgres://alms:alms@localhost:5432/alms_db?sslmode=disable ./deploy/deploy.sh"
echo "  2. ssh data 'migrate -path /opt/alms/migrations -database \"postgres://alms:alms@localhost:5432/alms_db?sslmode=disable\" up'"
echo "============================================"
