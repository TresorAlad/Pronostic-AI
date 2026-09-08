package predictions

import (
	"context"
	"fmt"

	"github.com/prono/backend/internal/db"
)

func (h *Handler) buildFeatures(ctx context.Context, match *db.Match) (map[string]interface{}, error) {
	enricher := h.historyEnricher

	dbHomeAvg, err := h.store.GetTeamRecentStatsAverages(ctx, match.HomeTeam.ID, match.KickoffAt, 5)
	if err != nil {
		return nil, fmt.Errorf("historique domicile: %w", err)
	}
	dbAwayAvg, err := h.store.GetTeamRecentStatsAverages(ctx, match.AwayTeam.ID, match.KickoffAt, 5)
	if err != nil {
		return nil, fmt.Errorf("historique extérieur: %w", err)
	}

	homeAvg, homeHistory, homeSource := enricher.resolveTeamStats(ctx, match.HomeTeam, match.KickoffAt, dbHomeAvg)
	awayAvg, awayHistory, awaySource := enricher.resolveTeamStats(ctx, match.AwayTeam, match.KickoffAt, dbAwayAvg)
	h2hHistory, h2hSource := enricher.resolveH2H(ctx, match, match.KickoffAt)

	leagueAvgGoals, err := h.store.GetLeagueAverageGoals(ctx, match.LeagueID, match.KickoffAt, 200)
	if err != nil {
		return nil, fmt.Errorf("moyenne ligue: %w", err)
	}
	if leagueAvgGoals <= 0 {
		leagueAvgGoals = (homeAvg.GoalsAvg + awayAvg.GoalsAvg) / 2
	}
	if leagueAvgGoals <= 0 {
		leagueAvgGoals = 2.6
	}

	homeHomeForm, _ := h.store.GetTeamVenueForm(ctx, match.HomeTeam.ID, "home", match.KickoffAt, 5)
	awayAwayForm, _ := h.store.GetTeamVenueForm(ctx, match.AwayTeam.ID, "away", match.KickoffAt, 5)
	if homeHomeForm <= 0 && len(homeHistory) > 0 {
		homeHomeForm = venueFormFromHistory(homeHistory, true)
	}
	if awayAwayForm <= 0 && len(awayHistory) > 0 {
		awayAwayForm = venueFormFromHistory(awayHistory, false)
	}

	homeAvailability, err := h.store.GetTeamAvailability(ctx, match.HomeTeam.ID)
	if err != nil {
		return nil, fmt.Errorf("disponibilité domicile: %w", err)
	}
	awayAvailability, err := h.store.GetTeamAvailability(ctx, match.AwayTeam.ID)
	if err != nil {
		return nil, fmt.Errorf("disponibilité extérieur: %w", err)
	}

	dataQuality := dataQualityLabel(homeAvg.MatchCount, awayAvg.MatchCount, homeSource+"+"+awaySource)

	features := map[string]interface{}{
		"home_goals_avg_5":          homeAvg.GoalsAvg,
		"away_goals_avg_5":          awayAvg.GoalsAvg,
		"home_goals_conceded_avg_5": homeAvg.GoalsConcededAvg,
		"away_goals_conceded_avg_5": awayAvg.GoalsConcededAvg,
		"home_form":                 homeAvg.Form,
		"away_form":                 awayAvg.Form,
		"home_home_form":            homeHomeForm,
		"away_away_form":            awayAwayForm,
		"home_attack_strength":      homeAvg.GoalsAvg / leagueAvgGoals,
		"away_attack_strength":      awayAvg.GoalsAvg / leagueAvgGoals,
		"home_defense_strength":     homeAvg.GoalsConcededAvg / leagueAvgGoals,
		"away_defense_strength":     awayAvg.GoalsConcededAvg / leagueAvgGoals,
		"home_advantage":            0.15,
		"home_availability":         homeAvailability,
		"away_availability":         awayAvailability,
		"home_team_id":              match.HomeTeam.ID,
		"away_team_id":              match.AwayTeam.ID,
		"data_source":               "database",
		"data_quality":              dataQuality,
		"match_history_home":        homeAvg.MatchCount,
		"match_history_away":        awayAvg.MatchCount,
		"stats_history_home":        homeAvg.StatsMatchCount,
		"stats_history_away":        awayAvg.StatsMatchCount,
		"history_source_home":       homeSource,
		"history_source_away":       awaySource,
		"league_avg_goals":          leagueAvgGoals,
	}

	if len(homeHistory) > 0 {
		features["home_recent_matches"] = homeHistory
	}
	if len(awayHistory) > 0 {
		features["away_recent_matches"] = awayHistory
	}
	if len(h2hHistory) > 0 {
		features["h2h_history"] = h2hHistory
		features["h2h_source"] = h2hSource
	}

	setIfPositive := func(key string, value float64) {
		if value > 0 {
			features[key] = value
		}
	}

	setIfPositive("home_shots_avg_5", homeAvg.ShotsAvg)
	setIfPositive("away_shots_avg_5", awayAvg.ShotsAvg)
	setIfPositive("home_shots_on_target_avg_5", homeAvg.ShotsOnTargetAvg)
	setIfPositive("away_shots_on_target_avg_5", awayAvg.ShotsOnTargetAvg)
	setIfPositive("home_corners_avg_5", homeAvg.CornersAvg)
	setIfPositive("away_corners_avg_5", awayAvg.CornersAvg)
	setIfPositive("home_possession_avg_5", homeAvg.PossessionAvg)
	setIfPositive("away_possession_avg_5", awayAvg.PossessionAvg)
	setIfPositive("home_xg_avg_5", homeAvg.XGAvg)
	setIfPositive("away_xg_avg_5", awayAvg.XGAvg)
	setIfPositive("home_fouls_avg_5", homeAvg.FoulsAvg)
	setIfPositive("away_fouls_avg_5", awayAvg.FoulsAvg)
	setIfPositive("home_yellow_cards_avg_5", homeAvg.YellowCardsAvg)
	setIfPositive("away_yellow_cards_avg_5", awayAvg.YellowCardsAvg)
	setIfPositive("home_offsides_avg_5", homeAvg.OffsidesAvg)
	setIfPositive("away_offsides_avg_5", awayAvg.OffsidesAvg)

	if match.Status == "live" {
		features["minute"] = match.Minute
		hs, aws := 0, 0
		if match.HomeScore != nil {
			hs = *match.HomeScore
		}
		if match.AwayScore != nil {
			aws = *match.AwayScore
		}
		features["current_total_goals"] = hs + aws
		features["home_score"] = hs
		features["away_score"] = aws

		stats, err := h.store.GetMatchStats(ctx, match.ID)
		if err == nil {
			for _, s := range stats {
				prefix := "home"
				if s.TeamID == match.AwayTeam.ID {
					prefix = "away"
				}
				if s.TotalShots != nil {
					features[prefix+"_live_shots"] = *s.TotalShots
				}
				if s.ShotsOnGoal != nil {
					features[prefix+"_live_shots_on_target"] = *s.ShotsOnGoal
				}
				if s.CornerKicks != nil {
					features[prefix+"_live_corners"] = *s.CornerKicks
				}
				if s.BallPossession != nil {
					features[prefix+"_live_possession"] = *s.BallPossession
				}
				if s.Fouls != nil {
					features[prefix+"_live_fouls"] = *s.Fouls
				}
				if s.YellowCards != nil {
					features[prefix+"_live_yellow_cards"] = *s.YellowCards
				}
				if s.Offsides != nil {
					features[prefix+"_live_offsides"] = *s.Offsides
				}
			}
		}
	}

	return features, nil
}

func venueFormFromHistory(entries []db.TeamMatchHistoryEntry, homeOnly bool) float64 {
	var pts float64
	var n int
	for _, e := range entries {
		if homeOnly && !e.IsHome {
			continue
		}
		if !homeOnly && e.IsHome {
			continue
		}
		switch e.Result {
		case "W":
			pts += 3
		case "D":
			pts += 1
		}
		n++
	}
	if n == 0 {
		return 0
	}
	return pts / (float64(n) * 3)
}
