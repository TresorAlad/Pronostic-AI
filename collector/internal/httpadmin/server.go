package httpadmin

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/prono/collector/internal/sync"
)

type Server struct {
	svc        *sync.Service
	backendURL string
}

func New(svc *sync.Service, backendURL string) *Server {
	return &Server{svc: svc, backendURL: backendURL}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("POST /sync/today", s.handleSyncToday)
	mux.HandleFunc("POST /sync/date", s.handleSyncDate)
	return mux
}

func (s *Server) handleSyncToday(w http.ResponseWriter, r *http.Request) {
	s.runSync(w, r, time.Now().Format("2006-01-02"))
}

func (s *Server) handleSyncDate(w http.ResponseWriter, r *http.Request) {
	date := r.URL.Query().Get("date")
	if date == "" {
		var body struct {
			Date string `json:"date"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		date = body.Date
	}
	if date == "" {
		http.Error(w, `{"error":"date required"}`, http.StatusBadRequest)
		return
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		http.Error(w, `{"error":"invalid date"}`, http.StatusBadRequest)
		return
	}
	s.runSync(w, r, date)
}

func (s *Server) runSync(w http.ResponseWriter, r *http.Request, date string) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Minute)
	defer cancel()

	if err := s.svc.SyncByDate(ctx, date); err != nil {
		log.Printf("Sync date %s error: %v", date, err)
		http.Error(w, `{"error":"sync failed"}`, http.StatusInternalServerError)
		return
	}
	sync.TriggerPrewarm(ctx, s.backendURL)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "date": date})
}

func (s *Server) RunSyncDate(ctx context.Context, date string) error {
	if err := s.svc.SyncByDate(ctx, date); err != nil {
		return err
	}
	sync.TriggerPrewarm(ctx, s.backendURL)
	return nil
}
