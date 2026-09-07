"""1X2 result model - LightGBM multiclass."""

from __future__ import annotations

import json
import os

import joblib
import lightgbm as lgb
import numpy as np
import pandas as pd
from sklearn.calibration import CalibratedClassifierCV
from sklearn.preprocessing import LabelEncoder

FEATURE_COLS = [
    "home_goals_avg_5", "away_goals_avg_5",
    "home_goals_conceded_avg_5", "away_goals_conceded_avg_5",
    "home_form", "away_form", "home_home_form", "away_away_form",
    "home_attack_strength", "home_defense_strength",
    "away_attack_strength", "away_defense_strength",
    "home_advantage", "home_availability", "away_availability",
]


def train_model(train_df: pd.DataFrame, val_df: pd.DataFrame) -> dict:
    train_df = train_df.copy()
    train_df["result"] = train_df.apply(
        lambda r: "home" if r["result_home"] else ("draw" if r["result_draw"] else "away"),
        axis=1,
    )
    val_df = val_df.copy()
    val_df["result"] = val_df.apply(
        lambda r: "home" if r["result_home"] else ("draw" if r["result_draw"] else "away"),
        axis=1,
    )

    le = LabelEncoder()
    y_train = le.fit_transform(train_df["result"])
    y_val = le.transform(val_df["result"])

    X_train = train_df[FEATURE_COLS].fillna(0)
    X_val = val_df[FEATURE_COLS].fillna(0)

    base = lgb.LGBMClassifier(
        n_estimators=200,
        learning_rate=0.05,
        max_depth=6,
        num_leaves=31,
        random_state=42,
        verbose=-1,
    )
    model = CalibratedClassifierCV(base, cv=3, method="isotonic")
    model.fit(X_train, y_train)

    probs = model.predict_proba(X_val)
    classes = le.classes_

    return {
        "model": model,
        "label_encoder": le,
        "feature_cols": FEATURE_COLS,
        "classes": classes.tolist(),
    }


def predict(artifact: dict, features: dict) -> dict:
    X = pd.DataFrame([{c: features.get(c, 0) for c in artifact["feature_cols"]}])
    probs = artifact["model"].predict_proba(X)[0]
    classes = artifact["classes"]

    result = {}
    for cls, prob in zip(classes, probs):
        if cls == "home":
            result["home_win"] = float(prob)
        elif cls == "draw":
            result["draw"] = float(prob)
        elif cls == "away":
            result["away_win"] = float(prob)

    result["double_chance_1x"] = result.get("home_win", 0) + result.get("draw", 0)
    result["double_chance_x2"] = result.get("draw", 0) + result.get("away_win", 0)
    result["double_chance_12"] = result.get("home_win", 0) + result.get("away_win", 0)
    return result


def save(artifact: dict, path: str) -> None:
    os.makedirs(os.path.dirname(path), exist_ok=True)
    joblib.dump(artifact, path)
    meta = {"feature_cols": artifact["feature_cols"], "classes": artifact["classes"]}
    with open(path.replace(".joblib", ".json"), "w") as f:
        json.dump(meta, f)


def load(path: str) -> dict:
    return joblib.load(path)
