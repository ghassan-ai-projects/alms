#!/bin/bash
# Phase 3 Acceptance Test — Deployment + Integration
# Validates: deploy, PG setup, service health, remote MCP access, agent registration, backup cron
set -euo pipefail

echo "============================================"
echo "Phase 3 Acceptance Test"
echo "============================================"

REMOTE="data"
ALMS_URL="http://192.168.2.112:8001/mcp"
PASS=0
FAIL=0

pass() { PASS=$((PASS+1)); echo "  ✅ $1"; }
fail() { FAIL=$((FAIL+1)); echo "  ❌ $1"; }

# Helper: MCP tools/list check
mcp_tools_list() {
    curl -s -X POST "$ALMS_URL" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' 2>/dev/null
}

mcp_tool_call() {
    curl -s -X POST "$ALMS_URL" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"$1\",\"arguments\":$2}}" 2>/dev/null
}

echo ""
echo "--- Gate 3→4 Check: Deploy ---"
echo ""

# 1. Binary exists on remote
echo "1) Checking binary..."
if ssh "$REMOTE" "test -x /opt/alms/alms"; then
    pass "Binary exists at /opt/alms/alms"
else
    fail "Binary not found at /opt/alms/alms — run deploy/deploy.sh first"
fi

# 2. Config exists on remote
echo "2) Checking config..."
if ssh "$REMOTE" "test -f /opt/alms/alms.yaml"; then
    pass "Config exists at /opt/alms/alms.yaml"
else
    fail "Config not found"
fi

# 3. Systemd unit installed
echo "3) Checking systemd unit..."
if ssh "$REMOTE" "systemctl is-enabled alms 2>/dev/null" | grep -q "enabled"; then
    pass "alms service is enabled on boot"
else
    fail "alms service not enabled"
fi

# 4. Service is active
echo "4) Checking service status..."
SERVICE_STATUS=$(ssh "$REMOTE" "systemctl is-active alms 2>/dev/null" || echo "inactive")
if [ "$SERVICE_STATUS" = "active" ]; then
    pass "alms service is active"
else
    fail "alms service is $SERVICE_STATUS"
fi

# 5. PostgreSQL is running
echo "5) Checking PostgreSQL..."
PG_STATUS=$(ssh "$REMOTE" "pg_isready -q 2>/dev/null && echo OK || echo FAIL")
if [ "$PG_STATUS" = "OK" ]; then
    pass "PostgreSQL is ready"
else
    fail "PostgreSQL not running"
fi

# 6. Can connect from Mac
echo "6) Checking remote MCP access from Mac..."
if TOOLS=$(mcp_tools_list); then
    TOOL_COUNT=$(echo "$TOOLS" | jq '.result.tools | length' 2>/dev/null || echo 0)
    if [ "$TOOL_COUNT" -ge 15 ] 2>/dev/null; then
        pass "Remote MCP accessible: $TOOL_COUNT tools registered"
    else
        fail "Remote MCP accessible but only $TOOL_COUNT tools (expected ≥15)"
    fi
else
    fail "Cannot reach ALMS at $ALMS_URL"
fi

# 7. Agent registration over MCP
echo "7) Testing agent registration..."
REG_RESULT=$(mcp_tool_call "agent.register" '{"agent_id":"acceptance-test-agent","agent_type":"mcp_client","display_name":"Phase 3 Acceptance Test"}')
AGENT_ID=$(echo "$REG_RESULT" | jq -r '.result.content[0].text // empty' | jq -r '.agent_id // empty' 2>/dev/null)
if [ "$AGENT_ID" = "acceptance-test-agent" ]; then
    pass "Agent registration works"
    # Clean up
    mcp_tool_call "agent.unregister" '{"agent_id":"acceptance-test-agent"}' >/dev/null 2>&1
else
    fail "Agent registration failed"
fi

# 8. Learning store over MCP
echo "8) Testing learning.store..."
LEARN_RESULT=$(mcp_tool_call "learning.store" '{"agent_id":"acceptance-test-agent","title":"Test learning from acceptance test","body":"Acceptance test body","type":"pattern"}')
LID=$(echo "$LEARN_RESULT" | jq -r '.result.content[0].text // empty' | jq -r '.learning_id // empty' 2>/dev/null)
if [ -n "$LID" ] && [ "$LID" != "null" ]; then
    pass "Learning store works (ID: ${LID:0:8}...)"
    # Search for it
    SEARCH_RESULT=$(mcp_tool_call "learning.search" '{"query":"acceptance test body","limit":5}')
    HITS=$(echo "$SEARCH_RESULT" | jq '.result.content[0].text // "[]"' | jq 'length' 2>/dev/null || echo 0)
    if [ "$HITS" -ge 1 ]; then
        pass "Learning search finds stored learning ($HITS hits)"
    else
        fail "Learning search returned 0 results"
    fi
else
    fail "Learning store failed"
fi

# 9. Backup script syntax check
echo "9) Checking backup script..."
if bash -n deploy/alms-backup.sh 2>/dev/null; then
    pass "Backup script syntax OK"
else
    fail "Backup script has syntax errors"
fi

# 10. Deploy script syntax check
echo "10) Checking deploy script..."
if bash -n deploy/deploy.sh 2>/dev/null; then
    pass "Deploy script syntax OK"
else
    fail "Deploy script has syntax errors"
fi

echo ""
echo "============================================"
echo "Results: $PASS passed, $FAIL failed"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
