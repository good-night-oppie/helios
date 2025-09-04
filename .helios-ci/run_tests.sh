#!/bin/bash
set -euo pipefail

# Simple end-to-end test for Helios store functionality
echo "=== Running Helios E2E Test ==="

# Create test content
TEST_DIR=$(mktemp -d)
trap 'rm -rf $TEST_DIR' EXIT

mkdir -p "$TEST_DIR/demo"
echo 'test content a' > "$TEST_DIR/demo/a.txt"
echo 'test content b' > "$TEST_DIR/demo/b.txt"

export HELIOS_DEBUG=1
export HELIOS_STORE_DIR="$TEST_DIR/.helios-store"

# Commit test content
cd "$TEST_DIR"
echo "Creating snapshot..."
RAW=$(helios commit --work "$PWD" | tr -d '\n')
SNAP_ID=$(printf "%s" "$RAW" | sed -n 's/.*"snapshot_id":"\([^"]*\)".*/\1/p')
if [ -z "$SNAP_ID" ]; then
  echo "Failed to parse snapshot ID from output: $RAW"
  exit 1
fi
echo "Snapshot ID: $SNAP_ID"

# Print debug stats
echo -e "\nStore statistics:"
helios stats

# Materialize to new location
OUT_DIR="$TEST_DIR/out"
echo -e "\nMaterializing to $OUT_DIR..."
helios materialize --id "$SNAP_ID" --out "$OUT_DIR" --include "demo/**"

# Verify content
echo -e "\nVerifying content..."
diff -ru "$TEST_DIR/demo" "$OUT_DIR/demo"

echo "=== All tests passed! ==="