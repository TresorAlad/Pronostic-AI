"""Shots model - Over/Under tirs and tirs cadrés."""

from __future__ import annotations

import os

import joblib
import pandas as pd
from sklearn.calibration import CalibratedClassifierCV
from sklearn.ensemble import GradientBoostingClassifier

FEATURE_COLS = [
    "home_shots_avg_5", "away_shots_avg_5",
    "home_shots_on_target_avg_5", "away_shots_on_target_avg_5",
    "home_attack_strength", "away_attack_strength",
    "home_possession_avg_5", "away_possession_avg_5",
]

MARKETS = ["over_shots_22_5", "over_shots_on_target_8_5"]


def train_models(train_df: pd.DataFrame, val_df: pd.DataFrame) -> dict:
    models = {}
    for market in MARKETS:
        if market not in train_df.columns:
            if market == "over_shots_22_5" and "total_shots" in train_df.columns:
                train_df[market] = (train_df["total_shots"] > 22.5).astype(int)
                val_df[market] = (val_df["total_shots"] > 22.5).astype(int)
            elif market == "over_shots_on_target_8_5" and "total_shots_on_target" in train_df.columns:
                train_df[market] = (train_df["total_shots_on_target"] > 8.5).astype(int)
                val_df[market] = (val_df["total_shots_on_target"] > 8.5).astype(int)
            else:
                continue

        X_train = train_df[FEATURE_COLS].fillna(0)
        y_train = train_df[market]

        clf = GradientBoostingClassifier(n_estimators=100, max_depth=4, random_state=42)
        clf = CalibratedClassifierCV(clf, cv=3)
        clf.fit(X_train, y_train)
        models[market] = clf

    return {"models": models, "feature_cols": FEATURE_COLS, "markets": MARKETS}


def predict(artifact: dict, features: dict) -> dict:
    X = pd.DataFrame([{c: features.get(c, 0) for c in artifact["feature_cols"]}])
    result = {}
    for market, model in artifact["models"].items():
        prob = model.predict_proba(X)[0]
        result[market] = float(prob[1]) if len(prob) > 1 else float(prob[0])
    return result


def save(artifact: dict, path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    joblib.dump(artifact, path)


def load(path: str) -> dict:
    return joblib.load(path)
