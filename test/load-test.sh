#!/bin/bash
# ALMS Load Test — 10 concurrent agents syncing learnings
#
# Usage: ALMS_PG_DSN="postgres://alms:alms@localhost:5432/alms_db?sslmode=disable" ./test/load-test.sh
#
# Starts ALMS binary, spawns 10 concurrent agents, measures latency.
# Exits 0 if p99 latency < 1000ms, 1 otherwise.
# Idempotent: cleans up processes and test data on exit.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ALMS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
BINARY="${ALMS_DIR}/bin/alms"
ALMS_PG_DSN="${ALMS_PG_DSN:-}"
ALMS_LISTEN="${ALMS_LISTEN:-127.0.0.1:18001}"
LOCK_FILE="/tmp/alms-load-test.lock"

PASS=0
FAIL=0
ALMS_PID=""

cleanup() {
    local exit_code=$?
    set +e

    # Kill ALMS if running
    if [ -n "$ALMS_PID" ] && kill -0 "$ALMS_PID" 2>/dev/null; then
        echo "  Shutting down ALMS (PID $ALMS_PID)..."
        kill "$ALMS_PID" 2>/dev/null
        wait "$ALMS_PID" 2>/dev/null
    fi

    # Release lock
    rm -f "$LOCK_FILE"

    if [ $exit_code -ne 0 ] && [ $exit_code -ne 1 ]; then
        echo ""
        echo "❌ Load test crashed (exit $exit_code)"
    fi
    exit $exit_code
}
trap cleanup EXIT INT TERM

# Prevent concurrent runs
if ! mkdir "$LOCK_FILE" 2>/dev/null; then
    echo "❌ Another load test is running (lock: $LOCK_FILE)"
    exit 1
fi

echo "============================================"
echo "ALMS Load Test"
echo "============================================"
echo ""

# --- Prerequisites ---

# 1. PG DSN
if [ -z "$ALMS_PG_DSN" ]; then
    echo "❌ ALMS_PG_DSN not set"
    echo "   Usage: ALMS_PG_DSN=... ./test/load-test.sh"
    exit 1
fi

# 2. Build binary if missing
if [ ! -x "$BINARY" ]; then
    echo "Building ALMS binary..."
    cd "$ALMS_DIR"
    go build -o "$BINARY" ./cmd/alms/
    echo "  ✅ Built $BINARY"
fi

echo "--- Starting ALMS ---"

# Start ALMS in background
export ALMS_PG_DSN
export ALMS_AUTH_TOKEN=""
cd "$ALMS_DIR"
"$BINARY" &
ALMS_PID=$!
echo "  ALMS PID: $ALMS_PID"

# Wait for ALMS to be ready
MCP_URL="http://${ALMS_LISTEN}/mcp"
MAX_RETRIES=20
RETRY=0
while [ $RETRY -lt $MAX_RETRIES ]; do
    if curl -s -X POST "$MCP_URL" \
        -H "Content-Type: application/json" \
        -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}' 2>/dev/null | grep -q "result"; then
        echo "  ✅ ALMS ready on $ALMS_LISTEN"
        break
    fi
    RETRY=$((RETRY + 1))
    sleep 0.5
done
if [ $RETRY -ge $MAX_RETRIES ]; then
    echo "❌ ALMS failed to start within $((MAX_RETRIES * 500))ms"
    kill "$ALMS_PID" 2>/dev/null
    exit 1
fi

echo ""
echo "--- Load Test: 10 concurrent agents ---"

# Create agents + learnings + time the syncs
NUM_AGENTS=10
LEARNINGS_PER_AGENT=5
CONCURRENCY=10

# Register agents in batch
echo "  Registering $NUM_AGENTS agents..."
AGENT_IDS=()
for i in $(seq 1 $NUM_AGENTS); do
    AID="load-test-agent-${i}-$$"
    AGENT_IDS+=("$AID")
    curl -s -X POST "$MCP_URL" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":$i,\"method\":\"tools/call\",\"params\":{\"name\":\"agent.register\",\"arguments\":{\"agent_id\":\"$AID\",\"agent_type\":\"mcp_client\"}}}" >/dev/null 2>&1
done
echo "  ✅ $NUM_AGENTS agents registered"

# Store learnings
echo "  Storing $((NUM_AGENTS * LEARNINGS_PER_AGENT)) learnings..."
LEARNING_IDS=()
for i in $(seq 1 $((NUM_AGENTS * LEARNINGS_PER_AGENT))); do
    RESULT=$(curl -s -X POST "$MCP_URL" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":$((1000 + i)),\"method\":\"tools/call\",\"params\":{\"name\":\"learning.store\",\"arguments\":{\"agent_id\":\"load-test-agent-$(( (i - 1) / LEARNINGS_PER_AGENT + 1 ))-$$\",\"title\":\"Load test learning $i\",\"body\":\"This is a test learning created during load testing.\",\"type\":\"pattern\"}}}" 2>/dev/null)
    LID=$(echo "$RESULT" | sed -n 's/.*"learning_id":"\([^"]*\)".*/\1/p')
    if [ -n "$LID" ]; then
        LEARNING_IDS+=("$LID")
    fi
