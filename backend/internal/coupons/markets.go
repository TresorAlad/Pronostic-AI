package coupons

import "strings"

func marketCategory(market string) string {
	switch {
	case market == "home_win" || market == "draw" || market == "away_win":
		return "Temps réglementaire"
	case strings.HasPrefix(market, "double_chance_"):
		return "Temps réglementaire"
	case strings.HasPrefix(market, "over_corners") || strings.HasPrefix(market, "under_corners"):
		return "Corners"
	case strings.Contains(market, "shots"):
		return "Tirs"
	case strings.Contains(market, "cards"):
		return "Cartons"
	case strings.Contains(market, "fouls"):
		return "Fautes"
	case strings.Contains(market, "offsides"):
		return "Hors-jeu"
	case strings.Contains(market, "possession"):
		return "Possession"
	case strings.HasPrefix(market, "over_") || strings.HasPrefix(market, "under_") ||
		market == "btts" || market == "btts_no":
		return "Buts"
	default:
		return "Autre"
	}
}

func marketLabel(market string) string {
	labels := map[string]string{
		"home_win":                   "Victoire domicile",
		"draw":                       "Match nul",
		"away_win":                   "Victoire extérieur",
		"over_1_5":                   "Plus de 1,5 buts",
		"over_2_5":                   "Plus de 2,5 buts",
		"over_3_5":                   "Plus de 3,5 buts",
		"under_1_5":                  "Moins de 1,5 buts",
		"under_2_5":                  "Moins de 2,5 buts",
		"btts":                       "Les deux équipes marquent",
		"btts_no":                    "Les deux équipes ne marquent pas",
		"over_corners_9_5":           "Plus de 9,5 corners",
		"over_shots_22_5":            "Plus de 22,5 tirs",
		"over_shots_on_target_8_5":   "Plus de 8,5 tirs cadrés",
		"over_fouls_20_5":            "Plus de 20,5 fautes",
		"over_fouls_22_5":            "Plus de 22,5 fautes",
		"over_fouls_25_5":            "Plus de 25,5 fautes",
		"over_cards_3_5":             "Plus de 3,5 cartons",
		"over_cards_4_5":             "Plus de 4,5 cartons",
		"over_cards_5_5":             "Plus de 5,5 cartons",
		"over_offsides_2_5":          "Plus de 2,5 hors-jeu",
		"over_offsides_3_5":          "Plus de 3,5 hors-jeu",
		"over_offsides_4_5":          "Plus de 4,5 hors-jeu",
		"home_possession_over_50":    "Domicile > 50 % possession",
		"away_possession_over_50":    "Extérieur > 50 % possession",
		"double_chance_1x":           "Double chance 1X (domicile ou nul)",
		"double_chance_x2":           "Double chance X2 (nul ou extérieur)",
		"double_chance_12":           "Double chance 12 (pas de nul)",
		"over_corners_9.5":           "Plus de 9,5 corners",
	}
	if label, ok := labels[market]; ok {
		return label
	}
	if label, ok := labels[strings.Replace(market, ".", "_", 1)]; ok {
		return label
	}
	return strings.ReplaceAll(market, "_", " ")
}
