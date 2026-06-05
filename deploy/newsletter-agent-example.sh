#!/bin/bash
# Example: Newsletter agent registration + learning sync via ALMS MCP
# Demonstrates the lifecycle: register → store learning → sync → ack
# Requires: curl, jq

set -euo pipefail

ALMS_URL="${ALMS_URL:-http://192.168.2.112:8001/mcp}"
ALMS_TOKEN="${ALMS_TOKEN:-}"

HEADERS=(-H "Content-Type: application/json")
if [ -n "$ALMS_TOKEN" ]; then
    HEADERS+=(-H "X-ALMS-TOKEN: $ALMS_TOKEN")
fi

mcp_call() {
    local method="$1"
    local params="${2:-{}}"
    local id="${3:-1}"

    curl -s "${HEADERS[@]}" -X POST "$ALMS_URL" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":$id,\"method\":\"$method\",\"params\":$params}" | jq .
}

echo "============================================"
echo "Newsletter Agent — ALMS Registration Example"
echo "============================================"

# 1. Register the newsletter agent
echo ""
echo "==> 1. Registering newsletter-scout agent..."
mcp_call "tools/call" '{"name":"agent.register","arguments":{"agent_id":"newsletter-scout","agent_type":"mcp_client","display_name":"Newsletter Scout Agent"}}' 10

# 2. Send a heartbeat
echo ""
echo "==> 2. Sending heartbeat..."
mcp_call "tools/call" '{"name":"agent.heartbeat","arguments":{"agent_id":"newsletter-scout"}}' 11

# 3. Store a learning
echo ""
echo "==> 3. Storing a learning..."
mcp_call "tools/call" '{"name":"learning.store","arguments":{"agent_id":"newsletter-scout","title":"Newsletter content scheduling pattern","body":"Best time to send newsletters is Tuesday 10am UTC based on 30-day analysis of open rates.","type":"pattern","tags":["newsletter","scheduling","engagement"]}}' 12

# 4. Push a protocol
echo ""
echo "==> 4. Pushing a protocol..."
mcp_call "tools/call" '{"name":"protocol.push","arguments":{"title":"Newsletter format requirements","body":"All newsletters must include: (1) TLDR section, (2) 2-3 key topics, (3) CTA at end.","target_tags":["newsletter"]}}' 13

# 5. List all agents
echo ""
echo "==> 5. Listing all agents..."
mcp_call "tools/call" '{"name":"agent.list","arguments":{}}' 14

# 6. List tools
echo ""
echo "==> 6. Listing MCP tools..."
mcp_call "tools/list" '{}' 15

# 7. Pull protocols for this agent
echo ""
echo "==> 7. Pulling protocols..."
mcp_call "tools/call" '{"name":"protocol.pull","arguments":{"agent_id":"newsletter-scout"}}' 16

# 8. Search learnings
echo ""
echo "==> 8. Searching learnings..."
mcp_call "tools/call" '{"name":"learning.search","arguments":{"query":"newsletter scheduling","limit":10}}' 17

echo ""
echo "============================================"
echo "Newsletter agent registration complete!"
echo "============================================"
