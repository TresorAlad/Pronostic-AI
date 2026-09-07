package matches

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prono/backend/internal/db"
	oddsutil "github.com/prono/backend/internal/odds"
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
	r.Get("/{id}/odds", h.GetOdds)
	return r
}

func (h *Handler) GetToday(w http.ResponseWriter, r *http.Request) {
	var matches []db.Match
	var err error
	if r.URL.Query().Get("scheduled_only") == "true" {
		matches, err = h.store.GetMatchesTodayScheduled(r.Context())
	} else {
		matches, err = h.store.GetMatchesToday(r.Context())
	}
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

func (h *Handler) GetOdds(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	odds, err := h.store.GetMatchOdds(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if odds == nil {
		odds = []db.MatchOdd{}
	}
	pred, _ := h.store.GetLatestPrediction(r.Context(), id)
	type oddRow struct {
		db.MatchOdd
		ImpliedProbability float64 `json:"implied_probability,omitempty"`
		MLProbability      float64 `json:"ml_probability,omitempty"`
		ValueEdge          float64 `json:"value_edge,omitempty"`
	}
	var rows []oddRow
	var preds map[string]float64
	if pred != nil {
		_ = json.Unmarshal(pred.Predictions, &preds)
	}
	for _, o := range odds {
		row := oddRow{MatchOdd: o}
		if o.Odd > 1 {
			row.ImpliedProbability = 1 / o.Odd
		}
		if ml := oddsutil.MapToMLKey(o.Market, o.Selection); ml != "" {
			if p, ok := preds[ml]; ok {
				row.MLProbability = p
				if row.ImpliedProbability > 0 {
					row.ValueEdge = p - row.ImpliedProbability
				}
			}
		}
		rows = append(rows, row)
	}
	json.NewEncoder(w).Encode(rows)
}

func (h *Handler) ListLeagues(w http.ResponseWriter, r *http.Request) {
	leagues, err := h.store.ListLeagues(r.Context())
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(leagues)
}
