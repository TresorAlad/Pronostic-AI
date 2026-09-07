package predictions

import (
	"github.com/prono/backend/internal/db"
	"github.com/prono/backend/internal/odds"
)

func buildOddsForAI(matchOdds []db.MatchOdd, mlPredictions map[string]float64) []map[string]interface{} {
	if len(matchOdds) == 0 {
		return nil
	}
	rows := make([]map[string]interface{}, 0, len(matchOdds))
	for _, o := range matchOdds {
		row := map[string]interface{}{
			"bookmaker": o.Bookmaker,
			"market":    o.Market,
			"selection": o.Selection,
			"odd":       o.Odd,
		}
		if o.Odd > 1 {
			row["implied_probability"] = 1 / o.Odd
		}
		if mlKey := odds.MapToMLKey(o.Market, o.Selection); mlKey != "" {
			if p, ok := mlPredictions[mlKey]; ok {
				row["ml_market"] = mlKey
				row["ml_probability"] = p
				if o.Odd > 1 {
					row["value_edge"] = p - (1 / o.Odd)
				}
			}
		}
		rows = append(rows, row)
	}
	return rows
}
