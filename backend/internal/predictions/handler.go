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
	store           *db.Store
	mlClient        *clients.MLClient
	aiClient        *clients.AIAgentClient
	redis           *redis.Client
	historyEnricher *HistoryEnricher
}

func NewHandler(store *db.Store, ml *clients.MLClient, ai *clients.AIAgentClient, redis *redis.Client, enricher *HistoryEnricher) *Handler {
	return &Handler{store: store, mlClient: ml, aiClient: ai, redis: redis, historyEnricher: enricher}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/performance", h.GetPerformance)
	r.Get("/performance/trend", h.GetPerformanceTrend)
	r.Post("/prewarm", h.PreWarm)
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

func (h *Handler) GetPerformanceTrend(w http.ResponseWriter, r *http.Request) {
	trend, err := h.store.GetPerformanceTrend(r.Context())
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if trend == nil {
		trend = []db.PerformanceTrend{}
	}
	json.NewEncoder(w).Encode(trend)
}

func (h *Handler) PreWarm(w http.ResponseWriter, r *http.Request) {
	n, err := h.PreWarmUpcoming(r.Context(), 15)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"warmed": n})
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

	features, err := h.buildFeatures(ctx, match)
	if err != nil {
		return nil, err
	}

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
		matchOdds, _ := h.store.GetMatchOdds(ctx, matchID)
		aiPayload := map[string]interface{}{
			"match_id":           matchID,
			"home_team":          match.HomeTeam.Name,
			"away_team":          match.AwayTeam.Name,
			"home_team_id":       match.HomeTeam.ID,
			"away_team_id":       match.AwayTeam.ID,
			"predictions":        mlResult.Predictions,
			"confidence":         mlResult.Confidence,
			"features":           features,
			"no_bet_recommended": mlResult.NoBetRecommended,
			"odds":               buildOddsForAI(matchOdds, mlResult.Predictions),
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

func (h *Handler) RunLivePrediction(ctx context.Context, matchID string) (*db.Prediction, error) {
	return h.runPrediction(ctx, matchID, false)
}

func (h *Handler) PredictForCoupon(ctx context.Context, matchID string) (*db.Prediction, error) {
	return h.GetOrCreatePrediction(ctx, matchID)
}

func (h *Handler) InvalidatePredictionCache(ctx context.Context, matchID string) {
	h.redis.Del(ctx, fmt.Sprintf("prediction:%s", matchID))
}

func (h *Handler) PreWarmUpcoming(ctx context.Context, limit int) (int, error) {
	matches, err := h.store.GetTop5UpcomingMatches(ctx, limit)
	if err != nil {
		return 0, err
	}
	warmed := 0
	for _, m := range matches {
		if _, err := h.GetOrCreatePrediction(ctx, m.ID); err == nil {
			warmed++
		}
	}
	return warmed, nil
}

func (h *Handler) GetOrCreatePrediction(ctx context.Context, matchID string) (*db.Prediction, error) {
	cacheKey := fmt.Sprintf("prediction:%s", matchID)
	cached, err := h.redis.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var pred db.Prediction
		if json.Unmarshal([]byte(cached), &pred) == nil {
			return &pred, nil
		}
	}

	if pred, err := h.store.GetLatestPrediction(ctx, matchID); err == nil {
		return pred, nil
	}

	return h.runPrediction(ctx, matchID, false)
}
