package sync

import (
	"context"
	"log"
	"time"

	"github.com/prono/collector/internal/leagues"
	"github.com/redis/go-redis/v9"
	"github.com/robfig/cron/v3"
)

func RunTodaySyncJob(ctx context.Context, svc *Service, backendURL string) {
	date := time.Now().Format("2006-01-02")
	if err := svc.SyncByDate(ctx, date); err != nil {
		log.Printf("Today sync error (%s): %v", date, err)
		return
	}
	TriggerPrewarm(ctx, backendURL)
}

func maybeSyncNewDay(ctx context.Context, svc *Service, backendURL string, redisClient *redis.Client) {
	if redisClient == nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	last, err := redisClient.Get(ctx, "collector:last_sync_date").Result()
	if err == nil && last == today {
		return
	}
	log.Printf("New calendar day detected (%s), syncing fixtures...", today)
	RunTodaySyncJob(ctx, svc, backendURL)
	_ = redisClient.Set(ctx, "collector:last_sync_date", today, 48*time.Hour).Err()
}

func RegisterDaemonJobs(c *cron.Cron, ctx context.Context, svc *Service, backendURL string, syncInterval, liveInterval time.Duration, redisClient *redis.Client) error {
	RunTodaySyncJob(ctx, svc, backendURL)
	if redisClient != nil {
		today := time.Now().Format("2006-01-02")
		_ = redisClient.Set(ctx, "collector:last_sync_date", today, 48*time.Hour).Err()
	}

	if _, err := c.AddFunc("5 0 * * *", func() {
		log.Println("Daily midnight sync (new day)...")
		RunTodaySyncJob(ctx, svc, backendURL)
	}); err != nil {
		return err
	}

	syncSpec := "@every " + syncInterval.String()
	if _, err := c.AddFunc(syncSpec, func() {
		log.Println("Running scheduled sync...")
		RunTodaySyncJob(ctx, svc, backendURL)
		seasonYear := time.Now().Year()
		leagueCfg, err := leagues.Load()
		if err != nil {
			log.Printf("Leagues config error: %v", err)
			return
		}
		for _, leagueID := range leagueCfg.TrackedExternalIDs() {
			svc.SyncInjuries(ctx, leagueID, seasonYear)
		}
		svc.SyncMatchDetails(ctx, 200)
	}); err != nil {
		return err
	}

	liveSpec := "@every " + liveInterval.String()
	if _, err := c.AddFunc(liveSpec, func() {
		if err := svc.SyncLive(ctx); err != nil {
			log.Printf("Live sync error: %v", err)
		} else {
			TriggerEvaluation(ctx, backendURL)
		}
		maybeSyncNewDay(ctx, svc, backendURL, redisClient)
	}); err != nil {
		return err
	}

	if _, err := c.AddFunc("@every 6h", func() {
		log.Println("Scheduled odds sync (coupon pool)...")
		if err := svc.SyncOdds(ctx, 200); err != nil {
			log.Printf("Odds sync error: %v", err)
		}
	}); err != nil {
		return err
	}

	return nil
}
