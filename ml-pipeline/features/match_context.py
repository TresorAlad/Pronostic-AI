"""Match context features: fatigue, home advantage, rest days."""

from __future__ import annotations

import pandas as pd


def compute_match_context(
    matches_df: pd.DataFrame,
    home_id: str,
    away_id: str,
    kickoff: pd.Timestamp,
) -> dict:
    home_rest = _days_since_last_match(matches_df, home_id, kickoff)
    away_rest = _days_since_last_match(matches_df, away_id, kickoff)

    return {
        "home_rest_days": home_rest,
        "away_rest_days": away_rest,
        "home_fatigue": 1.0 if home_rest is not None and home_rest < 4 else 0.0,
        "away_fatigue": 1.0 if away_rest is not None and away_rest < 4 else 0.0,
        "rest_advantage": (away_rest or 7) - (home_rest or 7),
    }


def _days_since_last_match(matches_df: pd.DataFrame, team_id: str, before: pd.Timestamp) -> float | None:
    team_matches = matches_df[
        (matches_df["kickoff_at"] < before)
        & (matches_df["status"] == "finished")
        & ((matches_df["home_team_id"] == team_id) | (matches_df["away_team_id"] == team_id))
    ].sort_values("kickoff_at", ascending=False)

    if team_matches.empty:
        return None

    last = team_matches.iloc[0]["kickoff_at"]
    return (before - last).days
