package markets

import "strings"

// Entry describes a canonical ML market for UI and coupons.
type Entry struct {
	Key              string
	LabelFR          string
	Category         string
	BookmakerAliases []string
}

var goalKeys = []string{
	"over_1_5", "over_2_5", "over_3_5", "under_1_5", "under_2_5",
	"btts", "btts_no",
	"team_over_0_5_home", "team_over_1_5_home", "team_over_0_5_away", "team_over_1_5_away",
	"result_btts_home_yes", "result_btts_draw_yes", "result_btts_away_yes",
	"over_0_5_ht",
}

// Catalog is the single source of truth for French labels (sync with frontend marketLabels.ts).
var Catalog = []Entry{
	{Key: "home_win", LabelFR: "Victoire domicile", Category: "Temps réglementaire"},
	{Key: "draw", LabelFR: "Match nul", Category: "Temps réglementaire"},
	{Key: "away_win", LabelFR: "Victoire extérieure", Category: "Temps réglementaire"},
	{Key: "double_chance_1x", LabelFR: "Double chance 1X (domicile ou nul)", Category: "Temps réglementaire"},
	{Key: "double_chance_x2", LabelFR: "Double chance X2 (nul ou extérieur)", Category: "Temps réglementaire"},
	{Key: "double_chance_12", LabelFR: "Double chance 12 (pas de nul)", Category: "Temps réglementaire"},
	{Key: "over_1_5", LabelFR: "Plus de 1,5 buts", Category: "Buts", BookmakerAliases: []string{"goals over/under", "total goals"}},
	{Key: "over_2_5", LabelFR: "Plus de 2,5 buts", Category: "Buts", BookmakerAliases: []string{"goals over/under", "total goals"}},
	{Key: "over_3_5", LabelFR: "Plus de 3,5 buts", Category: "Buts", BookmakerAliases: []string{"goals over/under", "total goals"}},
	{Key: "under_1_5", LabelFR: "Moins de 1,5 buts", Category: "Buts"},
	{Key: "under_2_5", LabelFR: "Moins de 2,5 buts", Category: "Buts"},
	{Key: "btts", LabelFR: "Les deux équipes marquent", Category: "Buts", BookmakerAliases: []string{"both teams score"}},
	{Key: "btts_no", LabelFR: "Les deux équipes ne marquent pas", Category: "Buts", BookmakerAliases: []string{"both teams score"}},
	{Key: "team_over_0_5_home", LabelFR: "Domicile marque au moins 1 but", Category: "Buts"},
	{Key: "team_over_1_5_home", LabelFR: "Domicile marque plus de 1,5 buts", Category: "Buts"},
	{Key: "team_over_0_5_away", LabelFR: "Extérieur marque au moins 1 but", Category: "Buts"},
	{Key: "team_over_1_5_away", LabelFR: "Extérieur marque plus de 1,5 buts", Category: "Buts"},
	{Key: "result_btts_home_yes", LabelFR: "Victoire domicile et les deux équipes marquent", Category: "Buts"},
	{Key: "result_btts_draw_yes", LabelFR: "Match nul et les deux équipes marquent", Category: "Buts"},
	{Key: "result_btts_away_yes", LabelFR: "Victoire extérieure et les deux équipes marquent", Category: "Buts"},
	{Key: "over_0_5_ht", LabelFR: "Plus de 0,5 but en première mi-temps", Category: "Buts"},
	{Key: "draw_no_bet_home", LabelFR: "Victoire domicile (remboursé si nul)", Category: "Temps réglementaire"},
	{Key: "draw_no_bet_away", LabelFR: "Victoire extérieure (remboursé si nul)", Category: "Temps réglementaire"},
	{Key: "over_corners_9_5", LabelFR: "Plus de 9,5 corners", Category: "Corners"},
	{Key: "over_corners_9.5", LabelFR: "Plus de 9,5 corners", Category: "Corners"},
	{Key: "corner_winner_home", LabelFR: "Domicile gagne aux corners", Category: "Corners"},
	{Key: "corner_winner_away", LabelFR: "Extérieur gagne aux corners", Category: "Corners"},
	{Key: "over_shots_22_5", LabelFR: "Plus de 22,5 tirs", Category: "Tirs"},
	{Key: "over_shots_on_target_8_5", LabelFR: "Plus de 8,5 tirs cadrés", Category: "Tirs"},
	{Key: "over_fouls_20_5", LabelFR: "Plus de 20,5 fautes", Category: "Fautes"},
	{Key: "over_fouls_22_5", LabelFR: "Plus de 22,5 fautes", Category: "Fautes"},
	{Key: "over_fouls_25_5", LabelFR: "Plus de 25,5 fautes", Category: "Fautes"},
	{Key: "over_cards_3_5", LabelFR: "Plus de 3,5 cartons", Category: "Cartons"},
	{Key: "over_cards_4_5", LabelFR: "Plus de 4,5 cartons", Category: "Cartons"},
	{Key: "over_cards_5_5", LabelFR: "Plus de 5,5 cartons", Category: "Cartons"},
	{Key: "over_offsides_2_5", LabelFR: "Plus de 2,5 hors-jeu", Category: "Hors-jeu"},
	{Key: "over_offsides_3_5", LabelFR: "Plus de 3,5 hors-jeu", Category: "Hors-jeu"},
	{Key: "over_offsides_4_5", LabelFR: "Plus de 4,5 hors-jeu", Category: "Hors-jeu"},
	{Key: "home_possession_over_50", LabelFR: "Domicile > 50 % possession", Category: "Possession"},
	{Key: "away_possession_over_50", LabelFR: "Extérieur > 50 % possession", Category: "Possession"},
}

var byKey map[string]Entry

func init() {
	byKey = make(map[string]Entry, len(Catalog))
	for _, e := range Catalog {
		byKey[e.Key] = e
	}
}

func Label(key string) string {
	if e, ok := byKey[key]; ok {
		return e.LabelFR
	}
	if e, ok := byKey[normalizeKey(key)]; ok {
		return e.LabelFR
	}
	return ""
}

func Category(key string) string {
	if e, ok := byKey[key]; ok {
		return e.Category
	}
	if e, ok := byKey[normalizeKey(key)]; ok {
		return e.Category
	}
	return "Autre"
}

func GoalKeys() []string {
	return append([]string(nil), goalKeys...)
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(key, ".", "_")
}
