package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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
	Offsides       *int     `json:"offsides"`
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	DisplayName  string    `json:"display_name"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
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
	MatchID        string  `json:"match_id"`
	HomeTeam       string  `json:"home_team"`
	AwayTeam       string  `json:"away_team"`
	LeagueName     string  `json:"league_name,omitempty"`
	Market         string  `json:"market"`
	Selection      string  `json:"selection"`
	MarketCategory string  `json:"market_category,omitempty"`
	MarketLabel    string  `json:"market_label,omitempty"`
	Confidence     float64 `json:"confidence"`
	ValueEdge      float64 `json:"value_edge,omitempty"`
	BookmakerOdd   float64 `json:"bookmaker_odd,omitempty"`
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
		  AND (
		    m.status = 'live'
		    OR (
		      m.status != 'live'
		      AND (
		        m.kickoff_at::date = CURRENT_DATE
		        OR (m.status = 'scheduled' AND m.kickoff_at BETWEEN NOW() AND NOW() + INTERVAL '7 days')
		      )
		    )
		  )
		ORDER BY
			CASE m.status WHEN 'live' THEN 0 WHEN 'scheduled' THEN 1 ELSE 2 END,
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
			ms.corner_kicks, ms.ball_possession, ms.expected_goals, ms.fouls, ms.yellow_cards, ms.offsides
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
			&st.CornerKicks, &st.BallPossession, &st.ExpectedGoals, &st.Fouls, &st.YellowCards, &st.Offsides); err != nil {
			return nil, err
		}
		stats = append(stats, st)
	}
	return stats, rows.Err()
}

type TeamStatsAverages struct {
	MatchCount        int
	StatsMatchCount   int
	ShotsAvg          float64
	ShotsOnTargetAvg  float64
	CornersAvg        float64
	PossessionAvg     float64
	XGAvg             float64
	FoulsAvg          float64
	YellowCardsAvg    float64
	OffsidesAvg       float64
	GoalsAvg          float64
	GoalsConcededAvg  float64
	Form              float64
}

func (s *Store) GetTeamRecentStatsAverages(ctx context.Context, teamID string, before time.Time, limit int) (*TeamStatsAverages, error) {
	if limit <= 0 {
		limit = 5
	}
	row := s.pool.QueryRow(ctx, `
		WITH recent AS (
			SELECT m.id, m.home_team_id, m.away_team_id, m.home_score, m.away_score
			FROM matches m
			WHERE m.status = 'finished'
			  AND m.kickoff_at < $2
			  AND (m.home_team_id = $1 OR m.away_team_id = $1)
			ORDER BY m.kickoff_at DESC
			LIMIT $3
		)
		SELECT
			COUNT(*)::int,
			COUNT(ms.id)::int,
			COALESCE(AVG(ms.total_shots) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(ms.shots_on_goal) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(ms.corner_kicks) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(ms.ball_possession) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(ms.expected_goals) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(ms.fouls) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(ms.yellow_cards) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(ms.offsides) FILTER (WHERE ms.id IS NOT NULL), 0),
			COALESCE(AVG(
				CASE WHEN r.home_team_id = $1 THEN r.home_score::float ELSE r.away_score::float END
			), 0),
			COALESCE(AVG(
				CASE WHEN r.home_team_id = $1 THEN r.away_score::float ELSE r.home_score::float END
			), 0),
			COALESCE(
				AVG(
					CASE
						WHEN r.home_team_id = $1 AND r.home_score > r.away_score THEN 3.0
						WHEN r.away_team_id = $1 AND r.away_score > r.home_score THEN 3.0
						WHEN r.home_score = r.away_score THEN 1.0
						ELSE 0.0
					END
				) / 3.0,
				0
			)
		FROM recent r
		LEFT JOIN match_statistics ms ON ms.match_id = r.id AND ms.team_id = $1
	`, teamID, before, limit)

	var avg TeamStatsAverages
	if err := row.Scan(
		&avg.MatchCount, &avg.StatsMatchCount,
		&avg.ShotsAvg, &avg.ShotsOnTargetAvg, &avg.CornersAvg, &avg.PossessionAvg, &avg.XGAvg,
		&avg.FoulsAvg, &avg.YellowCardsAvg, &avg.OffsidesAvg,
		&avg.GoalsAvg, &avg.GoalsConcededAvg, &avg.Form,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if avg.MatchCount == 0 {
		return nil, nil
	}

	return &avg, nil
}

func (s *Store) GetTeamVenueForm(ctx context.Context, teamID string, venue string, before time.Time, limit int) (float64, error) {
	if limit <= 0 {
		limit = 5
	}
	var venueFilter string
	switch venue {
	case "home":
		venueFilter = "AND m.home_team_id = $1"
	case "away":
		venueFilter = "AND m.away_team_id = $1"
	default:
		return 0, fmt.Errorf("invalid venue: %s", venue)
	}

	query := fmt.Sprintf(`
		WITH recent AS (
			SELECT m.home_team_id, m.away_team_id, m.home_score, m.away_score
			FROM matches m
			WHERE m.status = 'finished'
			  AND m.kickoff_at < $2
			  %s
			ORDER BY m.kickoff_at DESC
			LIMIT $3
		)
		SELECT COALESCE(AVG(pts), 0) / 3.0 FROM (
			SELECT CASE
				WHEN home_team_id = $1 AND home_score > away_score THEN 3.0
				WHEN away_team_id = $1 AND away_score > home_score THEN 3.0
				WHEN home_score = away_score THEN 1.0
				ELSE 0.0
			END AS pts
			FROM recent
		) f
	`, venueFilter)

	var form float64
	err := s.pool.QueryRow(ctx, query, teamID, before, limit).Scan(&form)
	return form, err
}

func (s *Store) GetLeagueAverageGoals(ctx context.Context, leagueID string, before time.Time, limit int) (float64, error) {
	if limit <= 0 {
		limit = 200
	}
	var avg float64
	err := s.pool.QueryRow(ctx, `
		WITH recent AS (
			SELECT m.home_score, m.away_score
			FROM matches m
			WHERE m.league_id = $1
			  AND m.status = 'finished'
			  AND m.kickoff_at < $2
			  AND m.home_score IS NOT NULL
			  AND m.away_score IS NOT NULL
			ORDER BY m.kickoff_at DESC
			LIMIT $3
		)
		SELECT COALESCE(AVG((home_score + away_score)::float / 2.0), 0) FROM recent
	`, leagueID, before, limit).Scan(&avg)
	return avg, err
}

func (s *Store) GetTeamAvailability(ctx context.Context, teamID string) (float64, error) {
	var injuryCount int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM injuries
		WHERE team_id = $1 AND is_active = true
	`, teamID).Scan(&injuryCount)
	if err != nil {
		return 0, err
	}
	score := 1.0 - float64(injuryCount)*0.05
	if score < 0.5 {
		score = 0.5
	}
	return score, nil
}

