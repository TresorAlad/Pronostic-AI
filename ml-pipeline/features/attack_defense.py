"""Attack and defense strength features."""

from __future__ import annotations


def compute_strength(home_form: dict, away_form: dict) -> dict:
    home_attack = home_form.get("goals_avg_5", 0) or home_form.get("home_goals_avg_5", 0)
    home_defense = home_form.get("goals_conceded_avg_5", 0) or home_form.get("home_goals_conceded_avg_5", 0)
    away_attack = away_form.get("goals_avg_5", 0) or away_form.get("away_goals_avg_5", 0)
    away_defense = away_form.get("goals_conceded_avg_5", 0) or away_form.get("away_goals_conceded_avg_5", 0)

    league_avg_goals = 1.35

    return {
        "home_attack_strength": home_attack / league_avg_goals if league_avg_goals else 1.0,
        "home_defense_strength": home_defense / league_avg_goals if league_avg_goals else 1.0,
        "away_attack_strength": away_attack / league_avg_goals if league_avg_goals else 1.0,
        "away_defense_strength": away_defense / league_avg_goals if league_avg_goals else 1.0,
        "home_advantage": 0.15,
    }
