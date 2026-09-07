#!/usr/bin/env python3
"""Train all ML models with temporal split and MLflow tracking."""

from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

import mlflow
from dotenv import load_dotenv

from datasets.builder import build_dataset, temporal_split
from evaluation.persist_metrics import collect_validation_metrics, persist_metrics
from models.model_1x2 import train_model as train_1x2, save as save_1x2
from models.model_goals import train_models as train_goals, save as save_goals
from models.model_corners import train_model as train_corners, save as save_corners
from models.model_shots import train_models as train_shots, save as save_shots

load_dotenv()

MODELS_DIR = os.getenv("ML_MODELS_DIR", "../ml-service/models")


def main():
    print("Building dataset...")
    df = build_dataset()
    if df.empty:
        print("Aucune donnée disponible. Lancez d'abord le backfill collector sur la base PostgreSQL.")
        sys.exit(1)

    train, val, test = temporal_split(df)
    print(f"Split: train={len(train)}, val={len(val)}, test={len(test)}")

    mlflow.set_experiment("football-ai-predictor")
    mlflow.set_tracking_uri(os.getenv("MLFLOW_TRACKING_URI", "file:///tmp/prono-mlruns"))

    os.makedirs(MODELS_DIR, exist_ok=True)

    with mlflow.start_run(run_name="train_all_models"):
        # 1X2
        print("Training 1X2 model...")
        artifact_1x2 = train_1x2(train, val)
        save_1x2(artifact_1x2, os.path.join(MODELS_DIR, "model_1x2.joblib"))
        mlflow.log_param("model_1x2_features", len(artifact_1x2["feature_cols"]))

        # Goals
        print("Training Goals model...")
        artifact_goals = train_goals(train, val)
        save_goals(artifact_goals, os.path.join(MODELS_DIR, "model_goals.joblib"))

        # Corners
        print("Training Corners model...")
        artifact_corners = train_corners(train, val)
        save_corners(artifact_corners, os.path.join(MODELS_DIR, "model_corners.joblib"))

        # Shots
        print("Training Shots model...")
        artifact_shots = train_shots(train, val)
        save_shots(artifact_shots, os.path.join(MODELS_DIR, "model_shots.joblib"))

        mlflow.log_param("train_size", len(train))
        mlflow.log_param("val_size", len(val))
        mlflow.log_param("test_size", len(test))

    artifacts = {
        "1x2": artifact_1x2,
        "goals": artifact_goals,
        "corners": artifact_corners,
        "shots": artifact_shots,
    }
    metric_rows = collect_validation_metrics(val, artifacts)
    saved = persist_metrics(metric_rows)
    print(f"Saved {saved} performance metrics to database")

    print(f"Models saved to {MODELS_DIR}")


if __name__ == "__main__":
    main()
