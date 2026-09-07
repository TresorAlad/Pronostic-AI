package sync

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/prono/collector/internal/apifootball"
	"github.com/prono/collector/internal/config"
	"github.com/prono/collector/internal/db"
	"github.com/redis/go-redis/v9"
)

type Service struct {
	api             *apifootball.Client
	store           *db.Store
	redis           *redis.Client
	liveScope       string
	liveMaxFixtures int
}

func NewService(api *apifootball.Client, store *db.Store, redisClient *redis.Client, liveScope string, liveMaxFixtures int) *Service {
	if liveScope == "" {
		liveScope = "all"
	}
	if liveMaxFixtures <= 0 {
		liveMaxFixtures = 20
	}
	return &Service{
		api:             api,
		store:           store,
		redis:           redisClient,
		liveScope:       liveScope,
		liveMaxFixtures: liveMaxFixtures,
	}
}

func (s *Service) SyncLeagueSeason(ctx context.Context, leagueID, season int) error {
	log.Printf("Syncing league %d season %d", leagueID, season)

	fixtures, err := s.api.GetFixtures(ctx, leagueID, season)
	if err != nil {
		return fmt.Errorf("get fixtures: %w", err)
	}

	dbLeagueID, err := s.store.GetLeagueID(ctx, leagueID)
	if err != nil {
		return fmt.Errorf("get league id: %w", err)
	}

	seasonID, err := s.store.UpsertSeason(ctx, dbLeagueID, season)
	if err != nil {
		return fmt.Errorf("upsert season: %w", err)
	}

	for _, f := range fixtures {
		if err := s.processFixture(ctx, f, dbLeagueID, &seasonID); err != nil {
			log.Printf("Error processing fixture %d: %v", f.Fixture.ID, err)
		}
	}

	return nil
}

func (s *Service) SyncToday(ctx context.Context) error {
	today := time.Now().Format("2006-01-02")
	fixtures, err := s.api.GetFixturesByDate(ctx, today)
	if err != nil {
		return err
	}

	for _, f := range fixtures {
		if !isTop5League(f.League.ID) {
			continue
		}
		dbLeagueID, err := s.store.GetLeagueID(ctx, f.League.ID)
		if err != nil {
			continue
		}
		seasonID, _ := s.store.UpsertSeason(ctx, dbLeagueID, f.League.Season)
		if err := s.processFixture(ctx, f, dbLeagueID, &seasonID); err != nil {
			log.Printf("Error processing today's fixture %d: %v", f.Fixture.ID, err)
		}
	}
	return nil
}

func (s *Service) SyncLive(ctx context.Context) error {
	fixtures, err := s.api.GetLiveFixtures(ctx)
	if err != nil {
		return err
	}

	synced := 0
	for _, f := range fixtures {
		if s.liveScope == "top5" && !isTop5League(f.League.ID) {
			continue
		}

		dbLeagueID, err := s.store.UpsertLeague(ctx, f.League.ID, f.League.Name, f.League.Country, f.League.Logo)
		if err != nil {
			log.Printf("Error upserting league %d: %v", f.League.ID, err)
			continue
		}
		seasonID, _ := s.store.UpsertSeason(ctx, dbLeagueID, f.League.Season)
		if err := s.processFixture(ctx, f, dbLeagueID, &seasonID); err != nil {
			log.Printf("Error processing live fixture %d: %v", f.Fixture.ID, err)
			continue
		}

		matchID, err := s.store.GetMatchIDByExternal(ctx, f.Fixture.ID)
		if err == nil {
			liveKey := "live:match:" + matchID
			liveData := fmt.Sprintf(`{"minute":%d,"home_score":%v,"away_score":%v}`,
				f.Fixture.Status.Elapsed, ptrInt(f.Goals.Home), ptrInt(f.Goals.Away))
			s.redis.Set(ctx, liveKey, liveData, 5*time.Minute)
			s.redis.Del(ctx, "prediction:"+matchID)
		}

		synced++
		if synced >= s.liveMaxFixtures {
			break
		}
	}

	log.Printf("Synced %d live fixtures (scope=%s)", synced, s.liveScope)
	return nil
}

