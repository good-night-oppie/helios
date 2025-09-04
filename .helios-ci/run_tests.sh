#!/bin/bash

set -euo pipefail
tmp=$(mktemp -d)
mkdir -p "$tmp/src/demo"
echo 'x' > "$tmp/src/demo/x.txt"
echo 'y' > "$tmp/src/demo/y.txt"

RAW=$(helios commit --work "$tmp/src" | tr -d '\n')
SNAP_ID=$(printf "%s" "$RAW" | sed -n 's/.*"snapshot_id":"\([^"]*\)".*/\1/p'); [ -z "$SNAP_ID" ] && SNAP_ID="$RAW"
[ -n "$SNAP_ID" ] || { echo "empty SNAP_ID"; exit 1; }

helios materialize --id "$SNAP_ID" --out "$tmp/out" --include "demo/**"
diff -ru "$tmp/src/demo" "$tmp/out/demo"
echo "OK"