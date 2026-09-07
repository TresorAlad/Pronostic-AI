"""Marchés dérivés basés uniquement sur des statistiques réelles en base."""

from __future__ import annotations

import math

MIN_STATS_MATCHES = 2


def _has_stat(features: dict, key: str) -> bool:
    return key in features and features[key] is not None and float(features[key]) > 0


def _stats_ready(features: dict) -> bool:
    home = int(features.get("stats_history_home", 0))
    away = int(features.get("stats_history_away", 0))
    return home >= MIN_STATS_MATCHES and away >= MIN_STATS_MATCHES


def _over_prob(expected_total: float, line: float, scale: float = 2.0) -> float:
    diff = expected_total - line
    prob = 1.0 / (1.0 + math.exp(-diff / scale))
    return max(0.05, min(0.95, prob))


def _side_prob(avg_value: float, line: float, scale: float = 8.0) -> float:
    diff = avg_value - line
    prob = 1.0 / (1.0 + math.exp(-diff / scale))
    return max(0.05, min(0.95, prob))


def enrich_derived_markets(predictions: dict, features: dict) -> tuple[dict, dict]:
    """Ajoute cartons, fautes, hors-jeu et possession si les données réelles existent."""
    extra_preds: dict[str, float] = {}
    extra_conf: dict[str, float] = {}

    if features.get("data_source") != "database":
        return predictions, {}

    if _stats_ready(features):
        if _has_stat(features, "home_fouls_avg_5") and _has_stat(features, "away_fouls_avg_5"):
            total_fouls = float(features["home_fouls_avg_5"]) + float(features["away_fouls_avg_5"])
            extra_preds["predicted_total_fouls"] = round(total_fouls, 1)
            for market, line in {
                "over_fouls_20_5": 20.5,
                "over_fouls_22_5": 22.5,
                "over_fouls_25_5": 25.5,
            }.items():
                prob = _over_prob(total_fouls, line, scale=2.5)
                extra_preds[market] = prob
                extra_conf[market] = prob

        if _has_stat(features, "home_yellow_cards_avg_5") and _has_stat(features, "away_yellow_cards_avg_5"):
            total_cards = float(features["home_yellow_cards_avg_5"]) + float(features["away_yellow_cards_avg_5"])
            extra_preds["predicted_total_cards"] = round(total_cards, 1)
            for market, line in {
                "over_cards_3_5": 3.5,
                "over_cards_4_5": 4.5,
                "over_cards_5_5": 5.5,
            }.items():
                prob = _over_prob(total_cards, line, scale=1.2)
                extra_preds[market] = prob
                extra_conf[market] = prob

        if _has_stat(features, "home_offsides_avg_5") and _has_stat(features, "away_offsides_avg_5"):
            total_offsides = float(features["home_offsides_avg_5"]) + float(features["away_offsides_avg_5"])
            extra_preds["predicted_total_offsides"] = round(total_offsides, 1)
            for market, line in {
                "over_offsides_2_5": 2.5,
                "over_offsides_3_5": 3.5,
                "over_offsides_4_5": 4.5,
            }.items():
                prob = _over_prob(total_offsides, line, scale=1.0)
                extra_preds[market] = prob
                extra_conf[market] = prob

        if _has_stat(features, "home_possession_avg_5"):
            home_poss_prob = _side_prob(float(features["home_possession_avg_5"]), 50.0, scale=6.0)
            extra_preds["home_possession_over_50"] = home_poss_prob
            extra_conf["home_possession_over_50"] = home_poss_prob

        if _has_stat(features, "away_possession_avg_5"):
            away_poss_prob = _side_prob(float(features["away_possession_avg_5"]), 50.0, scale=6.0)
            extra_preds["away_possession_over_50"] = away_poss_prob
            extra_conf["away_possession_over_50"] = away_poss_prob

    if "over_2_5" in predictions:
        under_25 = 1.0 - predictions["over_2_5"]
        extra_preds["under_2_5"] = under_25
        extra_conf["under_2_5"] = under_25

    if "over_1_5" in predictions:
        under_15 = 1.0 - predictions["over_1_5"]
        extra_preds["under_1_5"] = under_15
        extra_conf["under_1_5"] = under_15

    if "btts" in predictions:
        extra_preds["btts_no"] = 1.0 - predictions["btts"]
        extra_conf["btts_no"] = extra_preds["btts_no"]

    if "home_win" in predictions and "draw" in predictions:
        home = predictions["home_win"]
        draw = predictions["draw"]
        dnb_home = home / max(home + draw, 0.01)
        dnb_away = predictions.get("away_win", 0) / max(predictions.get("away_win", 0) + draw, 0.01)
        extra_preds["draw_no_bet_home"] = min(0.95, dnb_home)
        extra_preds["draw_no_bet_away"] = min(0.95, dnb_away)
        extra_conf["draw_no_bet_home"] = extra_preds["draw_no_bet_home"]
        extra_conf["draw_no_bet_away"] = extra_preds["draw_no_bet_away"]

    if _stats_ready(features):
        home_goals = float(features.get("home_goals_avg_5", 0))
        away_goals = float(features.get("away_goals_avg_5", 0))
        if home_goals > 0:
            for line, key in ((0.5, "team_over_0_5_home"), (1.5, "team_over_1_5_home")):
                prob = _over_prob(home_goals, line, scale=0.8)
                extra_preds[key] = prob
                extra_conf[key] = prob
        if away_goals > 0:
            for line, key in ((0.5, "team_over_0_5_away"), (1.5, "team_over_1_5_away")):
                prob = _over_prob(away_goals, line, scale=0.8)
                extra_preds[key] = prob
                extra_conf[key] = prob

        if _has_stat(features, "home_corners_avg_5") and _has_stat(features, "away_corners_avg_5"):
            if float(features["home_corners_avg_5"]) > float(features["away_corners_avg_5"]):
                prob = _side_prob(float(features["home_corners_avg_5"]) - float(features["away_corners_avg_5"]), 0.5, scale=1.5)
            else:
                prob = 1.0 - _side_prob(float(features["away_corners_avg_5"]) - float(features["home_corners_avg_5"]), 0.5, scale=1.5)
            extra_preds["corner_winner_home"] = prob
            extra_preds["corner_winner_away"] = 1.0 - prob
            extra_conf["corner_winner_home"] = prob
            extra_conf["corner_winner_away"] = 1.0 - prob

        if "home_win" in predictions and "btts" in predictions:
            hw, bt = predictions["home_win"], predictions["btts"]
            extra_preds["result_btts_home_yes"] = hw * bt
            extra_conf["result_btts_home_yes"] = extra_preds["result_btts_home_yes"]
            if "draw" in predictions:
                extra_preds["result_btts_draw_yes"] = predictions["draw"] * bt
                extra_conf["result_btts_draw_yes"] = extra_preds["result_btts_draw_yes"]
            if "away_win" in predictions:
                extra_preds["result_btts_away_yes"] = predictions["away_win"] * bt
                extra_conf["result_btts_away_yes"] = extra_preds["result_btts_away_yes"]

    merged_preds = {**predictions, **extra_preds}
    return merged_preds, extra_conf
