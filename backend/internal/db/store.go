package db

import (
	"context"
	"encoding/json"
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

type League struct {
	ID         string `json:"id"`
	ExternalID int    `json:"external_id"`
	Name       string `json:"name"`
	Country    string `json:"country"`
	LogoURL    string `json:"logo_url"`
}

type Team struct {
	ID         string `json:"id"`
	ExternalID int    `json:"external_id"`
	Name       string `json:"name"`
	LogoURL    string `json:"logo_url"`
}

type Match struct {
	ID          string    `json:"id"`
	ExternalID  int       `json:"external_id"`
	LeagueID    string    `json:"league_id"`
	LeagueName  string    `json:"league_name"`
	HomeTeam    Team      `json:"home_team"`
	AwayTeam    Team      `json:"away_team"`
	KickoffAt   time.Time `json:"kickoff_at"`
	Status      string    `json:"status"`
	Minute      *int      `json:"minute,omitempty"`
	HomeScore   *int      `json:"home_score,omitempty"`
	AwayScore   *int      `json:"away_score,omitempty"`
	Venue       string    `json:"venue,omitempty"`
	Round       string    `json:"round,omitempty"`
}

type MatchStats struct {
	TeamID         string   `json:"team_id"`
	TeamName       string   `json:"team_name"`
	TotalShots     *int     `json:"total_shots"`
	ShotsOnGoal    *int     `json:"shots_on_goal"`
	CornerKicks    *int     `json:"corner_kicks"`
	BallPossession *float64 `json:"ball_possession"`
	ExpectedGoals  *float64 `json:"expected_goals"`
	Fouls          *int     `json:"fouls"`
	YellowCards    *int     `json:"yellow_cards"`
}

type User struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	PasswordHash string `json:"-"`
}

type Prediction struct {
	ID               string          `json:"id"`
	MatchID          string          `json:"match_id"`
	ModelVersion     string          `json:"model_version"`
	Predictions      json.RawMessage `json:"predictions"`
	Confidence       json.RawMessage `json:"confidence"`
	NoBetRecommended bool            `json:"no_bet_recommended"`
	DataSnapshotAt   time.Time       `json:"data_snapshot_at"`
	IsLive           bool            `json:"is_live"`
	AIAnalysis       *string         `json:"ai_analysis,omitempty"`
	AIReasons        json.RawMessage `json:"ai_reasons,omitempty"`
	AIAbstain        bool            `json:"ai_abstain"`
	CreatedAt        time.Time       `json:"created_at"`
}

type CouponSelection struct {
	MatchID    string  `json:"match_id"`
	HomeTeam   string  `json:"home_team"`
	AwayTeam   string  `json:"away_team"`
	Market     string  `json:"market"`
	Selection  string  `json:"selection"`
	Confidence float64 `json:"confidence"`
}

type ModelPerformance struct {
	ModelName   string   `json:"model_name"`
	ModelVersion string  `json:"model_version"`
	Market      string   `json:"market"`
	Accuracy    *float64 `json:"accuracy"`
	LogLoss     *float64 `json:"log_loss"`
	BrierScore  *float64 `json:"brier_score"`
	SampleSize  *int     `json:"sample_size"`
}

func (s *Store) ListLeagues(ctx context.Context) ([]League, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, external_id, name, country, COALESCE(logo_url,'') FROM leagues ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var leagues []League
	for rows.Next() {
		var l League
		if err := rows.Scan(&l.ID, &l.ExternalID, &l.Name, &l.Country, &l.LogoURL); err != nil {
			return nil, err
		}
		leagues = append(leagues, l)
	}
	return leagues, rows.Err()
}

func (s *Store) GetMatchesToday(ctx context.Context) ([]Match, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.external_id, m.league_id, l.name,
			ht.id, ht.external_id, ht.name, COALESCE(ht.logo_url,''),
			at.id, at.external_id, at.name, COALESCE(at.logo_url,''),
			m.kickoff_at, m.status::text, m.minute, m.home_score, m.away_score,
			COALESCE(m.venue,''), COALESCE(m.round,'')
		FROM matches m
		JOIN leagues l ON l.id = m.league_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		WHERE l.external_id IN (39, 140, 135, 78, 61)
		  AND m.status != 'live'
		  AND (
		    m.kickoff_at::date = CURRENT_DATE
		    OR (m.status = 'scheduled' AND m.kickoff_at BETWEEN NOW() AND NOW() + INTERVAL '7 days')
		  )
		ORDER BY
			CASE m.status WHEN 'scheduled' THEN 0 ELSE 1 END,
			m.kickoff_at
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches, err := scanMatches(rows)
	if err != nil {
		return nil, err
	}
	if len(matches) > 0 {
		return matches, nil
	}
	return s.getRecentMatches(ctx, 20)
}

