"""Goals model - Over/Under and BTTS."""

from __future__ import annotations

import json
import os

import joblib
import lightgbm as lgb
import pandas as pd
from sklearn.calibration import CalibratedClassifierCV

FEATURE_COLS = [
    "home_goals_avg_5", "away_goals_avg_5",
    "home_goals_conceded_avg_5", "away_goals_conceded_avg_5",
    "home_attack_strength", "away_attack_strength",
    "home_defense_strength", "away_defense_strength",
    "home_form", "away_form",
    "home_shots_avg_5", "away_shots_avg_5",
    "home_xg_avg_5", "away_xg_avg_5",
]

MARKETS = ["over_1_5", "over_2_5", "over_3_5", "btts"]


def train_models(train_df: pd.DataFrame, val_df: pd.DataFrame) -> dict:
    models = {}
    for market in MARKETS:
        if market not in train_df.columns:
            continue
        X_train = train_df[FEATURE_COLS].fillna(0)
        y_train = train_df[market]
        X_val = val_df[FEATURE_COLS].fillna(0)

        base = lgb.LGBMClassifier(
            n_estimators=150, learning_rate=0.05, max_depth=5, random_state=42, verbose=-1
        )
        model = CalibratedClassifierCV(base, cv=3, method="isotonic")
        model.fit(X_train, y_train)
        models[market] = model

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
