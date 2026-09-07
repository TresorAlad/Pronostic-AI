package coupons

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"

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
		req.MinConfidence = 0.55
	}
	if req.MaxSelections == 0 {
		req.MaxSelections = 5
	}
	if req.Name == "" {
		req.Name = "Coupon IA"
	}

	matches, err := h.store.GetTop5UpcomingMatches(r.Context(), 15)
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

	predictionsByMatch := make([]*db.Prediction, len(matches))
	var wg sync.WaitGroup
	for i, m := range matches {
		wg.Add(1)
		go func(idx int, matchID string) {
			defer wg.Done()
			pred, err := h.predictions.GetOrCreatePrediction(r.Context(), matchID)
			if err == nil {
				predictionsByMatch[idx] = pred
			}
		}(i, m.ID)
	}
	wg.Wait()

	var candidates []candidate
	for i, m := range matches {
		pred := predictionsByMatch[i]
		if pred == nil || pred.NoBetRecommended {
			continue
		}

		var conf map[string]float64
		json.Unmarshal(pred.Confidence, &conf)

		bestMarket := ""
		bestConf := 0.0
		for market, confidence := range conf {
			if !isCouponMarket(market) {
				continue
			}
			if confidence >= req.MinConfidence && confidence > bestConf {
				bestMarket = market
				bestConf = confidence
			}
		}
		if bestMarket != "" {
			candidates = append(candidates, candidate{
				MatchID: m.ID, HomeTeam: m.HomeTeam.Name, AwayTeam: m.AwayTeam.Name,
				Market: bestMarket, Selection: bestMarket, Confidence: bestConf,
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Confidence > candidates[j].Confidence
	})

	if len(candidates) > req.MaxSelections {
		candidates = candidates[:req.MaxSelections]
	}

	var selections = make([]db.CouponSelection, 0)
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

func isCouponMarket(market string) bool {
	if strings.HasPrefix(market, "predicted_") || strings.HasPrefix(market, "double_chance_") {
		return false
	}
	return true
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
