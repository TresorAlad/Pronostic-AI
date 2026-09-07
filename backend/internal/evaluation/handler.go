package evaluation

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/prono/backend/internal/db"
)

type evaluationStore interface {
	EvaluateFinishedPredictions(ctx context.Context) (int, error)
	EvaluateUserCoupons(ctx context.Context) (int, error)
	RefreshModelPerformance(ctx context.Context) error
	GetEvaluationOutcomes(ctx context.Context, limit int) ([]db.EvaluationOutcome, error)
	NotifyRecentCouponEvaluations(ctx context.Context) error
}

type Handler struct {
	store evaluationStore
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RunEvaluation(w http.ResponseWriter, r *http.Request) {
	count, err := h.store.EvaluateFinishedPredictions(r.Context())
	if err != nil {
		http.Error(w, `{"error":"evaluation failed"}`, http.StatusInternalServerError)
		return
	}
	couponCount, _ := h.store.EvaluateUserCoupons(r.Context())
	_ = h.store.NotifyRecentCouponEvaluations(r.Context())
	if err := h.store.RefreshModelPerformance(r.Context()); err != nil {
		http.Error(w, `{"error":"metrics refresh failed"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"evaluated":        count,
		"coupon_evaluated": couponCount,
		"message":          "Evaluation post-match terminee",
	})
}

func (h *Handler) ListOutcomes(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	outcomes, err := h.store.GetEvaluationOutcomes(r.Context(), limit)
	if err != nil {
		http.Error(w, `{"error":"outcomes unavailable"}`, http.StatusInternalServerError)
		return
	}
	if outcomes == nil {
		outcomes = []db.EvaluationOutcome{}
	}
	json.NewEncoder(w).Encode(outcomes)
}
