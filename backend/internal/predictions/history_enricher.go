package predictions

import (
	"context"
	"strings"
	"time"

	"github.com/prono/backend/internal/apifootball"
	"github.com/prono/backend/internal/db"
)

const aiHistoryLimit = 10
const h2hLimit = 8

type HistoryEnricher struct {
	store *db.Store
	api   *apifootball.Client
}

func NewHistoryEnricher(store *db.Store, api *apifootball.Client) *HistoryEnricher {
	return &HistoryEnricher{store: store, api: api}
}

func defaultTeamStats() *db.TeamStatsAverages {
	return &db.TeamStatsAverages{
		GoalsAvg:         1.35,
		GoalsConcededAvg: 1.25,
		Form:             0.5,
		MatchCount:       0,
	}
}

func (e *HistoryEnricher) resolveTeamStats(
	ctx context.Context,
	team db.Team,
	before time.Time,
	dbAvg *db.TeamStatsAverages,
) (*db.TeamStatsAverages, []db.TeamMatchHistoryEntry, string) {
	history, source := e.resolveTeamHistory(ctx, team, before, aiHistoryLimit)
	if dbAvg != nil && dbAvg.MatchCount >= 1 {
		return dbAvg, history, source
	}
	if computed := averagesFromHistory(history); computed != nil && computed.MatchCount > 0 {
		return computed, history, source
	}
	if dbAvg != nil && dbAvg.MatchCount > 0 {
		return dbAvg, history, source
	}
	return defaultTeamStats(), history, source
}

func (e *HistoryEnricher) resolveTeamHistory(
	ctx context.Context,
	team db.Team,
	before time.Time,
	limit int,
) ([]db.TeamMatchHistoryEntry, string) {
	entries, _ := e.store.GetTeamRecentMatchHistory(ctx, team.ID, before, limit)
	if len(entries) >= 3 || e.api == nil || !e.api.Enabled() || team.ExternalID <= 0 {
		if len(entries) > 0 {
			return entries, "database"
		}
		if e.api != nil && e.api.Enabled() && team.ExternalID > 0 {
			apiEntries, err := e.api.GetLastFixturesByTeam(ctx, team.ExternalID, limit)
			if err == nil && len(apiEntries) > 0 {
				return fixturesToTeamHistory(team.ExternalID, apiEntries), "api"
			}
		}
		return entries, "none"
	}

	apiEntries, err := e.api.GetLastFixturesByTeam(ctx, team.ExternalID, limit)
	if err != nil || len(apiEntries) == 0 {
		if len(entries) > 0 {
			return entries, "database"
		}
		return nil, "none"
	}
	apiHistory := fixturesToTeamHistory(team.ExternalID, apiEntries)
	return mergeTeamHistory(entries, apiHistory, limit), "mixed"
}

func (e *HistoryEnricher) resolveH2H(
	ctx context.Context,
	match *db.Match,
	before time.Time,
) ([]db.HeadToHeadEntry, string) {
	entries, _ := e.store.GetHeadToHeadHistory(ctx, match.HomeTeam.ID, match.AwayTeam.ID, before, h2hLimit)
	if len(entries) >= 2 || e.api == nil || !e.api.Enabled() {
		if len(entries) > 0 {
			return entries, "database"
		}
	}
	homeExt := match.HomeTeam.ExternalID
	awayExt := match.AwayTeam.ExternalID
	if homeExt <= 0 || awayExt <= 0 {
		return entries, "none"
	}
	fixtures, err := e.api.GetHeadToHead(ctx, homeExt, awayExt, h2hLimit)
	if err != nil || len(fixtures) == 0 {
		if len(entries) > 0 {
			return entries, "database"
		}
		return nil, "none"
	}
	apiH2H := fixturesToH2H(fixtures)
	if len(entries) == 0 {
		return apiH2H, "api"
	}
	return mergeH2H(entries, apiH2H, h2hLimit), "mixed"
}

