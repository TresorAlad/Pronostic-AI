package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/prono/collector/internal/apifootball"
	"github.com/prono/collector/internal/config"
	"github.com/prono/collector/internal/db"
	"github.com/prono/collector/internal/sync"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

func main() {
	mode := flag.String("mode", "daemon", "Mode: daemon, backfill, sync-today, sync-live")
	startYear := flag.Int("start-year", 2018, "Backfill start year")
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
		log.Println("Done")

	case "sync-today":
		if err := svc.SyncToday(ctx); err != nil {
			log.Fatalf("Sync today error: %v", err)
		}
		log.Println("Today's fixtures synced")

	case "sync-live":
		if err := svc.SyncLive(ctx); err != nil {
			log.Fatalf("Sync live error: %v", err)
		}
		log.Println("Live fixtures synced")

	default:
		runDaemon(ctx, cfg, svc)
	}
}

func runDaemon(ctx context.Context, cfg *config.Config, svc *sync.Service) {
	c := cron.New()

	c.AddFunc("@every 6h", func() {
		log.Println("Running scheduled sync...")
		if err := svc.SyncToday(ctx); err != nil {
			log.Printf("Scheduled sync error: %v", err)
		}
		for _, leagueID := range config.Top5LeagueIDs {
			year := 2025
			svc.SyncInjuries(ctx, leagueID, year)
		}
		svc.SyncMatchDetails(ctx, 100)
	})

	c.AddFunc("@every 1m", func() {
		if err := svc.SyncLive(ctx); err != nil {
			log.Printf("Live sync error: %v", err)
		}
	})

	c.Start()
	log.Println("Collector daemon started")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	log.Println("Shutting down collector...")
	c.Stop()
}
