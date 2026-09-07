"""Model loader and predictor."""

from __future__ import annotations

import os
import sys
from datetime import datetime, timezone

import joblib

MODELS_DIR = os.getenv("ML_MODELS_DIR", "./models")
CONFIDENCE_THRESHOLD = float(os.getenv("ML_CONFIDENCE_THRESHOLD", "0.55"))
APP_ENV = os.getenv("APP_ENV", "development")


def _find_pipeline_path() -> str:
    base_dir = os.path.dirname(__file__)
    candidates = [
        os.path.join(base_dir, "..", "ml-pipeline"),
        os.path.join(base_dir, "..", "..", "ml-pipeline"),
    ]
    for path in candidates:
        abs_path = os.path.abspath(path)
        if os.path.isdir(abs_path):
            return abs_path
    return os.path.abspath(candidates[-1])


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
            "live": "model_live.joblib",
        }
        for name, filename in model_files.items():
            path = os.path.join(MODELS_DIR, filename)
            if os.path.exists(path):
                artifact = joblib.load(path)
                version = str(artifact.get("model_version", f"{name}-v1.0"))
                if APP_ENV == "production" and version.startswith("demo-"):
                    continue
                self.models[name] = artifact
                self.versions[name] = version

    def reload(self):
        self.models.clear()
        self.versions.clear()
        self._load_models()

    def predict_match(self, features: dict, is_live: bool = False) -> dict:
        if features.get("data_source") != "database":
            return {
                "predictions": {},
                "confidence": {},
                "no_bet_recommended": True,
                "model_version": "none",
                "data_snapshot_at": datetime.now(timezone.utc).isoformat(),
                "error": "features must come from database history",
            }

        if not self.models:
            return {
                "predictions": {},
                "confidence": {},
                "no_bet_recommended": True,
                "model_version": "none",
                "data_snapshot_at": datetime.now(timezone.utc).isoformat(),
                "error": "no ML models loaded",
            }

        predictions = {}
        confidence = {}

        pipeline_path = _find_pipeline_path()
        if pipeline_path not in sys.path:
            sys.path.insert(0, pipeline_path)

        if "1x2" in self.models:
            from models.model_1x2 import predict as predict_1x2

            preds = predict_1x2(self.models["1x2"], features)
            predictions.update(preds)
            for k, v in preds.items():
                if k.startswith(("home_win", "draw", "away_win")):
                    confidence[k] = v

        if "goals" in self.models:
            from models.model_goals import predict as predict_goals

            preds = predict_goals(self.models["goals"], features)
            predictions.update(preds)
            confidence.update({k: v for k, v in preds.items()})

        if "corners" in self.models:
            from models.model_corners import predict as predict_corners

            preds = predict_corners(self.models["corners"], features)
            predictions.update(preds)
            confidence.update({k: v for k, v in preds.items() if k.startswith("over_")})

        if "shots" in self.models:
            from models.model_shots import predict as predict_shots

            preds = predict_shots(self.models["shots"], features)
            predictions.update(preds)
            confidence.update({k: v for k, v in preds.items()})

        if is_live:
            predictions = _adjust_live_predictions(predictions, features)
            if "live" in self.models:
                live_art = self.models["live"]
                cols = live_art.get("feature_cols", [])
                row = {c: features.get(c, 0) for c in cols}
                import pandas as pd

                prob = float(live_art["model"].predict_proba(pd.DataFrame([row]))[0][1])
                predictions["over_2_5"] = prob
                confidence["over_2_5"] = prob
                self.versions["live"] = str(live_art.get("model_version", "live"))

        from app.market_estimates import enrich_derived_markets

        predictions, derived_conf = enrich_derived_markets(predictions, features)
        confidence.update(derived_conf)

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
