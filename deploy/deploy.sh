#!/bin/bash
set -euo pipefail

REMOTE_HOST="${1:-data}"
REMOTE_DIR="/opt/alms"
BINARY="bin/alms-linux"

echo "==> Building Linux binary..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    -ldflags="-X main.Version=$(git describe --tags 2>/dev/null || echo dev) -X main.Commit=$(git rev-parse --short HEAD)" \
    -o "$BINARY" ./cmd/alms/

echo "==> Copying to ${REMOTE_HOST}:${REMOTE_DIR}/..."
scp "$BINARY" "${REMOTE_HOST}:${REMOTE_DIR}/alms"
scp deploy/alms.yaml "${REMOTE_HOST}:${REMOTE_DIR}/"

echo "==> Reloading systemd and restarting..."
ssh "$REMOTE_HOST" "
    sudo cp ${REMOTE_DIR}/alms.service /etc/systemd/system/almps.service 2>/dev/null || true
    sudo systemctl daemon-reload
    sudo systemctl restart alms
    sudo systemctl status alms --no-pager
"

echo "==> Done."