func (s *Store) CreateUser(ctx context.Context, email, passwordHash, displayName string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, display_name)
		VALUES ($1, $2, $3) RETURNING id, email, display_name, created_at
	`, email, passwordHash, displayName).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	return &u, err
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, COALESCE(display_name,''), created_at
		FROM users WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) GetUserByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		SELECT id, email, COALESCE(display_name,''), created_at
		FROM users WHERE id = $1
	`, id).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) CountCouponsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM saved_coupons WHERE user_id = $1
	`, userID).Scan(&count)
	return count, err
}

func (s *Store) UpdateUserDisplayName(ctx context.Context, userID, displayName string) (*User, error) {
	var u User
	err := s.pool.QueryRow(ctx, `
		UPDATE users SET display_name = $2 WHERE id = $1
		RETURNING id, email, COALESCE(display_name,''), created_at
	`, userID, displayName).Scan(&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

type PublicStats struct {
	MatchesToday      int `json:"matches_today"`
	LiveMatches       int `json:"live_matches"`
	FinishedMatches   int `json:"finished_matches"`
	MatchStatistics   int `json:"match_statistics"`
	Predictions       int `json:"predictions"`
	Outcomes          int `json:"outcomes"`
}

func (s *Store) GetPublicStats(ctx context.Context) (*PublicStats, error) {
	var st PublicStats
	err := s.pool.QueryRow(ctx, `
		SELECT
			(SELECT COUNT(*)::int FROM matches m
			 JOIN leagues l ON l.id = m.league_id
			 WHERE l.external_id IN (39,140,135,78,61) AND m.kickoff_at::date = CURRENT_DATE),
			(SELECT COUNT(*)::int FROM matches WHERE status = 'live'),
			(SELECT COUNT(*)::int FROM matches WHERE status = 'finished'),
			(SELECT COUNT(*)::int FROM match_statistics),
			(SELECT COUNT(*)::int FROM predictions),
			(SELECT COUNT(*)::int FROM prediction_outcomes)
	`).Scan(&st.MatchesToday, &st.LiveMatches, &st.FinishedMatches, &st.MatchStatistics, &st.Predictions, &st.Outcomes)
	if err != nil {
		return nil, err
	}
	return &st, nil
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
	return s.getTop5UpcomingMatches(ctx, limit, true)
}

func (s *Store) GetTop5UpcomingMatches(ctx context.Context, limit int) ([]Match, error) {
	return s.getTop5UpcomingMatches(ctx, limit, false)
}

func (s *Store) GetScheduledMatchesForCoupon(ctx context.Context, minMatches, maxMatches int) ([]Match, error) {
	if minMatches <= 0 {
		minMatches = 10
	}
	if maxMatches <= 0 {
		maxMatches = 30
	}
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
		WHERE m.status = 'scheduled'
		  AND l.external_id IN (39, 140, 135, 78, 61, 3, 848, 40)
		  AND m.kickoff_at BETWEEN CURRENT_DATE AND NOW() + INTERVAL '7 days'
		ORDER BY
			CASE WHEN l.external_id IN (39, 140, 135, 78, 61) THEN 0 ELSE 1 END,
			CASE WHEN m.kickoff_at::date = CURRENT_DATE THEN 0 ELSE 1 END,
			m.kickoff_at
		LIMIT $1
	`, maxMatches)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	matches, err := scanMatches(rows)
	if err != nil {
		return nil, err
	}
	if len(matches) >= minMatches {
		return matches, nil
	}
	return matches, nil
}

