#!/usr/bin/env bash
# Réentraîne les modèles ML et signale le reload au service Flask.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

PYTHON="${PYTHON:-/tmp/prono-ml-venv/bin/python}"
if [[ ! -x "$PYTHON" ]]; then
  PYTHON="$(command -v python3)"
fi

echo "==> Entraînement ML"
cd ml-pipeline
ML_MODELS_DIR=../ml-service/models \
MLFLOW_TRACKING_URI="${MLFLOW_TRACKING_URI:-sqlite:////tmp/prono-mlruns/mlflow.db}" \
"$PYTHON" train.py

ML_PORT="${ML_SERVICE_PORT:-5002}"
echo "==> Reload modèles (POST /models/reload si disponible)"
curl -sf -X POST "http://localhost:${ML_PORT}/models/reload" >/dev/null 2>&1 \
  && echo "  ML service rechargé" \
  || echo "  Redémarrez le service ML pour charger les nouveaux .joblib"

echo "Terminé."
