package coupons

import "testing"

func TestBuildPoolIncludesCornersKey(t *testing.T) {
	conf := map[string]float64{"over_1_5": 0.69}
	preds := map[string]float64{"over_corners_9.5": 0.477, "double_chance_12": 0.88}
	pool := buildCandidatePool(conf, preds, "m4", "Home", "Away")
	found := false
	for _, c := range pool {
		if c.Market == "over_corners_9.5" {
			found = true
		}
	}
	if !found {
		t.Fatalf("pool missing corners: %#v", pool)
	}
}

func TestApplyCombinedOddCap(t *testing.T) {
	selected := []candidate{
		{MatchID: "m1", Market: "home_win", Confidence: 0.7, BookmakerOdd: 5},
		{MatchID: "m2", Market: "over_2_5", Confidence: 0.65, BookmakerOdd: 6},
		{MatchID: "m3", Market: "btts", Confidence: 0.6, BookmakerOdd: 2},
	}
	out := applyCombinedOddCap(selected)
	product := combinedOddProduct(out)
	if product > maxCombinedOdd {
		t.Fatalf("expected product <= 50, got %f with %d picks", product, len(out))
	}
}

func TestPickTipsterFourMatches(t *testing.T) {
	candidates := []candidate{
		{MatchID: "m1", Market: "double_chance_12", Confidence: 0.88},
		{MatchID: "m2", Market: "over_1_5", Confidence: 0.69},
		{MatchID: "m3", Market: "over_shots_22_5", Confidence: 0.53},
		{MatchID: "m4", Market: "over_corners_9.5", Confidence: 0.477},
	}
	picked := pickTipsterSelections(candidates, 8, 0.55)
	if len(picked) < 4 {
		t.Fatalf("expected 4 picks, got %d: %#v", len(picked), picked)
	}
	cats := map[string]bool{}
	for _, p := range picked {
		cats[marketCategory(p.Market)] = true
	}
	if !cats["Corners"] {
		t.Fatalf("expected Corners in %#v", picked)
	}
}
