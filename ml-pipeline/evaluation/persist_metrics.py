"""Persist validation metrics to model_performance table."""

from __future__ import annotations

import os

import numpy as np
import pandas as pd
import psycopg2

from evaluation.metrics import compute_metrics
from models.model_1x2 import predict as predict_1x2
from models.model_corners import predict as predict_corners
from models.model_goals import predict as predict_goals
from models.model_shots import predict as predict_shots

MODEL_VERSION = "1x2-v1.0-goals-v1.0-corners-v1.0-shots-v1.0"


def _row_features(row: pd.Series) -> dict:
    return row.to_dict()


def _metrics_for_binary(y_true: np.ndarray, y_prob: np.ndarray) -> dict:
    if len(y_true) == 0:
        return {"accuracy": 0.0, "log_loss": 0.0, "brier_score": 0.0, "sample_size": 0}
    m = compute_metrics(y_true, y_prob)
    m["sample_size"] = int(len(y_true))
    return m


def collect_validation_metrics(val_df: pd.DataFrame, artifacts: dict) -> list[dict]:
    rows: list[dict] = []

    # 1X2 markets
    home_probs, draw_probs, away_probs = [], [], []
    y_home, y_draw, y_away = [], [], []
    for _, row in val_df.iterrows():
        feats = _row_features(row)
        preds = predict_1x2(artifacts["1x2"], feats)
        home_probs.append(preds.get("home_win", 0))
        draw_probs.append(preds.get("draw", 0))
        away_probs.append(preds.get("away_win", 0))
        y_home.append(int(row.get("result_home", 0)))
        y_draw.append(int(row.get("result_draw", 0)))
        y_away.append(int(row.get("result_away", 0)))

    for market, y_true, y_prob in [
        ("home_win", y_home, home_probs),
        ("draw", y_draw, draw_probs),
        ("away_win", y_away, away_probs),
    ]:
        m = _metrics_for_binary(np.array(y_true), np.array(y_prob))
        rows.append({"model_name": "1x2", "model_version": MODEL_VERSION, "market": market, **m})

    # Goals markets
    for market in artifacts["goals"]["markets"]:
        if market not in val_df.columns:
            continue
        y_true, y_prob = [], []
        for _, row in val_df.iterrows():
            preds = predict_goals(artifacts["goals"], _row_features(row))
            y_prob.append(preds.get(market, 0))
            y_true.append(int(row[market]))
        m = _metrics_for_binary(np.array(y_true), np.array(y_prob))
        rows.append({"model_name": "goals", "model_version": MODEL_VERSION, "market": market, **m})

    # Corners
    if "over_corners_9_5" in val_df.columns:
        y_true, y_prob = [], []
        for _, row in val_df.iterrows():
            preds = predict_corners(artifacts["corners"], _row_features(row))
            y_prob.append(preds.get("over_corners_9_5", 0))
            y_true.append(int(row["over_corners_9_5"]))
        m = _metrics_for_binary(np.array(y_true), np.array(y_prob))
        rows.append({
            "model_name": "corners",
            "model_version": MODEL_VERSION,
            "market": "over_corners_9_5",
            **m,
        })

    # Shots markets
    for market in ["over_shots_22_5", "over_shots_on_target_8_5"]:
        if market not in val_df.columns:
            continue
        y_true, y_prob = [], []
        for _, row in val_df.iterrows():
            preds = predict_shots(artifacts["shots"], _row_features(row))
            y_prob.append(preds.get(market, 0))
            y_true.append(int(row[market]))
        m = _metrics_for_binary(np.array(y_true), np.array(y_prob))
        rows.append({"model_name": "shots", "model_version": MODEL_VERSION, "market": market, **m})

    return rows


def persist_metrics(rows: list[dict]) -> int:
    db_url = os.getenv("DATABASE_URL", "postgres://prono:prono_secret@localhost:5432/prono")
    db_url = db_url.replace("postgres://", "postgresql://", 1)

    conn = psycopg2.connect(db_url)
    cur = conn.cursor()
    cur.execute("DELETE FROM model_performance")
    inserted = 0
    for row in rows:
        cur.execute(
            """
            INSERT INTO model_performance
                (model_name, model_version, market, accuracy, log_loss, brier_score, sample_size)
            VALUES (%s, %s, %s, %s, %s, %s, %s)
            """,
            (
                row["model_name"],
                row["model_version"],
                row["market"],
                row["accuracy"],
                row["log_loss"],
                row["brier_score"],
                row["sample_size"],
            ),
        )
        inserted += 1
    conn.commit()
    cur.close()
    conn.close()
    return inserted
