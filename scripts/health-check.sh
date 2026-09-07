#!/usr/bin/env bash
# Vérifie la santé de la stack Pronostic-AI.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

BACKEND_PORT="${BACKEND_PORT:-8082}"
ML_PORT="${ML_SERVICE_PORT:-5002}"
AI_PORT="${AI_AGENT_PORT:-5001}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"

fail=0

check() {
  local name="$1"
  local url="$2"
  if curl -sf "$url" >/dev/null; then
    echo "OK  $name ($url)"
  else
    echo "FAIL $name ($url)"
    fail=1
  fi
}

check "Backend" "http://localhost:${BACKEND_PORT}/health"
check "ML" "http://localhost:${ML_PORT}/health"
check "AI Agent" "http://localhost:${AI_PORT}/health"
check "Frontend" "http://127.0.0.1:${FRONTEND_PORT}"

if command -v redis-cli >/dev/null 2>&1; then
  if redis-cli -u "${REDIS_URL:-redis://localhost:6379/0}" ping 2>/dev/null | grep -q PONG; then
    echo "OK  Redis"
  else
    echo "FAIL Redis"
    fail=1
  fi
fi

if [[ -n "${DATABASE_URL:-}" ]] && command -v psql >/dev/null 2>&1; then
  if psql "$DATABASE_URL" -c "SELECT 1" >/dev/null 2>&1; then
    echo "OK  PostgreSQL"
  else
    echo "FAIL PostgreSQL"
    fail=1
  fi
fi

exit "$fail"