func (s *Store) getRecentMatches(ctx context.Context, limit int) ([]Match, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.external_id, m.league_id, l.name,
			ht.id, ht.external_id, ht.name, COALESCE(ht.logo_url,''),
			at.id, at.external_id, at.name, COALESCE(at.logo_url,''),
			m.kickoff_at, m.status::text, m.minute, m.home_score, m.away_score,
			COALESCE(m.venue,''), COALESCE(m.round,'')
		FROM matches m
		JOIN leagues l ON l.id = m.league_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		WHERE m.status = 'finished'
		  AND l.external_id IN (39, 140, 135, 78, 61)
		ORDER BY m.kickoff_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

func (s *Store) GetMatchByID(ctx context.Context, id string) (*Match, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT m.id, m.external_id, m.league_id, l.name,
			ht.id, ht.external_id, ht.name, COALESCE(ht.logo_url,''),
			at.id, at.external_id, at.name, COALESCE(at.logo_url,''),
			m.kickoff_at, m.status::text, m.minute, m.home_score, m.away_score,
			COALESCE(m.venue,''), COALESCE(m.round,'')
		FROM matches m
		JOIN leagues l ON l.id = m.league_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		WHERE m.id = $1
	`, id)
	return scanMatch(row)
}

func (s *Store) GetMatchStats(ctx context.Context, matchID string) ([]MatchStats, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT ms.team_id, t.name, ms.total_shots, ms.shots_on_goal,
			ms.corner_kicks, ms.ball_possession, ms.expected_goals, ms.fouls, ms.yellow_cards
		FROM match_statistics ms
		JOIN teams t ON t.id = ms.team_id
		WHERE ms.match_id = $1
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []MatchStats
	for rows.Next() {
		var st MatchStats
		if err := rows.Scan(&st.TeamID, &st.TeamName, &st.TotalShots, &st.ShotsOnGoal,
			&st.CornerKicks, &st.BallPossession, &st.ExpectedGoals, &st.Fouls, &st.YellowCards); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3) RETURNING id, email, display_name
	`, email, passwordHash, displayName).Scan(&u.ID, &u.Email, &u.DisplayName)
	return &u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, COALESCE(display_name,'') FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) SavePrediction(ctx context.Context, p *Prediction) error {
	return s.pool.QueryRow(ctx, `
		INSERT INTO predictions (match_id, model_version, predictions, confidence,
			no_bet_recommended, data_snapshot_at, is_live, ai_analysis, ai_reasons, ai_abstain)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, created_at
	`, p.MatchID, p.ModelVersion, p.Predictions, p.Confidence,
		p.NoBetRecommended, p.DataSnapshotAt, p.IsLive, p.AIAnalysis, p.AIReasons, p.AIAbstain,
	).Scan(&p.ID, &p.CreatedAt)
}

