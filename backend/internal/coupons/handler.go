package coupons

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/prono/backend/internal/auth"
	"github.com/prono/backend/internal/db"
	"github.com/prono/backend/internal/odds"
	"github.com/prono/backend/internal/predictions"
)

type Handler struct {
	store       *db.Store
	predictions *predictions.Handler
}

type candidate struct {
	MatchID       string
	HomeTeam      string
	AwayTeam      string
	LeagueName    string
	Market        string
	Selection     string
	Confidence    float64
	BookmakerOdd  float64
	AvgOdd        float64
	BookmakerName string
}

func NewHandler(store *db.Store, predHandler *predictions.Handler) *Handler {
	return &Handler{store: store, predictions: predHandler}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/generate", h.Generate)
	r.Get("/mine", h.ListMine)
	r.Get("/mine/{id}", h.GetMineByID)
	r.Get("/mine/{id}/export", h.ExportMine)
	r.Get("/{id}", h.GetByID)
	return r
}

func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}

	var req struct {
		MinConfidence  float64 `json:"min_confidence"`
		MaxSelections  int     `json:"max_selections"`
		MinCombinedOdd float64 `json:"min_combined_odd"`
		MaxCombinedOdd float64 `json:"max_combined_odd"`
		Name           string  `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.MinConfidence == 0 {
		req.MinConfidence = 0.55
	}
	if req.MaxSelections == 0 {
		req.MaxSelections = 10
	}
	if req.MinCombinedOdd == 0 {
		req.MinCombinedOdd = 5
	}
	if req.MaxCombinedOdd == 0 {
		req.MaxCombinedOdd = 50
	}
	if req.Name == "" {
		req.Name = "Coupon IA"
	}

	if req.MinCombinedOdd < 5 || req.MinCombinedOdd > 50 ||
		req.MaxCombinedOdd < 5 || req.MaxCombinedOdd > 50 ||
		req.MaxCombinedOdd < req.MinCombinedOdd {
		http.Error(w, `{"error":"invalid combined odd range (5-50)"}`, http.StatusBadRequest)
		return
	}
	if req.MaxSelections < 1 || req.MaxSelections > 10 {
		http.Error(w, `{"error":"max_selections must be between 1 and 10"}`, http.StatusBadRequest)
		return
	}

	matches, err := h.store.GetScheduledMatchesForCoupon(r.Context(), 0)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
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
		if pred == nil {
			continue
		}

		var conf map[string]float64
		var preds map[string]float64
		json.Unmarshal(pred.Confidence, &conf)
		json.Unmarshal(pred.Predictions, &preds)

		pool := buildCandidatePool(conf, preds, m.ID, m.HomeTeam.Name, m.AwayTeam.Name)
		for j := range pool {
			pool[j].LeagueName = m.LeagueName
			if matchOdds, err := h.store.GetMatchOdds(r.Context(), m.ID); err == nil {
				summary := odds.OddsSummaryForMarket(matchOdds, pool[j].Market)
				pool[j].BookmakerOdd = summary.Best
				if summary.BestBookmaker != "" {
					pool[j].BookmakerName = summary.BestBookmaker
				}
				pool[j].AvgOdd = summary.Average
			}
		}
		candidates = append(candidates, pool...)
	}

	picked := pickForOddsTarget(candidates, req.MinCombinedOdd, req.MaxCombinedOdd, req.MaxSelections, req.MinConfidence)
	combinedOdd := CombinedOddProductFromCandidates(picked)

	selections := make([]db.CouponSelection, 0, len(picked))
	for _, c := range picked {
		sel := db.CouponSelection{
			MatchID: c.MatchID, HomeTeam: c.HomeTeam, AwayTeam: c.AwayTeam,
			LeagueName: c.LeagueName,
			Market: c.Market, Selection: c.Selection, Confidence: c.Confidence,
			MarketCategory: marketCategory(c.Market),
			MarketLabel:    marketLabel(c.Market),
			BookmakerOdd:   c.BookmakerOdd,
			AvgOdd:         c.AvgOdd,
			BookmakerName:  c.BookmakerName,
		}
		if matchOdds, err := h.store.GetMatchOdds(r.Context(), c.MatchID); err == nil {
			sel.ValueEdge = odds.ValueEdgeForMarket(matchOdds, c.Market, c.Confidence)
			if sel.BookmakerOdd <= 1 {
				summary := odds.OddsSummaryForMarket(matchOdds, c.Market)
				sel.BookmakerOdd = summary.Best
				sel.AvgOdd = summary.Average
				sel.BookmakerName = summary.BestBookmaker
			}
			if history, err := h.store.GetMatchOddsHistory(r.Context(), c.MatchID, 200); err == nil {
				trend := odds.OddTrendForMarket(history, c.Market)
				sel.OddTrend = trend.Direction
			}
		}
		selections = append(selections, sel)
	}

	couponID, err := h.store.SaveCoupon(r.Context(), &userID, req.Name, combinedOdd, selections)
	if err != nil {
		http.Error(w, `{"error":"save error"}`, http.StatusInternalServerError)
		return
	}
	_ = h.store.CreateNotification(r.Context(), userID, "coupon", "Coupon généré",
		fmt.Sprintf("Votre coupon « %s » contient %d sélection(s).", req.Name, len(selections)))

	warning := ""
	if !CombinedOddInRange(combinedOdd, req.MinCombinedOdd, req.MaxCombinedOdd) {
		warning = fmt.Sprintf("Cote combinée hors intervalle demandé : %.2f (objectif %.0f-%.0f).",
			combinedOdd, req.MinCombinedOdd, req.MaxCombinedOdd)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": couponID, "name": req.Name, "selections": selections,
		"combined_odd":      combinedOdd,
		"min_combined_odd":  req.MinCombinedOdd,
		"max_combined_odd":  req.MaxCombinedOdd,
		"disclaimer":        "Estimations statistiques, aucune garantie de gain.",
		"warning":           warning,
	})
}

func (h *Handler) ListMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}
	coupons, err := h.store.ListCouponsByUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	if coupons == nil {
		coupons = []map[string]interface{}{}
	}
	json.NewEncoder(w).Encode(coupons)
}

func (h *Handler) GetMineByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := auth.UserIDFromContext(r.Context())
	if !ok {
		http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
		return
	}
	id := chi.URLParam(r, "id")
	coupon, err := h.store.GetCouponForUser(r.Context(), id, userID)
	if err != nil {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(coupon)
}

func isCouponMarket(market string) bool {
	if strings.HasPrefix(market, "predicted_") {
		return false
	}
	return strings.HasPrefix(market, "over_") ||
		strings.HasPrefix(market, "under_") ||
		strings.HasPrefix(market, "double_chance_") ||
		strings.HasPrefix(market, "team_over_") ||
		strings.HasPrefix(market, "result_btts_") ||
		strings.HasPrefix(market, "draw_no_bet_") ||
		strings.HasPrefix(market, "corner_winner_") ||
		market == "home_win" || market == "draw" || market == "away_win" ||
		market == "btts" || market == "btts_no" ||
		market == "over_0_5_ht" ||
		strings.HasPrefix(market, "home_possession_") || strings.HasPrefix(market, "away_possession_")
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

func (h *Handler) ExportMine(w http.ResponseWriter, r *http.Request) {
	http.Error(w, `{"error":"use client PDF export"}`, http.StatusGone)
}
