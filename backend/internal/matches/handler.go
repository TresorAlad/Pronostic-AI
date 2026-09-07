package matches

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	collectorhook "github.com/prono/backend/internal/collector"
	"github.com/prono/backend/internal/db"
	"github.com/prono/backend/internal/leagues"
	oddsutil "github.com/prono/backend/internal/odds"
)

type Handler struct {
	store     *db.Store
	collector *collectorhook.Trigger
}

func NewHandler(store *db.Store, collector *collectorhook.Trigger) *Handler {
	return &Handler{store: store, collector: collector}
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

type matchDayQuery struct {
	date             time.Time
	leagueID         string
	leagueExternalID int
	status           string
}

func parseMatchDayQuery(r *http.Request) (matchDayQuery, error) {
	q := r.URL.Query()
	dateStr := q.Get("date")
	if dateStr == "" {
		dateStr = time.Now().UTC().Format("2006-01-02")
	}
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return matchDayQuery{}, err
	}

	status := q.Get("status")
	if status == "" && q.Get("scheduled_only") == "true" {
		status = "scheduled"
	}
	switch status {
	case "", "scheduled", "live", "finished":
	default:
		status = ""
	}

	out := matchDayQuery{
		date:     parsed,
		leagueID: q.Get("league_id"),
		status:   status,
	}
	if extStr := q.Get("league_external_id"); extStr != "" {
		if ext, err := strconv.Atoi(extStr); err == nil && ext > 0 {
			out.leagueExternalID = ext
		}
	}
	return out, nil
}

func (h *Handler) GetToday(w http.ResponseWriter, r *http.Request) {
	query, err := parseMatchDayQuery(r)
	if err != nil {
		http.Error(w, `{"error":"invalid date"}`, http.StatusBadRequest)
		return
	}

	dateStr := query.date.Format("2006-01-02")
	if h.collector != nil {
		h.collector.RequestDateSync(r.Context(), dateStr)
	}

	opts := db.MatchQueryOpts{Status: query.status}
	if query.leagueID != "" {
		opts.LeagueID = &query.leagueID
	}
	if query.leagueExternalID > 0 {
		opts.LeagueExternalID = &query.leagueExternalID
	}

	matches, err := h.store.GetMatchesByDate(r.Context(), query.date, opts)
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
	leaguesList, err := h.store.ListLeagues(r.Context())
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(leaguesList)
}

func (h *Handler) ListActiveLeagues(w http.ResponseWriter, r *http.Request) {
	dateStr := r.URL.Query().Get("date")
	if dateStr == "" {
		dateStr = time.Now().UTC().Format("2006-01-02")
	}
	parsed, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, `{"error":"invalid date"}`, http.StatusBadRequest)
		return
	}

	options, err := h.store.ListLeagueFilterOptions(r.Context(), parsed)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if options == nil {
		options = []db.LeagueFilterOption{}
	}
	json.NewEncoder(w).Encode(options)
}

type trackedLeagueResponse struct {
	ExternalID int    `json:"external_id"`
	Label      string `json:"label"`
	Priority   int    `json:"priority"`
}

func (h *Handler) ListTrackedLeagues(w http.ResponseWriter, r *http.Request) {
	cfg, err := leagues.Load()
	if err != nil {
		http.Error(w, `{"error":"leagues config unavailable"}`, http.StatusInternalServerError)
		return
	}
	out := make([]trackedLeagueResponse, len(cfg.Leagues))
	for i, l := range cfg.Leagues {
		out[i] = trackedLeagueResponse{
			ExternalID: l.ExternalID,
			Label:      l.Label,
			Priority:   l.Priority,
		}
	}
	json.NewEncoder(w).Encode(out)
}