func (s *Store) GetMatchesTodayScheduled(ctx context.Context) ([]Match, error) {
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
		  AND m.status = 'scheduled'
		  AND (
		    m.kickoff_at::date = CURRENT_DATE
		    OR (m.kickoff_at BETWEEN NOW() AND NOW() + INTERVAL '7 days')
		  )
		ORDER BY m.kickoff_at
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMatches(rows)
}

func (s *Store) getTop5UpcomingMatches(ctx context.Context, limit int, includeLive bool) ([]Match, error) {
	statusFilter := "m.status = 'scheduled'"
	if includeLive {
		statusFilter = "m.status IN ('scheduled', 'live')"
	}

	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
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
		  AND %s
		  AND m.kickoff_at BETWEEN NOW() - INTERVAL '1 day' AND NOW() + INTERVAL '7 days'
		ORDER BY m.kickoff_at
		LIMIT $1
	`, statusFilter), limit)
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

func (s *Store) ListCouponsByUser(ctx context.Context, userID string) ([]map[string]interface{}, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, created_at,
			(SELECT COUNT(*) FROM coupon_selections cs WHERE cs.coupon_id = saved_coupons.id)
		FROM saved_coupons
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT 50
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var coupons []map[string]interface{}
	for rows.Next() {
		var id, name string
		var createdAt time.Time
		var selectionCount int
		if err := rows.Scan(&id, &name, &createdAt, &selectionCount); err != nil {
			return nil, err
		}
		coupons = append(coupons, map[string]interface{}{
			"id":              id,
			"name":            name,
			"created_at":      createdAt,
			"selection_count": selectionCount,
		})
	}
	return coupons, rows.Err()
}

func (s *Store) GetCouponForUser(ctx context.Context, couponID, userID string) (map[string]interface{}, error) {
	var ownerID *string
	err := s.pool.QueryRow(ctx, `SELECT user_id FROM saved_coupons WHERE id = $1`, couponID).Scan(&ownerID)
	if err != nil || ownerID == nil || *ownerID != userID {
		return nil, fmt.Errorf("not found")
	}
	return s.GetCoupon(ctx, couponID)
}

type PerformanceTrend struct {
	Period   string  `json:"period"`
	Market   string  `json:"market"`
	Accuracy float64 `json:"accuracy"`
	Samples  int     `json:"sample_size"`
}

func (s *Store) GetPerformanceTrend(ctx context.Context) ([]PerformanceTrend, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(date_trunc('week', m.kickoff_at), 'YYYY-MM-DD') AS period,
			po.market,
			AVG(CASE WHEN po.is_correct THEN 1.0 ELSE 0.0 END) AS accuracy,
			COUNT(*)::int AS samples
		FROM prediction_outcomes po
		JOIN predictions p ON p.id = po.prediction_id
		JOIN matches m ON m.id = p.match_id
		GROUP BY 1, po.market
		ORDER BY 1 DESC, po.market
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trend []PerformanceTrend
	for rows.Next() {
		var t PerformanceTrend
		if err := rows.Scan(&t.Period, &t.Market, &t.Accuracy, &t.Samples); err != nil {
			return nil, err
		}
		trend = append(trend, t)
	}
	return trend, rows.Err()
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
	scoreCount, err := s.evaluateScoreMarkets(ctx)
	if err != nil {
		return 0, err
	}
	statsCount, err := s.evaluateStatsMarkets(ctx)
	if err != nil {
		return scoreCount, err
	}
	return scoreCount + statsCount, nil
}

func (s *Store) evaluateScoreMarkets(ctx context.Context) (int, error) {
	result, err := s.pool.Exec(ctx, `
		INSERT INTO prediction_outcomes (prediction_id, market, predicted_probability, actual_outcome, is_correct)
		SELECT p.id, key, (p.predictions->>key)::decimal,
			CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
				WHEN key = 'draw' THEN (m.home_score = m.away_score)
				WHEN key = 'away_win' THEN (m.home_score < m.away_score)
				WHEN key = 'over_1_5' THEN ((m.home_score + m.away_score) > 1)
				WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
				WHEN key = 'over_3_5' THEN ((m.home_score + m.away_score) > 3)
				WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
				WHEN key = 'btts_no' THEN NOT (m.home_score > 0 AND m.away_score > 0)
				ELSE false END,
			CASE WHEN (p.predictions->>key)::decimal >= 0.5 THEN
				CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
					WHEN key = 'draw' THEN (m.home_score = m.away_score)
					WHEN key = 'away_win' THEN (m.home_score < m.away_score)
					WHEN key = 'over_1_5' THEN ((m.home_score + m.away_score) > 1)
					WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
					WHEN key = 'over_3_5' THEN ((m.home_score + m.away_score) > 3)
					WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
					WHEN key = 'btts_no' THEN NOT (m.home_score > 0 AND m.away_score > 0)
					ELSE false END
			ELSE NOT CASE WHEN key = 'home_win' THEN (m.home_score > m.away_score)
				WHEN key = 'draw' THEN (m.home_score = m.away_score)
				WHEN key = 'away_win' THEN (m.home_score < m.away_score)
				WHEN key = 'over_1_5' THEN ((m.home_score + m.away_score) > 1)
				WHEN key = 'over_2_5' THEN ((m.home_score + m.away_score) > 2)
				WHEN key = 'over_3_5' THEN ((m.home_score + m.away_score) > 3)
				WHEN key = 'btts' THEN (m.home_score > 0 AND m.away_score > 0)
				WHEN key = 'btts_no' THEN NOT (m.home_score > 0 AND m.away_score > 0)
				ELSE false END END
		FROM predictions p
		JOIN matches m ON m.id = p.match_id
		CROSS JOIN LATERAL jsonb_object_keys(p.predictions) AS key
		WHERE m.status = 'finished' AND m.home_score IS NOT NULL
		AND key IN ('home_win','draw','away_win','over_1_5','over_2_5','over_3_5','btts','btts_no')
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

func (s *Store) evaluateStatsMarkets(ctx context.Context) (int, error) {
	result, err := s.pool.Exec(ctx, `
		INSERT INTO prediction_outcomes (prediction_id, market, predicted_probability, actual_outcome, is_correct)
		SELECT p.id, market.key,
			(p.predictions->>market.key)::decimal,
			market.actual,
			CASE WHEN (p.predictions->>market.key)::decimal >= 0.5 THEN market.actual ELSE NOT market.actual END
		FROM predictions p
		JOIN matches m ON m.id = p.match_id
		JOIN (
			SELECT match_id,
				COALESCE(SUM(corner_kicks), 0) AS total_corners,
				COALESCE(SUM(total_shots), 0) AS total_shots,
				COALESCE(SUM(shots_on_goal), 0) AS total_shots_on_target
			FROM match_statistics
			GROUP BY match_id
		) ms ON ms.match_id = m.id
		CROSS JOIN LATERAL (
			SELECT 'over_corners_9_5' AS key, (ms.total_corners > 9.5) AS actual
			UNION ALL SELECT 'over_corners_9.5', (ms.total_corners > 9.5)
			UNION ALL SELECT 'over_shots_22_5', (ms.total_shots > 22.5)
			UNION ALL SELECT 'over_shots_on_target_8_5', (ms.total_shots_on_target > 8.5)
		) market
		WHERE m.status = 'finished'
		AND p.predictions ? market.key
		AND NOT EXISTS (
			SELECT 1 FROM prediction_outcomes po
			WHERE po.prediction_id = p.id AND po.market = market.key
		)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

func (s *Store) RefreshModelPerformance(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM model_performance;
		INSERT INTO model_performance (model_name, model_version, market, accuracy, log_loss, brier_score, sample_size, period_start, period_end)
		SELECT
			split_part(p.model_version, '-', 1),
			p.model_version,
			po.market,
			AVG(CASE WHEN po.is_correct THEN 1.0 ELSE 0.0 END),
			AVG(
				CASE WHEN po.actual_outcome THEN
					-ln(GREATEST(po.predicted_probability::float8, 1e-15))
				ELSE
					-ln(GREATEST(1.0 - po.predicted_probability::float8, 1e-15))
				END
			),
			AVG(POWER(po.predicted_probability::float8 - CASE WHEN po.actual_outcome THEN 1.0 ELSE 0.0 END, 2)),
			COUNT(*)::int,
			MIN(m.kickoff_at::date),
			MAX(m.kickoff_at::date)
		FROM prediction_outcomes po
		JOIN predictions p ON p.id = po.prediction_id
		JOIN matches m ON m.id = p.match_id
		GROUP BY p.model_version, po.market
	`)
	return err
}

type EvaluationOutcome struct {
	PredictionID         string  `json:"prediction_id"`
	Market               string  `json:"market"`
	PredictedProbability float64 `json:"predicted_probability"`
	ActualOutcome        bool    `json:"actual_outcome"`
	IsCorrect            bool    `json:"is_correct"`
	ModelVersion         string  `json:"model_version"`
	EvaluatedAt          string  `json:"evaluated_at"`
}

func (s *Store) GetEvaluationOutcomes(ctx context.Context, limit int) ([]EvaluationOutcome, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT po.prediction_id::text, po.market, po.predicted_probability, po.actual_outcome,
			po.is_correct, p.model_version, po.evaluated_at::text
		FROM prediction_outcomes po
		JOIN predictions p ON p.id = po.prediction_id
		ORDER BY po.evaluated_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []EvaluationOutcome
	for rows.Next() {
		var o EvaluationOutcome
		if err := rows.Scan(&o.PredictionID, &o.Market, &o.PredictedProbability, &o.ActualOutcome, &o.IsCorrect, &o.ModelVersion, &o.EvaluatedAt); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
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
