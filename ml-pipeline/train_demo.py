#!/usr/bin/env python3
"""Entraîne des modèles de démonstration sur données synthétiques (CI / dev sans backfill)."""

from __future__ import annotations

import os
import sys

import numpy as np
import pandas as pd

sys.path.insert(0, os.path.dirname(__file__))

from models.model_1x2 import train_model as train_1x2, save as save_1x2
from models.model_goals import train_models as train_goals, save as save_goals
from models.model_corners import train_model as train_corners, save as save_corners
from models.model_shots import train_models as train_shots, save as save_shots

MODELS_DIR = os.getenv("ML_MODELS_DIR", "../ml-service/models")
N_ROWS = 400


def synthetic_dataset() -> pd.DataFrame:
    rng = np.random.default_rng(42)
    rows = []
    for i in range(N_ROWS):
        home_goals = rng.uniform(0.5, 2.5)
        away_goals = rng.uniform(0.5, 2.5)
        hs = int(rng.integers(0, 5))
        aws = int(rng.integers(0, 5))
        total_corners = int(rng.integers(6, 14))
        total_shots = int(rng.integers(18, 28))
        total_sot = int(rng.integers(5, 12))
        row = {
            "match_id": f"demo-{i}",
            "kickoff_at": pd.Timestamp("2024-01-01") + pd.Timedelta(days=i),
            "home_goals_avg_5": home_goals,
            "away_goals_avg_5": away_goals,
            "home_goals_conceded_avg_5": rng.uniform(0.5, 2.0),
            "away_goals_conceded_avg_5": rng.uniform(0.5, 2.0),
            "home_form": rng.uniform(0.3, 2.5),
            "away_form": rng.uniform(0.3, 2.5),
            "home_home_form": rng.uniform(0.3, 2.5),
            "away_away_form": rng.uniform(0.3, 2.5),
            "home_attack_strength": home_goals / 1.4,
            "away_attack_strength": away_goals / 1.4,
            "home_defense_strength": rng.uniform(0.6, 1.4),
            "away_defense_strength": rng.uniform(0.6, 1.4),
            "home_advantage": 0.15,
            "home_availability": rng.uniform(0.7, 1.0),
            "away_availability": rng.uniform(0.7, 1.0),
            "home_shots_avg_5": rng.uniform(8, 16),
            "away_shots_avg_5": rng.uniform(8, 16),
            "home_shots_on_target_avg_5": rng.uniform(3, 7),
            "away_shots_on_target_avg_5": rng.uniform(3, 7),
            "home_corners_avg_5": rng.uniform(3, 7),
            "away_corners_avg_5": rng.uniform(3, 7),
            "home_possession_avg_5": rng.uniform(42, 58),
            "away_possession_avg_5": rng.uniform(42, 58),
            "home_xg_avg_5": rng.uniform(0.8, 2.0),
            "away_xg_avg_5": rng.uniform(0.8, 2.0),
            "result_home": 1 if hs > aws else 0,
            "result_draw": 1 if hs == aws else 0,
            "result_away": 1 if hs < aws else 0,
            "over_1_5": 1 if hs + aws > 1.5 else 0,
            "over_2_5": 1 if hs + aws > 2.5 else 0,
            "over_3_5": 1 if hs + aws > 3.5 else 0,
            "btts": 1 if hs > 0 and aws > 0 else 0,
            "total_corners": total_corners,
            "total_shots": total_shots,
            "total_shots_on_target": total_sot,
            "over_corners_9_5": 1 if total_corners > 9.5 else 0,
            "over_shots_22_5": 1 if total_shots > 22.5 else 0,
            "over_shots_on_target_8_5": 1 if total_sot > 8.5 else 0,
        }
        rows.append(row)
    return pd.DataFrame(rows)


def main():
    df = synthetic_dataset()
    split_idx = int(len(df) * 0.8)
    train = df.iloc[:split_idx].copy()
    val = df.iloc[split_idx:].copy()
    os.makedirs(MODELS_DIR, exist_ok=True)

    print(f"Demo train: {len(train)} rows, val: {len(val)} rows")

    artifact_1x2 = train_1x2(train, val)
    save_1x2(artifact_1x2, os.path.join(MODELS_DIR, "model_1x2.joblib"))

    artifact_goals = train_goals(train, val)
    save_goals(artifact_goals, os.path.join(MODELS_DIR, "model_goals.joblib"))

    artifact_corners = train_corners(train, val)
    save_corners(artifact_corners, os.path.join(MODELS_DIR, "model_corners.joblib"))

    artifact_shots = train_shots(train, val)
    save_shots(artifact_shots, os.path.join(MODELS_DIR, "model_shots.joblib"))

    print(f"Demo models saved to {MODELS_DIR}")


if __name__ == "__main__":
    main()
