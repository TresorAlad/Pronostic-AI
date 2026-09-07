package coupons

import (
	"math"
	"sort"
)

var categoryPriority = []string{
	"Temps réglementaire",
	"Buts",
	"Tirs",
	"Corners",
	"Cartons",
	"Fautes",
	"Hors-jeu",
	"Possession",
}

var secondaryCategories = map[string]bool{
	"Corners":    true,
	"Cartons":    true,
	"Fautes":     true,
	"Hors-jeu":   true,
	"Possession": true,
}

func thresholdForCategory(category string, pass int, minConfidence float64) float64 {
	relaxed := math.Max(0.50, minConfidence-0.05)
	switch pass {
	case 0:
		return minConfidence
	case 1:
		return relaxed
	default:
		if secondaryCategories[category] {
			return 0.45
		}
		return relaxed
	}
}

func collapseBestPerMatchCategory(candidates []candidate) []candidate {
	best := make(map[string]candidate)
	for _, c := range candidates {
		key := c.MatchID + "|" + marketCategory(c.Market)
		if existing, ok := best[key]; !ok || c.Confidence > existing.Confidence {
			best[key] = c
		}
	}
	out := make([]candidate, 0, len(best))
	for _, c := range best {
		out = append(out, c)
	}
	return out
}

func pickTipsterSelections(candidates []candidate, max int, minConfidence float64) []candidate {
	if len(candidates) == 0 || max <= 0 {
		return nil
	}

	candidates = collapseBestPerMatchCategory(candidates)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	selected := make([]candidate, 0, max)
	usedMatches := map[string]bool{}
	usedCategories := map[string]bool{}

	for pass := 0; pass < 3; pass++ {
		for _, cat := range categoryPriority {
			if len(selected) >= max {
				return selected
			}
			if usedCategories[cat] {
				continue
			}
			threshold := thresholdForCategory(cat, pass, minConfidence)
			if c, ok := bestForCategory(candidates, cat, threshold, usedMatches); ok {
				selected = append(selected, c)
				usedMatches[c.MatchID] = true
				usedCategories[cat] = true
			}
		}
	}

	return selected
}

func bestForCategory(candidates []candidate, category string, minConf float64, usedMatches map[string]bool) (candidate, bool) {
	var best candidate
	found := false
	for _, c := range candidates {
		if marketCategory(c.Market) != category {
			continue
		}
		if c.Confidence < minConf {
			continue
		}
		if usedMatches[c.MatchID] {
			continue
		}
		if !found || c.Confidence > best.Confidence {
			best = c
			found = true
		}
	}
	return best, found
}

func buildCandidatePool(confidence, predictions map[string]float64, matchID, home, away string) []candidate {
	merged := mergeMarketScores(confidence, predictions)
	pool := make([]candidate, 0, len(merged))
	for market, score := range merged {
		if !isCouponMarket(market) {
			continue
		}
		if score < 0.45 {
			continue
		}
		pool = append(pool, candidate{
			MatchID: matchID, HomeTeam: home, AwayTeam: away,
			Market: market, Selection: market, Confidence: score,
		})
	}
	return pool
}

func mergeMarketScores(confidence, predictions map[string]float64) map[string]float64 {
	merged := make(map[string]float64, len(confidence)+len(predictions))
	for k, v := range predictions {
		if isCouponMarket(k) {
			merged[k] = v
		}
	}
	for k, v := range confidence {
		if !isCouponMarket(k) {
			continue
		}
		if existing, ok := merged[k]; !ok || v > existing {
			merged[k] = v
		}
	}
	return merged
}