func (s *Service) Backfill(ctx context.Context, startYear int) error {
	currentYear := time.Now().Year()
	for _, leagueID := range config.Top5LeagueIDs {
		for year := startYear; year <= currentYear; year++ {
			if err := s.SyncLeagueSeason(ctx, leagueID, year); err != nil {
				log.Printf("Backfill error league %d year %d: %v", leagueID, year, err)
			}
		}
	}
	return nil
}

func (s *Service) SyncMatchDetails(ctx context.Context, limit int) error {
	matches, err := s.store.GetFinishedMatchesWithoutStats(ctx, limit)
	if err != nil {
		return err
	}

	for _, m := range matches {
		if err := s.syncFixtureDetails(ctx, m.ID, m.ExternalID); err != nil {
			log.Printf("Error syncing details for match %d: %v", m.ExternalID, err)
		}
	}
	return nil
}

func (s *Service) SyncInjuries(ctx context.Context, leagueID, season int) error {
	injuries, err := s.api.GetInjuries(ctx, leagueID, season)
	if err != nil {
		return err
	}

	for _, inj := range injuries {
		playerID, err := s.store.UpsertPlayer(ctx, inj.Player.ID, inj.Player.Name)
		if err != nil {
			continue
		}
		teamID, err := s.store.UpsertTeam(ctx, inj.Team.ID, inj.Team.Name, inj.Team.Logo)
		if err != nil {
			continue
		}
		s.store.UpsertInjury(ctx, playerID, teamID, inj.Reason, inj.Type)
	}
	return nil
}

func (s *Service) processFixture(ctx context.Context, f apifootball.FixtureResponse, leagueID string, seasonID *string) error {
	homeTeamID, err := s.store.UpsertTeam(ctx, f.Teams.Home.ID, f.Teams.Home.Name, f.Teams.Home.Logo)
	if err != nil {
		return err
	}
	awayTeamID, err := s.store.UpsertTeam(ctx, f.Teams.Away.ID, f.Teams.Away.Name, f.Teams.Away.Logo)
	if err != nil {
		return err
	}

	kickoff, err := time.Parse(time.RFC3339, f.Fixture.Date)
	if err != nil {
		return fmt.Errorf("invalid kickoff date for fixture %d: %w", f.Fixture.ID, err)
	}
	status := mapStatus(f.Fixture.Status.Short)
	var minute *int
	if f.Fixture.Status.Elapsed > 0 {
		m := f.Fixture.Status.Elapsed
		minute = &m
	}

	_, err = s.store.UpsertMatch(ctx, db.MatchParams{
		ExternalID:  f.Fixture.ID,
		LeagueID:    leagueID,
		SeasonID:    seasonID,
		HomeTeamID:  homeTeamID,
		AwayTeamID:  awayTeamID,
		KickoffAt:   kickoff,
		Status:      status,
		Minute:      minute,
		HomeScore:   f.Goals.Home,
		AwayScore:   f.Goals.Away,
		HomeScoreHT: f.Score.Halftime.Home,
		AwayScoreHT: f.Score.Halftime.Away,
		Venue:       f.Fixture.Venue.Name,
		Referee:     f.Fixture.Referee,
		Round:       f.League.Round,
	})
	if err != nil {
		return err
	}

	if status == "finished" {
		// Details synced separately via SyncMatchDetails
	}

	return nil
}

