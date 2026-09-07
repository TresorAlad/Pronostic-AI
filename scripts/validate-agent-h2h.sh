#!/usr/bin/env bash
# Valide que l'agent IA peut lire le graphe Neo4j (H2H).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
AI_URL="${AI_AGENT_URL:-http://localhost:5001}"

echo "==> Health ai-agent"
curl -sf "${AI_URL}/health" | tee /tmp/agent-health.json
if ! grep -q '"neo4j": true' /tmp/agent-health.json; then
  echo "ATTENTION: Neo4j non connecte. Lancez: make sync-neo4j"
  exit 1
fi

echo "==> Sync Neo4j"
curl -sf -X POST "${AI_URL}/sync/neo4j" || true

echo "OK: agent H2H pret"
