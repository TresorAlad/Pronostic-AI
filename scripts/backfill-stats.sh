#!/usr/bin/env bash
# Rattrapage massif des match_statistics par batches (quota API-Football).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

BATCHES="${1:-10}"
LIMIT="${2:-500}"
PAUSE="${3:-45}"

echo "==> Backfill stats: ${BATCHES} batches x ${LIMIT} matchs (pause ${PAUSE}s)"

for ((i = 1; i <= BATCHES; i++)); do
  echo "--- Batch ${i}/${BATCHES} ---"
  cd collector
  go run ./cmd/collector -mode=sync-stats -limit="$LIMIT"
  cd "$ROOT"
  if [[ "$i" -lt "$BATCHES" ]]; then
    echo "Pause ${PAUSE}s (quota API)..."
    sleep "$PAUSE"
  fi
done

echo "Terminé. Vérifiez: SELECT COUNT(*) FROM match_statistics;"
