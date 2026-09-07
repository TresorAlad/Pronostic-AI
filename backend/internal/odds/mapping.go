package odds

import (
	"strings"

	"github.com/prono/backend/internal/db"
)

// MapToMLKey maps bookmaker market/selection labels to internal ML market keys.
func MapToMLKey(market, selection string) string {
	m := strings.ToLower(market)
	s := strings.ToLower(selection)
	if strings.Contains(m, "match winner") || m == "1x2" {
		switch s {
		case "home":
			return "home_win"
		case "draw":
			return "draw"
		case "away":
			return "away_win"
		}
	}
	if strings.Contains(m, "both teams") && s == "yes" {
		return "btts"
	}
	return ""
}

// ValueEdgeForMarket returns ML probability minus implied probability for a mapped market.
func ValueEdgeForMarket(matchOdds []db.MatchOdd, mlMarket string, mlProb float64) float64 {
	for _, o := range matchOdds {
		if MapToMLKey(o.Market, o.Selection) != mlMarket || o.Odd <= 1 {
			continue
		}
		return mlProb - (1 / o.Odd)
	}
	return 0
}
