package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type Notification struct {
	ID        string     `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Body      string     `json:"body"`
	ReadAt    *time.Time `json:"read_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type UserPerformanceSummary struct {
	TotalSelections int                `json:"total_selections"`
	CorrectCount    int                `json:"correct_count"`
	Accuracy        float64            `json:"accuracy"`
	ByMarket        map[string]float64 `json:"by_market"`
}

type UserPerformanceTrend struct {
	Period   string  `json:"period"`
	Accuracy float64 `json:"accuracy"`
	Sample   int     `json:"sample_size"`
}

type MatchOdd struct {
	Bookmaker string  `json:"bookmaker"`
	Market    string  `json:"market"`
	Selection string  `json:"selection"`
	Odd       float64 `json:"odd"`
}

func (s *Store) CreateNotification(ctx context.Context, userID, nType, title, body string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body)
		VALUES ($1, $2, $3, $4)
	`, userID, nType, title, body)
	return err
}

func (s *Store) ListNotifications(ctx context.Context, userID string, limit int) ([]Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, type, title, body, read_at, created_at
		FROM notifications WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.Type, &n.Title, &n.Body, &n.ReadAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, n)
	}
	return items, rows.Err()
}

func (s *Store) MarkNotificationRead(ctx context.Context, userID, notificationID string) error {
	_, err := s.pool.Exec(ctx, `
		UPDATE notifications SET read_at = NOW()
		WHERE id = $1 AND user_id = $2 AND read_at IS NULL
	`, notificationID, userID)
	return err
}

func (s *Store) EvaluateUserCoupons(ctx context.Context) (int, error) {
	result, err := s.pool.Exec(ctx, `
		INSERT INTO coupon_selection_outcomes
			(saved_coupon_id, match_id, market, selection, predicted_probability, actual_outcome, is_correct)
		SELECT cs.coupon_id, cs.match_id, cs.market, cs.selection,
			po.predicted_probability, po.actual_outcome, po.is_correct
		FROM coupon_selections cs
		JOIN matches m ON m.id = cs.match_id AND m.status = 'finished'
		JOIN LATERAL (
			SELECT p.id FROM predictions p
			WHERE p.match_id = cs.match_id
			ORDER BY p.created_at DESC LIMIT 1
		) lp ON true
		JOIN prediction_outcomes po ON po.prediction_id = lp.id AND po.market = cs.market
		WHERE NOT EXISTS (
			SELECT 1 FROM coupon_selection_outcomes cso
			WHERE cso.saved_coupon_id = cs.coupon_id
			  AND cso.match_id = cs.match_id AND cso.market = cs.market
		)
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

func (s *Store) GetUserPerformanceSummary(ctx context.Context, userID string) (*UserPerformanceSummary, error) {
	var total, correct int
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int,
			COALESCE(SUM(CASE WHEN cso.is_correct THEN 1 ELSE 0 END), 0)::int
		FROM coupon_selection_outcomes cso
		JOIN saved_coupons sc ON sc.id = cso.saved_coupon_id
		WHERE sc.user_id = $1
	`, userID).Scan(&total, &correct)
	if err != nil {
		return nil, err
	}
	summary := &UserPerformanceSummary{
		TotalSelections: total,
		CorrectCount:    correct,
		ByMarket:        map[string]float64{},
	}
	if total > 0 {
		summary.Accuracy = float64(correct) / float64(total)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT cso.market,
			AVG(CASE WHEN cso.is_correct THEN 1.0 ELSE 0.0 END)
		FROM coupon_selection_outcomes cso
		JOIN saved_coupons sc ON sc.id = cso.saved_coupon_id
		WHERE sc.user_id = $1
		GROUP BY cso.market
	`, userID)
	if err != nil {
		return summary, nil
	}
	defer rows.Close()
	for rows.Next() {
		var market string
		var acc float64
		if err := rows.Scan(&market, &acc); err != nil {
			return summary, err
		}
		summary.ByMarket[market] = acc
	}
	return summary, rows.Err()
}