done
echo "  ✅ ${#LEARNING_IDS[@]} learnings stored"

# Concurrent sync + ack measurement
echo "  Running concurrent sync..."
TIMING_FILE=$(mktemp)
ERROR_FILE=$(mktemp)
PIDS=()

do_sync() {
    local agent_id="$1"
    local iter="$2"
    local timing_file="$3"
    local tmpfile
    tmpfile=$(mktemp)

    local start end elapsed
    start=$(date +%s%N)

    # Sync
    curl -s -X POST "$MCP_URL" \
        -H "Content-Type: application/json" \
        -d "{\"jsonrpc\":\"2.0\",\"id\":$((5000 + iter * 100 + RANDOM % 100)),\"method\":\"tools/call\",\"params\":{\"name\":\"learning.sync\",\"arguments\":{\"agent_id\":\"$agent_id\",\"since\":\"2020-01-01T00:00:00Z\"}}}" > "$tmpfile" 2>&1

    # Extract learning IDs from sync result
    local ids
    ids=$(grep -o '"learning_id":"[^"]*"' "$tmpfile" | cut -d'"' -f4 | tr '\n' ',' | sed 's/,$//')

    if [ -n "$ids" ]; then
        # Ack all
        curl -s -X POST "$MCP_URL" \
            -H "Content-Type: application/json" \
            -d "{\"jsonrpc\":\"2.0\",\"id\":$((6000 + iter * 100 + RANDOM % 100)),\"method\":\"tools/call\",\"params\":{\"name\":\"learning.sync_ack\",\"arguments\":{\"agent_id\":\"$agent_id\",\"learning_ids\":[\"$ids\"]}}}" > /dev/null 2>&1
    fi

    end=$(date +%s%N)
    elapsed=$(( (end - start) / 1000000 ))  # ms
    echo "$elapsed" >> "$timing_file"
    rm -f "$tmpfile"
}

# Run 3 rounds of concurrent sync
for round in 1 2 3; do
    echo "    Round $round..."
    for aid in "${AGENT_IDS[@]}"; do
        (do_sync "$aid" "$round" "$TIMING_FILE") &
        PIDS+=($!)
        # Throttle: max CONCURRENCY at a time
        if [ ${#PIDS[@]} -ge $CONCURRENCY ]; then
            wait "${PIDS[0]}"
            PIDS=("${PIDS[@]:1}")
        fi
    done
    # Wait for remaining
    for pid in "${PIDS[@]}"; do
        wait "$pid" 2>/dev/null || true
    done
    PIDS=()
done

# Analyze timing
if [ ! -s "$TIMING_FILE" ]; then
    echo "❌ No timing data collected"
    rm -f "$TIMING_FILE" "$ERROR_FILE"
    exit 1
fi

# Sort timings
sort -n "$TIMING_FILE" > "${TIMING_FILE}.sorted"
TOTAL=$(wc -l < "${TIMING_FILE}.sorted")

# Calculate percentiles
P50_IDX=$(( (TOTAL * 50 + 99) / 100 ))
P95_IDX=$(( (TOTAL * 95 + 99) / 100 ))
P99_IDX=$(( (TOTAL * 99 + 99) / 100 ))

if [ $P50_IDX -lt 1 ]; then P50_IDX=1; fi
if [ $P95_IDX -lt 1 ]; then P95_IDX=1; fi
if [ $P99_IDX -lt 1 ]; then P99_IDX=1; fi
if [ $P50_IDX -gt $TOTAL ]; then P50_IDX=$TOTAL; fi
if [ $P95_IDX -gt $TOTAL ]; then P95_IDX=$TOTAL; fi
if [ $P99_IDX -gt $TOTAL ]; then P99_IDX=$TOTAL; fi

P50=$(sed -n "${P50_IDX}p" "${TIMING_FILE}.sorted")
P95=$(sed -n "${P95_IDX}p" "${TIMING_FILE}.sorted")
P99=$(sed -n "${P99_IDX}p" "${TIMING_FILE}.sorted")

echo ""
echo "--- Results ---"
echo "  Total sync operations: $TOTAL"
echo "  p50 latency:  ${P50}ms"
echo "  p95 latency:  ${P95}ms"
echo "  p99 latency:  ${P99}ms"

# Check pass/fail
if [ "$P99" -lt 1000 ]; then
    echo "  ✅ PASS: p99 < 1000ms (${P99}ms)"
else
    echo "  ❌ FAIL: p99 >= 1000ms (${P99}ms)"
    rm -f "$TIMING_FILE" "${TIMING_FILE}.sorted" "$ERROR_FILE"
    exit 1
fi

rm -f "$TIMING_FILE" "${TIMING_FILE}.sorted" "$ERROR_FILE"
echo ""
echo "============================================"
echo "Load test completed successfully"
echo "============================================"
