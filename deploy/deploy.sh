#!/bin/bash
set -euo pipefail

REMOTE_HOST="${1:-data}"
REMOTE_DIR="/opt/alms"
BINARY="bin/alms-linux"

echo "==> Building Linux binary (CGO_ENABLED=0)..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -ldflags="-X main.Version=$(git describe --tags 2>/dev/null || echo dev) -X main.Commit=$(git rev-parse --short HEAD)" \
    -o "$BINARY" ./cmd/alms/

echo "==> Ensuring remote dir exists..."
ssh "$REMOTE_HOST" "mkdir -p $REMOTE_DIR"

echo "==> Copying binary to ${REMOTE_HOST}:${REMOTE_DIR}/alms..."
scp "$BINARY" "${REMOTE_HOST}:${REMOTE_DIR}/alms"

echo "==> Copying config to ${REMOTE_HOST}:${REMOTE_DIR}/alms.yaml..."
scp deploy/alms.yaml "${REMOTE_HOST}:${REMOTE_DIR}/alms.yaml"

echo "==> Copying systemd unit..."
scp deploy/alms.service "${REMOTE_HOST}:${REMOTE_DIR}/alms.service"

echo "==> Copying migrations..."
scp -r internal/store/migrations "${REMOTE_HOST}:${REMOTE_DIR}/migrations"

echo "==> Installing systemd unit and restarting..."
ssh "$REMOTE_HOST" "
    sudo cp ${REMOTE_DIR}/alms.service /etc/systemd/system/alms.service
    sudo systemctl daemon-reload
    sudo systemctl enable alms
    sudo systemctl restart alms
    echo '--- alms status ---'
    sudo systemctl status alms --no-pager 2>&1 | head -20
"

echo "==> Verifying service is active..."
ssh "$REMOTE_HOST" "systemctl is-active alms"

echo "==> Done."
echo ""
echo "If migrations haven't been run yet, SSH in and run:"
echo "  migrate -path /opt/alms/migrations -database \"postgres://alms:alms@localhost:5432/alms_db?sslmode=disable\" up"
