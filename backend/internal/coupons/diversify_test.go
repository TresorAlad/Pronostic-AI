package coupons

import (
	"testing"
)

func TestPickForOddsTargetRespectsMaxSelections(t *testing.T) {
	candidates := make([]candidate, 0, 12)
	for i := 0; i < 12; i++ {
		candidates = append(candidates, candidate{
			MatchID:      string(rune('a' + i)),
			Market:       "home_win",
			Selection:    "home_win",
			Confidence:   0.7,
			BookmakerOdd: 1.5,
		})
	}
	picked := pickForOddsTarget(candidates, 5, 50, 10, 0.55)
	if len(picked) > 10 {
		t.Fatalf("expected at most 10 selections, got %d", len(picked))
	}
}

func TestPickForOddsTargetShrinksAboveMaxOdd(t *testing.T) {
	candidates := []candidate{
		{MatchID: "m1", Market: "home_win", Selection: "home_win", Confidence: 0.7, BookmakerOdd: 3},
		{MatchID: "m2", Market: "btts", Selection: "btts", Confidence: 0.65, BookmakerOdd: 4},
		{MatchID: "m3", Market: "over_2_5", Selection: "over_2_5", Confidence: 0.6, BookmakerOdd: 5},
	}
	picked := pickForOddsTarget(candidates, 5, 10, 10, 0.55)
	product := CombinedOddProductFromCandidates(picked)
	if product > 10.01 {
		t.Fatalf("expected combined odd <= 10, got %.2f", product)
	}
}

func TestCombinedOddInRange(t *testing.T) {
	if !CombinedOddInRange(12, 10, 25) {
		t.Fatal("12 should be in 10-25")
	}
	if CombinedOddInRange(8, 10, 25) {
		t.Fatal("8 should be outside 10-25")
	}
}

func TestPickForOddsTargetReachesBalancedInterval(t *testing.T) {
	candidates := []candidate{
		{MatchID: "m1", Market: "home_win", Confidence: 0.89, BookmakerOdd: 1.15},
		{MatchID: "m2", Market: "home_win", Confidence: 0.85, BookmakerOdd: 1.20},
		{MatchID: "m3", Market: "away_win", Confidence: 0.58, BookmakerOdd: 3.10},
		{MatchID: "m4", Market: "over_2_5", Confidence: 0.62, BookmakerOdd: 2.05},
		{MatchID: "m5", Market: "btts", Confidence: 0.60, BookmakerOdd: 1.95},
		{MatchID: "m6", Market: "home_win", Confidence: 0.57, BookmakerOdd: 2.40},
	}
	picked := pickForOddsTarget(candidates, 10, 25, 10, 0.55)
	product := CombinedOddProductFromCandidates(picked)
	if product < 10 {
		t.Fatalf("expected combined odd >= 10 for balanced profile, got %.2f with %d picks", product, len(picked))
	}
	if product > 25.01 {
		t.Fatalf("expected combined odd <= 25, got %.2f", product)
	}
}

func TestBestPerMatchPrefersHigherOddWhenTargetHigh(t *testing.T) {
	candidates := []candidate{
		{MatchID: "m1", Market: "home_win", Confidence: 0.89, BookmakerOdd: 1.15},
		{MatchID: "m1", Market: "away_win", Confidence: 0.40, BookmakerOdd: 4.50},
		{MatchID: "m1", Market: "over_2_5", Confidence: 0.58, BookmakerOdd: 2.10},
	}
	filtered := filterByConfidence(candidates, 0.55)
	out := bestPerMatchForTarget(filtered, 10)
	if len(out) != 1 {
		t.Fatalf("expected 1 pick per match, got %d", len(out))
	}
	if out[0].Market != "over_2_5" {
		t.Fatalf("expected higher-odd market over_2_5, got %s (odd %.2f)", out[0].Market, out[0].BookmakerOdd)
	}
}
