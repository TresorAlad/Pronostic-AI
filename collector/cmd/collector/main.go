package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prono/collector/internal/apifootball"
	"github.com/prono/collector/internal/config"
	"github.com/prono/collector/internal/db"
	"github.com/prono/collector/internal/httpadmin"
	"github.com/prono/collector/internal/sync"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

func main() {
	mode := flag.String("mode", "daemon", "Mode: daemon, backfill, sync-today, sync-live, sync-stats, sync-stats-all, sync-odds")
	date := flag.String("date", "", "Date YYYY-MM-DD for sync-today (default: today)")
	startYear := flag.Int("start-year", 2018, "Backfill start year")
	statsLimit := flag.Int("limit", 500, "Max matches for sync-stats mode")
	statsBatches := flag.Int("batches", 10, "Batches for sync-stats-all mode")
	statsPause := flag.Int("pause", 45, "Pause seconds between sync-stats-all batches")
	flag.Parse()

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
	defer redisClient.Close()

	apiClient := apifootball.NewClient(cfg.APIFootballBase, cfg.APIFootballHost, cfg.APIFootballKey, cfg.APIFootballAuth)
	svc := sync.NewService(apiClient, store, redisClient, cfg.LiveScope, cfg.LiveMaxFixtures)

	switch *mode {
	case "backfill":
		log.Printf("Starting backfill from %d", *startYear)
		if err := svc.Backfill(ctx, *startYear); err != nil {
			log.Fatalf("Backfill error: %v", err)
		}
		log.Println("Backfill complete, syncing match details...")
		svc.SyncMatchDetails(ctx, 500)
		sync.TriggerNeo4jSync(ctx, cfg.AIAgentURL)
		log.Println("Done. Graphe Neo4j synchronise si ai-agent disponible.")

	case "sync-today":
		syncDate := *date
		if syncDate == "" {
			syncDate = time.Now().Format("2006-01-02")
		}
		if err := svc.SyncByDate(ctx, syncDate); err != nil {
			log.Fatalf("Sync today error: %v", err)
		}
		log.Printf("Fixtures synced for %s", syncDate)
		sync.TriggerPrewarm(ctx, cfg.BackendURL)

	case "sync-live":
		if err := svc.SyncLive(ctx); err != nil {
			log.Fatalf("Sync live error: %v", err)
		}
		log.Println("Live fixtures synced")
		sync.TriggerEvaluation(ctx, cfg.BackendURL)

	case "sync-stats":
		if err := svc.SyncMatchDetails(ctx, *statsLimit); err != nil {
			log.Fatalf("Sync stats error: %v", err)
		}
		log.Printf("Match statistics synced (limit=%d)", *statsLimit)

	case "sync-stats-all":
		for i := 1; i <= *statsBatches; i++ {
			log.Printf("Sync stats batch %d/%d (limit=%d)", i, *statsBatches, *statsLimit)
			if err := svc.SyncMatchDetails(ctx, *statsLimit); err != nil {
				log.Fatalf("Sync stats error: %v", err)
			}
			if i < *statsBatches {
				log.Printf("Pause %ds (quota API)...", *statsPause)
				time.Sleep(time.Duration(*statsPause) * time.Second)
			}
		}
		log.Printf("Match statistics synced (%d batches)", *statsBatches)

	case "sync-odds":
		if err := svc.SyncOdds(ctx, *statsLimit); err != nil {
			log.Fatalf("Sync odds error: %v", err)
		}
		log.Printf("Odds synced (limit=%d)", *statsLimit)

	default:
		runDaemon(ctx, cfg, svc, redisClient)
	}
}

func runDaemon(ctx context.Context, cfg *config.Config, svc *sync.Service, redisClient *redis.Client) {
	admin := httpadmin.New(svc, cfg.BackendURL)
	httpSrv := &http.Server{
		Addr:    ":" + cfg.HTTPPort,
		Handler: admin.Handler(),
	}
	go func() {
		log.Printf("Collector HTTP listening on :%s", cfg.HTTPPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Collector HTTP error: %v", err)
		}
	}()

	c := cron.New()
	if err := sync.RegisterDaemonJobs(c, ctx, svc, cfg.BackendURL, cfg.SyncInterval, cfg.LiveInterval, redisClient); err != nil {
		log.Fatalf("Invalid cron spec: %v", err)
	}
	c.Start()
	log.Println("Collector daemon started (auto sync-today on start, daily at 00:05, periodic)")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down collector...")
	c.Stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	httpSrv.Shutdown(shutdownCtx)
}
