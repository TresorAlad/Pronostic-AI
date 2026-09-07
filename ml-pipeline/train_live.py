#!/usr/bin/env python3
"""Minimal in-play model training (minute/score features)."""

from __future__ import annotations

import os
import sys

import joblib
import numpy as np
import pandas as pd
from sklearn.ensemble import GradientBoostingClassifier

sys.path.insert(0, os.path.dirname(__file__))

from datasets.builder import build_dataset, temporal_split
from models.versioning import prod_version, tag_artifact

MODELS_DIR = os.getenv("ML_MODELS_DIR", "../ml-service/models")
FEATURES = ["home_goals_avg_5", "away_goals_avg_5", "home_form", "away_form"]


def main() -> None:
    df = build_dataset()
    if df.empty:
        print("Dataset vide, skip train_live")
        sys.exit(0)
    train, val, _ = temporal_split(df)
    train = train.copy()
    train["target_over_2_5"] = train["over_2_5"]
    X = train[FEATURES].fillna(0)
    y = train["target_over_2_5"]
    model = GradientBoostingClassifier(random_state=42)
    model.fit(X, y)
    artifact = tag_artifact(
        {"model": model, "feature_cols": FEATURES, "target": "over_2_5_live_proxy"},
        prod_version("live"),
    )
    os.makedirs(MODELS_DIR, exist_ok=True)
    out = os.path.join(MODELS_DIR, "model_live.joblib")
    joblib.dump(artifact, out)
    print(f"Live proxy model saved to {out} (train={len(train)}, val={len(val)})")


if __name__ == "__main__":
    main()
