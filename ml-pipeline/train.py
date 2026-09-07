#!/usr/bin/env python3
"""Train all ML models with temporal split and MLflow tracking."""

from __future__ import annotations

import os
import sys

sys.path.insert(0, os.path.dirname(__file__))

import mlflow
import pandas as pd
from dotenv import load_dotenv

from datasets.builder import build_dataset, temporal_split
from evaluation.metrics import compute_metrics
from models.model_1x2 import train_model as train_1x2, save as save_1x2
from models.model_goals import train_models as train_goals, save as save_goals
from models.model_corners import train_model as train_corners, save as save_corners
from models.model_shots import train_models as train_shots, save as save_shots

load_dotenv()

MODELS_DIR = os.getenv("ML_MODELS_DIR", "../ml-service/models")


def main():
    print("Building dataset...")
    try:
        df = build_dataset()
    except Exception as exc:
        print(f"Database unavailable ({exc}). Using synthetic demo dataset.")
        df = pd.DataFrame()

    if df.empty:
        print("No data available. Run collector backfill first.")
        print("Generating synthetic demo dataset for model structure validation...")
        df = _generate_demo_dataset()

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

    print(f"Models saved to {MODELS_DIR}")


def _generate_demo_dataset():
    """Minimal synthetic data so training pipeline runs without DB."""
    import numpy as np
    import pandas as pd

    np.random.seed(42)
    n = 1000
    dates = pd.date_range("2019-01-01", periods=n, freq="3D")
    data = {
        "match_id": [f"m{i}" for i in range(n)],
        "kickoff_at": dates,
        "home_goals_avg_5": np.random.uniform(0.5, 2.5, n),
        "away_goals_avg_5": np.random.uniform(0.5, 2.5, n),
        "home_goals_conceded_avg_5": np.random.uniform(0.5, 2.5, n),
        "away_goals_conceded_avg_5": np.random.uniform(0.5, 2.5, n),
        "home_form": np.random.uniform(0.2, 0.8, n),
        "away_form": np.random.uniform(0.2, 0.8, n),
        "home_home_form": np.random.uniform(0.2, 0.8, n),
        "away_away_form": np.random.uniform(0.2, 0.8, n),
        "home_attack_strength": np.random.uniform(0.5, 1.5, n),
        "away_attack_strength": np.random.uniform(0.5, 1.5, n),
        "home_defense_strength": np.random.uniform(0.5, 1.5, n),
        "away_defense_strength": np.random.uniform(0.5, 1.5, n),
        "home_advantage": np.full(n, 0.15),
        "home_availability": np.random.uniform(0.7, 1.0, n),
        "away_availability": np.random.uniform(0.7, 1.0, n),
        "home_shots_avg_5": np.random.uniform(8, 18, n),
        "away_shots_avg_5": np.random.uniform(8, 18, n),
        "home_shots_on_target_avg_5": np.random.uniform(3, 8, n),
        "away_shots_on_target_avg_5": np.random.uniform(3, 8, n),
        "home_corners_avg_5": np.random.uniform(3, 8, n),
        "away_corners_avg_5": np.random.uniform(3, 8, n),
        "home_possession_avg_5": np.random.uniform(40, 60, n),
        "away_possession_avg_5": np.random.uniform(40, 60, n),
        "home_xg_avg_5": np.random.uniform(0.8, 2.0, n),
        "away_xg_avg_5": np.random.uniform(0.8, 2.0, n),
    }
    df = pd.DataFrame(data)
    df["result_home"] = (np.random.rand(n) > 0.55).astype(int)
    df["result_draw"] = (np.random.rand(n) > 0.75).astype(int)
    df["result_away"] = 1 - df["result_home"] - df["result_draw"]
    df["result_away"] = df["result_away"].clip(0, 1)
    df["over_1_5"] = (np.random.rand(n) > 0.3).astype(int)
    df["over_2_5"] = (np.random.rand(n) > 0.5).astype(int)
    df["over_3_5"] = (np.random.rand(n) > 0.7).astype(int)
    df["btts"] = (np.random.rand(n) > 0.45).astype(int)
    df["total_corners"] = np.random.randint(6, 14, n)
    df["total_shots"] = np.random.randint(15, 30, n)
    df["total_shots_on_target"] = np.random.randint(5, 12, n)
    df["over_corners_9_5"] = (df["total_corners"] > 9.5).astype(int)
    df["over_shots_22_5"] = (df["total_shots"] > 22.5).astype(int)
    df["over_shots_on_target_8_5"] = (df["total_shots_on_target"] > 8.5).astype(int)
    return df


if __name__ == "__main__":
    main()