func fixturesToTeamHistory(teamExternalID int, fixtures []apifootball.Fixture) []db.TeamMatchHistoryEntry {
	out := make([]db.TeamMatchHistoryEntry, 0, len(fixtures))
	for _, f := range fixtures {
		if f.Goals.Home == nil || f.Goals.Away == nil {
			continue
		}
		isHome := f.Teams.Home.ID == teamExternalID
		gf, ga := *f.Goals.Away, *f.Goals.Home
		opponent := f.Teams.Home.Name
		if isHome {
			gf, ga = *f.Goals.Home, *f.Goals.Away
			opponent = f.Teams.Away.Name
		}
		date := f.Fixture.Date
		if len(date) >= 10 {
			date = date[:10]
		}
		entry := db.TeamMatchHistoryEntry{
			Date:         date,
			LeagueName:   f.League.Name,
			Opponent:     opponent,
			IsHome:       isHome,
			GoalsFor:     gf,
			GoalsAgainst: ga,
			Result:       resultFromGoals(gf, ga),
		}
		if f.League.Round != "" {
			entry.Round = f.League.Round
		}
		out = append(out, entry)
	}
	return out
}

func fixturesToH2H(fixtures []apifootball.Fixture) []db.HeadToHeadEntry {
	out := make([]db.HeadToHeadEntry, 0, len(fixtures))
	for _, f := range fixtures {
		if f.Goals.Home == nil || f.Goals.Away == nil {
			continue
		}
		date := f.Fixture.Date
		if len(date) >= 10 {
			date = date[:10]
		}
		out = append(out, db.HeadToHeadEntry{
			Date:       date,
			LeagueName: f.League.Name,
			HomeTeam:   f.Teams.Home.Name,
			AwayTeam:   f.Teams.Away.Name,
			HomeScore:  *f.Goals.Home,
			AwayScore:  *f.Goals.Away,
		})
	}
	return out
}

func resultFromGoals(gf, ga int) string {
	if gf > ga {
		return "W"
	}
	if gf < ga {
		return "L"
	}
	return "D"
}

func averagesFromHistory(entries []db.TeamMatchHistoryEntry) *db.TeamStatsAverages {
	if len(entries) == 0 {
		return nil
	}
	var goalsFor, goalsAgainst, formPts float64
	for _, e := range entries {
		goalsFor += float64(e.GoalsFor)
		goalsAgainst += float64(e.GoalsAgainst)
		switch e.Result {
		case "W":
			formPts += 3
		case "D":
			formPts += 1
		}
	}
	n := float64(len(entries))
	return &db.TeamStatsAverages{
		MatchCount:       len(entries),
		GoalsAvg:         goalsFor / n,
		GoalsConcededAvg: goalsAgainst / n,
		Form:             formPts / (n * 3),
	}
}

func mergeTeamHistory(a, b []db.TeamMatchHistoryEntry, limit int) []db.TeamMatchHistoryEntry {
	seen := make(map[string]struct{})
	out := make([]db.TeamMatchHistoryEntry, 0, limit)
	appendUnique := func(list []db.TeamMatchHistoryEntry) {
		for _, e := range list {
			key := e.Date + "|" + e.Opponent + "|" + e.LeagueName
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, e)
			if len(out) >= limit {
				return
			}
		}
	}
	appendUnique(a)
	appendUnique(b)
	return out
}

func mergeH2H(a, b []db.HeadToHeadEntry, limit int) []db.HeadToHeadEntry {
	seen := make(map[string]struct{})
	out := make([]db.HeadToHeadEntry, 0, limit)
	appendUnique := func(list []db.HeadToHeadEntry) {
		for _, e := range list {
			key := e.Date + "|" + e.HomeTeam + "|" + e.AwayTeam
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, e)
			if len(out) >= limit {
				return
			}
		}
	}
	appendUnique(a)
	appendUnique(b)
	return out
}

func dataQualityLabel(homeCount, awayCount int, source string) string {
	minCount := homeCount
	if awayCount < minCount {
		minCount = awayCount
	}
	switch {
	case minCount >= 5:
		return "high"
	case minCount >= 2:
		return "medium"
	case minCount >= 1 || strings.Contains(source, "api"):
		return "low"
	default:
		return "estimated"
	}
}
