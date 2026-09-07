package coupons

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/go-chi/chi/v5"
	"github.com/prono/backend/internal/db"
	"github.com/prono/backend/internal/predictions"
)

type Handler struct {
	store       *db.Store
	predictions *predictions.Handler
}

func NewHandler(store *db.Store, predHandler *predictions.Handler) *Handler {
	return &Handler{store: store, predictions: predHandler}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/generate", h.Generate)
	r.Get("/{id}", h.GetByID)
	return r
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MinConfidence float64 `json:"min_confidence"`
		MaxSelections int     `json:"max_selections"`
		Name          string  `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.MinConfidence == 0 {
		req.MinConfidence = 0.65
	}
	if req.MaxSelections == 0 {
		req.MaxSelections = 5
	}
	if req.Name == "" {
		req.Name = "Coupon IA"
	}

	matches, err := h.store.GetUpcomingMatchesWithPredictions(r.Context(), 20)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}

	type candidate struct {
		MatchID    string
		HomeTeam   string
		AwayTeam   string
		Market     string
		Selection  string
		Confidence float64
	}

	var candidates []candidate

	for _, m := range matches {
		pred, err := h.predictions.RunLivePrediction(r.Context(), m.ID)
		if err != nil {
			continue
		}
		if pred.NoBetRecommended {
			continue
		}

		var conf map[string]float64
		json.Unmarshal(pred.Confidence, &conf)

		for market, confidence := range conf {
			if confidence >= req.MinConfidence {
				candidates = append(candidates, candidate{
					MatchID: m.ID, HomeTeam: m.HomeTeam.Name, AwayTeam: m.AwayTeam.Name,
					Market: market, Selection: market, Confidence: confidence,
				})
			}
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	if len(candidates) > req.MaxSelections {
		candidates = candidates[:req.MaxSelections]
	}

	var selections []db.CouponSelection
	for _, c := range candidates {
		selections = append(selections, db.CouponSelection{
			MatchID: c.MatchID, HomeTeam: c.HomeTeam, AwayTeam: c.AwayTeam,
			Market: c.Market, Selection: c.Selection, Confidence: c.Confidence,
		})
	}

	couponID, err := h.store.SaveCoupon(r.Context(), nil, req.Name, selections)
	if err != nil {
		http.Error(w, `{"error":"save error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": couponID, "name": req.Name, "selections": selections,
		"disclaimer": "Estimations statistiques, aucune garantie de gain.",
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	coupon, err := h.store.GetCoupon(r.Context(), id)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(coupon)
}
