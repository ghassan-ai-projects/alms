#!/bin/bash
# Phase 4 Acceptance Test — CI, Integration, Load, Operations, Coverage
#
# Validates:
# 1. `make ci-check` passes
# 2. Integration tests pass (if PG available)
# 3. Load test passes (if PG available)
# 4. Operations doc exists
# 5. Coverage: service >80%, store >60%, server >40%
#
# Run: ./test/phase-4-acceptance.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ALMS_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ALMS_DIR"

PASS=0
FAIL=0
SKIP=0

pass() { PASS=$((PASS+1)); echo "  ✅ $1"; }
fail() { FAIL=$((FAIL+1)); echo "  ❌ $1"; }
skip() { SKIP=$((SKIP+1)); echo "  ⏭️  $1"; }

echo "============================================"
echo "Phase 4 Acceptance Test"
echo "============================================"
echo ""

# --- 1. CI Check ---
echo "--- 1. CI Check ---"
echo ""

# Must: go.mod tidy, build, vet, lint, test-short, deadcode, vulncheck
echo "  Running: go mod tidy..."
if go mod tidy 2>&1; then
    pass "go mod tidy"
else
    fail "go mod tidy"
fi

echo "  Running: go build..."
if go build -o /dev/null ./cmd/alms/ 2>&1; then
    pass "go build"
else
    fail "go build"
fi

echo "  Running: go vet..."
if go vet ./... 2>&1; then
    pass "go vet"
else
    fail "go vet"
fi

echo "  Running: golangci-lint..."
if golangci-lint run ./... --timeout=3m 2>&1; then
    pass "golangci-lint"
else
    fail "golangci-lint"
fi

echo "  Running: go test -short..."
if go test -short -race -count=1 -shuffle=on ./... 2>&1; then
    pass "go test -short"
else
    fail "go test -short"
fi

echo "  Running: deadcode..."
if command -v deadcode &>/dev/null; then
    if deadcode ./... 2>&1; then
        pass "deadcode"
    else
        fail "deadcode"
    fi
else
    skip "deadcode (not installed)"
fi

echo "  Running: govulncheck..."
if command -v govulncheck &>/dev/null; then
    if govulncheck ./... 2>&1; then
        pass "govulncheck"
    else
        fail "govulncheck"
    fi
else
    skip "govulncheck (not installed)"
fi

echo ""
echo "--- 2. Integration Tests ---"
echo ""

ALMS_PG_DSN="${ALMS_PG_DSN:-}"
if [ -n "$ALMS_PG_DSN" ]; then
    echo "  Running: go test -tags=integration -race -count=1 ./internal/integration/..."
    if go test -tags=integration -race -count=1 -v ./internal/integration/... 2>&1; then
        pass "Integration tests"
    else
        fail "Integration tests"
    fi
else
    skip "Integration tests (ALMS_PG_DSN not set)"
fi

echo ""
echo "--- 3. Load Test ---"
echo ""

if [ -n "$ALMS_PG_DSN" ]; then
    echo "  Running: test/load-test.sh (p99 < 1000ms)..."
    if ALMS_PG_DSN="$ALMS_PG_DSN" bash test/load-test.sh 2>&1; then
        pass "Load test"
    else
        fail "Load test"
    fi
else
    skip "Load test (ALMS_PG_DSN not set)"
fi

echo ""
echo "--- 4. Operations Doc ---"
echo ""

if [ -f docs/operations.md ]; then
    SIZE=$(wc -c < docs/operations.md)
    if [ "$SIZE" -gt 100 ]; then
        pass "docs/operations.md exists ($SIZE bytes)"
    else
        fail "docs/operations.md exists but is too small ($SIZE bytes)"
    fi
else
    fail "docs/operations.md is missing"
fi

echo ""
echo "--- 5. Coverage ---"
echo ""

echo "  Measuring coverage..."

# Measure each package individually for accurate per-package coverage
measure_pkg() {
    local path="$1"
    local output
    output=$(go test -race -count=1 -coverprofile=/dev/null "$path" 2>&1)
    # Extract coverage percentage from: "ok  	pkg	1.234s	coverage: X.X% of statements"
    echo "$output" | sed -n 's/.*coverage: \([0-9.]*\)%.*/\1/p' || echo "0"
}

SERVICE_COV=$(measure_pkg "./internal/service/")
STORE_COV=$(measure_pkg "./internal/store/")
SERVER_COV=$(measure_pkg "./internal/server/")
MODELS_COV=$(measure_pkg "./internal/models/")

echo "  service/  coverage: ${SERVICE_COV}%  (target: >80%)"
echo "  store/    coverage: ${STORE_COV}%  (target: >60%)"
echo "  server/   coverage: ${SERVER_COV}%  (target: >40%)"
echo "  models/   coverage: ${MODELS_COV}%  (target: >90%)"

# Check targets (note: store coverage uses t.Skip on no-PG, so may be 0% without PG)
if [ -n "$ALMS_PG_DSN" ]; then
    STORE_TARGET=60
else
    STORE_TARGET=0  # Accept 0 when PG unavailable
fi

SERVICE_COV_INT=${SERVICE_COV%%.*}
STORE_COV_INT=${STORE_COV%%.*}
SERVER_COV_INT=${SERVER_COV%%.*}

if [ -n "$SERVICE_COV_INT" ] && [ "$SERVICE_COV_INT" -ge 80 ] 2>/dev/null; then
    pass "service coverage ${SERVICE_COV}% ≥ 80%"
else
    fail "service coverage ${SERVICE_COV}% < 80%"
fi

if [ -n "$STORE_COV_INT" ] && [ "$STORE_COV_INT" -ge "$STORE_TARGET" ] 2>/dev/null; then
    pass "store coverage ${STORE_COV}% ≥ ${STORE_TARGET}%"
else
    fail "store coverage ${STORE_COV}% < ${STORE_TARGET}%"
fi

if [ -n "$SERVER_COV_INT" ] && [ "$SERVER_COV_INT" -ge 40 ] 2>/dev/null; then
    pass "server coverage ${SERVER_COV}% ≥ 40%"
else
    fail "server coverage ${SERVER_COV}% < 40%"
fi

echo ""
echo "============================================"
echo "Results: $PASS passed, $FAIL failed, $SKIP skipped"
echo "============================================"

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
