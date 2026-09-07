package performance

import (
	"encoding/json"
	"net/http"

	"github.com/prono/backend/internal/auth"
	"github.com/prono/backend/internal/db"
)

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}
	summary, err := h.store.GetUserPerformanceSummary(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	trend, _ := h.store.GetUserPerformanceTrend(r.Context(), userID)
	recent, _ := h.store.GetUserRecentOutcomes(r.Context(), userID, 20)
	if trend == nil {
		trend = []db.UserPerformanceTrend{}
	}
	if recent == nil {
		recent = []map[string]interface{}{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"summary": summary,
		"trend":   trend,
		"recent":  recent,
	})
}
