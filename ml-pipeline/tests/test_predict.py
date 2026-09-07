"""Smoke test predict() with minimal feature vector."""

from __future__ import annotations

import os
import sys

import pytest

ML_SERVICE = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", "ml-service"))
sys.path.insert(0, ML_SERVICE)

from app.predictor import ModelRegistry  # noqa: E402


@pytest.fixture
def registry():
    models_dir = os.path.join(ML_SERVICE, "models")
    os.environ["ML_MODELS_DIR"] = models_dir
    return ModelRegistry()


def test_registry_status(registry):
    assert isinstance(registry.models, dict)


def test_predict_with_features(registry):
    if not registry.models:
        pytest.skip("No .joblib models loaded")

    features = {
        "home_goals_avg_5": 1.5,
        "away_goals_avg_5": 1.2,
        "home_goals_conceded_avg_5": 1.0,
        "away_goals_conceded_avg_5": 1.1,
        "home_form": 1.8,
        "away_form": 1.4,
        "home_home_form": 2.0,
        "away_away_form": 1.2,
        "home_attack_strength": 1.1,
        "away_attack_strength": 0.9,
        "home_defense_strength": 0.9,
        "away_defense_strength": 1.0,
        "home_advantage": 0.15,
        "home_availability": 0.95,
        "away_availability": 0.92,
        "home_shots_avg_5": 12.0,
        "away_shots_avg_5": 11.0,
        "home_corners_avg_5": 5.0,
        "away_corners_avg_5": 4.5,
        "home_possession_avg_5": 52.0,
        "away_possession_avg_5": 48.0,
        "data_source": "database",
        "stats_history_home": 5,
        "stats_history_away": 5,
    }

    result = registry.predict_match(features, is_live=False)
    assert result["predictions"]
    assert len(result["predictions"]) > 0
