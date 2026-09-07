"""Tests for derived betting markets."""

from app.market_estimates import enrich_derived_markets


def test_btts_no_derived():
    preds, conf = enrich_derived_markets({"btts": 0.6}, {"data_source": "database"})
    assert "btts_no" in preds
    assert abs(preds["btts_no"] - 0.4) < 0.01


def test_draw_no_bet_from_1x2():
    preds, _ = enrich_derived_markets(
        {"home_win": 0.5, "draw": 0.3, "away_win": 0.2},
        {"data_source": "database"},
    )
    assert "draw_no_bet_home" in preds
    assert "draw_no_bet_away" in preds
