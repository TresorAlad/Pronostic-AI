"""Team form features with strict temporal anti-leakage."""

from __future__ import annotations

import pandas as pd


def compute_team_form(
    matches_df: pd.DataFrame,
    stats_df: pd.DataFrame,
    team_id: str,
    before_date: pd.Timestamp,
    n_matches: int = 5,
    venue: str | None = None,
) -> dict:
    """
    Compute form features using only matches finished before before_date.
    venue: 'home', 'away', or None for all venues.
    """
    team_matches = matches_df[
        (matches_df["kickoff_at"] < before_date)
        & (matches_df["status"] == "finished")
        & (
            (matches_df["home_team_id"] == team_id)
            | (matches_df["away_team_id"] == team_id)
        )
    ].copy()

    if venue == "home":
        team_matches = team_matches[team_matches["home_team_id"] == team_id]
    elif venue == "away":
        team_matches = team_matches[team_matches["away_team_id"] == team_id]

    team_matches = team_matches.sort_values("kickoff_at", ascending=False).head(n_matches)

    if team_matches.empty:
        return _empty_form(prefix="home" if venue == "home" else "away" if venue == "away" else "")

    goals_scored = []
    goals_conceded = []
    wins = draws = losses = 0

    for _, m in team_matches.iterrows():
        is_home = m["home_team_id"] == team_id
        hs, aws = m["home_score"] or 0, m["away_score"] or 0
        scored = hs if is_home else aws
        conceded = aws if is_home else hs
        goals_scored.append(scored)
        goals_conceded.append(conceded)
        if scored > conceded:
            wins += 1
        elif scored == conceded:
            draws += 1
        else:
            losses += 1

    prefix = ""
    if venue:
        prefix = f"{venue}_"

    form_points = wins * 3 + draws
    n = len(team_matches)

    result = {
        f"{prefix}goals_avg_{n_matches}": sum(goals_scored) / n,
        f"{prefix}goals_conceded_avg_{n_matches}": sum(goals_conceded) / n,
        f"{prefix}wins_last_{n_matches}": wins,
        f"{prefix}draws_last_{n_matches}": draws,
        f"{prefix}losses_last_{n_matches}": losses,
        f"{prefix}form_points": form_points,
        f"{prefix}form": form_points / (n * 3) if n > 0 else 0,
    }

    # Merge stats if available
    match_ids = team_matches["id"].tolist()
    team_stats = stats_df[
        (stats_df["match_id"].isin(match_ids)) & (stats_df["team_id"] == team_id)
    ]

    if not team_stats.empty:
        result[f"{prefix}shots_avg_{n_matches}"] = team_stats["total_shots"].mean()
        result[f"{prefix}shots_on_target_avg_{n_matches}"] = team_stats["shots_on_goal"].mean()
        result[f"{prefix}corners_avg_{n_matches}"] = team_stats["corner_kicks"].mean()
        result[f"{prefix}possession_avg_{n_matches}"] = team_stats["ball_possession"].mean()
        if team_stats["expected_goals"].notna().any():
            result[f"{prefix}xg_avg_{n_matches}"] = team_stats["expected_goals"].mean()
            result["has_xg"] = True
        else:
            result["has_xg"] = False
        if "fouls" in team_stats.columns and team_stats["fouls"].notna().any():
            result[f"{prefix}fouls_avg_{n_matches}"] = team_stats["fouls"].mean()
        if "yellow_cards" in team_stats.columns and team_stats["yellow_cards"].notna().any():
            result[f"{prefix}yellow_cards_avg_{n_matches}"] = team_stats["yellow_cards"].mean()
        if "offsides" in team_stats.columns and team_stats["offsides"].notna().any():
            result[f"{prefix}offsides_avg_{n_matches}"] = team_stats["offsides"].mean()

    return result


def _empty_form(prefix: str = "") -> dict:
    p = f"{prefix}_" if prefix and not prefix.endswith("_") else prefix
    return {
        f"{p}goals_avg_5": 0.0,
        f"{p}goals_conceded_avg_5": 0.0,
        f"{p}wins_last_5": 0,
        f"{p}form": 0.0,
        "has_xg": False,
    }
