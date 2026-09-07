"""Corners model - XGBoost regression + threshold."""

from __future__ import annotations

import os

import joblib
import pandas as pd
import xgboost as xgb
from sklearn.calibration import CalibratedClassifierCV
from sklearn.ensemble import GradientBoostingClassifier

FEATURE_COLS = [
    "home_corners_avg_5", "away_corners_avg_5",
    "home_shots_avg_5", "away_shots_avg_5",
    "home_possession_avg_5", "away_possession_avg_5",
    "home_attack_strength", "away_attack_strength",
]

CORNER_THRESHOLD = 9.5


def train_model(train_df: pd.DataFrame, val_df: pd.DataFrame) -> dict:
    if "over_corners_9_5" not in train_df.columns:
        train_df["over_corners_9_5"] = (train_df.get("total_corners", 0) > CORNER_THRESHOLD).astype(int)
        val_df["over_corners_9_5"] = (val_df.get("total_corners", 0) > CORNER_THRESHOLD).astype(int)

    X_train = train_df[FEATURE_COLS].fillna(0)
    y_train = train_df["over_corners_9_5"]

    reg = xgb.XGBRegressor(n_estimators=100, max_depth=4, learning_rate=0.1, random_state=42)
    if "total_corners" in train_df.columns:
        reg.fit(X_train, train_df["total_corners"].fillna(0))

    clf = GradientBoostingClassifier(n_estimators=100, max_depth=4, random_state=42)
    clf = CalibratedClassifierCV(clf, cv=3)
    clf.fit(X_train, y_train)

    return {
        "regressor": reg,
        "classifier": clf,
        "feature_cols": FEATURE_COLS,
        "threshold": CORNER_THRESHOLD,
    }


def predict(artifact: dict, features: dict) -> dict:
    X = pd.DataFrame([{c: features.get(c, 0) for c in artifact["feature_cols"]}])
    prob = artifact["classifier"].predict_proba(X)[0]
    predicted_total = float(artifact["regressor"].predict(X)[0]) if hasattr(artifact["regressor"], "predict") else 0
    return {
        "over_corners_9_5": float(prob[1]) if len(prob) > 1 else float(prob[0]),
        "predicted_total_corners": predicted_total,
    }


def save(artifact: dict, path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    joblib.dump(artifact, path)


def load(path: str) -> dict:
    return joblib.load(path)
