package evaluation

import (
	"encoding/json"
	"net/http"

	"github.com/prono/backend/internal/db"
)

type Handler struct {
	store *db.Store
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
	if err := h.store.RefreshModelPerformance(r.Context()); err != nil {
		http.Error(w, `{"error":"metrics refresh failed"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"evaluated": count,
		"message":   "Evaluation post-match terminee",
	})
}
