#!/usr/bin/env bash
# Demarre toute la stack Pronostic-AI en local (temps reel).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

if [[ ! -f .env ]]; then
  echo "Fichier .env manquant. Copiez .env.example vers .env"
  exit 1
fi

set -a
# shellcheck disable=SC1091
source .env
set +a

PYTHON="${PYTHON:-/tmp/prono-ml-venv/bin/python}"
if [[ ! -x "$PYTHON" ]]; then
  PYTHON="$(command -v python3)"
fi

mkdir -p /tmp/prono-logs

if [[ -n "${DATABASE_URL:-}" ]]; then
  echo "==> Verification PostgreSQL"
  if command -v psql >/dev/null 2>&1; then
    if ! psql "$DATABASE_URL" -c "SELECT 1" >/dev/null 2>&1; then
      echo "ATTENTION: PostgreSQL inaccessible ($DATABASE_URL)"
      echo "  Local: make infra"
      echo "  Neon: verifiez DATABASE_URL dans .env"
    else
      echo "  PostgreSQL OK"
    fi
  else
    echo "  psql absent, skip check DB"
  fi
fi

echo "==> Redis (Docker)"
docker compose up -d redis 2>&1 | tail -5

echo "==> Arret des anciens processus"
for port in 8082 5001 5002 5173; do
  fuser -k "${port}/tcp" 2>/dev/null || true
done
pkill -f "go run ./cmd/collector -mode=daemon" 2>/dev/null || true
sleep 1

echo "==> Backend Go (:${BACKEND_PORT:-8082})"
cd "$ROOT/backend"
nohup go run ./cmd/server > /tmp/prono-logs/backend.log 2>&1 &
echo $! > /tmp/prono-logs/backend.pid

echo "==> ML Service (:${ML_SERVICE_PORT:-5002})"
cd "$ROOT/ml-service"
ML_SERVICE_PORT="${ML_SERVICE_PORT:-5002}" nohup "$PYTHON" -m app.main > /tmp/prono-logs/ml-service.log 2>&1 &
echo $! > /tmp/prono-logs/ml-service.pid

echo "==> AI Agent (:${AI_AGENT_PORT:-5001})"
cd "$ROOT/ai-agent"
AI_AGENT_PORT="${AI_AGENT_PORT:-5001}" nohup "$PYTHON" -m app.main > /tmp/prono-logs/ai-agent.log 2>&1 &
echo $! > /tmp/prono-logs/ai-agent.pid

echo "==> Collector live (daemon, sync chaque minute)"
cd "$ROOT/collector"
nohup go run ./cmd/collector -mode=daemon > /tmp/prono-logs/collector.log 2>&1 &
echo $! > /tmp/prono-logs/collector.pid

echo "==> Frontend Vite (:5173)"
cd "$ROOT/frontend"
nohup npm run dev -- --host 127.0.0.1 --port 5173 > /tmp/prono-logs/frontend.log 2>&1 &
echo $! > /tmp/prono-logs/frontend.pid

echo "==> Attente des services..."
for i in {1..30}; do
  ok=0
  curl -sf "http://localhost:${BACKEND_PORT:-8082}/health" >/dev/null && ok=$((ok + 1))
  curl -sf "http://localhost:${ML_SERVICE_PORT:-5002}/health" >/dev/null && ok=$((ok + 1))
  curl -sf "http://localhost:${AI_AGENT_PORT:-5001}/health" >/dev/null && ok=$((ok + 1))
  curl -sf "http://127.0.0.1:5173" >/dev/null && ok=$((ok + 1))
  if [[ "$ok" -ge 4 ]]; then
    break
  fi
  sleep 1
done

echo ""
echo "Stack demarree."
echo "  Frontend : http://localhost:5173"
echo "  Backend  : http://localhost:${BACKEND_PORT:-8082}"
echo "  ML       : http://localhost:${ML_SERVICE_PORT:-5002}"
echo "  AI Agent : http://localhost:${AI_AGENT_PORT:-5001}"
echo "  Logs     : /tmp/prono-logs/"
echo ""
curl -sf "http://localhost:${BACKEND_PORT:-8082}/health" && echo " backend OK"
curl -sf "http://localhost:${ML_SERVICE_PORT:-5002}/health" && echo " ml OK"
curl -sf "http://localhost:${AI_AGENT_PORT:-5001}/health" && echo " ai OK"
