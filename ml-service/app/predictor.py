"""Model loader and predictor."""

from __future__ import annotations

import os
from datetime import datetime, timezone

import joblib

MODELS_DIR = os.getenv("ML_MODELS_DIR", "./models")
CONFIDENCE_THRESHOLD = float(os.getenv("ML_CONFIDENCE_THRESHOLD", "0.55"))


class ModelRegistry:
    def __init__(self):
        self.models: dict = {}
        self.versions: dict = {}
        self._load_models()

    def _load_models(self):
        model_files = {
            "1x2": "model_1x2.joblib",
            "goals": "model_goals.joblib",
            "corners": "model_corners.joblib",
            "shots": "model_shots.joblib",
        }
        for name, filename in model_files.items():
            path = os.path.join(MODELS_DIR, filename)
            if os.path.exists(path):
                self.models[name] = joblib.load(path)
                self.versions[name] = f"{name}-v1.0"

    def reload(self):
        self.models.clear()
        self.versions.clear()
        self._load_models()

    def predict_match(self, features: dict, is_live: bool = False) -> dict:
        predictions = {}
        confidence = {}

        if "1x2" in self.models:
            from sys import path as syspath
            syspath.insert(0, os.path.join(os.path.dirname(__file__), "..", "..", "ml-pipeline"))
            try:
                from models.model_1x2 import predict as predict_1x2
                preds = predict_1x2(self.models["1x2"], features)
                predictions.update(preds)
                for k, v in preds.items():
                    if k.startswith(("home_win", "draw", "away_win")):
                        confidence[k] = v
            except ImportError:
                predictions.update(_fallback_1x2(features))

        if "goals" in self.models:
            try:
                from models.model_goals import predict as predict_goals
                preds = predict_goals(self.models["goals"], features)
                predictions.update(preds)
                confidence.update({k: v for k, v in preds.items()})
            except ImportError:
                pass

        if "corners" in self.models:
            try:
                from models.model_corners import predict as predict_corners
                preds = predict_corners(self.models["corners"], features)
                predictions.update(preds)
                confidence.update({k: v for k, v in preds.items() if k.startswith("over_")})
            except ImportError:
                pass

        if "shots" in self.models:
            try:
                from models.model_shots import predict as predict_shots
                preds = predict_shots(self.models["shots"], features)
                predictions.update(preds)
                confidence.update({k: v for k, v in preds.items()})
            except ImportError:
                pass

        if is_live:
            predictions = _adjust_live_predictions(predictions, features)

        max_conf = max(confidence.values()) if confidence else 0
        no_bet = max_conf < CONFIDENCE_THRESHOLD

        return {
            "predictions": predictions,
            "confidence": confidence,
            "no_bet_recommended": no_bet,
            "model_version": "-".join(self.versions.values()) if self.versions else "none",
            "data_snapshot_at": datetime.now(timezone.utc).isoformat(),
        }

    def status(self) -> dict:
        return {
            "models_loaded": list(self.models.keys()),
            "versions": self.versions,
            "confidence_threshold": CONFIDENCE_THRESHOLD,
        }


def _fallback_1x2(features: dict) -> dict:
    home_strength = features.get("home_attack_strength", 1.0) * features.get("home_form", 0.5)
    away_strength = features.get("away_attack_strength", 1.0) * features.get("away_form", 0.5)
    total = home_strength + away_strength + 0.3
    return {
        "home_win": home_strength / total,
        "draw": 0.3 / total,
        "away_win": away_strength / total,
    }


def _adjust_live_predictions(predictions: dict, features: dict) -> dict:
    minute = features.get("minute", 0)
    current_goals = features.get("current_total_goals", 0)
    if minute > 0 and "over_2_5" in predictions:
        remaining_factor = (90 - minute) / 90
        adjusted = predictions["over_2_5"]
        if current_goals >= 3:
            predictions["over_2_5"] = 1.0
        elif current_goals == 2:
            predictions["over_2_5"] = max(adjusted, 0.7)
        else:
            predictions["over_2_5"] = adjusted * (1 + (1 - remaining_factor) * 0.2)
    return predictions
