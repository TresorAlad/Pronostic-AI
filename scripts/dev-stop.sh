#!/usr/bin/env bash
# Arrete la stack locale Pronostic-AI.
set -euo pipefail

for port in 8082 5001 5002 5173; do
  fuser -k "${port}/tcp" 2>/dev/null || true
done

pkill -f "go run ./cmd/collector -mode=daemon" 2>/dev/null || true
pkill -f "go run ./cmd/server" 2>/dev/null || true

echo "Services arretes. Redis Docker laisse tourner (docker compose stop redis pour l'arreter)."
