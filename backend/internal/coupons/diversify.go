package coupons

import (
	"math"
	"sort"

	"github.com/prono/backend/internal/markets"
)

const maxCombinedOdd = 50.0

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
	return pickForOddsTarget(candidates, 5, maxCombinedOdd, max, minConfidence)
}

func pickForOddsTarget(candidates []candidate, minOdd, maxOdd float64, max int, minConfidence float64) []candidate {
	if len(candidates) == 0 || max <= 0 {
		return nil
	}
	if minOdd <= 0 {
		minOdd = 5
	}
	if maxOdd <= 0 {
		maxOdd = maxCombinedOdd
	}
	if maxOdd < minOdd {
		maxOdd = minOdd
	}

	eligible := filterByConfidence(candidates, minConfidence)
	selected := buildTowardOddsInterval(eligible, minOdd, maxOdd, max, minConfidence)

	if combinedOddProduct(selected) < minOdd {
		relaxed := filterByConfidence(candidates, math.Max(0.50, minConfidence-0.08))
		selected = buildTowardOddsInterval(relaxed, minOdd, maxOdd, max, math.Max(0.50, minConfidence-0.08))
	}
	if combinedOddProduct(selected) < minOdd && minOdd >= 10 {
		highOdd := make([]candidate, 0, len(candidates))
		for _, c := range candidates {
			if c.Confidence >= 0.50 && effectiveOdd(c) >= 1.75 {
				highOdd = append(highOdd, c)
			}
		}
		if len(highOdd) > 0 {
			selected = buildTowardOddsInterval(highOdd, minOdd, maxOdd, max, 0.50)
		}
	}

	selected = shrinkToMaxOdd(selected, maxOdd)
	return selected
}

func filterByConfidence(candidates []candidate, minConfidence float64) []candidate {
	out := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.Confidence >= minConfidence {
			out = append(out, c)
		}
	}
	return out
}

// bestPerMatchForTarget keeps one candidate per match, favouring higher odds when the target interval is ambitious.
func bestPerMatchForTarget(candidates []candidate, minOdd float64) []candidate {
	byMatch := make(map[string][]candidate)
	for _, c := range candidates {
		byMatch[c.MatchID] = append(byMatch[c.MatchID], c)
	}
	preferHighOdds := minOdd >= 10
	out := make([]candidate, 0, len(byMatch))
	for _, list := range byMatch {
		sort.Slice(list, func(i, j int) bool {
			if preferHighOdds {
				oi := effectiveOdd(list[i])
				oj := effectiveOdd(list[j])
				if math.Abs(oi-oj) > 0.05 {
					return oi > oj
				}
			}
			return candidateScore(list[i]) > candidateScore(list[j])
		})
		out = append(out, list[0])
	}
	return out
}

func buildTowardOddsInterval(candidates []candidate, minOdd, maxOdd float64, max int, minConfidence float64) []candidate {
	pool := bestPerMatchForTarget(candidates, minOdd)
	sort.Slice(pool, func(i, j int) bool {
		return effectiveOdd(pool[i]) > effectiveOdd(pool[j])
	})

	selected := greedyFillTowardMin(pool, minOdd, maxOdd, max)
	selected = swapTowardInterval(selected, candidates, minOdd, maxOdd, max)
	return selected
}

func greedyFillTowardMin(pool []candidate, minOdd, maxOdd float64, max int) []candidate {
	selected := make([]candidate, 0, max)
	usedMatches := map[string]bool{}
	remaining := append([]candidate{}, pool...)

	for len(selected) < max {
		product := combinedOddProduct(selected)
		if product >= minOdd && product <= maxOdd {
			break
		}

		bestIdx := -1
		var bestProduct float64
		for i, c := range remaining {
			if usedMatches[c.MatchID] {
				continue
			}
			trialProduct := combinedOddProduct(append(selected, c))
			if trialProduct > maxOdd {
				continue
			}
			if bestIdx < 0 || trialProduct > bestProduct {
				bestIdx = i
				bestProduct = trialProduct
			}
		}
		if bestIdx < 0 {
			break
		}

		selected = append(selected, remaining[bestIdx])
		usedMatches[remaining[bestIdx].MatchID] = true
		remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)

		if combinedOddProduct(selected) >= minOdd {
			break
		}
	}
	return selected
}

