package stats

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

func (h *Handler) Public(w http.ResponseWriter, r *http.Request) {
	st, err := h.store.GetPublicStats(r.Context())
	if err != nil {
		http.Error(w, `{"error":"stats unavailable"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}
