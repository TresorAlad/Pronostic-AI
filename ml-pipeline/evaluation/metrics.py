"""Evaluation metrics for ML models."""

from __future__ import annotations

import numpy as np
from sklearn.metrics import (
    accuracy_score,
    brier_score_loss,
    f1_score,
    log_loss,
    precision_score,
    recall_score,
)


def compute_metrics(y_true: np.ndarray, y_prob: np.ndarray, threshold: float = 0.5) -> dict:
    y_pred = (y_prob >= threshold).astype(int)
    metrics = {
        "accuracy": float(accuracy_score(y_true, y_pred)),
        "precision": float(precision_score(y_true, y_pred, zero_division=0)),
        "recall": float(recall_score(y_true, y_pred, zero_division=0)),
        "f1": float(f1_score(y_true, y_pred, zero_division=0)),
        "log_loss": float(log_loss(y_true, np.clip(y_prob, 1e-7, 1 - 1e-7))),
        "brier_score": float(brier_score_loss(y_true, y_prob)),
    }
    return metrics


def calibration_bins(y_true: np.ndarray, y_prob: np.ndarray, n_bins: int = 10) -> list[dict]:
    bins = np.linspace(0, 1, n_bins + 1)
    result = []
    for i in range(n_bins):
        mask = (y_prob >= bins[i]) & (y_prob < bins[i + 1])
        if mask.sum() == 0:
            continue
        result.append({
            "bin_low": float(bins[i]),
            "bin_high": float(bins[i + 1]),
            "predicted_mean": float(y_prob[mask].mean()),
            "actual_rate": float(y_true[mask].mean()),
            "count": int(mask.sum()),
        })
    return result
