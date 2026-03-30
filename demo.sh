#!/usr/bin/env bash
# demo.sh — spin up a local FastAPI server and run BuzzBench against it.
#
# Requirements:
#   - Go (to build the binary)
#   - Python 3 with pip
#
# Run from the repo root:
#   ./demo.sh

set -euo pipefail

BINARY="./buzzbench_demo"
SERVER_PID=""

# ── cleanup ──────────────────────────────────────────────────────────────────

cleanup() {
    echo ""
    if [[ -n "$SERVER_PID" ]]; then
        kill "$SERVER_PID" 2>/dev/null && echo "Demo server stopped."
    fi
    rm -f "$BINARY"
    echo "Done."
}
trap cleanup EXIT

# ── 1. build ─────────────────────────────────────────────────────────────────

echo "=== Building BuzzBench ==="
go build -o "$BINARY" ./cmd/buzzbench
echo "Built: $BINARY"

# ── 2. python deps ───────────────────────────────────────────────────────────

echo ""
echo "=== Checking Python dependencies ==="
if command -v pip3 &>/dev/null; then
    pip3 install --quiet fastapi uvicorn
elif command -v pip &>/dev/null; then
    pip install --quiet fastapi uvicorn
elif command -v python3 &>/dev/null; then
    python3 -m pip install --quiet fastapi uvicorn
else
    echo "ERROR: No pip or python3 found. Install Python 3 first."
    exit 1
fi
echo "fastapi + uvicorn ready."

# ── 3. start server ──────────────────────────────────────────────────────────

echo ""
echo "=== Starting demo server on http://127.0.0.1:8000 ==="
python3 -m uvicorn demo.server:app --host 127.0.0.1 --port 8000 --log-level warning &
SERVER_PID=$!

echo "Waiting for server to be ready..."
for i in $(seq 1 15); do
    if curl -sf http://127.0.0.1:8000/health > /dev/null 2>&1; then
        echo "Server is ready."
        break
    fi
    if [[ $i -eq 15 ]]; then
        echo "ERROR: server did not start within 15 seconds."
        exit 1
    fi
    sleep 1
done

# ── 4. mode 1: local flag mode ───────────────────────────────────────────────

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  MODE 1 — Local flag mode"
echo "  Single test defined entirely on the command line."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo ""
echo "--- Quick GET test ---"
"$BINARY" \
    -url         http://127.0.0.1:8000/health \
    -name        "Health check (flag mode)" \
    -requests    100 \
    -concurrency 10

echo ""
echo "--- POST test with a JSON body ---"
"$BINARY" \
    -url         http://127.0.0.1:8000/orders \
    -name        "Create order (flag mode)" \
    -method      POST \
    -body        '{"product_id": 42, "quantity": 1}' \
    -requests    30 \
    -concurrency 5

echo ""
echo "--- Save results to a JSON file ---"
"$BINARY" \
    -url         http://127.0.0.1:8000/health \
    -name        "Health check (saved to file)" \
    -requests    50 \
    -concurrency 5 \
    -out         /tmp/buzzbench_result.json
echo "Saved to /tmp/buzzbench_result.json"

# ── 5. mode 2: local config-file mode ────────────────────────────────────────

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  MODE 2 — Config file mode"
echo "  Five tests loaded from demo/tests.json"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

"$BINARY" -config demo/tests.json

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "  Demo complete."
echo "  Try it yourself:"
echo ""
echo "  go build -o buzzbench ./cmd/buzzbench"
echo ""
echo "  ./buzzbench -url http://localhost:8000/health"
echo "  ./buzzbench -url http://localhost:8000/orders -method POST -body '{\"product_id\":1,\"quantity\":2}'"
echo "  ./buzzbench -config demo/tests.json"
echo "  ./buzzbench -config demo/tests.json -out results.json"
echo "  ./buzzbench -h"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
