package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prono/backend/internal/auth"
	"github.com/prono/backend/internal/clients"
	"github.com/prono/backend/internal/config"
	"github.com/prono/backend/internal/coupons"
	"github.com/prono/backend/internal/db"
	"github.com/prono/backend/internal/evaluation"
	"github.com/prono/backend/internal/matches"
	"github.com/prono/backend/internal/predictions"
	"github.com/prono/backend/internal/ws"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	ctx := context.Background()
	store, err := db.NewStore(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("DB error: %v", err)
	}
	defer store.Close()

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Redis parse error: %v", err)
	}
	redisClient := redis.NewClient(opt)

	mlClient := clients.NewMLClient(cfg.MLServiceURL)
	aiClient := clients.NewAIAgentClient(cfg.AIAgentURL)

	authSvc := auth.NewService(store, cfg.JWTSecret, cfg.JWTExpiration)
	matchHandler := matches.NewHandler(store)
	predHandler := predictions.NewHandler(store, mlClient, aiClient, redisClient)
	couponHandler := coupons.NewHandler(store, predHandler)
	evalHandler := evaluation.NewHandler(store)

	hub := ws.NewHub()
	go hub.Run()

	liveSvc := ws.NewLiveService(hub, store, predHandler, redisClient)
	go liveSvc.StartPolling(ctx, 30*time.Second)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authSvc.Middleware)

		r.Post("/auth/register", authSvc.Register)
		r.Post("/auth/login", authSvc.Login)

		r.Get("/leagues", matchHandler.ListLeagues)
		r.Mount("/matches", matchHandler.Routes())
		r.Mount("/", predHandler.Routes())
		r.Mount("/coupons", couponHandler.Routes())
		r.Post("/evaluation/run", evalHandler.RunEvaluation)

		r.Get("/ws/live", liveSvc.HandleWebSocket)
	})

	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	go func() {
		log.Printf("Backend API listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}
