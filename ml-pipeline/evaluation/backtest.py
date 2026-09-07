"""Temporal walk-forward backtesting."""

from __future__ import annotations

import pandas as pd

from evaluation.metrics import compute_metrics


def walk_forward_backtest(
    df: pd.DataFrame,
    feature_cols: list[str],
    label_col: str,
    train_fn,
    predict_fn,
    min_train_size: int = 500,
) -> pd.DataFrame:
    df = df.sort_values("kickoff_at").reset_index(drop=True)
    results = []

    for i in range(min_train_size, len(df)):
        train = df.iloc[:i]
        test_row = df.iloc[i : i + 1]

        X_train = train[feature_cols].fillna(0)
        y_train = train[label_col]
        X_test = test_row[feature_cols].fillna(0)
        y_true = test_row[label_col].values[0]

        model = train_fn(X_train, y_train)
        y_prob = predict_fn(model, X_test)[0]

        results.append({
            "kickoff_at": test_row["kickoff_at"].values[0],
            "match_id": test_row["match_id"].values[0],
            "y_true": y_true,
            "y_prob": y_prob,
        })

    results_df = pd.DataFrame(results)
    if not results_df.empty:
        overall = compute_metrics(
            results_df["y_true"].values,
            results_df["y_prob"].values,
        )
        results_df.attrs["overall_metrics"] = overall

    return results_df
