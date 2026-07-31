#!/bin/bash
# ============================================================
# P2P File Sharing System - Day 1 Integration Test
# Tests: Bootstrap server + 5 peer nodes + network graph
# ============================================================

set -e

echo ""
echo "========================================"
echo "  P2P File Sharing - Day 1 Test"
echo "========================================"
echo ""

PROJECT_ROOT="$(cd "$(dirname "$0")" && pwd)"
BIN_DIR="$PROJECT_ROOT/bin"

# ---- Build Phase ----
echo "[1/5] Building binaries..."
mkdir -p "$BIN_DIR"

echo "  Building bootstrap server..."
cd "$PROJECT_ROOT/cmd/bootstrap"
go build -o "$BIN_DIR/bootstrap"
cd "$PROJECT_ROOT"

echo "  Building peer application..."
cd "$PROJECT_ROOT/cmd/peer"
go build -o "$BIN_DIR/peer"
cd "$PROJECT_ROOT"

echo "  Build complete!"
echo ""

# ---- Cleanup Function ----
PIDS=()
cleanup() {
    echo ""
    echo "[5/5] Cleaning up processes..."
    for pid in "${PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
            echo "  Stopped PID $pid"
        fi
    done
    echo "  Cleanup complete!"
}
trap cleanup EXIT

# ---- Start Bootstrap ----
echo "[2/5] Starting bootstrap server on :5000..."
"$BIN_DIR/bootstrap" > "$BIN_DIR/bootstrap_out.log" 2>"$BIN_DIR/bootstrap_err.log" &
BOOTSTRAP_PID=$!
PIDS+=($BOOTSTRAP_PID)
sleep 2

if ! kill -0 "$BOOTSTRAP_PID" 2>/dev/null; then
    echo "  Bootstrap server failed to start!"
    cat "$BIN_DIR/bootstrap_err.log"
    exit 1
fi
echo "  Bootstrap running (PID: $BOOTSTRAP_PID)"
echo ""

# ---- Start Peers ----
echo "[3/5] Starting 5 peer nodes..."

for i in $(seq 1 5); do
    PORT=$((9000 + i))
    echo "  Starting peer-$i on port $PORT..."
    "$BIN_DIR/peer" --id="$i" --port="$PORT" --bootstrap="localhost:5000" \
        > "$BIN_DIR/peer${i}_out.log" 2>"$BIN_DIR/peer${i}_err.log" &
    PEER_PID=$!
    PIDS+=($PEER_PID)
    
    # Stagger peer startups
    sleep 2
done

echo "  All peers started!"
echo ""

# ---- Wait for Discovery ----
echo "[4/5] Waiting for peer discovery (5 seconds)..."
sleep 5

# ---- Display Results ----
echo ""
echo "========================================"
echo "  RESULTS"
echo "========================================"
echo ""

# Show bootstrap log
echo "--- Bootstrap Server Log ---"
cat "$BIN_DIR/bootstrap_out.log" 2>/dev/null || true
echo ""

# Show each peer's output
ALL_SUCCESS=true
for i in $(seq 1 5); do
    echo "--- Peer-$i Output ---"
    if [ -f "$BIN_DIR/peer${i}_out.log" ]; then
        cat "$BIN_DIR/peer${i}_out.log"
        
        if grep -q "NETWORK GRAPH" "$BIN_DIR/peer${i}_out.log"; then
            echo "  [PASS] Network graph found!"
        else
            echo "  [WARN] No network graph in output"
        fi
    else
        echo "  [FAIL] No output file found"
        ALL_SUCCESS=false
    fi
    
    # Check for errors
    if [ -s "$BIN_DIR/peer${i}_err.log" ]; then
        echo "  Errors:"
        cat "$BIN_DIR/peer${i}_err.log"
        ALL_SUCCESS=false
    fi
    echo ""
done

# ---- Summary ----
echo "========================================"
if [ "$ALL_SUCCESS" = true ]; then
    echo "  ALL TESTS PASSED!"
else
    echo "  SOME TESTS FAILED!"
fi
echo "========================================"
echo ""
