package odds

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/prono/backend/internal/db"
)

var goalLineRe = regexp.MustCompile(`(over|under)\s*([0-9]+(?:[.,][0-9]+)?)`)

// MapToMLKey maps bookmaker market/selection labels to internal ML market keys.
func MapToMLKey(market, selection string) string {
	m := strings.ToLower(market)
	s := strings.ToLower(strings.TrimSpace(selection))

	if strings.Contains(m, "match winner") || m == "1x2" || strings.Contains(m, "full time result") {
		switch {
		case s == "home" || strings.Contains(s, "1"):
			return "home_win"
		case s == "draw" || s == "x":
			return "draw"
		case s == "away" || strings.Contains(s, "2"):
			return "away_win"
		}
	}

	if strings.Contains(m, "double chance") {
		switch {
		case strings.Contains(s, "home/draw") || s == "1x":
			return "double_chance_1x"
		case strings.Contains(s, "draw/away") || s == "x2":
			return "double_chance_x2"
		case strings.Contains(s, "home/away") || s == "12":
			return "double_chance_12"
		}
	}

	if strings.Contains(m, "both teams") || strings.Contains(m, "btts") {
		if s == "yes" || s == "oui" {
			return "btts"
		}
		if s == "no" || s == "non" {
			return "btts_no"
		}
	}

	if strings.Contains(m, "draw no bet") || strings.Contains(m, "dnb") {
		if s == "home" || strings.Contains(s, "1") {
			return "draw_no_bet_home"
		}
		if s == "away" || strings.Contains(s, "2") {
			return "draw_no_bet_away"
		}
	}

	if strings.Contains(m, "goal") || strings.Contains(m, "total") {
		if strings.Contains(m, "corner") {
			return mapTotalLine(s, "corners")
		}
		if strings.Contains(m, "home") || strings.Contains(m, "team 1") {
			return mapTeamTotal(s, "home")
		}
		if strings.Contains(m, "away") || strings.Contains(m, "team 2") {
			return mapTeamTotal(s, "away")
		}
		if strings.Contains(m, "half") || strings.Contains(m, "1st") {
			if strings.Contains(s, "over") && strings.Contains(s, "0.5") {
				return "over_0_5_ht"
			}
		}
		return mapTotalLine(s, "goals")
	}

	if strings.Contains(m, "corner") {
		if strings.Contains(m, "winner") || strings.Contains(m, "most") {
			if s == "home" || strings.Contains(s, "1") {
				return "corner_winner_home"
			}
			if s == "away" || strings.Contains(s, "2") {
				return "corner_winner_away"
			}
		}
		return mapTotalLine(s, "corners")
	}

	return ""
}

func mapTotalLine(selection, kind string) string {
	s := strings.ToLower(selection)
	prefix := "over"
	if strings.Contains(s, "under") {
		prefix = "under"
	}
	m := goalLineRe.FindStringSubmatch(s)
	if len(m) < 3 {
		return ""
	}
	line := strings.ReplaceAll(m[2], ".", "_")
	if kind == "corners" {
		if prefix == "over" {
			return "over_corners_" + line
		}
		return ""
	}
	return prefix + "_" + line
}

func mapTeamTotal(selection, side string) string {
	s := strings.ToLower(selection)
	if !strings.Contains(s, "over") {
		return ""
	}
	m := goalLineRe.FindStringSubmatch(s)
	if len(m) < 3 {
		if strings.Contains(s, "0.5") || strings.Contains(s, "0,5") {
			return "team_over_0_5_" + side
		}
		return ""
	}
	line := strings.ReplaceAll(strings.ReplaceAll(m[2], ".", "_"), ",", "_")
	return "team_over_" + line + "_" + side
}

// BestOddForMarket returns the highest decimal odd for a mapped ML market.
func BestOddForMarket(matchOdds []db.MatchOdd, mlMarket string) float64 {
	var best float64
	for _, o := range matchOdds {
		if MapToMLKey(o.Market, o.Selection) != mlMarket || o.Odd <= 1 {
			continue
		}
		if o.Odd > best {
			best = o.Odd
		}
	}
	return best
}

// ValueEdgeForMarket returns ML probability minus implied probability for a mapped market.
func ValueEdgeForMarket(matchOdds []db.MatchOdd, mlMarket string, mlProb float64) float64 {
	best := BestOddForMarket(matchOdds, mlMarket)
	if best <= 1 {
		return 0
	}
	return mlProb - (1 / best)
}

// ParseOddLine extracts numeric line from selection like "Over 2.5".
func ParseOddLine(selection string) float64 {
	m := goalLineRe.FindStringSubmatch(strings.ToLower(selection))
	if len(m) < 3 {
		return 0
	}
	v, _ := strconv.ParseFloat(strings.ReplaceAll(m[2], ",", "."), 64)
	return v
}