func swapTowardInterval(selected []candidate, all []candidate, minOdd, maxOdd float64, max int) []candidate {
	if len(selected) == 0 {
		return selected
	}

	byMatch := make(map[string][]candidate)
	for _, c := range all {
		byMatch[c.MatchID] = append(byMatch[c.MatchID], c)
	}

	for attempt := 0; attempt < max*2; attempt++ {
		product := combinedOddProduct(selected)
		if product >= minOdd && product <= maxOdd {
			return selected
		}

		improved := false
		if product < minOdd {
			for i, current := range selected {
				alternatives := byMatch[current.MatchID]
				for _, alt := range alternatives {
					if alt.Market == current.Market && alt.Selection == current.Selection {
						continue
					}
					trial := append([]candidate{}, selected...)
					trial[i] = alt
					trialProduct := combinedOddProduct(trial)
					if trialProduct > product && trialProduct <= maxOdd {
						selected = trial
						product = trialProduct
						improved = true
						break
					}
				}
				if improved {
					break
				}
			}
		}

		if !improved && product < minOdd {
			usedMatches := map[string]bool{}
			for _, c := range selected {
				usedMatches[c.MatchID] = true
			}
			for _, c := range all {
				if usedMatches[c.MatchID] {
					continue
				}
				if len(selected) >= max {
					worstIdx := -1
					var worstOdd float64 = math.MaxFloat64
					for i, s := range selected {
						o := effectiveOdd(s)
						if o < worstOdd {
							worstOdd = o
							worstIdx = i
						}
					}
					if worstIdx < 0 {
						break
					}
					trial := append([]candidate{}, selected...)
					trial[worstIdx] = c
					trialProduct := combinedOddProduct(trial)
					if trialProduct > product && trialProduct <= maxOdd {
						selected = trial
						improved = true
						break
					}
				} else {
					trial := append(selected, c)
					trialProduct := combinedOddProduct(trial)
					if trialProduct > product && trialProduct <= maxOdd {
						selected = trial
						improved = true
						break
					}
				}
			}
		}

		if !improved {
			break
		}
	}
	return selected
}

func pickDiversified(candidates []candidate, max int, minConfidence float64) []candidate {
	candidates = collapseBestPerMatchCategory(candidates)
	sort.Slice(candidates, func(i, j int) bool {
		return candidateScore(candidates[i]) > candidateScore(candidates[j])
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

func candidateScore(c candidate) float64 {
	score := c.Confidence
	if c.BookmakerOdd > 1 {
		edge := c.Confidence - (1 / c.BookmakerOdd)
		if edge > 0 {
			score += edge * 0.5
		}
	}
	return score
}

func effectiveOdd(c candidate) float64 {
	if c.BookmakerOdd > 1 {
		return c.BookmakerOdd
	}
	return 1 / math.Max(c.Confidence, 0.01)
}

func shrinkToMaxOdd(selected []candidate, maxOdd float64) []candidate {
	if len(selected) == 0 {
		return selected
	}
	for {
		product := combinedOddProduct(selected)
		if product <= maxOdd || len(selected) <= 1 {
			return selected
		}
		worstIdx := -1
		var worstScore float64
		for i, c := range selected {
			score := c.Confidence
			if c.BookmakerOdd > 0 {
				score = c.Confidence / effectiveOdd(c)
			}
			if worstIdx < 0 || score < worstScore {
				worstIdx = i
				worstScore = score
			}
		}
		if worstIdx < 0 {
			return selected[:len(selected)-1]
		}
		selected = append(selected[:worstIdx], selected[worstIdx+1:]...)
	}
}

func growToMinOdd(selected []candidate, candidates []candidate, minOdd, maxOdd float64, max int) []candidate {
	if len(selected) >= max {
		return selected
	}

	usedMatches := map[string]bool{}
	for _, c := range selected {
		usedMatches[c.MatchID] = true
	}

	remaining := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if usedMatches[c.MatchID] {
			continue
		}
		remaining = append(remaining, c)
	}
	sort.Slice(remaining, func(i, j int) bool {
		return effectiveOdd(remaining[i]) > effectiveOdd(remaining[j])
	})

	for combinedOddProduct(selected) < minOdd && len(selected) < max && len(remaining) > 0 {
		added := false
		for i, c := range remaining {
			trial := append(append([]candidate{}, selected...), c)
			if combinedOddProduct(trial) > maxOdd {
				continue
			}
			selected = trial
			remaining = append(remaining[:i], remaining[i+1:]...)
			added = true
			break
		}
		if !added {
			break
		}
	}
	return selected
}

func CombinedOddInRange(product, minOdd, maxOdd float64) bool {
	return product >= minOdd && product <= maxOdd
}

func applyCombinedOddCap(selected []candidate) []candidate {
	if len(selected) == 0 {
		return selected
	}
	for {
		product := combinedOddProduct(selected)
		if product <= maxCombinedOdd || len(selected) <= 1 {
			return selected
		}
		worstIdx := -1
		var worstScore float64
		for i, c := range selected {
			score := c.Confidence
			if c.BookmakerOdd > 0 {
				score = c.Confidence / c.BookmakerOdd
			}
			if worstIdx < 0 || score < worstScore {
				worstIdx = i
				worstScore = score
			}
		}
		if worstIdx < 0 {
			return selected[:len(selected)-1]
		}
		selected = append(selected[:worstIdx], selected[worstIdx+1:]...)
	}
}

func combinedOddProduct(selected []candidate) float64 {
	product := 1.0
	for _, c := range selected {
		odd := c.BookmakerOdd
		if odd <= 1 {
			odd = 1 / math.Max(c.Confidence, 0.01)
		}
		product *= odd
	}
	return product
}

func CombinedOddProductFromCandidates(selected []candidate) float64 {
	return combinedOddProduct(selected)
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
		if !found || candidateScore(c) > candidateScore(best) {
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

func marketCategory(market string) string {
	return markets.Category(market)
}

func marketLabel(market string) string {
	if label := markets.Label(market); label != "" {
		return label
	}
	return market
}
