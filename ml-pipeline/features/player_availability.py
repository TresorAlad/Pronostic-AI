"""Player availability score based on injuries and suspensions."""

from __future__ import annotations

import pandas as pd


def compute_availability(
    injuries_df: pd.DataFrame,
    team_id: str,
    match_date: pd.Timestamp,
) -> float:
    if injuries_df.empty:
        return 1.0

    active = injuries_df[
        (injuries_df["team_id"] == team_id)
        & (injuries_df["is_active"] == True)  # noqa: E712
    ]
    if active.empty:
        return 1.0

    # Simple score: fewer injuries = higher availability
    count = len(active)
    return max(0.5, 1.0 - count * 0.05)
