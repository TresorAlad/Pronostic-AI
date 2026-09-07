package predictions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prono/backend/internal/clients"
	"github.com/prono/backend/internal/db"
	"github.com/redis/go-redis/v9"
)

type Handler struct {
	store    *db.Store
	mlClient *clients.MLClient
	aiClient *clients.AIAgentClient
	redis    *redis.Client
}

func NewHandler(store *db.Store, ml *clients.MLClient, ai *clients.AIAgentClient, redis *redis.Client) *Handler {
	return &Handler{store: store, mlClient: ml, aiClient: ai, redis: redis}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/performance", h.GetPerformance)
	r.Get("/matches/{id}/prediction", h.GetPrediction)
	r.Post("/matches/{id}/analyze", h.AnalyzeMatch)
	return r
}

func (h *Handler) GetPerformance(w http.ResponseWriter, r *http.Request) {
	perf, err := h.store.GetModelPerformance(r.Context())
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if perf == nil {
		perf = []db.ModelPerformance{}
	}
	json.NewEncoder(w).Encode(perf)
}

func (h *Handler) GetPrediction(w http.ResponseWriter, r *http.Request) {
	matchID := chi.URLParam(r, "id")

	cached, err := h.redis.Get(r.Context(), fmt.Sprintf("prediction:%s", matchID)).Result()
	if err == nil && cached != "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cached))
		return
	}

	pred, err := h.store.GetLatestPrediction(r.Context(), matchID)
	if err == nil {
		json.NewEncoder(w).Encode(pred)
		return
	}

	result, err := h.runPrediction(r.Context(), matchID, false)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) AnalyzeMatch(w http.ResponseWriter, r *http.Request) {
	matchID := chi.URLParam(r, "id")
	result, err := h.runPrediction(r.Context(), matchID, true)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func (h *Handler) runPrediction(ctx context.Context, matchID string, withAI bool) (*db.Prediction, error) {
	match, err := h.store.GetMatchByID(ctx, matchID)
	if err != nil {
		return nil, fmt.Errorf("match not found")
	}

	features := h.buildFeatures(ctx, match)

	isLive := match.Status == "live"
	mlResult, err := h.mlClient.PredictMatch(matchID, features, isLive)
	if err != nil {
		return nil, fmt.Errorf("ml prediction failed: %w", err)
	}

	predJSON, _ := json.Marshal(mlResult.Predictions)
	confJSON, _ := json.Marshal(mlResult.Confidence)
	snapshotAt, _ := time.Parse(time.RFC3339, mlResult.DataSnapshotAt)
	if snapshotAt.IsZero() {
		snapshotAt = time.Now()
	}

	pred := &db.Prediction{
		MatchID:          matchID,
		ModelVersion:     mlResult.ModelVersion,
		Predictions:      predJSON,
		Confidence:       confJSON,
		NoBetRecommended: mlResult.NoBetRecommended,
		DataSnapshotAt:   snapshotAt,
		IsLive:           isLive,
	}

	if withAI {
		aiPayload := map[string]interface{}{
			"match_id":   matchID,
			"home_team":  match.HomeTeam.Name,
			"away_team":  match.AwayTeam.Name,
			"predictions": mlResult.Predictions,
			"confidence":  mlResult.Confidence,
			"features":    features,
			"no_bet_recommended": mlResult.NoBetRecommended,
		}
		aiResult, err := h.aiClient.AnalyzeMatch(aiPayload)
		if err == nil {
			pred.AIAnalysis = &aiResult.Analysis
			reasonsJSON, _ := json.Marshal(aiResult.Reasons)
			pred.AIReasons = reasonsJSON
			pred.AIAbstain = aiResult.Abstain
			if aiResult.Abstain {
				pred.NoBetRecommended = true
			}
		}
	}

	if err := h.store.SavePrediction(ctx, pred); err != nil {
		return nil, fmt.Errorf("save prediction: %w", err)
	}

	cached, _ := json.Marshal(pred)
	h.redis.Set(ctx, fmt.Sprintf("prediction:%s", matchID), cached, time.Hour)

	return pred, nil
}

func (h *Handler) buildFeatures(ctx context.Context, match *db.Match) map[string]interface{} {
	features := map[string]interface{}{
		"home_attack_strength": 1.0,
		"away_attack_strength": 1.0,
		"home_defense_strength": 1.0,
		"away_defense_strength": 1.0,
		"home_form": 0.5,
		"away_form": 0.5,
		"home_goals_avg_5": 1.3,
		"away_goals_avg_5": 1.1,
		"home_advantage": 0.15,
		"home_availability": 1.0,
		"away_availability": 1.0,
	}

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
	}

	stats, err := h.store.GetMatchStats(ctx, match.ID)
	if err == nil && len(stats) >= 2 {
		for _, s := range stats {
			prefix := "home"
			if s.TeamID == match.AwayTeam.ID {
				prefix = "away"
			}
			if s.TotalShots != nil {
				features[prefix+"_shots_avg_5"] = float64(*s.TotalShots)
			}
			if s.ShotsOnGoal != nil {
				features[prefix+"_shots_on_target_avg_5"] = float64(*s.ShotsOnGoal)
			}
			if s.CornerKicks != nil {
				features[prefix+"_corners_avg_5"] = float64(*s.CornerKicks)
			}
			if s.BallPossession != nil {
				features[prefix+"_possession_avg_5"] = *s.BallPossession
			}
			if s.ExpectedGoals != nil {
				features[prefix+"_xg_avg_5"] = *s.ExpectedGoals
			}
		}
	}

	return features
}

func (h *Handler) RunLivePrediction(ctx context.Context, matchID string) (*db.Prediction, error) {
	return h.runPrediction(ctx, matchID, false)
}
