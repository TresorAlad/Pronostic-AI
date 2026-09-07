"""Home and away specific form features."""

from __future__ import annotations

from features.team_form import compute_team_form


def compute_home_away_form(matches_df, stats_df, home_id, away_id, before_date):
    return {
        **compute_team_form(matches_df, stats_df, home_id, before_date, venue="home"),
        **compute_team_form(matches_df, stats_df, away_id, before_date, venue="away"),
    }
