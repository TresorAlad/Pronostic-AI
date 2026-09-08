#!/bin/sh
set -e

MODELS_DIR="${ML_MODELS_DIR:-/app/models}"
MARKER="${MODELS_DIR}/model_1x2.joblib"

mkdir -p "${MODELS_DIR}"

if [ ! -f "${MARKER}" ]; then
  echo "==> Aucun modèle ML trouvé dans ${MODELS_DIR}"
  echo "==> Entraînement des modèles demo (données synthétiques)..."
  python /app/ml-pipeline/train_demo.py
  echo "==> Modèles demo prêts: $(ls -1 "${MODELS_DIR}"/*.joblib 2>/dev/null | wc -l | tr -d ' ') fichiers"
fi

exec gunicorn --bind 0.0.0.0:5000 --workers 2 app.main:app
