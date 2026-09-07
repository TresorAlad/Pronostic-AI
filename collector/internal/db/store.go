package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect db: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() {
	s.pool.Close()
}

func (s *Store) UpsertTeam(ctx context.Context, externalID int, name, logo string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO teams (external_id, name, logo_url)
		VALUES ($1, $2, $3)
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			logo_url = EXCLUDED.logo_url,
			updated_at = NOW()
		RETURNING id
	`, externalID, name, logo).Scan(&id)
	return id, err
}

func (s *Store) GetLeagueID(ctx context.Context, externalID int) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM leagues WHERE external_id = $1`, externalID).Scan(&id)
	return id, err
}

func (s *Store) UpsertLeague(ctx context.Context, externalID int, name, country, logo string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO leagues (external_id, name, country, logo_url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			country = COALESCE(NULLIF(EXCLUDED.country, ''), leagues.country),
			logo_url = COALESCE(NULLIF(EXCLUDED.logo_url, ''), leagues.logo_url),
			updated_at = NOW()
		RETURNING id
	`, externalID, name, country, logo).Scan(&id)
	return id, err
}

func (s *Store) UpsertSeason(ctx context.Context, leagueID string, year int) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO seasons (league_id, year, is_current)
		VALUES ($1, $2, $3)
		ON CONFLICT (league_id, year) DO UPDATE SET is_current = EXCLUDED.is_current
		RETURNING id
	`, leagueID, year, year >= 2025).Scan(&id)
	return id, err
}

func (s *Store) UpsertMatch(ctx context.Context, params MatchParams) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO matches (
			external_id, league_id, season_id, home_team_id, away_team_id,
			kickoff_at, status, minute, home_score, away_score,
			home_score_ht, away_score_ht, venue, referee, round
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		ON CONFLICT (external_id) DO UPDATE SET
			status = EXCLUDED.status,
			minute = EXCLUDED.minute,
			home_score = EXCLUDED.home_score,
			away_score = EXCLUDED.away_score,
			home_score_ht = EXCLUDED.home_score_ht,
			away_score_ht = EXCLUDED.away_score_ht,
			updated_at = NOW()
		RETURNING id
	`, params.ExternalID, params.LeagueID, params.SeasonID, params.HomeTeamID, params.AwayTeamID,
		params.KickoffAt, params.Status, params.Minute, params.HomeScore, params.AwayScore,
		params.HomeScoreHT, params.AwayScoreHT, params.Venue, params.Referee, params.Round).Scan(&id)
	return id, err
}

func (s *Store) GetMatchIDByExternal(ctx context.Context, externalID int) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `SELECT id FROM matches WHERE external_id = $1`, externalID).Scan(&id)
	return id, err
}

type MatchParams struct {
	ExternalID  int
	LeagueID    string
	SeasonID    *string
	HomeTeamID  string
	AwayTeamID  string
	KickoffAt   time.Time
	Status      string
	Minute      *int
	HomeScore   *int
	AwayScore   *int
	HomeScoreHT *int
	AwayScoreHT *int
	Venue       string
	Referee     string
	Round       string
}

func (s *Store) UpsertMatchStatistics(ctx context.Context, matchID, teamID string, stats MatchStats) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO match_statistics (
			match_id, team_id, shots_on_goal, shots_off_goal, total_shots,
			blocked_shots, shots_inside_box, shots_outside_box, fouls,
			corner_kicks, offsides, ball_possession, yellow_cards, red_cards,
			goalkeeper_saves, total_passes, passes_accurate, passes_pct, expected_goals
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)
		ON CONFLICT (match_id, team_id) DO UPDATE SET
			shots_on_goal = EXCLUDED.shots_on_goal,
			total_shots = EXCLUDED.total_shots,
			corner_kicks = EXCLUDED.corner_kicks,
			ball_possession = EXCLUDED.ball_possession,
			expected_goals = EXCLUDED.expected_goals
	`, matchID, teamID, stats.ShotsOnGoal, stats.ShotsOffGoal, stats.TotalShots,
		stats.BlockedShots, stats.ShotsInsideBox, stats.ShotsOutsideBox, stats.Fouls,
		stats.CornerKicks, stats.Offsides, stats.BallPossession, stats.YellowCards, stats.RedCards,
		stats.GoalkeeperSaves, stats.TotalPasses, stats.PassesAccurate, stats.PassesPct, stats.ExpectedGoals)
	return err
}

type MatchStats struct {
	ShotsOnGoal      *int
	ShotsOffGoal     *int
	TotalShots       *int
	BlockedShots     *int
	ShotsInsideBox   *int
	ShotsOutsideBox  *int
	Fouls            *int
	CornerKicks      *int
	Offsides         *int
	BallPossession   *float64
	YellowCards      *int
	RedCards         *int
	GoalkeeperSaves  *int
	TotalPasses      *int
	PassesAccurate   *int
	PassesPct        *float64
	ExpectedGoals    *float64
}

func (s *Store) DeleteMatchEvents(ctx context.Context, matchID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM match_events WHERE match_id = $1`, matchID)
	return err
}

func (s *Store) InsertMatchEvent(ctx context.Context, matchID string, teamID, playerID *string, eventType, detail string, minute int, extraMinute *int) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO match_events (match_id, team_id, player_id, event_type, detail, minute, extra_minute)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, matchID, teamID, playerID, eventType, detail, minute, extraMinute)
	return err
}

func (s *Store) UpsertPlayer(ctx context.Context, externalID int, name string) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO players (external_id, name)
		VALUES ($1, $2)
		ON CONFLICT (external_id) DO UPDATE SET name = EXCLUDED.name, updated_at = NOW()
		RETURNING id
	`, externalID, name).Scan(&id)
	return id, err
}

func (s *Store) UpsertInjury(ctx context.Context, playerID, teamID, reason, injuryType string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO injuries (player_id, team_id, reason, type, is_active)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT DO NOTHING
	`, playerID, teamID, reason, injuryType)
	return err
}

func (s *Store) GetFinishedMatchesWithoutStats(ctx context.Context, limit int) ([]MatchRef, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.external_id
		FROM matches m
		LEFT JOIN match_statistics ms ON ms.match_id = m.id
		WHERE m.status = 'finished' AND ms.id IS NULL
		ORDER BY m.kickoff_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []MatchRef
	for rows.Next() {
		var ref MatchRef
		if err := rows.Scan(&ref.ID, &ref.ExternalID); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}

type MatchRef struct {
	ID         string
	ExternalID int
}

func (s *Store) GetMatchExternalID(ctx context.Context, matchID string) (int, error) {
	var extID int
	err := s.pool.QueryRow(ctx, `SELECT external_id FROM matches WHERE id = $1`, matchID).Scan(&extID)
	return extID, err
}

func (s *Store) GetLiveMatches(ctx context.Context) ([]MatchRef, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, external_id FROM matches WHERE status = 'live'
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var refs []MatchRef
	for rows.Next() {
		var ref MatchRef
		if err := rows.Scan(&ref.ID, &ref.ExternalID); err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, rows.Err()
}
