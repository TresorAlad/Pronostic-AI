package coupons

import (
	"encoding/csv"
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
	MatchID    string
	HomeTeam   string
	AwayTeam   string
	Market     string
	Selection  string
	Confidence float64
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
		MinConfidence float64 `json:"min_confidence"`
		MaxSelections int     `json:"max_selections"`
		Name          string  `json:"name"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.MinConfidence == 0 {
		req.MinConfidence = 0.55
	}
	if req.MaxSelections == 0 {
		req.MaxSelections = 8
	}
	if req.Name == "" {
		req.Name = "Coupon IA"
	}

	matches, err := h.store.GetTop5UpcomingMatches(r.Context(), 15)
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
		candidates = append(candidates, pool...)
	}

	picked := pickTipsterSelections(candidates, req.MaxSelections, req.MinConfidence)

	selections := make([]db.CouponSelection, 0, len(picked))
	for _, c := range picked {
		sel := db.CouponSelection{
			MatchID: c.MatchID, HomeTeam: c.HomeTeam, AwayTeam: c.AwayTeam,
			Market: c.Market, Selection: c.Selection, Confidence: c.Confidence,
			MarketCategory: marketCategory(c.Market),
			MarketLabel:    marketLabel(c.Market),
		}
		if matchOdds, err := h.store.GetMatchOdds(r.Context(), c.MatchID); err == nil {
			sel.ValueEdge = odds.ValueEdgeForMarket(matchOdds, c.Market, c.Confidence)
		}
		selections = append(selections, sel)
	}

	couponID, err := h.store.SaveCoupon(r.Context(), &userID, req.Name, selections)
	if err != nil {
		http.Error(w, `{"error":"save error"}`, http.StatusInternalServerError)
		return
	}
	_ = h.store.CreateNotification(r.Context(), userID, "coupon", "Coupon généré",
		fmt.Sprintf("Votre coupon « %s » contient %d sélection(s).", req.Name, len(selections)))

	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": couponID, "name": req.Name, "selections": selections,
		"disclaimer": "Estimations statistiques, aucune garantie de gain.",
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
		market == "home_win" || market == "draw" || market == "away_win" ||
		market == "btts" || market == "btts_no" ||
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
	format := strings.ToLower(r.URL.Query().Get("format"))
	if format == "" {
		format = "json"
	}
	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"coupon-%s.csv\"", id))
		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"home_team", "away_team", "market", "selection", "confidence"})
		selections, _ := coupon["selections"].([]map[string]interface{})
		for _, sel := range selections {
			_ = cw.Write([]string{
				fmt.Sprint(sel["home_team"]), fmt.Sprint(sel["away_team"]),
				fmt.Sprint(sel["market"]), fmt.Sprint(sel["selection"]),
				fmt.Sprint(sel["confidence"]),
			})
		}
		cw.Flush()
	default:
		data, err := db.CouponToExportJSON(coupon)
		if err != nil {
			http.Error(w, `{"error":"export failed"}`, http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"coupon-%s.json\"", id))
		w.Write(data)
	}
}