func (s *Store) GetLatestPrediction(ctx context.Context, matchID string) (*Prediction, error) {
	var p Prediction
	err := s.pool.QueryRow(ctx, `
		SELECT id, match_id, model_version, predictions, confidence,
			no_bet_recommended, data_snapshot_at, is_live, ai_analysis, ai_reasons, ai_abstain, created_at
		FROM predictions WHERE match_id = $1 ORDER BY created_at DESC LIMIT 1
	`, matchID).Scan(&p.ID, &p.MatchID, &p.ModelVersion, &p.Predictions, &p.Confidence,
		&p.NoBetRecommended, &p.DataSnapshotAt, &p.IsLive, &p.AIAnalysis, &p.AIReasons, &p.AIAbstain, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetUpcomingMatchesWithPredictions(ctx context.Context, limit int) ([]Match, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.external_id, m.league_id, l.name,
			ht.id, ht.external_id, ht.name, COALESCE(ht.logo_url,''),
			at.id, at.external_id, at.name, COALESCE(at.logo_url,''),
			m.kickoff_at, m.status::text, m.minute, m.home_score, m.away_score,
			COALESCE(m.venue,''), COALESCE(m.round,'')
		FROM matches m
		JOIN leagues l ON l.id = m.league_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		WHERE m.status IN ('scheduled', 'live')
		ORDER BY m.kickoff_at LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

func (s *Store) SaveCoupon(ctx context.Context, userID *string, name string, selections []CouponSelection) (string, error) {
	var couponID string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO saved_coupons (user_id, name) VALUES ($1, $2) RETURNING id
	`, userID, name).Scan(&couponID)
	if err != nil {
		return "", err
	}
	for _, sel := range selections {
		_, err := s.pool.Exec(ctx, `
			INSERT INTO coupon_selections (coupon_id, match_id, market, selection, confidence)
			VALUES ($1, $2, $3, $4, $5)
		`, couponID, sel.MatchID, sel.Market, sel.Selection, sel.Confidence)
		if err != nil {
			return "", err
		}
	}
	return couponID, nil
}

func (s *Store) GetCoupon(ctx context.Context, id string) (map[string]interface{}, error) {
	var name string
	var createdAt time.Time
	err := s.pool.QueryRow(ctx, `SELECT name, created_at FROM saved_coupons WHERE id = $1`, id).Scan(&name, &createdAt)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT cs.match_id, cs.market, cs.selection, cs.confidence,
			ht.name, at.name
		FROM coupon_selections cs
		JOIN matches m ON m.id = cs.match_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		WHERE cs.coupon_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var selections []map[string]interface{}
	for rows.Next() {
		var matchID, market, selection, home, away string
		var confidence float64
		if err := rows.Scan(&matchID, &market, &selection, &confidence, &home, &away); err != nil {
			return nil, err
		}
		selections = append(selections, map[string]interface{}{
			"match_id": matchID, "market": market, "selection": selection,
			"confidence": confidence, "home_team": home, "away_team": away,
		})
	}
	return map[string]interface{}{
		"id": id, "name": name, "created_at": createdAt, "selections": selections,
	}, rows.Err()
}

func (s *Store) GetModelPerformance(ctx context.Context) ([]ModelPerformance, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT model_name, model_version, market, accuracy, log_loss, brier_score, sample_size
		FROM model_performance ORDER BY calculated_at DESC LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var perf []ModelPerformance
	for rows.Next() {
		var p ModelPerformance
		if err := rows.Scan(&p.ModelName, &p.ModelVersion, &p.Market, &p.Accuracy, &p.LogLoss, &p.BrierScore, &p.SampleSize); err != nil {
			return nil, err
		}
		perf = append(perf, p)
	}
	return perf, rows.Err()
}

func (s *Store) EvaluateFinishedPredictions(ctx context.Context) (int, error) {
	result, err := s.pool.Exec(ctx, `
		INSERT INTO prediction_outcomes (prediction_id, market, predicted_probability, actual_outcome, is_correct)
		SELECT p.id, key, (p.predictions->>key)::decimal,
			CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
				WHEN key = 'draw' THEN (m.home_score = m.away_score)
				WHEN key = 'away_win' THEN (m.home_score < m.away_score)
				WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
				WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
				ELSE false END,
			CASE WHEN (p.predictions->>key)::decimal >= 0.5 THEN
				CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
					WHEN key = 'draw' THEN (m.home_score = m.away_score)
					WHEN key = 'away_win' THEN (m.home_score < m.away_score)
					WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
					WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
					ELSE false END
			ELSE NOT CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
				WHEN key = 'draw' THEN (m.home_score = m.away_score)
				WHEN key = 'away_win' THEN (m.home_score < m.away_score)
				WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
				WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
				ELSE false END END
		FROM predictions p
		JOIN matches m ON m.id = p.match_id
		CROSS JOIN LATERAL jsonb_object_keys(p.predictions) AS key
		WHERE m.status = 'finished' AND m.home_score IS NOT NULL
		AND NOT EXISTS (
			SELECT 1 FROM prediction_outcomes po
			WHERE po.prediction_id = p.id AND po.market = key
		)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

func (s *Store) GetLiveMatches(ctx context.Context) ([]Match, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.id, m.external_id, m.league_id, l.name,
			ht.id, ht.external_id, ht.name, COALESCE(ht.logo_url,''),
			at.id, at.external_id, at.name, COALESCE(at.logo_url,''),
			m.kickoff_at, m.status::text, m.minute, m.home_score, m.away_score,
			COALESCE(m.venue,''), COALESCE(m.round,'')
		FROM matches m
		JOIN leagues l ON l.id = m.league_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		WHERE m.status = 'live'
		ORDER BY m.kickoff_at
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

type scannable interface {
	Scan(dest ...any) error
}

func scanMatch(row scannable) (*Match, error) {
	var m Match
	err := row.Scan(
		&m.ID, &m.ExternalID, &m.LeagueID, &m.LeagueName,
		&m.HomeTeam.ID, &m.HomeTeam.ExternalID, &m.HomeTeam.Name, &m.HomeTeam.LogoURL,
		&m.AwayTeam.ID, &m.AwayTeam.ExternalID, &m.AwayTeam.Name, &m.AwayTeam.LogoURL,
		&m.KickoffAt, &m.Status, &m.Minute, &m.HomeScore, &m.AwayScore,
		&m.Venue, &m.Round,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

type rowScanner interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanMatches(rows rowScanner) ([]Match, error) {
	var matches []Match
	for rows.Next() {
		var m Match
		if err := rows.Scan(
			&m.ID, &m.ExternalID, &m.LeagueID, &m.LeagueName,
			&m.HomeTeam.ID, &m.HomeTeam.ExternalID, &m.HomeTeam.Name, &m.HomeTeam.LogoURL,
			&m.AwayTeam.ID, &m.AwayTeam.ExternalID, &m.AwayTeam.Name, &m.AwayTeam.LogoURL,
			&m.KickoffAt, &m.Status, &m.Minute, &m.HomeScore, &m.AwayScore,
			&m.Venue, &m.Round,
		); err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	return matches, rows.Err()
}
