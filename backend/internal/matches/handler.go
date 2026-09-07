package matches

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prono/backend/internal/db"
)

type Handler struct {
	store *db.Store
}

func NewHandler(store *db.Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/today", h.GetToday)
	r.Get("/live", h.GetLive)
	r.Get("/{id}", h.GetByID)
	r.Get("/{id}/stats", h.GetStats)
	return r
}

func (h *Handler) GetToday(w http.ResponseWriter, r *http.Request) {
	matches, err := h.store.GetMatchesToday(r.Context())
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if matches == nil {
		matches = []db.Match{}
	}
	json.NewEncoder(w).Encode(matches)
}

func (h *Handler) GetLive(w http.ResponseWriter, r *http.Request) {
	matches, err := h.store.GetLiveMatches(r.Context())
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if matches == nil {
		matches = []db.Match{}
	}
	json.NewEncoder(w).Encode(matches)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	match, err := h.store.GetMatchByID(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(match)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stats, err := h.store.GetMatchStats(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if stats == nil {
		stats = []db.MatchStats{}
	}
	json.NewEncoder(w).Encode(stats)
}

func (h *Handler) ListLeagues(w http.ResponseWriter, r *http.Request) {
	leagues, err := h.store.ListLeagues(r.Context())
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(leagues)
}