func (s *Store) GetUserPerformanceTrend(ctx context.Context, userID string) ([]UserPerformanceTrend, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT to_char(date_trunc('week', cso.evaluated_at), 'YYYY-MM-DD') AS period,
			AVG(CASE WHEN cso.is_correct THEN 1.0 ELSE 0.0 END),
			COUNT(*)::int
		FROM coupon_selection_outcomes cso
		JOIN saved_coupons sc ON sc.id = cso.saved_coupon_id
		WHERE sc.user_id = $1
		GROUP BY 1
		ORDER BY 1 DESC
		LIMIT 12
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var trend []UserPerformanceTrend
	for rows.Next() {
		var t UserPerformanceTrend
		if err := rows.Scan(&t.Period, &t.Accuracy, &t.Sample); err != nil {
			return nil, err
		}
		trend = append(trend, t)
	}
	return trend, rows.Err()
}

func (s *Store) GetUserRecentOutcomes(ctx context.Context, userID string, limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT cso.market, cso.selection, cso.is_correct, cso.evaluated_at,
			ht.name, at.name
		FROM coupon_selection_outcomes cso
		JOIN saved_coupons sc ON sc.id = cso.saved_coupon_id
		JOIN matches m ON m.id = cso.match_id
		JOIN teams ht ON ht.id = m.home_team_id
		JOIN teams at ON at.id = m.away_team_id
		WHERE sc.user_id = $1
		ORDER BY cso.evaluated_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []map[string]interface{}
	for rows.Next() {
		var market, selection, home, away string
		var isCorrect bool
		var evaluatedAt time.Time
		if err := rows.Scan(&market, &selection, &isCorrect, &evaluatedAt, &home, &away); err != nil {
			return nil, err
		}
		items = append(items, map[string]interface{}{
			"market": market, "selection": selection, "is_correct": isCorrect,
			"evaluated_at": evaluatedAt, "home_team": home, "away_team": away,
		})
	}
	return items, rows.Err()
}

func (s *Store) UpsertMatchOdds(ctx context.Context, matchID, bookmaker, market, selection string, odd float64) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO match_odds (match_id, bookmaker, market, selection, odd, fetched_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (match_id, bookmaker, market, selection)
		DO UPDATE SET odd = EXCLUDED.odd, fetched_at = NOW()
	`, matchID, bookmaker, market, selection, odd)
	return err
}

func (s *Store) GetMatchOdds(ctx context.Context, matchID string) ([]MatchOdd, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT bookmaker, market, selection, odd::float8
		FROM match_odds WHERE match_id = $1
		ORDER BY bookmaker, market, selection
	`, matchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var odds []MatchOdd
	for rows.Next() {
		var o MatchOdd
		if err := rows.Scan(&o.Bookmaker, &o.Market, &o.Selection, &o.Odd); err != nil {
			return nil, err
		}
		odds = append(odds, o)
	}
	return odds, rows.Err()
}

func (s *Store) GetUpcomingMatchIDsForOdds(ctx context.Context, limit int) ([]struct {
	ID         string
	ExternalID int
}, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := s.pool.Query(ctx, `
		SELECT m.id::text, m.external_id
		FROM matches m
		JOIN leagues l ON l.id = m.league_id
		WHERE l.external_id IN (39, 140, 135, 78, 61)
		  AND m.status IN ('scheduled', 'live')
		  AND m.kickoff_at BETWEEN NOW() - INTERVAL '1 day' AND NOW() + INTERVAL '3 days'
		ORDER BY m.kickoff_at
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []struct {
		ID         string
		ExternalID int
	}
	for rows.Next() {
		var row struct {
			ID         string
			ExternalID int
		}
		if err := rows.Scan(&row.ID, &row.ExternalID); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (s *Store) NotifyRecentCouponEvaluations(ctx context.Context) error {
	rows, err := s.pool.Query(ctx, `
		SELECT sc.user_id::text, sc.name,
			COUNT(*)::int,
			COALESCE(SUM(CASE WHEN cso.is_correct THEN 1 ELSE 0 END), 0)::int
		FROM coupon_selection_outcomes cso
		JOIN saved_coupons sc ON sc.id = cso.saved_coupon_id
		WHERE cso.evaluated_at > NOW() - INTERVAL '5 minutes'
		GROUP BY sc.user_id, sc.name
	`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, name string
		var total, correct int
		if err := rows.Scan(&userID, &name, &total, &correct); err != nil {
			return err
		}
		body := fmt.Sprintf("Coupon « %s » : %d/%d sélection(s) correcte(s) après les matchs terminés.", name, correct, total)
		if err := s.CreateNotification(ctx, userID, "eval", "Résultats de votre coupon", body); err != nil {
			return err
		}
	}
	return rows.Err()
}

func CouponToExportJSON(coupon map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(coupon, "", "  ")
}
