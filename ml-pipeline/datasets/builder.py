"""Dataset builder with temporal anti-leakage."""

from __future__ import annotations

import os

import pandas as pd
from sqlalchemy import create_engine

from features.attack_defense import compute_strength
from features.match_context import compute_match_context
from features.player_availability import compute_availability
from features.team_form import compute_team_form


def load_data(database_url: str | None = None) -> tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame, pd.DataFrame]:
    url = database_url or os.getenv(
        "DATABASE_URL", "postgresql://prono:prono_secret@localhost:5432/prono"
    )
    url = url.replace("postgres://", "postgresql://", 1)
    engine = create_engine(url)

    matches = pd.read_sql(
        """
        SELECT m.id, m.external_id, m.league_id, m.home_team_id, m.away_team_id,
               m.kickoff_at, m.status, m.home_score, m.away_score
        FROM matches m
        WHERE m.status = 'finished'
        ORDER BY m.kickoff_at
        """,
        engine,
    )
    matches["kickoff_at"] = pd.to_datetime(matches["kickoff_at"])

    stats = pd.read_sql("SELECT * FROM match_statistics", engine)
    injuries = pd.read_sql("SELECT * FROM injuries WHERE is_active = true", engine)

    return matches, stats, injuries, engine


def build_features_for_match(
    match_row: pd.Series,
    matches_df: pd.DataFrame,
    stats_df: pd.DataFrame,
    injuries_df: pd.DataFrame,
) -> dict:
    kickoff = match_row["kickoff_at"]
    home_id = match_row["home_team_id"]
    away_id = match_row["away_team_id"]
    league_id = match_row["league_id"]

    home_form = compute_team_form(matches_df, stats_df, home_id, kickoff)
    away_form = compute_team_form(matches_df, stats_df, away_id, kickoff)
    home_home_form = compute_team_form(matches_df, stats_df, home_id, kickoff, venue="home")
    away_away_form = compute_team_form(matches_df, stats_df, away_id, kickoff, venue="away")

    league_matches = matches_df[
        (matches_df["league_id"] == league_id)
        & (matches_df["kickoff_at"] < kickoff)
        & (matches_df["status"] == "finished")
    ].tail(200)
    if not league_matches.empty:
        league_avg_goals = (
            (league_matches["home_score"].fillna(0) + league_matches["away_score"].fillna(0)) / 2.0
        ).mean()
    else:
        league_avg_goals = None

    strength = compute_strength(home_form, away_form, league_avg_goals)
    context = compute_match_context(matches_df, home_id, away_id, kickoff)

    features = {
        "match_id": match_row["id"],
        "kickoff_at": kickoff,
        **{f"home_{k}": v for k, v in home_form.items()},
        **{f"away_{k}": v for k, v in away_form.items()},
        "home_home_form": home_home_form.get("home_form", 0),
        "away_away_form": away_away_form.get("away_form", 0),
        **strength,
        **context,
        "home_availability": compute_availability(injuries_df, home_id, kickoff),
        "away_availability": compute_availability(injuries_df, away_id, kickoff),
    }
    return features


def build_labels(match_row: pd.Series, stats_df: pd.DataFrame) -> dict:
    hs = match_row["home_score"] or 0
    aws = match_row["away_score"] or 0
    total_goals = hs + aws

    labels = {
        "result_home": 1 if hs > aws else 0,
        "result_draw": 1 if hs == aws else 0,
        "result_away": 1 if hs < aws else 0,
        "over_1_5": 1 if total_goals > 1.5 else 0,
        "over_2_5": 1 if total_goals > 2.5 else 0,
        "over_3_5": 1 if total_goals > 3.5 else 0,
        "btts": 1 if hs > 0 and aws > 0 else 0,
        "total_goals": total_goals,
    }

    match_stats = stats_df[stats_df["match_id"] == match_row["id"]]
    if not match_stats.empty:
        labels["total_corners"] = match_stats["corner_kicks"].sum()
        labels["total_shots"] = match_stats["total_shots"].sum()
        labels["total_shots_on_target"] = match_stats["shots_on_goal"].sum()
        labels["over_corners_9_5"] = 1 if labels.get("total_corners", 0) > 9.5 else 0
        labels["over_shots_22_5"] = 1 if labels.get("total_shots", 0) > 22.5 else 0
        labels["over_shots_on_target_8_5"] = 1 if labels.get("total_shots_on_target", 0) > 8.5 else 0

    return labels


def build_dataset(database_url: str | None = None) -> pd.DataFrame:
    matches, stats, injuries, _ = load_data(database_url)
    rows = []

    for _, match in matches.iterrows():
        try:
            features = build_features_for_match(match, matches, stats, injuries)
            labels = build_labels(match, stats)
            rows.append({**features, **labels})
        except Exception:
            continue

    return pd.DataFrame(rows)


def temporal_split(df: pd.DataFrame) -> tuple[pd.DataFrame, pd.DataFrame, pd.DataFrame]:
    df = df.sort_values("kickoff_at")
    train = df[df["kickoff_at"].dt.year <= 2022]
    val = df[(df["kickoff_at"].dt.year == 2023)]
    test = df[df["kickoff_at"].dt.year >= 2024]
    return train, val, test