func (s *Service) syncFixtureDetails(ctx context.Context, matchID string, externalID int) error {
	stats, err := s.api.GetFixtureStatistics(ctx, externalID)
	if err != nil {
		return err
	}

	for _, teamStats := range stats {
		teamID, err := s.store.UpsertTeam(ctx, teamStats.Team.ID, teamStats.Team.Name, teamStats.Team.Logo)
		if err != nil {
			continue
		}
		ms := parseStatistics(teamStats.Statistics)
		s.store.UpsertMatchStatistics(ctx, matchID, teamID, ms)
	}

	events, err := s.api.GetFixtureEvents(ctx, externalID)
	if err == nil {
		s.store.DeleteMatchEvents(ctx, matchID)
		for _, ev := range events {
			var teamID, playerID *string
			if ev.Team.ID > 0 {
				tid, _ := s.store.UpsertTeam(ctx, ev.Team.ID, ev.Team.Name, ev.Team.Logo)
				teamID = &tid
			}
			if ev.Player.ID > 0 {
				pid, _ := s.store.UpsertPlayer(ctx, ev.Player.ID, ev.Player.Name)
				playerID = &pid
			}
			eventType := mapEventType(ev.Type, ev.Detail)
			s.store.InsertMatchEvent(ctx, matchID, teamID, playerID, eventType, ev.Detail, ev.Time.Elapsed, ev.Time.Extra)
		}
	}

	return nil
}

func mapStatus(short string) string {
	switch short {
	case "NS", "TBD":
		return "scheduled"
	case "1H", "2H", "HT", "ET", "BT", "P", "LIVE":
		return "live"
	case "FT", "AET", "PEN":
		return "finished"
	case "PST":
		return "postponed"
	case "CANC":
		return "cancelled"
	case "SUSP", "INT":
		return "suspended"
	default:
		return "scheduled"
	}
}

func mapEventType(eventType, detail string) string {
	switch strings.ToLower(eventType) {
	case "goal":
		if strings.Contains(strings.ToLower(detail), "own") {
			return "own_goal"
		}
		return "goal"
	case "card":
		if strings.Contains(strings.ToLower(detail), "red") {
			return "red_card"
		}
		return "yellow_card"
	case "subst":
		return "substitution"
	case "var":
		return "var"
	default:
		return "unknown"
	}
}

func parseStatistics(items []apifootball.StatItem) db.MatchStats {
	var stats db.MatchStats
	for _, item := range items {
		val := item.Value
		switch item.Type {
		case "Shots on Goal":
			stats.ShotsOnGoal = toIntPtr(val)
		case "Shots off Goal":
			stats.ShotsOffGoal = toIntPtr(val)
		case "Total Shots":
			stats.TotalShots = toIntPtr(val)
		case "Blocked Shots":
			stats.BlockedShots = toIntPtr(val)
		case "Shots insidebox":
			stats.ShotsInsideBox = toIntPtr(val)
		case "Shots outsidebox":
			stats.ShotsOutsideBox = toIntPtr(val)
		case "Fouls":
			stats.Fouls = toIntPtr(val)
		case "Corner Kicks":
			stats.CornerKicks = toIntPtr(val)
		case "Offsides":
			stats.Offsides = toIntPtr(val)
		case "Ball Possession":
			stats.BallPossession = toFloatPtr(val)
		case "Yellow Cards":
			stats.YellowCards = toIntPtr(val)
		case "Red Cards":
			stats.RedCards = toIntPtr(val)
		case "Goalkeeper Saves":
			stats.GoalkeeperSaves = toIntPtr(val)
		case "Total passes":
			stats.TotalPasses = toIntPtr(val)
		case "Passes accurate":
			stats.PassesAccurate = toIntPtr(val)
		case "Passes %":
			stats.PassesPct = toFloatPtr(val)
		case "expected_goals":
			stats.ExpectedGoals = toFloatPtr(val)
		}
	}
	return stats
}

func toIntPtr(v interface{}) *int {
	switch val := v.(type) {
	case float64:
		i := int(val)
		return &i
	case string:
		val = strings.TrimSuffix(val, "%")
		if n, err := strconv.Atoi(val); err == nil {
			return &n
		}
	}
	return nil
}

func toFloatPtr(v interface{}) *float64 {
	switch val := v.(type) {
	case float64:
		return &val
	case string:
		val = strings.TrimSuffix(val, "%")
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return &f
		}
	}
	return nil
}

func isTop5League(id int) bool {
	for _, lid := range config.Top5LeagueIDs {
		if lid == id {
			return true
		}
	}
	return false
}

func ptrInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
